package handler

import (
	"strconv"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"promt-market/internal/domain"
	"promt-market/internal/middleware"
	"promt-market/internal/repository"
	"promt-market/internal/service"
)

type AdminHandler struct {
	service   *service.AdminService
	validator *validator.Validate
}

func NewAdminHandler(db *gorm.DB, redisClient *redis.Client) *AdminHandler {
	adminRepo := repository.NewAdminRepository(db)
	userRepo := repository.NewUserRepository(db)
	promptRepo := repository.NewPromptRepository(db)
	orderRepo := repository.NewOrderRepository(db)

	// Dependencies for reviewing author-application requests.
	appRepo := repository.NewAuthorApplicationRepository(db)
	verifyRepo := repository.NewVerificationRepository(redisClient)
	appService := service.NewAuthorApplicationService(appRepo, userRepo, verifyRepo)

	adminService := service.NewAdminService(adminRepo, userRepo, promptRepo, orderRepo, appService, db)

	return &AdminHandler{
		service:   adminService,
		validator: validator.New(),
	}
}

// ============================================================
//  DASHBOARD
// ============================================================

// GetDashboardStats returns admin dashboard statistics
func (h *AdminHandler) GetDashboardStats(c *fiber.Ctx) error {
	stats, err := h.service.GetDashboardStats(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data": stats,
	})
}

// ============================================================
//  USER MANAGEMENT
// ============================================================

// GetAllUsers returns all users with pagination
func (h *AdminHandler) GetAllUsers(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))

	users, total, err := h.service.GetAllUsers(c.Context(), page, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data": users,
		"pagination": fiber.Map{
			"page":  page,
			"limit": limit,
			"total": total,
			"pages": (total + int64(limit) - 1) / int64(limit),
		},
	})
}

// UpdateUserStatus activates or deactivates a user
func (h *AdminHandler) UpdateUserStatus(c *fiber.Ctx) error {
	userID := c.Params("id")

	var req struct {
		IsActive bool `json:"is_active"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if err := h.service.UpdateUserStatus(c.Context(), userID, req.IsActive); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "User status updated successfully",
	})
}

// ============================================================
//  PROMPT MANAGEMENT
// ============================================================

// GetPendingPrompts returns all pending prompts
func (h *AdminHandler) GetPendingPrompts(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))

	prompts, total, err := h.service.GetPromptsByStatus(c.Context(), "pending", page, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data": prompts,
		"pagination": fiber.Map{
			"page":  page,
			"limit": limit,
			"total": total,
			"pages": (total + int64(limit) - 1) / int64(limit),
		},
	})
}

// GetApprovedPrompts returns all approved prompts
func (h *AdminHandler) GetApprovedPrompts(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))

	prompts, total, err := h.service.GetPromptsByStatus(c.Context(), "approved", page, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data": prompts,
		"pagination": fiber.Map{
			"page":  page,
			"limit": limit,
			"total": total,
			"pages": (total + int64(limit) - 1) / int64(limit),
		},
	})
}

// GetRejectedPrompts returns all rejected prompts
func (h *AdminHandler) GetRejectedPrompts(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))

	prompts, total, err := h.service.GetPromptsByStatus(c.Context(), "rejected", page, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data": prompts,
		"pagination": fiber.Map{
			"page":  page,
			"limit": limit,
			"total": total,
			"pages": (total + int64(limit) - 1) / int64(limit),
		},
	})
}

// GetDeletedPrompts returns all soft-deleted prompts
func (h *AdminHandler) GetDeletedPrompts(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))

	prompts, total, err := h.service.GetPromptsByStatus(c.Context(), "deleted", page, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data": prompts,
		"pagination": fiber.Map{
			"page":  page,
			"limit": limit,
			"total": total,
			"pages": (total + int64(limit) - 1) / int64(limit),
		},
	})
}

// GetAllPromptsAdmin returns all prompts (any status)
func (h *AdminHandler) GetAllPromptsAdmin(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))

	prompts, total, err := h.service.GetAllPromptsAdmin(c.Context(), page, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data": prompts,
		"pagination": fiber.Map{
			"page":  page,
			"limit": limit,
			"total": total,
			"pages": (total + int64(limit) - 1) / int64(limit),
		},
	})
}

// ApprovePrompt approves a prompt.
func (h *AdminHandler) ApprovePrompt(c *fiber.Ctx) error {
	promptID := c.Params("id")

	if err := h.service.ApprovePrompt(c.Context(), promptID); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Prompt approved successfully",
	})
}

// DeletePromptAdmin deletes a prompt (admin only, no ownership check).
// Unlike PromptHandler.DeletePrompt (used by regular sellers), this path
// intentionally skips the "you are not the owner" check because an admin
// is expected to be able to remove any prompt, e.g. for moderation.
func (h *AdminHandler) DeletePromptAdmin(c *fiber.Ctx) error {
	promptID := c.Params("id")

	prompt, err := h.service.DeletePromptAdmin(c.Context(), promptID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Best-effort audit trail with domain-specific context (prompt title/seller).
	// Failing to write this log should not fail the delete itself, since the
	// delete already succeeded and the generic AuditLog middleware still ran.
	if adminID, ok := c.Locals("userID").(string); ok {
		_ = h.service.LogAdminAction(
			c.Context(),
			adminID,
			"delete_prompt",
			"prompt",
			promptID,
			domain.JSONMap{
				"title":     prompt.Title,
				"seller_id": prompt.SellerID,
			},
		)
	}

	return c.JSON(fiber.Map{
		"message": "Prompt deleted successfully",
	})
}

// ============================================================
//  PROMPT EDITING & MODERATION
// ============================================================

// RejectPromptWithNote rejects a prompt and stores an optional reason text
// that the seller will see on their dashboard. Body: { "note": "..." }.
func (h *AdminHandler) RejectPromptWithNote(c *fiber.Ctx) error {
	promptID := c.Params("id")
	adminID, _ := c.Locals("userID").(string)

	var req domain.RejectPromptRequest
	_ = c.BodyParser(&req) // note is optional

	if err := h.service.RejectPrompt(c.Context(), promptID, req.Note); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	_ = h.service.LogAdminAction(
		c.Context(), adminID, "reject_prompt", "prompt", promptID,
		domain.JSONMap{"note": req.Note},
	)

	return c.JSON(fiber.Map{"message": "Prompt rejected"})
}

// AdminUpdatePrompt allows the admin to patch any prompt while reviewing it.
// All fields are optional. Accepts a Status field to approve/reject in the
// same request as metadata edits.
func (h *AdminHandler) AdminUpdatePrompt(c *fiber.Ctx) error {
	promptID := c.Params("id")
	adminID, _ := c.Locals("userID").(string)

	var req domain.AdminUpdatePromptRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}
	if err := h.validator.Struct(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	prompt, err := h.service.AdminUpdatePrompt(c.Context(), promptID, &req)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	_ = h.service.LogAdminAction(
		c.Context(), adminID, "update_prompt", "prompt", promptID,
		domain.JSONMap{"title": prompt.Title},
	)

	return c.JSON(fiber.Map{
		"message": "Prompt updated successfully",
		"data":    prompt,
	})
}

// RemovePromptImage removes a single gallery image identified by its
// zero-based index in the URL path: DELETE /admin/prompts/:id/images/:index
func (h *AdminHandler) RemovePromptImage(c *fiber.Ctx) error {
	promptID := c.Params("id")
	adminID, _ := c.Locals("userID").(string)

	index, err := strconv.Atoi(c.Params("index"))
	if err != nil || index < 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid image index"})
	}

	prompt, err := h.service.RemovePromptImage(c.Context(), promptID, index)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	_ = h.service.LogAdminAction(
		c.Context(), adminID, "remove_prompt_image", "prompt", promptID,
		domain.JSONMap{"index": index},
	)

	return c.JSON(fiber.Map{
		"message": "Image removed",
		"data":    prompt,
	})
}

// RemovePromptCover clears the cover image on a prompt.
func (h *AdminHandler) RemovePromptCover(c *fiber.Ctx) error {
	promptID := c.Params("id")
	adminID, _ := c.Locals("userID").(string)

	prompt, err := h.service.RemovePromptCover(c.Context(), promptID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	_ = h.service.LogAdminAction(
		c.Context(), adminID, "remove_prompt_cover", "prompt", promptID,
		nil,
	)

	return c.JSON(fiber.Map{
		"message": "Cover image removed",
		"data":    prompt,
	})
}

// ============================================================
//  ORDER MANAGEMENT
// ============================================================

// GetAllOrders returns all orders with pagination
func (h *AdminHandler) GetAllOrders(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))

	orders, total, err := h.service.GetAllOrders(c.Context(), page, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data": orders,
		"pagination": fiber.Map{
			"page":  page,
			"limit": limit,
			"total": total,
			"pages": (total + int64(limit) - 1) / int64(limit),
		},
	})
}

// GetOrderDetail returns order details by ID
func (h *AdminHandler) GetOrderDetail(c *fiber.Ctx) error {
	orderID := c.Params("id")

	order, err := h.service.GetOrderByID(c.Context(), orderID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data": order,
	})
}

// UpdateOrderStatus updates order status
func (h *AdminHandler) UpdateOrderStatus(c *fiber.Ctx) error {
	orderID := c.Params("id")

	var req struct {
		Status string `json:"status" validate:"oneof=pending paid failed refunded"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if err := h.validator.Struct(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	if err := h.service.UpdateOrderStatus(c.Context(), orderID, req.Status); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Order status updated successfully",
	})
}

// ============================================================
//  CSRF
// ============================================================

// GetCSRFToken returns a new CSRF token
func (h *AdminHandler) GetCSRFToken(c *fiber.Ctx) error {
	token := middleware.GenerateCSRFToken()

	c.Cookie(&fiber.Cookie{
		Name:     "csrf_token",
		Value:    token,
		HTTPOnly: true,
		Secure:   true,
		SameSite: "Strict",
		MaxAge:   3600,
	})

	return c.JSON(fiber.Map{
		"csrf_token": token,
	})
}

// ============================================================
//  AUDIT LOGS
// ============================================================

// GetAdminLogs returns admin action logs
func (h *AdminHandler) GetAdminLogs(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))

	logs, total, err := h.service.GetAdminLogs(c.Context(), page, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data": logs,
		"pagination": fiber.Map{
			"page":  page,
			"limit": limit,
			"total": total,
			"pages": (total + int64(limit) - 1) / int64(limit),
		},
	})
}

// ============================================================
//  AUTHOR APPLICATIONS
// ============================================================

// GetAuthorApplications returns applications filtered by status via
// ?status=pending|approved|rejected|all (defaults to "pending").
func (h *AdminHandler) GetAuthorApplications(c *fiber.Ctx) error {
	status := c.Query("status", "pending")
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))

	apps, total, err := h.service.GetAuthorApplications(c.Context(), status, page, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data": apps,
		"pagination": fiber.Map{
			"page":  page,
			"limit": limit,
			"total": total,
			"pages": (total + int64(limit) - 1) / int64(limit),
		},
	})
}

// GetAuthorApplicationDetail returns a single application by ID, including
// the server-local path to the uploaded ID document — visible only to
// admins through this endpoint (not exposed in the public User/JSON output).
func (h *AdminHandler) GetAuthorApplicationDetail(c *fiber.Ctx) error {
	id := c.Params("id")

	app, err := h.service.GetAuthorApplicationByID(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	if app == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Application not found"})
	}

	return c.JSON(fiber.Map{
		"data": fiber.Map{
			"id":               app.ID,
			"user_id":          app.UserID,
			"expertise":        app.Expertise,
			"national_id":      app.NationalID,
			"status":           app.Status,
			"rejection_reason": app.RejectionReason,
			"created_at":       app.CreatedAt,
			"reviewed_at":      app.ReviewedAt,
			"document_url":     "/api/v1/admin/author-applications/" + app.ID + "/document",
		},
	})
}

// GetAuthorApplicationDocument streams the uploaded ID document image/PDF
// for admin review. Deliberately not a public route.
func (h *AdminHandler) GetAuthorApplicationDocument(c *fiber.Ctx) error {
	id := c.Params("id")

	app, err := h.service.GetAuthorApplicationByID(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	if app == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Application not found"})
	}

	return c.SendFile(app.IDDocumentPath, false)
}

// ApproveAuthorApplication approves an application and promotes the user
// to Role="prompt_author".
func (h *AdminHandler) ApproveAuthorApplication(c *fiber.Ctx) error {
	appID := c.Params("id")
	adminID, _ := c.Locals("userID").(string)

	app, err := h.service.ApproveAuthorApplication(c.Context(), appID, adminID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	_ = h.service.LogAdminAction(
		c.Context(), adminID, "approve_author_application", "author_application", appID,
		domain.JSONMap{"user_id": app.UserID},
	)

	return c.JSON(fiber.Map{"message": "Application approved, user promoted to prompt_author"})
}

// RejectAuthorApplication rejects an application with an optional reason.
func (h *AdminHandler) RejectAuthorApplication(c *fiber.Ctx) error {
	appID := c.Params("id")
	adminID, _ := c.Locals("userID").(string)

	var req struct {
		Reason string `json:"reason"`
	}
	_ = c.BodyParser(&req) // reason is optional

	app, err := h.service.RejectAuthorApplication(c.Context(), appID, adminID, req.Reason)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	_ = h.service.LogAdminAction(
		c.Context(), adminID, "reject_author_application", "author_application", appID,
		domain.JSONMap{"user_id": app.UserID, "reason": req.Reason},
	)

	return c.JSON(fiber.Map{"message": "Application rejected"})
}