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

	// ========== STATIC FILES (serve uploaded files) ==========
	// Any file under ./uploads is publicly reachable via /uploads/<...>.
	// Example: ./uploads/prompts/<uid>/xxx.jpg
	//        → http://localhost:8080/uploads/prompts/<uid>/xxx.jpg
	//
	// NOTE: this makes images publicly readable by URL. Do NOT serve
	// sensitive files (e.g. author ID documents) from this directory —
	// those go through authenticated endpoints instead.
	app.Static("/uploads", "./uploads")

	api := app.Group("/api/v1")

	api.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "ok",
			"message": "Prompt Market API is running",
		})
	})

	// ========== AUTH ROUTES (Public, strict rate limit) ==========
	// Register / login / verify / resend are prime targets for automated
	// abuse (brute force, email spam, code enumeration), so they get a
	// dedicated, tighter limiter per endpoint.
	emailService := service.NewEmailService()
	authHandler := handler.NewAuthHandler(db, redis, emailService)
	auth := api.Group("/auth")

	auth.Post("/register", middleware.AuthRateLimit(redis), authHandler.Register)
	auth.Post("/verify-email", middleware.AuthRateLimit(redis), authHandler.VerifyEmail)
	auth.Post("/resend-verification", middleware.AuthRateLimit(redis), authHandler.ResendVerification)
	auth.Post("/login", middleware.AuthRateLimit(redis), authHandler.Login)
	auth.Post("/refresh", middleware.StrictRateLimit(redis), authHandler.RefreshToken)

	// ========== PROMPT ROUTES (Public reads, general limit) ==========
	promptHandler := handler.NewPromptHandler(db)
	prompts := api.Group("/prompts",
		middleware.RateLimit(redis),
	)
	prompts.Get("/", promptHandler.GetAllPrompts)
	prompts.Get("/search", promptHandler.SearchPrompts)
	prompts.Get("/categories", promptHandler.GetCategories)
	prompts.Get("/:slug", promptHandler.GetPromptBySlug)

	// ========== PROTECTED ROUTES ==========
	protected := api.Group("/", middleware.JWTProtected())

	protected.Get("/profile", authHandler.GetProfile)

	// Create / update / delete prompts — restricted to prompt_author or admin.
	protected.Post("/prompts", middleware.PromptAuthorOnly(), promptHandler.CreatePrompt)
	protected.Put("/prompts/:id", middleware.PromptAuthorOnly(), promptHandler.UpdatePrompt)
	protected.Delete("/prompts/:id", middleware.PromptAuthorOnly(), promptHandler.DeletePrompt)
	protected.Get("/my-prompts", middleware.PromptAuthorOnly(), promptHandler.GetMyPrompts)

	// Author-only: view a single own prompt with any status (pending/rejected/deleted).
	// Ownership is enforced inside the handler, not just by middleware.
	protected.Get("/my-prompts/:id", middleware.PromptAuthorOnly(), promptHandler.GetMyPromptByID)

	// Admin-only: full prompt lookup including soft-deleted rows.
	protected.Get("/prompts/id/:id", middleware.AdminOnly(), promptHandler.GetPromptByID)

	// ========== UPLOAD ROUTES ==========
	// Only authenticated users can upload. Returns a public URL for the file.
	// Uploads are rate-limited per user (falls back to IP) to protect disk
	// space and prevent automated abuse.
	baseURL := os.Getenv("BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}
	uploadHandler := handler.NewUploadHandler("./uploads", baseURL)
	upload := protected.Group("/upload",
		middleware.UploadRateLimit(redis),
	)
	upload.Post("/prompt-image", uploadHandler.UploadPromptImage)   // single file (cover)
	upload.Post("/prompt-images", uploadHandler.UploadPromptImages) // multiple files (gallery)

	// ========== AUTHOR APPLICATION ROUTES ==========
	// Flow: authenticated user → verify phone (OTP) → submit ID documents
	// (expertise, national ID, ID card photo) → admin review → on approval,
	// the user's role is promoted to "prompt_author".
	//
	// Rate-limited per user+path so a single account can't spam OTPs or
	// flood the review queue with applications.
	authorAppHandler := handler.NewAuthorApplicationHandler(db, redis)
	authorApp := protected.Group("/author-application",
		middleware.AuthorAppRateLimit(redis),
	)
	authorApp.Post("/send-phone-otp", authorAppHandler.SendPhoneOTP)
	authorApp.Post("/verify-phone", authorAppHandler.VerifyPhone)
	authorApp.Post("/", authorAppHandler.SubmitApplication)
	authorApp.Get("/me", authorAppHandler.GetMyApplication)
}