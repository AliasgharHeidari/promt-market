package repository

import (
	"context"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"promt-market/internal/domain"
)

type PromptViewRepository struct {
	db *gorm.DB
}

func NewPromptViewRepository(db *gorm.DB) *PromptViewRepository {
	return &PromptViewRepository{db: db}
}

// IncrementToday adds 1 to today's row for the given prompt, creating it if
// it doesn't exist. Uses an UPSERT so concurrent hits don't fail with a
// unique-constraint violation.
func (r *PromptViewRepository) IncrementToday(ctx context.Context, promptID string) error {
	today := time.Now().Truncate(24 * time.Hour)
	row := &domain.PromptView{
		PromptID: promptID,
		Date:     today,
		Count:    1,
	}
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "prompt_id"}, {Name: "date"}},
			DoUpdates: clause.Assignments(map[string]interface{}{
				"count": gorm.Expr("prompt_views.count + 1"),
			}),
		}).
		Create(row).Error
}

// SumBetween returns the total views for a prompt between two dates.
func (r *PromptViewRepository) SumBetween(ctx context.Context, promptID string, from, to time.Time) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).
		Model(&domain.PromptView{}).
		Where("prompt_id = ? AND date >= ? AND date < ?", promptID, from, to).
		Select("COALESCE(SUM(count), 0)").
		Scan(&total).Error
	return total, err
}

// SumBetweenForSeller returns total views across all of a seller's prompts.
func (r *PromptViewRepository) SumBetweenForSeller(ctx context.Context, sellerID string, from, to time.Time) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).
		Model(&domain.PromptView{}).
		Joins("JOIN prompts ON prompts.id = prompt_views.prompt_id").
		Where("prompts.seller_id = ? AND prompt_views.date >= ? AND prompt_views.date < ?", sellerID, from, to).
		Select("COALESCE(SUM(prompt_views.count), 0)").
		Scan(&total).Error
	return total, err
}

// DailySeries returns a day-by-day list of (date, count) for a seller's
// prompts. Days with zero views are missing from the result; the caller
// (chart) should fill the gaps.
func (r *PromptViewRepository) DailySeries(ctx context.Context, sellerID string, from, to time.Time) ([]struct {
	Date  time.Time
	Count int64
}, error) {
	var rows []struct {
		Date  time.Time
		Count int64
	}
	err := r.db.WithContext(ctx).
		Model(&domain.PromptView{}).
		Joins("JOIN prompts ON prompts.id = prompt_views.prompt_id").
		Where("prompts.seller_id = ? AND prompt_views.date >= ? AND prompt_views.date < ?", sellerID, from, to).
		Select("prompt_views.date AS date, SUM(prompt_views.count) AS count").
		Group("prompt_views.date").
		Order("prompt_views.date ASC").
		Scan(&rows).Error
	return rows, err
}

// DailySeriesForPrompt is like DailySeries but scoped to a single prompt.
func (r *PromptViewRepository) DailySeriesForPrompt(ctx context.Context, promptID string, from, to time.Time) ([]struct {
	Date  time.Time
	Count int64
}, error) {
	var rows []struct {
		Date  time.Time
		Count int64
	}
	err := r.db.WithContext(ctx).
		Model(&domain.PromptView{}).
		Where("prompt_id = ? AND date >= ? AND date < ?", promptID, from, to).
		Select("date, count").
		Order("date ASC").
		Scan(&rows).Error
	return rows, err
}