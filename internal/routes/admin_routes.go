package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"promt-market/internal/handler"
	"promt-market/internal/middleware"
)

func SetupAdminRoutes(app *fiber.App, db *gorm.DB, redisClient *redis.Client) {
	adminHandler := handler.NewAdminHandler(db, redisClient)

	admin := app.Group("/api/v1/admin",
		middleware.JWTProtected(),
		middleware.AdminOnly(),
		middleware.AdminRateLimit(redisClient),
		middleware.AuditLog(db),
	)

	// CSRF token endpoint.
	admin.Get("/csrf-token", adminHandler.GetCSRFToken)

	// Dashboard
	admin.Get("/stats", adminHandler.GetDashboardStats)

	// User management
	admin.Get("/users", adminHandler.GetAllUsers)
	admin.Put("/users/:id/status", adminHandler.UpdateUserStatus)

	// Prompt management — listing
	admin.Get("/prompts/pending", adminHandler.GetPendingPrompts)
	admin.Get("/prompts/approved", adminHandler.GetApprovedPrompts)
	admin.Get("/prompts/rejected", adminHandler.GetRejectedPrompts)
	admin.Get("/prompts/deleted", adminHandler.GetDeletedPrompts)
	admin.Get("/prompts/all", adminHandler.GetAllPromptsAdmin)

	// Prompt management — moderation & editing
	admin.Put("/prompts/:id/approve", adminHandler.ApprovePrompt)
	admin.Put("/prompts/:id/reject", adminHandler.RejectPromptWithNote)
	admin.Patch("/prompts/:id", adminHandler.AdminUpdatePrompt)
	admin.Delete("/prompts/:id/images/:index", adminHandler.RemovePromptImage)
	admin.Delete("/prompts/:id/cover", adminHandler.RemovePromptCover)
	admin.Delete("/prompts/:id", adminHandler.DeletePromptAdmin)

	// Order management
	admin.Get("/orders", adminHandler.GetAllOrders)
	admin.Get("/orders/:id", adminHandler.GetOrderDetail)
	admin.Put("/orders/:id/status", adminHandler.UpdateOrderStatus)

	// Audit logs
	admin.Get("/logs", adminHandler.GetAdminLogs)

	// Author applications
	admin.Get("/author-applications", adminHandler.GetAuthorApplications)
	admin.Get("/author-applications/:id", adminHandler.GetAuthorApplicationDetail)
	admin.Get("/author-applications/:id/document", adminHandler.GetAuthorApplicationDocument)
	admin.Put("/author-applications/:id/approve", adminHandler.ApproveAuthorApplication)
	admin.Put("/author-applications/:id/reject", adminHandler.RejectAuthorApplication)
}
