package utils

import (
	"fmt"
	"log"

	"github.com/morgarakt/aurify/internal/config"
	"github.com/morgarakt/aurify/internal/models"

	_ "github.com/lib/pq"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func ConnectDB(cfg *config.Config) (*gorm.DB, error) {
	if err := createDatabaseIfNotExists(cfg); err != nil {
		return nil, fmt.Errorf("failed to create database: %w", err)
	}

	dsn := cfg.DBConnectionStringWName()
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	log.Printf("Connected to database: %s", cfg.DBName)
	return db, nil
}

func createDatabaseIfNotExists(cfg *config.Config) error {
	pgConnStr := cfg.DBConnectionStringWOName()

	db, err := gorm.Open(postgres.Open(pgConnStr), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}

	var exists bool
	db.Raw("SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = ?)", cfg.DBName).Scan(&exists)
	if !exists {
		log.Printf("Database %s does not exist. Creating...", cfg.DBName)
		if err := db.Exec(fmt.Sprintf("CREATE DATABASE %s;", cfg.DBName)).Error; err != nil {
			log.Fatalf("Failed to create database: %v", err)
		}
		log.Println("Database created successfully.")
	} else {
		log.Println("Database already exists.")
	}

	return nil
}

func MigrateDB(db *gorm.DB) error {
	log.Println("Running database migrations...")
	if err := db.Exec("CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\"").Error; err != nil {
		return fmt.Errorf("failed to create uuid extension: %w", err)
	}

	err := db.AutoMigrate(&models.User{}, &models.MusicType{}, &models.ModelType{}, &models.Music{})
	if err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}
	log.Println("Database migrations completed successfully")
	return nil
}

func CloseDB(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
