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
	repo         *repository.PromptRepository
	userRepo     *repository.UserRepository
	viewService  *PromptViewService
	purchaseSvc  *PurchaseService
}

func NewPromptService(
	repo *repository.PromptRepository,
	userRepo *repository.UserRepository,
	viewService *PromptViewService,
	purchaseSvc *PurchaseService,
) *PromptService {
	return &PromptService{
		repo:        repo,
		userRepo:    userRepo,
		viewService: viewService,
		purchaseSvc: purchaseSvc,
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

// GetBySlug returns a public prompt with the viewer's access level applied.
// Views are recorded for approved, non-paused prompts.
//
// userID may be empty (anonymous visitor). In that case only free prompts
// expose their content; everything else is stripped.
func (s *PromptService) GetBySlug(ctx context.Context, slug, userID string) (*domain.PromptResponse, error) {
	prompt, err := s.repo.FindBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	if prompt == nil {
		return nil, errors.New("prompt not found")
	}

	// Record a view for publicly visible prompts only.
	if prompt.Status == "approved" && !prompt.IsPaused {
		go func(id string) {
			bg := context.Background()
			_ = s.repo.IncrementViews(bg, id)
			if s.viewService != nil {
				_ = s.viewService.Record(bg, id)
			}
		}(prompt.ID)
	}

	return s.buildPromptResponse(ctx, prompt, userID)
}

// GetByID returns a prompt by ID (no access filtering; used internally).
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

// ─────────────────────────────────────────────────────────
//  Access-aware response builder
// ─────────────────────────────────────────────────────────

// buildPromptResponse wraps a raw prompt with the viewer's access decision
// and strips protected fields when access is not granted.
//
// This is the ONLY place where content gating happens, so every caller
// (slug lookup, ID lookup, purchases listing) can reuse it.
func (s *PromptService) buildPromptResponse(ctx context.Context, prompt *domain.Prompt, viewerID string) (*domain.PromptResponse, error) {
	resp := &domain.PromptResponse{
		Prompt: *prompt,
	}

	// If the purchase service isn't wired (legacy setups), default to public
	// view with no gating. This keeps tests and older deployments working.
	if s.purchaseSvc == nil {
		resp.HasAccess = false
		resp.AccessReason = ""
		s.stripProtectedFields(resp)
		return resp, nil
	}

	decision, err := s.purchaseSvc.CanAccessPrompt(ctx, viewerID, prompt.ID, prompt)
	if err != nil {
		return nil, err
	}

	resp.HasAccess = decision.Allowed
	resp.AccessReason = decision.Reason

	if !decision.Allowed {
		s.stripProtectedFields(resp)
	}

	return resp, nil
}

// stripProtectedFields blanks out the fields that should not be visible to
// non-owners. Called by buildPromptResponse when HasAccess is false.
func (s *PromptService) stripProtectedFields(resp *domain.PromptResponse) {
	resp.Content = ""
	resp.Instructions = ""
	// demo_output is intentionally left visible: sellers use it as a teaser.
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