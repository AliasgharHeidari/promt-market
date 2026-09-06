package routes

import (
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"promt-market/internal/handler"
	"promt-market/internal/middleware"
	"promt-market/pkg/jwt"
)

func SetupRoutes(app *fiber.App, db *gorm.DB, redis *redis.Client) {
	// Initialize JWT config
	jwt.Init(&jwt.JWTConfig{
		Secret:        os.Getenv("JWT_SECRET"),
		RefreshSecret: os.Getenv("JWT_REFRESH_SECRET"),
		AccessTTL:     15 * time.Minute,
		RefreshTTL:    7 * 24 * time.Hour,
	})

	// API v1 group
	api := app.Group("/api/v1")

	// Health check
	api.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "ok",
			"message": "Prompt Market API is running",
		})
	})

	// ========== AUTH ROUTES (Public) ==========
	authHandler := handler.NewAuthHandler(db)
	auth := api.Group("/auth")
	auth.Post("/register", authHandler.Register)
	auth.Post("/login", authHandler.Login)
	auth.Post("/refresh", authHandler.RefreshToken)

	// ========== PROMPT ROUTES (Public) ==========
	promptHandler := handler.NewPromptHandler(db)
	prompts := api.Group("/prompts")
	prompts.Get("/", promptHandler.GetAllPrompts)
	prompts.Get("/search", promptHandler.SearchPrompts)
	prompts.Get("/categories", promptHandler.GetCategories)
	prompts.Get("/:slug", promptHandler.GetPromptBySlug)

	// ========== PROTECTED ROUTES ==========
	protected := api.Group("/", middleware.JWTProtected())

	// User profile
	protected.Get("/profile", authHandler.GetProfile)

	// Prompt management (Seller only)
	protected.Post("/prompts", promptHandler.CreatePrompt)
	protected.Put("/prompts/:id", promptHandler.UpdatePrompt)
	protected.Delete("/prompts/:id", promptHandler.DeletePrompt)
	protected.Get("/my-prompts", promptHandler.GetMyPrompts)
}