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

	// ========== STATIC FILES ==========
	app.Static("/uploads", "./uploads")

	api := app.Group("/api/v1")

	api.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "ok",
			"message": "Prompt Market API is running",
		})
	})

	// ========== AUTH ROUTES ==========
	emailService := service.NewEmailService()
	authHandler := handler.NewAuthHandler(db, redis, emailService)
	auth := api.Group("/auth")

	auth.Post("/register", middleware.AuthRateLimit(redis), authHandler.Register)
	auth.Post("/verify-email", middleware.AuthRateLimit(redis), authHandler.VerifyEmail)
	auth.Post("/resend-verification", middleware.AuthRateLimit(redis), authHandler.ResendVerification)
	auth.Post("/login", middleware.AuthRateLimit(redis), authHandler.Login)
	auth.Post("/refresh", middleware.StrictRateLimit(redis), authHandler.RefreshToken)

	// ========== PROMPT ROUTES (Public) ==========
	promptHandler := handler.NewPromptHandler(db)
	reviewHandler := handler.NewReviewHandler(db)
	prompts := api.Group("/prompts",
		middleware.RateLimit(redis),
	)
	prompts.Get("/", promptHandler.GetAllPrompts)
	prompts.Get("/search", promptHandler.SearchPrompts)
	prompts.Get("/categories", promptHandler.GetCategories)

	// Reviews for a prompt — public read (only approved).
	prompts.Get("/:id/reviews", reviewHandler.ListReviews)

	// Catch-all slug route must stay LAST.
	prompts.Get("/:slug", promptHandler.GetPromptBySlug)

	// ========== PROTECTED ROUTES ==========
	protected := api.Group("/", middleware.JWTProtected())

	protected.Get("/profile", authHandler.GetProfile)

	// Prompt CRUD — restricted to prompt_author or admin.
	protected.Post("/prompts", middleware.PromptAuthorOnly(), promptHandler.CreatePrompt)
	protected.Put("/prompts/:id", middleware.PromptAuthorOnly(), promptHandler.UpdatePrompt)
	protected.Delete("/prompts/:id", middleware.PromptAuthorOnly(), promptHandler.DeletePrompt)
	protected.Get("/my-prompts", middleware.PromptAuthorOnly(), promptHandler.GetMyPrompts)
	protected.Get("/my-prompts/:id", middleware.PromptAuthorOnly(), promptHandler.GetMyPromptByID)
	protected.Get("/prompts/id/:id", middleware.AdminOnly(), promptHandler.GetPromptByID)

	// Reviews — any authenticated user can try; service enforces ownership,
	// purchase, and one-per-user rules.
	protected.Post("/prompts/:id/reviews", reviewHandler.CreateReview)
	protected.Delete("/reviews/:id", reviewHandler.DeleteOwnReview)

	// ========== AUTHOR DASHBOARD ROUTES ==========
	// Everything below is author-only. The dashboard shows sales, views,
	// profile management, pause/unpause, and edit proposals.
	authorDashHandler := handler.NewAuthorDashboardHandler(db)
	author := protected.Group("/author", middleware.PromptAuthorOnly())
	author.Get("/dashboard/stats", authorDashHandler.GetDashboardStats)
	author.Get("/dashboard/views-chart", authorDashHandler.GetViewsChart)
	author.Get("/dashboard/revenue-chart", authorDashHandler.GetRevenueChart)

	author.Get("/profile", authorDashHandler.GetProfile)
	author.Put("/profile", authorDashHandler.UpdateProfile)
	author.Post("/profile/avatar", authorDashHandler.UploadAvatar)

	author.Post("/prompts/:id/pause", authorDashHandler.PausePrompt)
	author.Post("/prompts/:id/unpause", authorDashHandler.UnpausePrompt)
	author.Post("/prompts/:id/edit-proposal", authorDashHandler.SubmitEdit)

	// ========== UPLOAD ROUTES ==========
	baseURL := os.Getenv("BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}
	uploadHandler := handler.NewUploadHandler("./uploads", baseURL)
	upload := protected.Group("/upload",
		middleware.UploadRateLimit(redis),
	)
	upload.Post("/prompt-image", uploadHandler.UploadPromptImage)
	upload.Post("/prompt-images", uploadHandler.UploadPromptImages)

	// ========== AUTHOR APPLICATION ROUTES ==========
	authorAppHandler := handler.NewAuthorApplicationHandler(db, redis)
	authorApp := protected.Group("/author-application",
		middleware.AuthorAppRateLimit(redis),
	)
	authorApp.Post("/send-phone-otp", authorAppHandler.SendPhoneOTP)
	authorApp.Post("/verify-phone", authorAppHandler.VerifyPhone)
	authorApp.Post("/", authorAppHandler.SubmitApplication)
	authorApp.Get("/me", authorAppHandler.GetMyApplication)
}