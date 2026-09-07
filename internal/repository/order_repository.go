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