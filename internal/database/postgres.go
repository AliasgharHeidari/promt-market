package database

import (
	"fmt"
	"log"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"promt-market/internal/domain"
)

func Connect(dsn string, maxConns int) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database instance: %w", err)
	}

	// Connection pool settings
	sqlDB.SetMaxOpenConns(maxConns)
	sqlDB.SetMaxIdleConns(maxConns / 5)
	sqlDB.SetConnMaxLifetime(time.Hour)

	// Auto migrate
	if err := autoMigrate(db); err != nil {
		return nil, fmt.Errorf("failed to migrate: %w", err)
	}

	log.Println("Database connected successfully")
	return db, nil
}

func autoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&domain.User{},
		&domain.Prompt{},
		&domain.Order{},
		&domain.Review{},
	)
}