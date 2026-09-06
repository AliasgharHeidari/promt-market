package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"
	"promt-market/internal/domain"
)

type PromptRepository struct {
	db *gorm.DB
}

func NewPromptRepository(db *gorm.DB) *PromptRepository {
	return &PromptRepository{db: db}
}

func (r *PromptRepository) Create(ctx context.Context, prompt *domain.Prompt) error {
	return r.db.WithContext(ctx).Create(prompt).Error
}

func (r *PromptRepository) FindByID(ctx context.Context, id string) (*domain.Prompt, error) {
	var prompt domain.Prompt
	err := r.db.WithContext(ctx).
		Preload("Seller").
		Where("id = ?", id).
		First(&prompt).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &prompt, nil
}

func (r *PromptRepository) FindBySlug(ctx context.Context, slug string) (*domain.Prompt, error) {
	var prompt domain.Prompt
	err := r.db.WithContext(ctx).
		Preload("Seller").
		Where("slug = ?", slug).
		First(&prompt).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &prompt, nil
}

func (r *PromptRepository) FindAll(ctx context.Context, limit, offset int) ([]domain.Prompt, int64, error) {
	var prompts []domain.Prompt
	var total int64

	query := r.db.WithContext(ctx).Model(&domain.Prompt{}).Where("status = ?", "approved")

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.
		Preload("Seller").
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&prompts).Error

	return prompts, total, err
}

func (r *PromptRepository) FindBySeller(ctx context.Context, sellerID string, limit, offset int) ([]domain.Prompt, int64, error) {
	var prompts []domain.Prompt
	var total int64

	query := r.db.WithContext(ctx).Model(&domain.Prompt{}).Where("seller_id = ?", sellerID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.
		Preload("Seller").
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&prompts).Error

	return prompts, total, err
}

func (r *PromptRepository) Update(ctx context.Context, prompt *domain.Prompt) error {
	return r.db.WithContext(ctx).Save(prompt).Error
}

func (r *PromptRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&domain.Prompt{}, "id = ?", id).Error
}

func (r *PromptRepository) IncrementViews(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Model(&domain.Prompt{}).
		Where("id = ?", id).
		Update("views", gorm.Expr("views + 1")).Error
}

func (r *PromptRepository) Search(ctx context.Context, filter *domain.PromptFilter) ([]domain.Prompt, int64, error) {
	var prompts []domain.Prompt
	var total int64

	query := r.db.WithContext(ctx).Model(&domain.Prompt{})

	if filter.Status == "approved" {
		query = query.Where("status = ?", "approved")
	}

	if filter.Category != "" {
		query = query.Where("category = ?", filter.Category)
	}

	if filter.SubCategory != "" {
		query = query.Where("sub_category = ?", filter.SubCategory)
	}

	if filter.MinPrice > 0 {
		query = query.Where("price >= ?", filter.MinPrice)
	}
	if filter.MaxPrice > 0 {
		query = query.Where("price <= ?", filter.MaxPrice)
	}

	if filter.Difficulty != "" {
		query = query.Where("difficulty = ?", filter.Difficulty)
	}

	if filter.Language != "" {
		query = query.Where("language = ?", filter.Language)
	}

	if filter.MinRating > 0 {
		query = query.Where("rating >= ?", filter.MinRating)
	}

	if len(filter.Tags) > 0 {
		query = query.Where("tags && ?", filter.Tags)
	}

	if filter.Search != "" {
		searchTerm := strings.TrimSpace(filter.Search)
		query = query.Where(
			"to_tsvector('english', title || ' ' || COALESCE(description, '')) @@ plainto_tsquery('english', ?)",
			searchTerm,
		)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	sortField := filter.SortBy
	sortOrder := filter.SortOrder

	validSortFields := map[string]bool{
		"price": true, "created_at": true, "rating": true,
		"sales_count": true, "views": true,
	}
	if !validSortFields[sortField] {
		sortField = "created_at"
	}

	if sortOrder != "asc" && sortOrder != "desc" {
		sortOrder = "desc"
	}

	query = query.Order(fmt.Sprintf("%s %s", sortField, sortOrder))

	query = query.
		Preload("Seller").
		Limit(filter.Limit).
		Offset(filter.GetOffset())

	err := query.Find(&prompts).Error

	return prompts, total, err
}

func (r *PromptRepository) GetCategories(ctx context.Context, categories *[]string) error {
	return r.db.WithContext(ctx).
		Model(&domain.Prompt{}).
		Where("status = ?", "approved").
		Distinct("category").
		Pluck("category", categories).Error
}