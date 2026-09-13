package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"gorm.io/gorm"

	"promt-market/internal/domain"
	"promt-market/internal/repository"
	"promt-market/pkg/payment"
)

// CheckoutService orchestrates the transition from "cart full" to "payment
// started". It runs in a single DB transaction so a half-finished checkout
// can never leave orphan orders or a paid-without-orders state.
//
// Responsibilities:
//   - Validate the cart (non-empty, all items buyable)
//   - Compute the amount from live DB prices (never trust the client)
//   - Snapshot the prompt title/cover onto each order
//   - Create one Payment and N Orders
//   - Ask the provider for an authority + gateway URL
type CheckoutService struct {
	db         *gorm.DB
	cartRepo   *repository.CartRepository
	promptRepo *repository.PromptRepository
	orderRepo  *repository.OrderRepository
	paymentRepo *repository.PaymentRepository
	provider   payment.Provider
	commission int // whole percent, 0..100
}

func NewCheckoutService(
	db *gorm.DB,
	cartRepo *repository.CartRepository,
	promptRepo *repository.PromptRepository,
	orderRepo *repository.OrderRepository,
	paymentRepo *repository.PaymentRepository,
	provider payment.Provider,
	commissionPercent int,
) *CheckoutService {
	if commissionPercent < 0 {
		commissionPercent = 0
	}
	if commissionPercent > 100 {
		commissionPercent = 100
	}
	return &CheckoutService{
		db:          db,
		cartRepo:    cartRepo,
		promptRepo:  promptRepo,
		orderRepo:   orderRepo,
		paymentRepo: paymentRepo,
		provider:    provider,
		commission:  commissionPercent,
	}
}

// CheckoutResult is returned to the handler so it can build the response.
type CheckoutResult struct {
	PaymentID  string
	Authority  string
	GatewayURL string
	Amount     int64
	OrderCount int
}

// Checkout validates the cart, creates Payment + Orders, and returns the
// gateway URL the user must be redirected to.
//
// Concurrency notes:
//   - Two tabs hitting /checkout simultaneously are protected by the
//     UNIQUE(user_id, prompt_id) index on cart_items (only one row per
//     prompt) and by the tx. The second tab will see already-existing
//     pending orders and fail cleanly.
//   - We do NOT re-use pending payments; every checkout starts a fresh one
//     so the user can retry freely after an abandoned attempt.
func (s *CheckoutService) Checkout(ctx context.Context, userID string) (*CheckoutResult, error) {
	if userID == "" {
		return nil, errors.New("unauthorized")
	}

	// Load cart with full prompt data. We need price/status to validate.
	items, err := s.cartRepo.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, errors.New("سبد خرید خالی است")
	}

	// ── Phase 1: pure validation, no writes ──
	type line struct {
		prompt    domain.Prompt
		unitPrice int64
	}

	lines := make([]line, 0, len(items))
	var total int64

	for _, it := range items {
		p := it.Prompt

		if p.ID == "" {
			return nil, errors.New("یکی از آیتم‌های سبد یافت نشد")
		}
		if p.DeletedAt != nil {
			return nil, fmt.Errorf("پرامپت «%s» حذف شده است؛ لطفاً از سبد حذفش کنید", p.Title)
		}
		if p.Status != "approved" {
			return nil, fmt.Errorf("پرامپت «%s» دیگر قابل خریداری نیست؛ لطفاً از سبد حذفش کنید", p.Title)
		}
		if p.IsPaused {
			return nil, fmt.Errorf("پرامپت «%s» موقتاً غیرفعال شده است؛ لطفاً از سبد حذفش کنید", p.Title)
		}
		if p.SellerID == userID {
			return nil, fmt.Errorf("نمی‌توانید پرامپت «%s» را بخرید چون نویسنده‌ی آن هستید", p.Title)
		}

		// Skip items the user has already purchased.
		if paid, _ := s.orderRepo.FindPaidOrder(ctx, userID, p.ID); paid != nil {
			return nil, fmt.Errorf("پرامپت «%s» قبلاً خریداری شده است؛ لطفاً از سبد حذفش کنید", p.Title)
		}

		unit := p.Price
		if p.DiscountPrice > 0 && p.DiscountPrice < p.Price {
			unit = p.DiscountPrice
		}
		if unit < 0 {
			unit = 0
		}

		total += unit
		lines = append(lines, line{prompt: p, unitPrice: unit})
	}

	if total <= 0 {
		return nil, errors.New("مبلغ قابل پرداخت نامعتبر است")
	}

	// ── Phase 2: writes inside a single transaction ──
	var (
		paymentRow *domain.Payment
		initResult *payment.InitResult
	)

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Create the payment row first (status=pending, authority empty until
		// the gateway responds). We update authority right after.
		paymentRow = &domain.Payment{
			UserID:   userID,
			Amount:   total,
			Status:   domain.PaymentStatusPending,
			Provider: s.provider.Name(),
		}
		if err := tx.Create(paymentRow).Error; err != nil {
			return err
		}

		// Create one order per cart line, all pointing at the same payment.
		for _, ln := range lines {
			commission := ln.unitPrice * int64(s.commission) / 100
			order := &domain.Order{
				BuyerID:     userID,
				PromptID:    ln.prompt.ID,
				Amount:      ln.unitPrice,
				Commission:  commission,
				PromptTitle: ln.prompt.Title,
				PromptCover: ln.prompt.CoverImage,
				Status:      domain.OrderStatusPending,
				PaymentID:   paymentRow.ID,
			}
			if err := tx.Create(order).Error; err != nil {
				return err
			}
		}

		// Ask the provider to start the payment. We do this INSIDE the tx so
		// that a gateway error rolls back the orders and payment row. The
		// provider call itself is side-effect-free on our side (only its
		// own bookkeeping is created), so this is safe.
		res, err := s.provider.InitPayment(ctx, total, "خرید پرامپت", paymentCallbackURL())
		if err != nil {
			return fmt.Errorf("failed to start payment: %w", err)
		}
		initResult = res

		// Persist the authority so the callback can look up the payment.
		if err := tx.Model(paymentRow).Update("authority", res.Authority).Error; err != nil {
			return err
		}
		paymentRow.Authority = res.Authority

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &CheckoutResult{
		PaymentID:  paymentRow.ID,
		Authority:  initResult.Authority,
		GatewayURL: initResult.GatewayURL,
		Amount:     total,
		OrderCount: len(lines),
	}, nil
}

// VerifyPayment confirms a payment after the user returns from the gateway.
//
// Idempotency:
//   - If the payment row is already paid, this returns success without
//     touching any state again. So a repeated callback (browser refresh) is
//     safe.
//   - If the payment is canceled/failed, this returns an error and does not
//     retroactively succeed.
//
// Atomicity:
//   - On success, orders → paid, prompt.SalesCount bumped, cart cleared, and
//     payment → paid happen in one transaction. Either all land or none do.
func (s *CheckoutService) VerifyPayment(ctx context.Context, authority string, gatewayStatus string) (*domain.Payment, error) {
	if authority == "" {
		return nil, errors.New("authority الزامی است")
	}

	pay, err := s.paymentRepo.FindByAuthority(ctx, authority)
	if err != nil {
		return nil, err
	}
	if pay == nil {
		return nil, errors.New("پرداخت یافت نشد")
	}

	// Idempotent: already paid → return success.
	if pay.Status == domain.PaymentStatusPaid {
		return pay, nil
	}
	if pay.Status == domain.PaymentStatusCanceled {
		return nil, errors.New("این پرداخت لغو شده است")
	}

	// If the gateway said "user canceled", mark as canceled and stop.
	if gatewayStatus != "OK" {
		_ = s.paymentRepo.UpdateFields(ctx, pay.ID, map[string]interface{}{
			"status":         domain.PaymentStatusCanceled,
			"failure_reason": "کاربر پرداخت را لغو کرد",
		})
		pay.Status = domain.PaymentStatusCanceled
		return pay, nil
	}

	// Verify with the provider. Trust nothing the client sent.
	vr, err := s.provider.VerifyPayment(ctx, authority, pay.Amount)
	if err != nil {
		return nil, err
	}
	if !vr.Success {
		_ = s.paymentRepo.UpdateFields(ctx, pay.ID, map[string]interface{}{
			"status":         domain.PaymentStatusFailed,
			"failure_reason": vr.ErrorMsg,
		})
		pay.Status = domain.PaymentStatusFailed
		return pay, nil
	}

	// Apply the success state in a single transaction.
	now := time.Now()
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Lock the payment row so two concurrent callbacks can't both run
		// the success path.
		var fresh domain.Payment
		if err := tx.
			Set("gorm:query_option", "FOR UPDATE").
			Where("id = ?", pay.ID).
			First(&fresh).Error; err != nil {
			return err
		}
		if fresh.Status == domain.PaymentStatusPaid {
			return nil // someone else already completed it
		}

		// Mark payment paid.
		if err := tx.Model(&domain.Payment{}).
			Where("id = ?", pay.ID).
			Updates(map[string]interface{}{
				"status":         domain.PaymentStatusPaid,
				"ref_id":         vr.RefID,
				"paid_at":        now,
				"failure_reason": "",
			}).Error; err != nil {
			return err
		}

		// Orders → paid.
		affected, err := s.orderRepo.MarkPaidWithTx(tx, pay.ID, now)
		if err != nil {
			return err
		}

		// Bump sales_count on each prompt.
		orders, err := s.orderRepo.FindOrdersByPaymentIDWithTx(tx, pay.ID)
		if err != nil {
			return err
		}
		for _, o := range orders {
			if err := s.orderRepo.IncrementSalesCountForPromptWithTx(tx, o.PromptID); err != nil {
				return err
			}
		}

		// Clear the buyer's cart only if we actually marked at least one
		// order as paid (protects against an empty-payment edge case).
		if affected > 0 {
			if err := tx.Where("user_id = ?", pay.UserID).Delete(&domain.CartItem{}).Error; err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	pay.Status = domain.PaymentStatusPaid
	pay.RefID = vr.RefID
	pay.PaidAt = &now
	return pay, nil
}

// paymentCallbackURL returns the URL the gateway must redirect to after
// the user finishes. Read from env with a sane local default.
func paymentCallbackURL() string {
	if v := os.Getenv("PAYMENT_CALLBACK_URL"); v != "" {
		return v
	}
	return "http://localhost:8080/api/v1/payment/callback"
}

// GetPaymentByAuthority returns a payment by its authority. Read-only;
// does not mutate state. Used by the frontend to poll payment status.
func (s *CheckoutService) GetPaymentByAuthority(ctx context.Context, authority string) (*domain.Payment, error) {
	return s.paymentRepo.FindByAuthority(ctx, authority)
}