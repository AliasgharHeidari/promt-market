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

	// ========== PROMPT ROUTES (Public, optional auth) ==========
	//
	// /prompts/:slug needs to know the viewer's identity to decide whether to
	// expose content. We add OptionalJWT (a middleware that sets Locals when
	// a valid Authorization header is present but doesn't reject otherwise)
	// to the whole prompts group. Anonymous visitors pass through cleanly.
	promptHandler := handler.NewPromptHandler(db)
	reviewHandler := handler.NewReviewHandler(db)
	prompts := api.Group("/prompts",
		middleware.RateLimit(redis),
		middleware.OptionalJWT(),
	)
	prompts.Get("/", promptHandler.GetAllPrompts)
	prompts.Get("/search", promptHandler.SearchPrompts)
	prompts.Get("/categories", promptHandler.GetCategories)
	prompts.Get("/:id/reviews", reviewHandler.ListReviews)

	// Catch-all slug route must stay LAST.
	prompts.Get("/:slug", promptHandler.GetPromptBySlug)

	// ========== PUBLIC PAYMENT CALLBACK ==========
	//
	// This MUST be registered before the `protected` group below. A Fiber
	// Group created with prefix "/" (protected := api.Group("/", JWTProtected()))
	// registers its middleware against the whole "/api/v1" prefix tree, so
	// ANY route added afterwards under `api` — not just under `protected` —
	// would still pass through JWTProtected(). Registering the callback here,
	// before `protected` exists, keeps it genuinely public.
	//
	// The gateway redirects the user's browser here without an Authorization
	// header. Security is enforced by verifying with the provider, not by JWT.
	commission := 20
	if v := os.Getenv("COMMISSION_PERCENT"); v != "" {
		// best-effort parse; a bad value falls back to 20
		if n, err := parseIntSafe(v); err == nil {
			commission = n
		}
	}
	checkoutHandler := handler.NewCheckoutHandler(db, commission)
	api.Get("/payment/callback", checkoutHandler.PaymentCallback)

	// ========== PROTECTED ROUTES ==========
	protected := api.Group("/", middleware.JWTProtected())

	protected.Get("/profile", authHandler.GetProfile)

	protected.Post("/prompts", middleware.PromptAuthorOnly(), promptHandler.CreatePrompt)
	protected.Put("/prompts/:id", middleware.PromptAuthorOnly(), promptHandler.UpdatePrompt)
	protected.Delete("/prompts/:id", middleware.PromptAuthorOnly(), promptHandler.DeletePrompt)
	protected.Get("/my-prompts", middleware.PromptAuthorOnly(), promptHandler.GetMyPrompts)
	protected.Get("/my-prompts/:id", middleware.PromptAuthorOnly(), promptHandler.GetMyPromptByID)
	protected.Get("/prompts/id/:id", middleware.AdminOnly(), promptHandler.GetPromptByID)

	protected.Post("/prompts/:id/reviews", reviewHandler.CreateReview)
	protected.Delete("/reviews/:id", reviewHandler.DeleteOwnReview)

	// ========== CART ROUTES ==========
	cartHandler := handler.NewCartHandler(db)
	cart := protected.Group("/cart",
		middleware.RateLimit(redis),
	)
	cart.Get("/", cartHandler.GetCart)
	cart.Get("/prompt-ids", cartHandler.GetCartPromptIDs)
	cart.Post("/items", cartHandler.AddToCart)
	cart.Delete("/items/:prompt_id", cartHandler.RemoveFromCart)
	cart.Delete("/", cartHandler.ClearCart)

	// ========== CHECKOUT & PURCHASES ==========
	//
	// checkoutHandler was already constructed above (before `protected`) so
	// the public callback route could be registered early. We reuse it here.
	purchaseHandler := handler.NewPurchaseHandler(db)

	// Any authenticated user can start a checkout or list their purchases.
	protected.Post("/checkout", middleware.StrictRateLimit(redis), checkoutHandler.Checkout)
	protected.Get("/my-purchases", purchaseHandler.ListMyPurchases)
	protected.Get("/my-purchases/:id", purchaseHandler.GetPurchaseDetail)
	protected.Get("/my-purchases/:id/content", purchaseHandler.GetPurchaseContent)

	// Payment status (read-only, for polling).
	protected.Get("/payment/status/:authority", checkoutHandler.GetPaymentStatus)

	// ========== AUTHOR DASHBOARD ROUTES ==========
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

// parseIntSafe is a tiny helper used only for the commission env read above.
// Kept local so we don't import strconv just for this one call elsewhere.
func parseIntSafe(s string) (int, error) {
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0, fiber.ErrBadRequest
		}
		n = n*10 + int(r-'0')
	}
	return n, nil
}