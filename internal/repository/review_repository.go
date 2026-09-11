package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"promt-market/internal/domain"
)

type ReviewRepository struct {
	db *gorm.DB
}

func NewReviewRepository(db *gorm.DB) *ReviewRepository {
	return &ReviewRepository{db: db}
}

func (r *ReviewRepository) Create(ctx context.Context, review *domain.Review) error {
	return r.db.WithContext(ctx).Create(review).Error
}

func (r *ReviewRepository) FindByID(ctx context.Context, id string) (*domain.Review, error) {
	var review domain.Review
	err := r.db.WithContext(ctx).
		Preload("User").
		Preload("Prompt").
		Where("id = ?", id).
		First(&review).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &review, nil
}

// FindByUserAndPrompt returns an existing review for the given (user, prompt)
// pair — used to enforce the "one review per user per prompt" rule.
func (r *ReviewRepository) FindByUserAndPrompt(ctx context.Context, userID, promptID string) (*domain.Review, error) {
	var review domain.Review
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND prompt_id = ?", userID, promptID).
		First(&review).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &review, nil
}

// FindApprovedByPrompt returns approved reviews for a prompt (public listing),
// newest first.
func (r *ReviewRepository) FindApprovedByPrompt(ctx context.Context, promptID string, limit, offset int) ([]domain.Review, int64, error) {
	var reviews []domain.Review
	var total int64

	query := r.db.WithContext(ctx).Model(&domain.Review{}).
		Where("prompt_id = ? AND status = ?", promptID, domain.ReviewStatusApproved)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.
		Preload("User").
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&reviews).Error

	return reviews, total, err
}

// FindByStatus returns reviews filtered by status (admin moderation list).
// Pass "all" to get every status.
func (r *ReviewRepository) FindByStatus(ctx context.Context, status string, limit, offset int) ([]domain.Review, int64, error) {
	var reviews []domain.Review
	var total int64

	query := r.db.WithContext(ctx).Model(&domain.Review{})
	if status != "" && status != "all" {
		query = query.Where("status = ?", status)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.
		Preload("User").
		Preload("Prompt").
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&reviews).Error

	return reviews, total, err
}

// UpdateFields applies a partial update to a review (status, rejection_reason).
func (r *ReviewRepository) UpdateFields(ctx context.Context, id string, fields map[string]interface{}) error {
	return r.db.WithContext(ctx).
		Model(&domain.Review{}).
		Where("id = ?", id).
		Updates(fields).Error
}

func (r *ReviewRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).
		Where("id = ?", id).
		Delete(&domain.Review{}).Error
}

// AggregateRating returns (avg, count) of approved reviews for a prompt.
// Used to keep Prompt.Rating and Prompt.ReviewCount in sync.
func (r *ReviewRepository) AggregateRating(ctx context.Context, promptID string) (float64, int64, error) {
	type result struct {
		Avg   float64
		Count int64
	}
	var res result
	err := r.db.WithContext(ctx).
		Model(&domain.Review{}).
		Select("COALESCE(AVG(rating), 0) AS avg, COUNT(*) AS count").
		Where("prompt_id = ? AND status = ?", promptID, domain.ReviewStatusApproved).
		Scan(&res).Error
	if err != nil {
		return 0, 0, err
	}
	return res.Avg, res.Count, nil
}