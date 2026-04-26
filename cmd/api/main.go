package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/seta-training/core/internal/config"
	deliveryHttp "github.com/seta-training/core/internal/delivery/http"
	"github.com/seta-training/core/internal/infrastructure"
	"github.com/seta-training/core/internal/repository"
	"github.com/seta-training/core/internal/usecase"
)

func main() {
	// 1. Load config
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 2. Init DB
	db := infrastructure.NewPostgresDB(cfg)

	// 3. Init Repositories
	userRepo := repository.NewUserRepository(db)
	teamRepo := repository.NewTeamRepository(db)
	assetRepo := repository.NewAssetRepository(db)

	// 4. Init UseCases
	userUseCase := usecase.NewUserUseCase(userRepo, cfg)
	teamUseCase := usecase.NewTeamUseCase(teamRepo, userRepo)
	assetUseCase := usecase.NewAssetUseCase(assetRepo)

	// 5. Init Gin App
	app := gin.Default()

	// 6. Setup Routes
	api := app.Group("/api/v1")
	deliveryHttp.NewUserHandler(api, userUseCase, cfg)
	deliveryHttp.NewTeamHandler(api, teamUseCase, cfg)
	deliveryHttp.NewAssetHandler(api, assetUseCase, cfg)

	// 7. Start server
	log.Printf("Starting server on port %s...", cfg.Port)
	if err := app.Run(":" + cfg.Port); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
