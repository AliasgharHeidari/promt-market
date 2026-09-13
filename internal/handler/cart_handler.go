package handler

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"promt-market/internal/repository"
	"promt-market/internal/service"
)

type CartHandler struct {
	service *service.CartService
}

func NewCartHandler(db *gorm.DB) *CartHandler {
	cartRepo := repository.NewCartRepository(db)
	promptRepo := repository.NewPromptRepository(db)
	orderRepo := repository.NewOrderRepository(db)
	cartService := service.NewCartService(cartRepo, promptRepo, orderRepo)

	return &CartHandler{service: cartService}
}

// GetCart returns the user's cart. Response includes only summary fields
// (title, price, cover) — never content/instructions.
//
// GET /api/v1/cart
func (h *CartHandler) GetCart(c *fiber.Ctx) error {
	userID, _ := c.Locals("userID").(string)
	if userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "unauthorized",
		})
	}

	cart, err := h.service.List(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "خطا در دریافت سبد خرید",
		})
	}

	return c.JSON(cart)
}

// AddToCart adds a prompt to the cart.
//
// POST /api/v1/cart/items
// Body: { "prompt_id": "uuid" }
func (h *CartHandler) AddToCart(c *fiber.Ctx) error {
	userID, _ := c.Locals("userID").(string)
	if userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "unauthorized",
		})
	}

	var body struct {
		PromptID string `json:"prompt_id"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "درخواست نامعتبر است",
		})
	}
	if body.PromptID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "شناسه پرامپت الزامی است",
		})
	}

	if err := h.service.Add(c.Context(), userID, body.PromptID); err != nil {
		// Domain-level validation errors are surfaced as 400/409 so the
		// frontend can show a specific message.
		status := fiber.StatusBadRequest
		msg := err.Error()

		// Conflict-ish cases get 409.
		if isConflictErr(msg) {
			status = fiber.StatusConflict
		}
		// Missing prompt → 404.
		if msg == "پرامپت یافت نشد" {
			status = fiber.StatusNotFound
		}

		return c.Status(status).JSON(fiber.Map{"error": msg})
	}

	// Return the fresh count so the frontend can update the header badge
	// in one round-trip.
	count, _ := h.service.Count(c.Context(), userID)

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "به سبد خرید اضافه شد",
		"count":   count,
	})
}

// RemoveFromCart deletes one item. Idempotent.
//
// DELETE /api/v1/cart/items/:prompt_id
func (h *CartHandler) RemoveFromCart(c *fiber.Ctx) error {
	userID, _ := c.Locals("userID").(string)
	if userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "unauthorized",
		})
	}

	promptID := c.Params("prompt_id")
	if promptID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "شناسه پرامپت الزامی است",
		})
	}

	if err := h.service.Remove(c.Context(), userID, promptID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "خطا در حذف آیتم",
		})
	}

	count, _ := h.service.Count(c.Context(), userID)

	return c.JSON(fiber.Map{
		"message": "از سبد خرید حذف شد",
		"count":   count,
	})
}

// ClearCart empties the cart.
//
// DELETE /api/v1/cart
func (h *CartHandler) ClearCart(c *fiber.Ctx) error {
	userID, _ := c.Locals("userID").(string)
	if userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "unauthorized",
		})
	}

	if err := h.service.Clear(c.Context(), userID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "خطا در خالی کردن سبد خرید",
		})
	}

	return c.JSON(fiber.Map{
		"message": "سبد خرید خالی شد",
		"count":   0,
	})
}

// GetCartPromptIDs returns just the list of prompt IDs in the cart. Used by
// the storefront overlay to show "already in cart" state without fetching
// the full cart payload.
//
// GET /api/v1/cart/prompt-ids
func (h *CartHandler) GetCartPromptIDs(c *fiber.Ctx) error {
	userID, _ := c.Locals("userID").(string)
	if userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "unauthorized",
		})
	}

	ids, err := h.service.ListPromptIDs(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "خطا در دریافت سبد خرید",
		})
	}
	return c.JSON(fiber.Map{"data": ids})
}

// isConflictErr classifies a subset of domain errors as HTTP 409.
// Kept as a small helper so all "this already exists / can't be done" cases
// have a consistent status code.
func isConflictErr(msg string) bool {
	switch msg {
	case "این پرامپت قبلاً به سبد اضافه شده است",
		"شما قبلاً این پرامپت را خریداری کرده‌اید":
		return true
	}
	return false
}

// keep errors import referenced (used by future handlers).
var _ = errors.New