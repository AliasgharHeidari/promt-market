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

// Connect opens a PostgreSQL connection, tunes the connection pool, and runs
// AutoMigrate for every model in the project.
//
// AutoMigrate lives here (not in main.go) so the schema is guaranteed to be
// up-to-date the moment any caller receives a usable *gorm.DB. Every new
// model added to the project must be registered in autoMigrate below.
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

	// Auto migrate — single source of truth for the schema.
	if err := autoMigrate(db); err != nil {
		return nil, fmt.Errorf("failed to migrate: %w", err)
	}

	log.Println("✅ Database connected and migrated successfully")
	return db, nil
}

// autoMigrate creates or updates all tables.
//
// Order: parent tables (users, prompts) before children (orders, cart_items,
// reviews). GORM is generally tolerant of order, but explicit ordering keeps
// the first run clean and avoids transient FK errors.
func autoMigrate(db *gorm.DB) error {
	log.Println("🔄 Running auto migration...")

	if err := db.AutoMigrate(
		// ── Core ─────────────────────────────────────────
		&domain.User{},
		&domain.Prompt{},

		// ── Transactions & moderation ────────────────────
		&domain.Order{},
		&domain.Payment{},
		&domain.Review{},
		&domain.AdminLog{},

		// ── Author features ──────────────────────────────
		&domain.AuthorApplication{},
		&domain.PromptView{},
		&domain.PromptEditProposal{},

		// ── Cart (Phase 1) ───────────────────────────────
		&domain.CartItem{},
	); err != nil {
		return err
	}

	log.Println("✅ All tables migrated")
	return nil
}