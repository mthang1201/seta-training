package infrastructure

import (
	"fmt"
	"log"

	"github.com/seta-training/core/internal/config"
	"github.com/seta-training/core/internal/domain"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func NewPostgresDB(cfg *config.Config) *gorm.DB {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=UTC",
		cfg.DBHost, cfg.DBUser, cfg.DBPassword, cfg.DBName, cfg.DBPort)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Auto Migrate
	err = db.AutoMigrate(&domain.User{}, &domain.Team{}, &domain.Folder{}, &domain.Note{}, &domain.AssetPermission{})
	if err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	return db
}
