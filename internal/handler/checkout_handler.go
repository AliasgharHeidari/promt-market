package handler

import (
	"os"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"promt-market/internal/repository"
	"promt-market/internal/service"
	"promt-market/pkg/payment"
)

type CheckoutHandler struct {
	service *service.CheckoutService
}

// NewCheckoutHandler wires the checkout flow. It reads the active payment
// provider from env via payment.NewFromEnv() so the same handler works in
// dev (mock) and prod (zarinpal) without changes.
func NewCheckoutHandler(db *gorm.DB, commissionPercent int) *CheckoutHandler {
	cartRepo := repository.NewCartRepository(db)
	promptRepo := repository.NewPromptRepository(db)
	orderRepo := repository.NewOrderRepository(db)
	paymentRepo := repository.NewPaymentRepository(db)

	provider, err := payment.NewFromEnv()
	if err != nil {
		// Fail fast at startup: a bad provider config should not silently
		// fall back to a broken state.
		panic("checkout: " + err.Error())
	}

	checkoutSvc := service.NewCheckoutService(
		db,
		cartRepo,
		promptRepo,
		orderRepo,
		paymentRepo,
		provider,
		commissionPercent,
	)

	return &CheckoutHandler{service: checkoutSvc}
}

// Checkout starts a payment for the user's cart.
//
// POST /api/v1/checkout
// → 200 { payment_id, authority, gateway_url, amount, order_count }
func (h *CheckoutHandler) Checkout(c *fiber.Ctx) error {
	userID, _ := c.Locals("userID").(string)
	if userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "unauthorized",
		})
	}

	result, err := h.service.Checkout(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data": fiber.Map{
			"payment_id":  result.PaymentID,
			"authority":   result.Authority,
			"gateway_url": result.GatewayURL,
			"amount":      result.Amount,
			"order_count": result.OrderCount,
		},
	})
}

// PaymentCallback is the endpoint the gateway redirects the user to after
// they finish (or cancel) the payment. It's a GET because gateways use
// browser redirects, not AJAX.
//
// Query params:
//
//	authority — the code we received at InitPayment
//	status    — "OK" or anything else (gateway-specific; mock uses OK/NOK)
//
// On success, the user is redirected to <FRONTEND_URL>/payment-result.html
// with a status query param. On failure, same but with status=failed.
//
// NOTE: it must NOT return JSON here — the browser is following this URL,
// so a 302 redirect is what the user actually sees.
//
// GET /api/v1/payment/callback?authority=...&status=OK
func (h *CheckoutHandler) PaymentCallback(c *fiber.Ctx) error {
	authority := c.Query("authority")
	status := c.Query("status")

	frontend := os.Getenv("FRONTEND_URL")
	if frontend == "" {
		frontend = "http://localhost:5500"
	}

	if authority == "" {
		return c.Redirect(frontend+"/payment-result.html?status=failed&reason=missing_authority", fiber.StatusFound)
	}

	pay, err := h.service.VerifyPayment(c.Context(), authority, status)
	if err != nil {
		return c.Redirect(frontend+"/payment-result.html?status=failed&reason="+err.Error(), fiber.StatusFound)
	}

	if pay == nil || pay.Status != "paid" {
		// Canceled or failed — send the user back with an appropriate label.
		label := "failed"
		if pay != nil && pay.Status == "canceled" {
			label = "canceled"
		}
		return c.Redirect(frontend+"/payment-result.html?status="+label, fiber.StatusFound)
	}

	return c.Redirect(
		frontend+"/payment-result.html?status=success&ref="+pay.RefID,
		fiber.StatusFound,
	)
}

// GetPaymentByAuthority is a small helper for the frontend to poll payment
// state without going through the redirect. Useful for the result page if
// the user has the URL open in more than one tab.
//
// GET /api/v1/payment/status/:authority
func (h *CheckoutHandler) GetPaymentStatus(c *fiber.Ctx) error {
	authority := c.Params("authority")
	if authority == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "authority required"})
	}

	// We reuse VerifyPayment's lookup but avoid state mutation. A dedicated
	// repository lookup would be cleaner; for now, we call a small read-only
	// path via the service.
	pay, err := h.service.GetPaymentByAuthority(c.Context(), authority)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	if pay == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "payment not found"})
	}

	return c.JSON(fiber.Map{
		"data": fiber.Map{
			"status": pay.Status,
			"ref_id": pay.RefID,
			"amount": pay.Amount,
		},
	})
}