package usecase

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/seta-training/core/internal/config"
	"github.com/seta-training/core/internal/domain"
	"golang.org/x/crypto/bcrypt"
)

type userUseCase struct {
	userRepo domain.UserRepository
	cfg      *config.Config
}

func NewUserUseCase(userRepo domain.UserRepository, cfg *config.Config) domain.UserUseCase {
	return &userUseCase{
		userRepo: userRepo,
		cfg:      cfg,
	}
}

func (u *userUseCase) Register(ctx context.Context, req *domain.RegisterRequest) (*domain.User, error) {
	// Check if user exists
	existingUser, err := u.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		return nil, err
	}
	if existingUser != nil {
		return nil, errors.New("email already in use")
	}

	// Validate role
	if req.Role != domain.RoleManager && req.Role != domain.RoleMember {
		return nil, errors.New("invalid role")
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &domain.User{
		Username: req.Username,
		Email:    req.Email,
		Password: string(hashedPassword),
		Role:     req.Role,
	}

	if err := u.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

func (u *userUseCase) Login(ctx context.Context, req *domain.LoginRequest) (string, error) {
	user, err := u.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		return "", err
	}
	if user == nil {
		return "", errors.New("invalid email or password")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return "", errors.New("invalid email or password")
	}

	// Generate JWT
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"userId": user.ID,
		"role":   user.Role,
		"exp":    time.Now().Add(time.Hour * 72).Unix(),
	})

	tokenString, err := token.SignedString([]byte(u.cfg.JWTSecret))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func (u *userUseCase) GetUsers(ctx context.Context) ([]*domain.User, error) {
	return u.userRepo.GetAll(ctx)
}

func (u *userUseCase) ImportUsers(ctx context.Context, csvData io.Reader) (*domain.ImportResult, error) {
	reader := csv.NewReader(csvData)
	
	// Read header
	_, err := reader.Read()
	if err != nil {
		return nil, errors.New("failed to read csv header")
	}

	type job struct {
		line int
		row  []string
	}
	type result struct {
		line int
		err  error
	}

	jobs := make(chan job, 100)
	results := make(chan result, 100)

	var wg sync.WaitGroup
	workerCount := 5

	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobs {
				if len(j.row) < 4 {
					results <- result{line: j.line, err: errors.New("invalid row length")}
					continue
				}

				username := strings.TrimSpace(j.row[0])
				email := strings.TrimSpace(j.row[1])
				password := strings.TrimSpace(j.row[2])
				role := domain.Role(strings.TrimSpace(j.row[3]))

				req := &domain.RegisterRequest{
					Username: username,
					Email:    email,
					Password: password,
					Role:     role,
				}

				// Using independent context so context cancellation doesn't break background inserts
				_, err := u.Register(context.Background(), req) 
				results <- result{line: j.line, err: err}
			}
		}()
	}

	// Read and dispatch jobs
	go func() {
		line := 2 // 1-based, header was line 1
		for {
			row, err := reader.Read()
			if err == io.EOF {
				break
			}
			if err != nil {
				results <- result{line: line, err: err}
				line++
				continue
			}
			jobs <- job{line: line, row: row}
			line++
		}
		close(jobs)
		wg.Wait()
		close(results)
	}()

	var summary domain.ImportResult
	for r := range results {
		if r.err != nil {
			summary.Failed++
			summary.Errors = append(summary.Errors, fmt.Sprintf("Line %d: %v", r.line, r.err))
		} else {
			summary.Succeeded++
		}
	}

	return &summary, nil
}
