package handler

import (
	"strconv"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
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

func NewAdminHandler(db *gorm.DB) *AdminHandler {
	adminRepo := repository.NewAdminRepository(db)
	userRepo := repository.NewUserRepository(db)
	promptRepo := repository.NewPromptRepository(db)
	orderRepo := repository.NewOrderRepository(db)
	adminService := service.NewAdminService(adminRepo, userRepo, promptRepo, orderRepo, db)

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

// ApprovePrompt approves a prompt
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

// RejectPrompt rejects a prompt
func (h *AdminHandler) RejectPrompt(c *fiber.Ctx) error {
	promptID := c.Params("id")

	if err := h.service.RejectPrompt(c.Context(), promptID); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Prompt rejected successfully",
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