package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"promt-market/internal/domain"
)

type AuthorApplicationRepository struct {
	db *gorm.DB
}

func NewAuthorApplicationRepository(db *gorm.DB) *AuthorApplicationRepository {
	return &AuthorApplicationRepository{db: db}
}

func (r *AuthorApplicationRepository) Create(ctx context.Context, app *domain.AuthorApplication) error {
	return r.db.WithContext(ctx).Create(app).Error
}

func (r *AuthorApplicationRepository) FindByID(ctx context.Context, id string) (*domain.AuthorApplication, error) {
	var app domain.AuthorApplication
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&app).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &app, nil
}

// FindLatestByUserID returns the user's most recent application, if any.
// Used to block duplicate applications while one is pending/approved.
func (r *AuthorApplicationRepository) FindLatestByUserID(ctx context.Context, userID string) (*domain.AuthorApplication, error) {
	var app domain.AuthorApplication
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		First(&app).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &app, nil
}

func (r *AuthorApplicationRepository) Update(ctx context.Context, app *domain.AuthorApplication) error {
	return r.db.WithContext(ctx).Save(app).Error
}

func (r *AuthorApplicationRepository) GetByStatus(ctx context.Context, status string, page, limit int) ([]domain.AuthorApplication, int64, error) {
	var apps []domain.AuthorApplication
	var total int64

	q := r.db.WithContext(ctx).Model(&domain.AuthorApplication{})
	if status != "" && status != "all" {
		q = q.Where("status = ?", status)
	}

	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	err := q.Order("created_at DESC").Offset(offset).Limit(limit).Find(&apps).Error
	if err != nil {
		return nil, 0, err
	}

	return apps, total, nil
}