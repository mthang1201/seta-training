package integration

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/seta-training/core/internal/config"
	deliveryHttp "github.com/seta-training/core/internal/delivery/http"
	"github.com/seta-training/core/internal/domain"
	"github.com/seta-training/core/internal/infrastructure"
	"github.com/seta-training/core/internal/repository"
	"github.com/seta-training/core/internal/usecase"
	"github.com/seta-training/core/internal/worker"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/modules/rabbitmq"
	"github.com/testcontainers/testcontainers-go/modules/redis"
	"github.com/testcontainers/testcontainers-go/wait"
	"gorm.io/gorm"
	amqp "github.com/rabbitmq/amqp091-go"
)

type TestServer struct {
	Router        *gin.Engine
	DB            *gorm.DB
	Config        *config.Config
	pgContainer   *postgres.PostgresContainer
	redisContainer *redis.RedisContainer
	rmqContainer  *rabbitmq.RabbitMQContainer
	amqpConn      *amqp.Connection
	RedisCache    domain.Cache
	UserRepo      domain.UserRepository
	TeamRepo      domain.TeamRepository
	AssetRepo     domain.AssetRepository
	ConsumerCancel context.CancelFunc
}

var ts *TestServer

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)

	// Setup logging for tests to not clutter stdout too much unless there's an error
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	slog.SetDefault(logger)

	ctx := context.Background()

	var err error
	ts, err = setupTestServer(ctx)
	if err != nil {
		slog.Error("Failed to setup test server", "error", err)
		os.Exit(1)
	}

	code := m.Run()

	ts.teardown(ctx)
	os.Exit(code)
}

func setupTestServer(ctx context.Context) (*TestServer, error) {
	ts := &TestServer{
		Config: &config.Config{
			JWTSecret: "test-secret",
		},
	}

	// 1. Setup Postgres Container
	pgContainer, err := postgres.Run(ctx,
		"postgres:15-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).WithStartupTimeout(5*time.Second),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to start postgres container: %w", err)
	}
	ts.pgContainer = pgContainer

	dbHost, _ := pgContainer.Host(ctx)
	dbPort, _ := pgContainer.MappedPort(ctx, "5432")
	ts.Config.DBHost = dbHost
	ts.Config.DBPort = dbPort.Port()
	ts.Config.DBName = "testdb"
	ts.Config.DBUser = "testuser"
	ts.Config.DBPassword = "testpass"

	// 2. Setup Redis Container
	redisContainer, err := redis.Run(ctx,
		"redis:7-alpine",
	)
	if err != nil {
		return nil, fmt.Errorf("failed to start redis container: %w", err)
	}
	ts.redisContainer = redisContainer

	redisHost, _ := redisContainer.Host(ctx)
	redisPort, _ := redisContainer.MappedPort(ctx, "6379")
	ts.Config.RedisURL = fmt.Sprintf("redis://%s:%s/0", redisHost, redisPort.Port())

	// 3. Setup RabbitMQ Container
	rmqContainer, err := rabbitmq.Run(ctx,
		"rabbitmq:3.13-management-alpine",
	)
	if err != nil {
		return nil, fmt.Errorf("failed to start rabbitmq container: %w", err)
	}
	ts.rmqContainer = rmqContainer

	rmqHost, _ := rmqContainer.Host(ctx)
	rmqPort, _ := rmqContainer.MappedPort(ctx, "5672")
	ts.Config.RabbitMQURL = fmt.Sprintf("amqp://guest:guest@%s:%s/", rmqHost, rmqPort.Port())

	// 4. Initialize Database
	ts.DB = infrastructure.NewPostgresDB(ts.Config)

	// 5. Initialize Repositories
	ts.UserRepo = repository.NewUserRepository(ts.DB)
	ts.TeamRepo = repository.NewTeamRepository(ts.DB)
	ts.AssetRepo = repository.NewAssetRepository(ts.DB)

	// 6. Initialize Redis
	ts.RedisCache, err = infrastructure.NewRedisCache(ts.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to init redis cache: %w", err)
	}

	// 7. Initialize RabbitMQ
	publisher, amqpConn, err := infrastructure.NewRabbitMQPublisher(ts.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to init rabbitmq publisher: %w", err)
	}
	ts.amqpConn = amqpConn

	consumerCtx, consumerCancel := context.WithCancel(context.Background())
	ts.ConsumerCancel = consumerCancel
	eventConsumer := worker.NewEventConsumer(amqpConn)
	if err := eventConsumer.Start(consumerCtx); err != nil {
		return nil, fmt.Errorf("failed to start event consumer: %w", err)
	}

	// 8. Initialize UseCases
	userUseCase := usecase.NewUserUseCase(ts.UserRepo, ts.Config)
	teamUseCase := usecase.NewTeamUseCase(ts.TeamRepo, ts.UserRepo, ts.RedisCache, publisher)
	assetUseCase := usecase.NewAssetUseCase(ts.AssetRepo, ts.TeamRepo, ts.RedisCache, publisher)

	// 9. Setup Gin Router
	ts.Router = gin.Default()
	ts.Router.Use(deliveryHttp.RequestIDMiddleware())
	ts.Router.Use(deliveryHttp.MetricsMiddleware())

	api := ts.Router.Group("/api/v1")
	deliveryHttp.NewUserHandler(api, userUseCase, ts.Config)
	deliveryHttp.NewTeamHandler(api, teamUseCase, ts.Config)
	deliveryHttp.NewAssetHandler(api, assetUseCase, ts.Config)

	return ts, nil
}

func (ts *TestServer) teardown(ctx context.Context) {
	ts.ConsumerCancel()
	if ts.amqpConn != nil {
		ts.amqpConn.Close()
	}
	if ts.RedisCache != nil {
		ts.RedisCache.Close()
	}
	if ts.pgContainer != nil {
		ts.pgContainer.Terminate(ctx)
	}
	if ts.redisContainer != nil {
		ts.redisContainer.Terminate(ctx)
	}
	if ts.rmqContainer != nil {
		ts.rmqContainer.Terminate(ctx)
	}
}

// Helper: Clear tables between tests
func (ts *TestServer) clearTables() {
	ts.DB.Exec("TRUNCATE TABLE users, teams, folders, notes, asset_permissions, team_members, team_managers CASCADE")
}

// Helper: Clear cache between tests
func (ts *TestServer) clearCache(ctx context.Context) {
}

// Helper: Perform HTTP request
func (ts *TestServer) PerformRequest(method, path string, body []byte, token string) *httptest.ResponseRecorder {
	req, _ := http.NewRequest(method, path, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	ts.Router.ServeHTTP(w, req)
	return w
}

// Factory: Create User
func (ts *TestServer) CreateTestUser(ctx context.Context, email, password string, role domain.Role) *domain.User {
	req := &domain.RegisterRequest{
		Email:    email,
		Password: password,
		Role:     role,
		Username: "test_" + email,
	}
	
	// Use UseCase to ensure proper hashing
	useCase := usecase.NewUserUseCase(ts.UserRepo, ts.Config)
	user, err := useCase.Register(ctx, req)
	if err != nil {
		slog.Error("Failed to create test user", "error", err)
	}
	return user
}

// Factory: Generate JWT
func (ts *TestServer) GenerateTestToken(userID uint, role domain.Role) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"userId": float64(userID),
		"role":   string(role),
		"exp":    time.Now().Add(time.Hour * 24).Unix(),
	})

	tokenString, _ := token.SignedString([]byte(ts.Config.JWTSecret))
	return tokenString
}
