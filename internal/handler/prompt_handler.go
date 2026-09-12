package handler

import (
	"strconv"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"promt-market/internal/domain"
	"promt-market/internal/repository"
	"promt-market/internal/service"
)

type PromptHandler struct {
	service   *service.PromptService
	validator *validator.Validate
}

func NewPromptHandler(db *gorm.DB) *PromptHandler {
	promptRepo := repository.NewPromptRepository(db)
	userRepo := repository.NewUserRepository(db)
	viewRepo := repository.NewPromptViewRepository(db)
	viewService := service.NewPromptViewService(viewRepo)
	promptService := service.NewPromptService(promptRepo, userRepo, viewService)

	return &PromptHandler{
		service:   promptService,
		validator: validator.New(),
	}
}

func (h *PromptHandler) CreatePrompt(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)

	var req domain.CreatePromptRequest
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

	prompt, err := h.service.Create(c.Context(), userID, &req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Prompt created successfully",
		"data":    prompt,
	})
}

func (h *PromptHandler) GetPromptBySlug(c *fiber.Ctx) error {
	slug := c.Params("slug")

	prompt, err := h.service.GetBySlug(c.Context(), slug)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data": prompt,
	})
}

func (h *PromptHandler) GetAllPrompts(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))

	prompts, total, err := h.service.GetAll(c.Context(), page, limit)
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

func (h *PromptHandler) GetMyPrompts(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))

	prompts, total, err := h.service.GetBySeller(c.Context(), userID, page, limit)
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

func (h *PromptHandler) UpdatePrompt(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)
	promptID := c.Params("id")

	var req domain.UpdatePromptRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	prompt, err := h.service.Update(c.Context(), promptID, userID, &req)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Prompt updated successfully",
		"data":    prompt,
	})
}

func (h *PromptHandler) DeletePrompt(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)
	promptID := c.Params("id")

	// ✅ Ownership check for regular users
	if err := h.service.Delete(c.Context(), promptID, userID); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Prompt deleted successfully",
	})
}

func (h *PromptHandler) SearchPrompts(c *fiber.Ctx) error {
	var filter domain.PromptFilter

	if err := c.QueryParser(&filter); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid query parameters",
		})
	}

	if err := h.validator.Struct(filter); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	prompts, total, err := h.service.Search(c.Context(), &filter)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data": prompts,
		"pagination": fiber.Map{
			"page":  filter.Page,
			"limit": filter.Limit,
			"total": total,
			"pages": (total + int64(filter.Limit) - 1) / int64(filter.Limit),
		},
		"filters": fiber.Map{
			"category":   filter.Category,
			"min_price":  filter.MinPrice,
			"max_price":  filter.MaxPrice,
			"difficulty": filter.Difficulty,
			"language":   filter.Language,
			"search":     filter.Search,
			"sort_by":    filter.SortBy,
			"sort_order": filter.SortOrder,
		},
	})
}

func (h *PromptHandler) GetCategories(c *fiber.Ctx) error {
	var categories []string

	err := h.service.GetCategories(c.Context(), &categories)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data": categories,
	})
}


// GetPromptByID returns a prompt by ID (for admin review)
func (h *PromptHandler) GetPromptByID(c *fiber.Ctx) error {
	id := c.Params("id")

	// ✅ تغییر: استفاده از FindByIDForAdmin به جای FindByID
	prompt, err := h.service.GetByIDForAdmin(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	if prompt == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Prompt not found",
		})
	}

	return c.JSON(fiber.Map{
		"data": prompt,
	})
}

// GetMyPromptByID returns a single prompt owned by the authenticated user,
// regardless of its status (pending/approved/rejected/deleted). Used by the
// author dashboard to preview their own drafts and rejected submissions.
//
// Admins should use the /admin/prompts/:id route via GetPromptByID instead.
func (h *PromptHandler) GetMyPromptByID(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)
	promptID := c.Params("id")

	prompt, err := h.service.GetByIDForAdmin(c.Context(), promptID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	if prompt == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Prompt not found",
		})
	}

	// Ownership check: an author can only view their own prompt through
	// this endpoint. Prevents sellers from peeking at other users' pending
	// or rejected drafts.
	if prompt.SellerID != userID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "You are not the owner of this prompt",
		})
	}

	return c.JSON(fiber.Map{
		"data": prompt,
	})
}