package service

import (
	"context"
	"errors"

	"promt-market/internal/domain"
	"promt-market/internal/repository"
)

// PurchaseService exposes read-only views over a user's paid orders:
//   - ListPurchases: for the "my purchases" page
//   - HasAccess:     for gating prompt content
//
// Keeping these reads in one place means the "has access?" rule is defined
// exactly once and reused by both the purchases listing and the prompt
// detail endpoint.
type PurchaseService struct {
	orderRepo  *repository.OrderRepository
	promptRepo *repository.PromptRepository
	userRepo   *repository.UserRepository
}

func NewPurchaseService(
	orderRepo *repository.OrderRepository,
	promptRepo *repository.PromptRepository,
	userRepo *repository.UserRepository,
) *PurchaseService {
	return &PurchaseService{
		orderRepo:  orderRepo,
		promptRepo: promptRepo,
		userRepo:   userRepo,
	}
}

// AccessDecision explains whether a viewer can see a prompt's content.
type AccessDecision struct {
	Allowed bool
	Reason  string // "purchased" | "free" | "owner" | "admin" | ""
}

// CanAccessPrompt decides whether userID may see content/instructions of the
// given prompt. The rule order matters:
//
//  1. No user (anonymous) → only free prompts are visible.
//  2. Admin → always allowed.
//  3. Owner (seller) → allowed (they authored it).
//  4. Free prompt (price == 0) → allowed.
//  5. Has a paid order → allowed.
//  6. Otherwise → denied.
//
// The function never errors on "not allowed"; that's a valid state. Errors
// are reserved for DB failures.
func (s *PurchaseService) CanAccessPrompt(ctx context.Context, userID, promptID string, prompt *domain.Prompt) (*AccessDecision, error) {
	if prompt == nil {
		return &AccessDecision{Allowed: false}, nil
	}

	// Free prompt: no login needed.
	if prompt.Price <= 0 {
		return &AccessDecision{Allowed: true, Reason: "free"}, nil
	}

	if userID == "" {
		return &AccessDecision{Allowed: false}, nil
	}

	// Owner check (also lets the author preview their own drafts).
	if prompt.SellerID == userID {
		return &AccessDecision{Allowed: true, Reason: "owner"}, nil
	}

	// Admin check.
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user != nil && user.Role == "admin" {
		return &AccessDecision{Allowed: true, Reason: "admin"}, nil
	}

	// Purchase check.
	paid, err := s.orderRepo.FindPaidOrder(ctx, userID, promptID)
	if err != nil {
		return nil, err
	}
	if paid != nil {
		return &AccessDecision{Allowed: true, Reason: "purchased"}, nil
	}

	return &AccessDecision{Allowed: false}, nil
}

// ListPurchases returns paid orders for the user with prompt data.
func (s *PurchaseService) ListPurchases(ctx context.Context, userID string, page, limit int) ([]domain.Order, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	return s.orderRepo.ListPaidByBuyer(ctx, userID, limit, (page-1)*limit)
}

// GetOwnedOrder returns an order only if it belongs to userID and is paid.
// Used by the "view purchased content" endpoint to fetch a specific order
// while enforcing ownership in one place.
func (s *PurchaseService) GetOwnedOrder(ctx context.Context, userID, orderID string) (*domain.Order, error) {
	order, err := s.orderRepo.FindByID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if order == nil {
		return nil, errors.New("سفارش یافت نشد")
	}
	if order.BuyerID != userID {
		return nil, errors.New("شما مالک این سفارش نیستید")
	}
	if order.Status != domain.OrderStatusPaid {
		return nil, errors.New("این سفارش پرداخت نشده است")
	}
	return order, nil
}