package routes

import (
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"promt-market/internal/handler"
	"promt-market/internal/middleware"
	"promt-market/internal/service"
	"promt-market/pkg/jwt"
)

func SetupRoutes(app *fiber.App, db *gorm.DB, redis *redis.Client) {
	jwt.Init(&jwt.JWTConfig{
		Secret:        os.Getenv("JWT_SECRET"),
		RefreshSecret: os.Getenv("JWT_REFRESH_SECRET"),
		AccessTTL:     15 * time.Minute,
		RefreshTTL:    7 * 24 * time.Hour,
	})

	api := app.Group("/api/v1")

	api.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "ok",
			"message": "Prompt Market API is running",
		})
	})

	// ========== AUTH ROUTES (Public) ==========
	emailService := service.NewEmailService()
	authHandler := handler.NewAuthHandler(db, redis, emailService)
	auth := api.Group("/auth")
	auth.Post("/register", authHandler.Register)
	auth.Post("/verify-email", authHandler.VerifyEmail)
	auth.Post("/resend-verification", authHandler.ResendVerification)
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

	protected.Get("/profile", authHandler.GetProfile)

	protected.Post("/prompts", promptHandler.CreatePrompt)
	protected.Put("/prompts/:id", promptHandler.UpdatePrompt)
	protected.Delete("/prompts/:id", promptHandler.DeletePrompt)
	protected.Get("/my-prompts", promptHandler.GetMyPrompts)

	protected.Get("/prompts/id/:id", middleware.AdminOnly(), promptHandler.GetPromptByID)
}