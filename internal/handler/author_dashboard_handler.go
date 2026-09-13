package handler

import (
	"os"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/microcosm-cc/bluemonday"
	"gorm.io/gorm"

	"promt-market/internal/repository"
	"promt-market/internal/service"
)

type AuthorDashboardHandler struct {
	service   *service.AuthorDashboardService
	sanitizer *bluemonday.Policy
}

func NewAuthorDashboardHandler(db *gorm.DB) *AuthorDashboardHandler {
	userRepo := repository.NewUserRepository(db)
	promptRepo := repository.NewPromptRepository(db)
	orderRepo := repository.NewOrderRepository(db)
	viewRepo := repository.NewPromptViewRepository(db)
	editRepo := repository.NewPromptEditRepository(db)

	viewService := service.NewPromptViewService(viewRepo)
	purchaseSvc := service.NewPurchaseService(orderRepo, promptRepo, userRepo)

	// Keep this if your handler actually uses PromptService; otherwise delete.
	_ = service.NewPromptService(promptRepo, userRepo, viewService, purchaseSvc)

	return &AuthorDashboardHandler{
		service:   service.NewAuthorDashboardService(userRepo, promptRepo, orderRepo, viewRepo, editRepo),
		sanitizer: bluemonday.StrictPolicy(),
	}
}

// ============================================================
//  STATS
// ============================================================

// GetDashboardStats returns aggregate sales, revenue, views, and prompt counts.
func (h *AuthorDashboardHandler) GetDashboardStats(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)

	stats, err := h.service.GetStats(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"data": stats})
}

// GetViewsChart returns daily views over the last N days (?days=30).
func (h *AuthorDashboardHandler) GetViewsChart(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)
	days, _ := strconv.Atoi(c.Query("days", "30"))

	points, err := h.service.ViewsChart(c.Context(), userID, days)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"data": points})
}

// GetRevenueChart returns daily paid orders over the last N days.
func (h *AuthorDashboardHandler) GetRevenueChart(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)
	days, _ := strconv.Atoi(c.Query("days", "30"))

	points, err := h.service.RevenueChart(c.Context(), userID, days)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"data": points})
}

// ============================================================
//  PROFILE
// ============================================================

func (h *AuthorDashboardHandler) GetProfile(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)

	u, err := h.service.GetProfile(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"data": u})
}

// UpdateProfile saves full_name / bio / expertise. All fields sanitized.
func (h *AuthorDashboardHandler) UpdateProfile(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)

	var req service.UpdateProfileRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	// Sanitize string fields before they hit the service.
	if req.FullName != nil {
		v := h.sanitizer.Sanitize(*req.FullName)
		req.FullName = &v
	}
	if req.Bio != nil {
		v := h.sanitizer.Sanitize(*req.Bio)
		req.Bio = &v
	}
	if req.Expertise != nil {
		v := h.sanitizer.Sanitize(*req.Expertise)
		req.Expertise = &v
	}

	u, err := h.service.UpdateProfile(c.Context(), userID, &req)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"message": "Profile updated", "data": u})
}

// UploadAvatar accepts a single image file and sets it as the user avatar.
func (h *AuthorDashboardHandler) UploadAvatar(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)

	up := NewUploadHandler("./uploads", baseURLFromEnv())
	file, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "فایل ارسال نشده است"})
	}

	// Avatars go into uploads/avatars/<userID>/ and are compressed
	// automatically (JPEG q85, max 1200px) by saveImageAs.
	url, err := up.saveImageAs(file, "avatars", userID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	if err := h.service.UpdateAvatar(c.Context(), userID, url); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"message": "Avatar updated",
		"data":    fiber.Map{"avatar": url},
	})
}

// ============================================================
//  PAUSE / UNPAUSE
// ============================================================

func (h *AuthorDashboardHandler) PausePrompt(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)
	promptID := c.Params("id")

	prompt, err := h.service.SetPaused(c.Context(), userID, promptID, true)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"message": "پرامپت موقتاً پنهان شد", "data": prompt})
}

func (h *AuthorDashboardHandler) UnpausePrompt(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)
	promptID := c.Params("id")

	prompt, err := h.service.SetPaused(c.Context(), userID, promptID, false)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"message": "پرامپت دوباره نمایش داده می‌شود", "data": prompt})
}

// ============================================================
//  EDIT PROPOSAL
// ============================================================

// SubmitEdit accepts a JSON body of fields to change and stores it as a
// pending proposal. The live prompt is untouched until admin approval.
func (h *AuthorDashboardHandler) SubmitEdit(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)
	promptID := c.Params("id")

	var changes map[string]interface{}
	if err := c.BodyParser(&changes); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	// Sanitize any string values inside the payload.
	for k, v := range changes {
		if s, ok := v.(string); ok {
			changes[k] = h.sanitizer.Sanitize(s)
		}
	}

	edit, err := h.service.SubmitEdit(c.Context(), userID, promptID, changes)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "پیشنهاد ویرایش ثبت شد و در انتظار تایید ادمین است",
		"data":    edit,
	})
}

// ============================================================
//  ADMIN HANDLERS (mounted under /admin)
// ============================================================

func (h *AuthorDashboardHandler) AdminListEdits(c *fiber.Ctx) error {
	status := c.Query("status", "pending")
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))

	edits, total, err := h.service.ListEditsByStatus(c.Context(), status, page, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{
		"data": edits,
		"pagination": fiber.Map{
			"page":  page,
			"limit": limit,
			"total": total,
			"pages": (total + int64(limit) - 1) / int64(limit),
		},
	})
}

func (h *AuthorDashboardHandler) AdminApproveEdit(c *fiber.Ctx) error {
	editID := c.Params("id")
	if err := h.service.ApproveEdit(c.Context(), editID); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"message": "Edit approved and applied"})
}

func (h *AuthorDashboardHandler) AdminRejectEdit(c *fiber.Ctx) error {
	editID := c.Params("id")

	var req struct {
		Note string `json:"note"`
	}
	_ = c.BodyParser(&req)

	if err := h.service.RejectEdit(c.Context(), editID, req.Note); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"message": "Edit rejected"})
}

// baseURLFromEnv is defined here so the handler compiles standalone. It
// mirrors the same fallback used in routes.
// baseURLFromEnv returns the server's public base URL for building file
// links. Reads BASE_URL from the environment with a sensible local default.
func baseURLFromEnv() string {
	if v := os.Getenv("BASE_URL"); v != "" {
		return v
	}
	return "http://localhost:8080"
}