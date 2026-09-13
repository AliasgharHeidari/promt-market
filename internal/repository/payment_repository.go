package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"promt-market/internal/domain"
)

type PaymentRepository struct {
	db *gorm.DB
}

func NewPaymentRepository(db *gorm.DB) *PaymentRepository {
	return &PaymentRepository{db: db}
}

// Create inserts a new payment row.
func (r *PaymentRepository) Create(ctx context.Context, p *domain.Payment) error {
	return r.db.WithContext(ctx).Create(p).Error
}

// FindByID returns a payment by ID, or nil if not found.
func (r *PaymentRepository) FindByID(ctx context.Context, id string) (*domain.Payment, error) {
	var p domain.Payment
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&p).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &p, nil
}

// FindByAuthority returns a payment by its gateway-issued authority string.
// This is the lookup key used when the user returns from the gateway.
func (r *PaymentRepository) FindByAuthority(ctx context.Context, authority string) (*domain.Payment, error) {
	if authority == "" {
		return nil, nil
	}
	var p domain.Payment
	err := r.db.WithContext(ctx).
		Where("authority = ?", authority).
		First(&p).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &p, nil
}

// UpdateFields applies a partial update to a payment row.
func (r *PaymentRepository) UpdateFields(ctx context.Context, id string, fields map[string]interface{}) error {
	return r.db.WithContext(ctx).
		Model(&domain.Payment{}).
		Where("id = ?", id).
		Updates(fields).Error
}

// UpdateFieldsWithTx is the same as UpdateFields but takes a *gorm.DB (which
// may be a transaction handle). Used by checkout/verify flows that must be
// atomic across multiple tables.
func (r *PaymentRepository) UpdateFieldsWithTx(tx *gorm.DB, id string, fields map[string]interface{}) error {
	return tx.WithContext(context.Background()).
		Model(&domain.Payment{}).
		Where("id = ?", id).
		Updates(fields).Error
}

// FindByIDWithTx returns a payment using a provided tx handle.
func (r *PaymentRepository) FindByIDWithTx(tx *gorm.DB, id string) (*domain.Payment, error) {
	var p domain.Payment
	err := tx.Where("id = ?", id).First(&p).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &p, nil
}

// DB returns the underlying *gorm.DB. Used by services that need to open a
// transaction spanning multiple repositories.
func (r *PaymentRepository) DB() *gorm.DB {
	return r.db
}