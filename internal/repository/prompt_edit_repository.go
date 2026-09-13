package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"promt-market/internal/domain"
)

type PromptEditRepository struct {
	db *gorm.DB
}

func NewPromptEditRepository(db *gorm.DB) *PromptEditRepository {
	return &PromptEditRepository{db: db}
}

func (r *PromptEditRepository) Create(ctx context.Context, edit *domain.PromptEditProposal) error {
	return r.db.WithContext(ctx).Create(edit).Error
}

func (r *PromptEditRepository) FindByID(ctx context.Context, id string) (*domain.PromptEditProposal, error) {
	var edit domain.PromptEditProposal
	err := r.db.WithContext(ctx).
		Preload("Prompt").
		Preload("Seller").
		Where("id = ?", id).
		First(&edit).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &edit, nil
}

// FindPendingByPrompt returns the current pending proposal for a prompt, if any.
func (r *PromptEditRepository) FindPendingByPrompt(ctx context.Context, promptID string) (*domain.PromptEditProposal, error) {
	var edit domain.PromptEditProposal
	err := r.db.WithContext(ctx).
		Where("prompt_id = ? AND status = ?", promptID, domain.EditProposalPending).
		First(&edit).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &edit, nil
}

// FindByStatus returns proposals filtered by status (admin moderation list).
func (r *PromptEditRepository) FindByStatus(ctx context.Context, status string, limit, offset int) ([]domain.PromptEditProposal, int64, error) {
	var edits []domain.PromptEditProposal
	var total int64

	query := r.db.WithContext(ctx).Model(&domain.PromptEditProposal{})
	if status != "" && status != "all" {
		query = query.Where("status = ?", status)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.
		Preload("Prompt").
		Preload("Seller").
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&edits).Error

	return edits, total, err
}

// UpdateFields applies a partial update to a proposal.
func (r *PromptEditRepository) UpdateFields(ctx context.Context, id string, fields map[string]interface{}) error {
	return r.db.WithContext(ctx).
		Model(&domain.PromptEditProposal{}).
		Where("id = ?", id).
		Updates(fields).Error
}

// DeletePendingByPrompt removes any pending proposal for a prompt. Used when
// the author resubmits (replaces the previous pending one) or deletes a draft.
func (r *PromptEditRepository) DeletePendingByPrompt(ctx context.Context, promptID string) error {
	return r.db.WithContext(ctx).
		Where("prompt_id = ? AND status = ?", promptID, domain.EditProposalPending).
		Delete(&domain.PromptEditProposal{}).Error
}