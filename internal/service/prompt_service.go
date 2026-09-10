package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"promt-market/internal/domain"
	"promt-market/internal/repository"
)

type PromptService struct {
	repo     *repository.PromptRepository
	userRepo *repository.UserRepository
}

func NewPromptService(repo *repository.PromptRepository, userRepo *repository.UserRepository) *PromptService {
	return &PromptService{
		repo:     repo,
		userRepo: userRepo,
	}
}

func (s *PromptService) Create(ctx context.Context, sellerID string, req *domain.CreatePromptRequest) (*domain.Prompt, error) {
	user, err := s.userRepo.FindByID(ctx, sellerID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.New("user not found")
	}

	slug := s.generateSlug(req.Title)

	prompt := &domain.Prompt{
		SellerID:        sellerID,
		Title:           req.Title,
		Description:     req.Description,
		Category:        req.Category,
		SubCategory:     req.SubCategory,
		Tags:            pq.StringArray(req.Tags),
		Content:         req.Content,
		DemoOutput:      req.DemoOutput,
		Instructions:    req.Instructions,
		CoverImage:      req.CoverImage,
		Images:          pq.StringArray(req.Images),
		Price:           req.Price,
		DiscountPrice:   req.DiscountPrice,
		Difficulty:      req.Difficulty,
		Language:        req.Language,
		Status:          "pending",
		Slug:            slug,
		MetaTitle:       req.Title,
		MetaDescription: s.truncate(req.Description, 160),
	}

	if err := s.repo.Create(ctx, prompt); err != nil {
		return nil, err
	}

	return prompt, nil
}

func (s *PromptService) GetBySlug(ctx context.Context, slug string) (*domain.Prompt, error) {
	prompt, err := s.repo.FindBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	if prompt == nil {
		return nil, errors.New("prompt not found")
	}

	go s.repo.IncrementViews(context.Background(), prompt.ID)

	return prompt, nil
}

func (s *PromptService) GetByID(ctx context.Context, id string) (*domain.Prompt, error) {
	return s.repo.FindByID(ctx, id)
}

// GetByIDForAdmin returns a prompt by ID including soft-deleted rows.
func (s *PromptService) GetByIDForAdmin(ctx context.Context, id string) (*domain.Prompt, error) {
	return s.repo.FindByIDForAdmin(ctx, id)
}

func (s *PromptService) GetAll(ctx context.Context, page, limit int) ([]domain.Prompt, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	offset := (page - 1) * limit
	return s.repo.FindAll(ctx, limit, offset)
}

func (s *PromptService) GetBySeller(ctx context.Context, sellerID string, page, limit int) ([]domain.Prompt, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	offset := (page - 1) * limit
	return s.repo.FindBySeller(ctx, sellerID, limit, offset)
}

func (s *PromptService) Update(ctx context.Context, promptID, userID string, req *domain.UpdatePromptRequest) (*domain.Prompt, error) {
	prompt, err := s.repo.FindByID(ctx, promptID)
	if err != nil {
		return nil, err
	}
	if prompt == nil {
		return nil, errors.New("prompt not found")
	}

	if prompt.SellerID != userID {
		return nil, errors.New("you are not the owner of this prompt")
	}

	if req.Title != nil {
		prompt.Title = *req.Title
	}
	if req.Description != nil {
		prompt.Description = *req.Description
	}
	if req.Category != nil {
		prompt.Category = *req.Category
	}
	if req.SubCategory != nil {
		prompt.SubCategory = *req.SubCategory
	}
	if req.Tags != nil {
		prompt.Tags = pq.StringArray(req.Tags)
	}
	if req.Content != nil {
		prompt.Content = *req.Content
	}
	if req.DemoOutput != nil {
		prompt.DemoOutput = *req.DemoOutput
	}
	if req.Instructions != nil {
		prompt.Instructions = *req.Instructions
	}
	if req.CoverImage != nil {
		prompt.CoverImage = *req.CoverImage
	}
	if req.Images != nil {
		prompt.Images = pq.StringArray(req.Images)
	}
	if req.Price != nil {
		prompt.Price = *req.Price
	}
	if req.DiscountPrice != nil {
		prompt.DiscountPrice = *req.DiscountPrice
	}
	if req.Difficulty != nil {
		prompt.Difficulty = *req.Difficulty
	}
	if req.Language != nil {
		prompt.Language = *req.Language
	}

	prompt.UpdatedAt = time.Now()

	if err := s.repo.Update(ctx, prompt); err != nil {
		return nil, err
	}

	return prompt, nil
}

func (s *PromptService) Delete(ctx context.Context, promptID, userID string) error {
	prompt, err := s.repo.FindByID(ctx, promptID)
	if err != nil {
		return err
	}
	if prompt == nil {
		return errors.New("prompt not found")
	}

	if prompt.SellerID != userID {
		return errors.New("you are not the owner of this prompt")
	}

	return s.repo.Delete(ctx, promptID)
}

func (s *PromptService) Search(ctx context.Context, filter *domain.PromptFilter) ([]domain.Prompt, int64, error) {
	filter.SetDefaults()
	return s.repo.Search(ctx, filter)
}

func (s *PromptService) GetCategories(ctx context.Context, categories *[]string) error {
	return s.repo.GetCategories(ctx, categories)
}

func (s *PromptService) generateSlug(title string) string {
	slug := strings.ToLower(title)
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = strings.ReplaceAll(slug, "?", "")
	slug = strings.ReplaceAll(slug, "!", "")
	slug = strings.ReplaceAll(slug, ".", "")
	slug = strings.ReplaceAll(slug, ",", "")
	slug = strings.ReplaceAll(slug, ":", "")
	slug = strings.ReplaceAll(slug, ";", "")

	return slug + "-" + uuid.New().String()[:8]
}

func (s *PromptService) truncate(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen-3] + "..."
}