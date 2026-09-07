// cmd/api/main.go
package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"gorm.io/gorm"

	"promt-market/internal/config"
	"promt-market/internal/database"
	"promt-market/internal/domain"
	"promt-market/internal/routes"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Connect to PostgreSQL
	db, err := database.Connect(cfg.GetDSN(), cfg.DB.MaxConns)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Auto migrate
	if err := autoMigrate(db); err != nil {
		log.Fatal("Failed to migrate database:", err)
	}

	// Connect to Redis
	redisClient, err := database.ConnectRedis(cfg.Redis)
	if err != nil {
		log.Fatal("Failed to connect to Redis:", err)
	}

	// Create Fiber app
	app := fiber.New(fiber.Config{
		AppName:      cfg.App.Name,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
		Prefork:      cfg.App.Env == "production",
	})

	//  CORS Middleware 
	app.Use(cors.New(cors.Config{
		AllowOrigins:     "http://localhost:5500, http://127.0.0.1:5500, http://localhost:3000, http://127.0.0.1:3000", // ✅ مشخص کردن دامنه‌ها
		AllowMethods:     "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization, X-CSRF-Token",
		ExposeHeaders:    "Content-Length, Content-Type",
		AllowCredentials: true,
	}))

	// Global middlewares
	app.Use(recover.New())
	app.Use(logger.New(logger.Config{
		Format: "[${time}] ${status} - ${method} ${path} ${latency}\n",
	}))

	// Setup routes
	routes.SetupRoutes(app, db, redisClient)
	routes.SetupAdminRoutes(app, db, redisClient)

	// Start server
	go func() {
		if err := app.Listen(":" + cfg.App.Port); err != nil {
			log.Fatal("Server error:", err)
		}
	}()

	log.Printf("🚀 Server started on port %s", cfg.App.Port)

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	if err := app.Shutdown(); err != nil {
		log.Fatal("Server shutdown error:", err)
	}
	log.Println("Server stopped")
}

// autoMigrate creates all tables automatically
func autoMigrate(db *gorm.DB) error {
	log.Println("🔄 Running auto migration...")

	if err := db.AutoMigrate(
		&domain.User{},
		&domain.Prompt{},
		&domain.Order{},
		&domain.Review{},
		&domain.AdminLog{},
	); err != nil {
		return err
	}

	log.Println("✅ Database migrated successfully!")
	return nil
}