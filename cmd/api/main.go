package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	_ "github.com/seta-training/core/docs"
	"github.com/seta-training/core/internal/config"
	deliveryHttp "github.com/seta-training/core/internal/delivery/http"
	"github.com/seta-training/core/internal/infrastructure"
	"github.com/seta-training/core/internal/repository"
	"github.com/seta-training/core/internal/usecase"
	"github.com/seta-training/core/internal/worker"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @title Seta Training API
// @version 1.0
// @description This is a sample server for a microservices challenge.
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:3000
// @BasePath /api/v1

// @securityDefinitions.apikey Bearer
// @in header
// @name Authorization


func main() {
	// 0. Init Logger
	infrastructure.InitLogger()
	slog.Info("Starting application...")

	// 1. Load config
	cfg, err := config.LoadConfig()
	if err != nil {
		slog.Error("Failed to load config", "error", err)
		log.Fatalf("Failed to load config: %v", err)
	}

	// 2. Init DB
	db := infrastructure.NewPostgresDB(cfg)

	// 3. Init Repositories
	userRepo := repository.NewUserRepository(db)
	teamRepo := repository.NewTeamRepository(db)
	assetRepo := repository.NewAssetRepository(db)

	// 4. Init Redis Cache
	redisCache, err := infrastructure.NewRedisCache(cfg)
	if err != nil {
		slog.Error("Failed to init Redis", "error", err)
		log.Fatalf("Failed to init Redis: %v", err)
	}
	defer redisCache.Close()

	// 5. Init RabbitMQ Publisher & Consumer
	publisher, amqpConn, err := infrastructure.NewRabbitMQPublisher(cfg)
	if err != nil {
		slog.Error("Failed to init RabbitMQ Publisher", "error", err)
		log.Fatalf("Failed to init RabbitMQ: %v", err)
	}
	defer amqpConn.Close()

	consumerCtx, consumerCancel := context.WithCancel(context.Background())
	defer consumerCancel()
	
	eventConsumer := worker.NewEventConsumer(amqpConn)
	if err := eventConsumer.Start(consumerCtx); err != nil {
		slog.Error("Failed to start event consumer", "error", err)
	}

	// 6. Init UseCases
	userUseCase := usecase.NewUserUseCase(userRepo, cfg)
	teamUseCase := usecase.NewTeamUseCase(teamRepo, userRepo, redisCache, publisher)
	assetUseCase := usecase.NewAssetUseCase(assetRepo, teamRepo, redisCache, publisher)

	// 7. Init Gin App
	app := gin.Default()
	
	// Add RequestID & Metrics Middleware
	app.Use(deliveryHttp.RequestIDMiddleware())
	app.Use(deliveryHttp.MetricsMiddleware())

	// Setup Prometheus Endpoint
	app.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// 8. Setup Routes
	api := app.Group("/api/v1")
	deliveryHttp.NewUserHandler(api, userUseCase, cfg)
	deliveryHttp.NewTeamHandler(api, teamUseCase, cfg)
	deliveryHttp.NewAssetHandler(api, assetUseCase, cfg)

	// Swagger route
	app.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// 9. Start server with Graceful Shutdown
	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: app,
	}

	go func() {
		slog.Info("Starting server", "port", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Server failed", "error", err)
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	slog.Info("Shutting down server...")

	// Cancel consumer context
	consumerCancel()

	// The context is used to inform the server it has 5 seconds to finish
	// the request it is currently handling
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("Server forced to shutdown", "error", err)
		log.Fatal("Server forced to shutdown: ", err)
	}

	slog.Info("Server exiting")
}
