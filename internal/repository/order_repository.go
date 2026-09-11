package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"promt-market/internal/domain"
)

type OrderRepository struct {
	db *gorm.DB
}

func NewOrderRepository(db *gorm.DB) *OrderRepository {
	return &OrderRepository{db: db}
}

func (r *OrderRepository) FindByID(ctx context.Context, id string) (*domain.Order, error) {
	var order domain.Order
	err := r.db.WithContext(ctx).
		Preload("Buyer").
		Preload("Prompt").
		Where("id = ?", id).
		First(&order).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &order, nil
}

func (r *OrderRepository) Update(ctx context.Context, order *domain.Order) error {
	return r.db.WithContext(ctx).Save(order).Error
}

// FindPaidOrder returns a paid order for the given (buyer, prompt) pair,
// or nil if the user hasn't purchased the prompt. Used to gate review
// creation on non-free prompts: only verified buyers can review.
func (r *OrderRepository) FindPaidOrder(ctx context.Context, buyerID, promptID string) (*domain.Order, error) {
	var order domain.Order
	err := r.db.WithContext(ctx).
		Where("buyer_id = ? AND prompt_id = ? AND status = ?", buyerID, promptID, "paid").
		First(&order).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &order, nil
}