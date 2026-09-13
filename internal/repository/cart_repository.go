package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"promt-market/internal/domain"
)

type CartRepository struct {
	db *gorm.DB
}

func NewCartRepository(db *gorm.DB) *CartRepository {
	return &CartRepository{db: db}
}

// Add inserts a cart item. Idempotent by design: if a row already exists for
// (user, prompt), the ON CONFLICT clause makes the insert a no-op instead of
// returning an error. This means concurrent "Add to cart" clicks don't cause
// a race — the result is a single row either way.
func (r *CartRepository) Add(ctx context.Context, userID, promptID string) error {
	item := &domain.CartItem{
		UserID:   userID,
		PromptID: promptID,
	}
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}, {Name: "prompt_id"}},
			DoNothing: true,
		}).
		Create(item).Error
}

// Remove deletes a single cart row. Scoped by user_id so a user can never
// delete another user's cart item even if they know the prompt id.
func (r *CartRepository) Remove(ctx context.Context, userID, promptID string) error {
	return r.db.WithContext(ctx).
		Where("user_id = ? AND prompt_id = ?", userID, promptID).
		Delete(&domain.CartItem{}).Error
}

// Clear removes all cart rows for a user. Used after a successful checkout.
func (r *CartRepository) Clear(ctx context.Context, userID string) error {
	return r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Delete(&domain.CartItem{}).Error
}

// ListByUser returns the user's cart items with the Prompt preloaded. The
// handler/service decides which fields to expose.
func (r *CartRepository) ListByUser(ctx context.Context, userID string) ([]domain.CartItem, error) {
	var items []domain.CartItem
	err := r.db.WithContext(ctx).
		Preload("Prompt").
		Preload("Prompt.Seller").
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&items).Error
	return items, err
}

// Exists reports whether a prompt is already in the user's cart.
func (r *CartRepository) Exists(ctx context.Context, userID, promptID string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&domain.CartItem{}).
		Where("user_id = ? AND prompt_id = ?", userID, promptID).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// CountByUser returns the number of items in the user's cart. Used for the
// header badge.
func (r *CartRepository) CountByUser(ctx context.Context, userID string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&domain.CartItem{}).
		Where("user_id = ?", userID).
		Count(&count).Error
	return count, err
}

// FindByUserAndPrompt returns a single cart row, or nil if not present.
// Used by the service for validation before Add (though Add itself is safe).
func (r *CartRepository) FindByUserAndPrompt(ctx context.Context, userID, promptID string) (*domain.CartItem, error) {
	var item domain.CartItem
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND prompt_id = ?", userID, promptID).
		First(&item).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &item, nil
}