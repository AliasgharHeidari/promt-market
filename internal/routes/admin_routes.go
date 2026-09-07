package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"promt-market/internal/handler"
	"promt-market/internal/middleware"
)

func SetupAdminRoutes(app *fiber.App, db *gorm.DB, redisClient *redis.Client) {
	adminHandler := handler.NewAdminHandler(db)

	admin := app.Group("/api/v1/admin",
		middleware.JWTProtected(),
		middleware.AdminOnly(),
		middleware.AdminRateLimit(redisClient),
		middleware.AuditLog(db),
	)

	// CSRF token endpoint.
	// ⚠️ Moved inside the protected `admin` group: previously this was
	// registered on `app` directly and bypassed JWTProtected/AdminOnly/
	// AdminRateLimit/AuditLog entirely, letting anyone (even unauthenticated
	// requests) mint a CSRF cookie tied to the admin panel.
	admin.Get("/csrf-token", adminHandler.GetCSRFToken)

	// Dashboard
	admin.Get("/stats", adminHandler.GetDashboardStats)

	// User management
	admin.Get("/users", adminHandler.GetAllUsers)
	admin.Put("/users/:id/status", adminHandler.UpdateUserStatus)

	// Prompt management
	admin.Get("/prompts/pending", adminHandler.GetPendingPrompts)
	admin.Get("/prompts/approved", adminHandler.GetApprovedPrompts)
	admin.Get("/prompts/rejected", adminHandler.GetRejectedPrompts)
	admin.Get("/prompts/all", adminHandler.GetAllPromptsAdmin)

	admin.Put("/prompts/:id/approve", adminHandler.ApprovePrompt)
	admin.Put("/prompts/:id/reject", adminHandler.RejectPrompt)
	
	// ✅ مسیر درست برای ادمین
	admin.Delete("/prompts/:id", adminHandler.DeletePromptAdmin)

	// Order management
	admin.Get("/orders", adminHandler.GetAllOrders)
	admin.Get("/orders/:id", adminHandler.GetOrderDetail)
	admin.Put("/orders/:id/status", adminHandler.UpdateOrderStatus)

	// Audit logs
	admin.Get("/logs", adminHandler.GetAdminLogs)
}