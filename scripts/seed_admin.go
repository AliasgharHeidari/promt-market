// scripts/seed_admin.go
package main

import (
	"log"
	"os"
	"fmt"

	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"promt-market/internal/domain"
)

func main() {
	// Load .env
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	// Get admin credentials from .env
	adminEmail := os.Getenv("ADMIN_EMAIL")
	adminPassword := os.Getenv("ADMIN_PASSWORD")
	adminFullName := os.Getenv("ADMIN_FULL_NAME")

	if adminEmail == "" || adminPassword == "" {
		log.Fatal("ADMIN_EMAIL and ADMIN_PASSWORD must be set in .env")
	}

	// ✅ Build DSN from environment variables (matching your .env)
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")
	dbSSLMode := os.Getenv("DB_SSL_MODE")

	if dbHost == "" {
		dbHost = "localhost"
	}
	if dbPort == "" {
		dbPort = "5432"
	}
	if dbUser == "" {
		dbUser = "db-admin"
	}
	if dbPassword == "" {
		dbPassword = "secret"
	}
	if dbName == "" {
		dbName = "promt_market"
	}
	if dbSSLMode == "" {
		dbSSLMode = "disable"
	}

	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		dbHost, dbPort, dbUser, dbPassword, dbName, dbSSLMode,
	)

	log.Printf("Connecting to database: %s", dsn)

	// Connect to database
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Auto migrate admin_log table
	if err := db.AutoMigrate(&domain.AdminLog{}); err != nil {
		log.Fatal("Failed to migrate:", err)
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(adminPassword), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal("Failed to hash password:", err)
	}

	// Create or update admin
	admin := domain.User{
		Email:        adminEmail,
		PasswordHash: string(hashedPassword),
		FullName:     adminFullName,
		Role:         "admin",
		IsActive:     true,
	}

	result := db.Where(domain.User{Email: adminEmail}).Assign(admin).FirstOrCreate(&admin)
	if result.Error != nil {
		log.Fatal("Failed to create admin:", result.Error)
	}

	log.Println("✅ Admin created/updated successfully!")
	log.Printf("📧 Email: %s", adminEmail)
	log.Printf("🔑 Password: %s", adminPassword)
	log.Printf("👤 Role: %s", admin.Role)
}