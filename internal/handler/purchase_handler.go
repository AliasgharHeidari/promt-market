package handler

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"promt-market/internal/repository"
	"promt-market/internal/service"
)

type PurchaseHandler struct {
	service *service.PurchaseService
}

func NewPurchaseHandler(db *gorm.DB) *PurchaseHandler {
	orderRepo := repository.NewOrderRepository(db)
	promptRepo := repository.NewPromptRepository(db)
	userRepo := repository.NewUserRepository(db)
	purchaseSvc := service.NewPurchaseService(orderRepo, promptRepo, userRepo)

	return &PurchaseHandler{service: purchaseSvc}
}

// ListMyPurchases returns the current user's paid orders.
//
// GET /api/v1/my-purchases?page=1&limit=20
func (h *PurchaseHandler) ListMyPurchases(c *fiber.Ctx) error {
	userID, _ := c.Locals("userID").(string)
	if userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))

	orders, total, err := h.service.ListPurchases(c.Context(), userID, page, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	// The response is intentionally a minimal shape: buyer-facing pages
	// don't need admin metadata. The content itself is fetched separately
	// through /prompts/:slug so a stale purchase listing never leaks it.
	type purchaseItem struct {
		OrderID     string `json:"order_id"`
		PromptID    string `json:"prompt_id"`
		PromptSlug  string `json:"prompt_slug"`
		Title       string `json:"title"`
		CoverImage  string `json:"cover_image"`
		SellerName  string `json:"seller_name"`
		Amount      int64  `json:"amount"`
		PurchasedAt string `json:"purchased_at"`
	}

	items := make([]purchaseItem, 0, len(orders))
	for _, o := range orders {
		// Prompt may be soft-deleted; the FK still resolves. Guard against
		// an empty prompt row anyway.
		slug := ""
		sellerName := ""
		if o.Prompt.ID != "" {
			slug = o.Prompt.Slug
			if o.Prompt.Seller.FullName != "" {
				sellerName = o.Prompt.Seller.FullName
			}
		}

		paidAt := ""
		if o.PaidAt != nil {
			paidAt = o.PaidAt.Format("2006-01-02T15:04:05Z07:00")
		}

		items = append(items, purchaseItem{
			OrderID:     o.ID,
			PromptID:    o.PromptID,
			PromptSlug:  slug,
			Title:       o.PromptTitle,
			CoverImage:  o.PromptCover,
			SellerName:  sellerName,
			Amount:      o.Amount,
			PurchasedAt: paidAt,
		})
	}

	return c.JSON(fiber.Map{
		"data": items,
		"pagination": fiber.Map{
			"page":  page,
			"limit": limit,
			"total": total,
			"pages": (total + int64(limit) - 1) / int64(limit),
		},
	})
}

// GetPurchaseDetail returns one order's summary — still no content, just
// the buyer-facing metadata. Used by a "purchase detail" page if needed.
//
// GET /api/v1/my-purchases/:id
func (h *PurchaseHandler) GetPurchaseDetail(c *fiber.Ctx) error {
	userID, _ := c.Locals("userID").(string)
	if userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	orderID := c.Params("id")
	order, err := h.service.GetOwnedOrder(c.Context(), userID, orderID)
	if err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"data": fiber.Map{
			"order_id":    order.ID,
			"prompt_id":   order.PromptID,
			"prompt_slug": order.Prompt.Slug,
			"title":       order.PromptTitle,
			"cover_image": order.PromptCover,
			"amount":      order.Amount,
			"paid_at":     order.PaidAt,
		},
	})
}

// GetPurchaseContent returns the protected content of a purchased prompt.
// This is the endpoint the buyer hits when they click "view content".
//
// Security:
//   - Ownership is enforced by GetOwnedOrder (buyer must own a paid order).
//   - The content is re-fetched from the live prompt row, not from a
//     snapshot, so seller edits are reflected. If the prompt was deleted,
//     the endpoints still return the snapshot fields it stored.
//
// GET /api/v1/my-purchases/:id/content
func (h *PurchaseHandler) GetPurchaseContent(c *fiber.Ctx) error {
	userID, _ := c.Locals("userID").(string)
	if userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	orderID := c.Params("id")
	order, err := h.service.GetOwnedOrder(c.Context(), userID, orderID)
	if err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": err.Error()})
	}

	// The prompt may have been deleted after purchase. We still have the FK
	// on the order, so the row is resolvable. If it was hard-deleted (which
	// this codebase doesn't do), we return only snapshot data.
	if order.Prompt.ID == "" {
		return c.JSON(fiber.Map{
			"data": fiber.Map{
				"title":     order.PromptTitle,
				"content":   "",
				"available": false,
				"message":   "این پرامپت دیگر در دسترس نیست",
			},
		})
	}

	return c.JSON(fiber.Map{
		"data": fiber.Map{
			"prompt_id":    order.Prompt.ID,
			"prompt_slug":  order.Prompt.Slug,
			"title":        order.Prompt.Title,
			"description":  order.Prompt.Description,
			"content":      order.Prompt.Content,
			"instructions": order.Prompt.Instructions,
			"demo_output":  order.Prompt.DemoOutput,
			"cover_image":  order.Prompt.CoverImage,
			"images":       order.Prompt.Images,
			"available":    true,
		},
	})
}