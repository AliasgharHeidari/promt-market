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

	// ========== STATIC FILES (سرو کردن فایل‌های آپلود شده) ==========
	// هر فایل داخل ./uploads از مسیر /uploads/<...> قابل دسترسی می‌شه.
	// مثلاً: ./uploads/prompts/<uid>/xxx.jpg → http://localhost:8080/uploads/prompts/<uid>/xxx.jpg
	app.Static("/uploads", "./uploads")

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

	// ✅ ساخت/ویرایش/حذف پرامپت فقط برای prompt_author یا admin
	protected.Post("/prompts", middleware.PromptAuthorOnly(), promptHandler.CreatePrompt)
	protected.Put("/prompts/:id", middleware.PromptAuthorOnly(), promptHandler.UpdatePrompt)
	protected.Delete("/prompts/:id", middleware.PromptAuthorOnly(), promptHandler.DeletePrompt)
	protected.Get("/my-prompts", middleware.PromptAuthorOnly(), promptHandler.GetMyPrompts)

	protected.Get("/prompts/id/:id", middleware.AdminOnly(), promptHandler.GetPromptByID)

	// ========== UPLOAD ROUTES (✅ جدید) ==========
	// فقط کاربر لاگین‌شده می‌تونه آپلود کنه. URL عمومی فایل برگردونده می‌شه.
	baseURL := os.Getenv("BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}
	uploadHandler := handler.NewUploadHandler("./uploads", baseURL)
	upload := protected.Group("/upload")
	upload.Post("/prompt-image", uploadHandler.UploadPromptImage)   // تک فایل (کاور)
	upload.Post("/prompt-images", uploadHandler.UploadPromptImages) // چند فایل (گالری)

	// ========== AUTHOR APPLICATION ROUTES ==========
	// فلو: کاربر لاگین‌شده → تایید شماره موبایل (OTP) → ارسال مدارک (تخصص،
	// کد ملی، عکس کارت ملی/شناسنامه) → بررسی توسط ادمین → در صورت تایید،
	// Role کاربر به "prompt_author" تغییر می‌کنه.
	authorAppHandler := handler.NewAuthorApplicationHandler(db, redis)
	authorApp := protected.Group("/author-application")
	authorApp.Post("/send-phone-otp", authorAppHandler.SendPhoneOTP)
	authorApp.Post("/verify-phone", authorAppHandler.VerifyPhone)
	authorApp.Post("/", authorAppHandler.SubmitApplication)
	authorApp.Get("/me", authorAppHandler.GetMyApplication)
}