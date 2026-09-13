package service

import (
	"context"
	"errors"

	"promt-market/internal/domain"
	"promt-market/internal/repository"
)

// CartService handles the "add / list / remove" workflow for the shopping
// cart. It performs all business-rule checks (prompt status, ownership,
// already-purchased, etc.) before touching the DB so error messages are
// user-facing and precise.
type CartService struct {
	cartRepo   *repository.CartRepository
	promptRepo *repository.PromptRepository
	orderRepo  *repository.OrderRepository
}

func NewCartService(
	cartRepo *repository.CartRepository,
	promptRepo *repository.PromptRepository,
	orderRepo *repository.OrderRepository,
) *CartService {
	return &CartService{
		cartRepo:   cartRepo,
		promptRepo: promptRepo,
		orderRepo:  orderRepo,
	}
}

// Add adds a prompt to the user's cart after validating business rules.
//
// Rejection rules (each returns a specific message):
//  1. prompt doesn't exist                → "پرامپت یافت نشد"
//  2. prompt not approved                  → "این پرامپت در حال حاضر قابل خریداری نیست"
//  3. prompt is paused                     → "این پرامپت موقتاً غیرفعال است"
//  4. user is the seller                   → "نمی‌توانید پرامپت خودتان را بخرید"
//  5. user already purchased this prompt   → "شما قبلاً این پرامپت را خریداری کرده‌اید"
//  6. prompt already in cart               → "این پرامپت قبلاً به سبد اضافه شده است"
func (s *CartService) Add(ctx context.Context, userID, promptID string) error {
	if userID == "" || promptID == "" {
		return errors.New("درخواست نامعتبر است")
	}

	prompt, err := s.promptRepo.FindByID(ctx, promptID)
	if err != nil {
		return err
	}
	if prompt == nil {
		return errors.New("پرامپت یافت نشد")
	}
	if prompt.Status != "approved" {
		return errors.New("این پرامپت در حال حاضر قابل خریداری نیست")
	}
	if prompt.IsPaused {
		return errors.New("این پرامپت موقتاً غیرفعال است")
	}
	if prompt.SellerID == userID {
		return errors.New("نمی‌توانید پرامپت خودتان را بخرید")
	}

	// Already purchased?
	paid, err := s.orderRepo.FindPaidOrder(ctx, userID, promptID)
	if err != nil {
		return err
	}
	if paid != nil {
		return errors.New("شما قبلاً این پرامپت را خریداری کرده‌اید")
	}

	// Already in cart?
	exists, err := s.cartRepo.Exists(ctx, userID, promptID)
	if err != nil {
		return err
	}
	if exists {
		return errors.New("این پرامپت قبلاً به سبد اضافه شده است")
	}

	return s.cartRepo.Add(ctx, userID, promptID)
}

// Remove deletes one item from the user's cart. Idempotent: removing an
// item that isn't there is not an error.
func (s *CartService) Remove(ctx context.Context, userID, promptID string) error {
	if userID == "" || promptID == "" {
		return errors.New("درخواست نامعتبر است")
	}
	return s.cartRepo.Remove(ctx, userID, promptID)
}

// Clear empties the user's cart.
func (s *CartService) Clear(ctx context.Context, userID string) error {
	return s.cartRepo.Clear(ctx, userID)
}

// List returns the cart contents with only summary fields exposed. Each item
// is annotated with Unavailable/AlreadyPurchased flags so the UI can render
// the row appropriately.
//
// Unavailability reasons:
//   - prompt was soft-deleted (DeletedAt != nil)
//   - prompt status is not "approved" anymore
//   - prompt is paused
//   - user purchased the prompt in a different session/tab
func (s *CartService) List(ctx context.Context, userID string) (*domain.CartResponse, error) {
	items, err := s.cartRepo.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	out := make([]domain.CartItemView, 0, len(items))
	var total int64

	for _, it := range items {
		p := it.Prompt

		// Compute the effective price: discount wins if set and non-zero.
		finalPrice := p.Price
		if p.DiscountPrice > 0 && p.DiscountPrice < p.Price {
			finalPrice = p.DiscountPrice
		}

		view := domain.CartItemView{
			PromptID:      it.PromptID,
			Title:         p.Title,
			CoverImage:    p.CoverImage,
			Category:      p.Category,
			Price:         p.Price,
			DiscountPrice: p.DiscountPrice,
			FinalPrice:    finalPrice,
			AddedAt:       it.CreatedAt,
		}
		if p.Seller.FullName != "" {
			view.SellerName = p.Seller.FullName
		}

		// Detect unavailability. Soft-deleted prompt rows still resolve
		// through the FK, so we check both DeletedAt and Status.
		switch {
		case p.DeletedAt != nil:
			view.Unavailable = true
			view.UnavailableMsg = "این پرامپت توسط مدیر حذف شده است"
		case p.Status != "approved":
			view.Unavailable = true
			view.UnavailableMsg = "این پرامپت دیگر قابل خریداری نیست"
		case p.IsPaused:
			view.Unavailable = true
			view.UnavailableMsg = "این پرامپت موقتاً غیرفعال شده است"
		}

		// Already purchased (e.g. user bought it on another device).
		if !view.Unavailable {
			if paid, _ := s.orderRepo.FindPaidOrder(ctx, userID, it.PromptID); paid != nil {
				view.AlreadyPurchased = true
			}
		}

		// Only count available + not-yet-purchased items toward the total.
		// This prevents checkout from summing items the user can't buy.
		if !view.Unavailable && !view.AlreadyPurchased {
			total += finalPrice
		}

		out = append(out, view)
	}

	return &domain.CartResponse{
		Data:        out,
		TotalAmount: total,
		Count:       len(items),
	}, nil
}

// Count returns the number of cart items (for the header badge).
func (s *CartService) Count(ctx context.Context, userID string) (int64, error) {
	return s.cartRepo.CountByUser(ctx, userID)
}

// ListPromptIDs returns just the prompt IDs in the user's cart. Used by the
// frontend to know which prompts to show the "already in cart" state for.
func (s *CartService) ListPromptIDs(ctx context.Context, userID string) ([]string, error) {
	items, err := s.cartRepo.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(items))
	for _, it := range items {
		ids = append(ids, it.PromptID)
	}
	return ids, nil
}