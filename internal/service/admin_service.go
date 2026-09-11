package service

import (
	"context"
	"errors"
	"strings"
	"time"
	"gorm.io/gorm"

	"promt-market/internal/domain"
	"promt-market/internal/repository"
)

type AdminService struct {
	adminRepo     *repository.AdminRepository
	userRepo      *repository.UserRepository
	promptRepo    *repository.PromptRepository
	orderRepo     *repository.OrderRepository
	appService    *AuthorApplicationService
	reviewService *ReviewService
	db            *gorm.DB
}

func NewAdminService(
	adminRepo *repository.AdminRepository,
	userRepo *repository.UserRepository,
	promptRepo *repository.PromptRepository,
	orderRepo *repository.OrderRepository,
	appService *AuthorApplicationService,
	reviewService *ReviewService,
	db *gorm.DB,
) *AdminService {
	return &AdminService{
		adminRepo:     adminRepo,
		userRepo:      userRepo,
		promptRepo:    promptRepo,
		orderRepo:     orderRepo,
		appService:    appService,
		reviewService: reviewService,
		db:            db,
	}
}

// ============================================================
//  USER MANAGEMENT
// ============================================================

func (s *AdminService) GetAllUsers(ctx context.Context, page, limit int) ([]domain.User, int64, error) {
	return s.adminRepo.GetAllUsers(ctx, page, limit)
}

func (s *AdminService) UpdateUserStatus(ctx context.Context, userID string, isActive bool) error {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return err
	}
	if user == nil {
		return errors.New("user not found")
	}

	user.IsActive = isActive
	return s.userRepo.Update(ctx, user)
}

// ============================================================
//  PROMPT MANAGEMENT
// ============================================================

func (s *AdminService) GetPromptsByStatus(ctx context.Context, status string, page, limit int) ([]domain.Prompt, int64, error) {
	return s.adminRepo.GetPromptsByStatus(ctx, status, page, limit)
}

func (s *AdminService) GetAllPromptsAdmin(ctx context.Context, page, limit int) ([]domain.Prompt, int64, error) {
	return s.adminRepo.GetAllPromptsAdmin(ctx, page, limit)
}

// ApprovePrompt marks a prompt as approved and stamps PublishedAt. Any
// previous rejection note is cleared so the seller no longer sees a stale
// "why was I rejected" message.
func (s *AdminService) ApprovePrompt(ctx context.Context, promptID string) error {
	prompt, err := s.promptRepo.FindByID(ctx, promptID)
	if err != nil {
		return err
	}
	if prompt == nil {
		return errors.New("prompt not found")
	}

	now := time.Now()
	fields := map[string]interface{}{
		"status":         "approved",
		"published_at":   &now,
		"rejection_note": "", // clear any prior rejection note
	}
	return s.promptRepo.UpdateFields(ctx, promptID, fields)
}

// RejectPrompt marks a prompt as rejected and stores an optional note that
// explains why. The note is what the author sees on their dashboard.
func (s *AdminService) RejectPrompt(ctx context.Context, promptID, note string) error {
	prompt, err := s.promptRepo.FindByID(ctx, promptID)
	if err != nil {
		return err
	}
	if prompt == nil {
		return errors.New("prompt not found")
	}

	fields := map[string]interface{}{
		"status":         "rejected",
		"rejection_note": strings.TrimSpace(note),
	}
	return s.promptRepo.UpdateFields(ctx, promptID, fields)
}

// DeletePromptAdmin removes any prompt regardless of ownership. Uses
// FindByIDForAdmin so that re-running on an already-deleted prompt returns
// a clear "already deleted" error instead of a misleading "not found".
func (s *AdminService) DeletePromptAdmin(ctx context.Context, promptID string) (*domain.Prompt, error) {
	prompt, err := s.promptRepo.FindByIDForAdmin(ctx, promptID)
	if err != nil {
		return nil, err
	}
	if prompt == nil {
		return nil, errors.New("prompt not found")
	}
	if prompt.DeletedAt != nil {
		return nil, errors.New("prompt already deleted")
	}

	if err := s.promptRepo.Delete(ctx, promptID); err != nil {
		return nil, err
	}

	return prompt, nil
}

// AdminUpdatePrompt applies a partial edit to any prompt while it is under
// review. Unlike the seller-facing Update, this bypasses ownership checks
// and accepts a Status field so the admin can approve/reject in the same
// request that fixes metadata.
func (s *AdminService) AdminUpdatePrompt(ctx context.Context, promptID string, req *domain.AdminUpdatePromptRequest) (*domain.Prompt, error) {
	prompt, err := s.promptRepo.FindByIDForAdmin(ctx, promptID)
	if err != nil {
		return nil, err
	}
	if prompt == nil {
		return nil, errors.New("prompt not found")
	}
	if prompt.DeletedAt != nil {
		return nil, errors.New("cannot edit a deleted prompt")
	}

	fields := map[string]interface{}{}

	if req.Title != nil {
		fields["title"] = *req.Title
	}
	if req.Description != nil {
		fields["description"] = *req.Description
		// Keep the SEO meta description in sync with the manual edit.
		fields["meta_description"] = truncate(*req.Description, 160)
	}
	if req.Category != nil {
		fields["category"] = *req.Category
	}
	if req.SubCategory != nil {
		fields["sub_category"] = *req.SubCategory
	}
	if req.Tags != nil {
		fields["tags"] = pqStringArray(req.Tags)
	}
	if req.Content != nil {
		fields["content"] = *req.Content
	}
	if req.DemoOutput != nil {
		fields["demo_output"] = *req.DemoOutput
	}
	if req.Instructions != nil {
		fields["instructions"] = *req.Instructions
	}
	if req.CoverImage != nil {
		fields["cover_image"] = *req.CoverImage
	}
	if req.Images != nil {
		fields["images"] = pqStringArray(req.Images)
	}
	if req.Price != nil {
		fields["price"] = *req.Price
	}
	if req.DiscountPrice != nil {
		fields["discount_price"] = *req.DiscountPrice
	}
	if req.Difficulty != nil {
		fields["difficulty"] = *req.Difficulty
	}
	if req.Language != nil {
		fields["language"] = *req.Language
	}
	if req.Status != nil {
		fields["status"] = *req.Status
		// When approving via edit, clear any prior rejection note and stamp
		// the publish time. Rejection via edit keeps the note field as-is
		// (the dedicated RejectPrompt endpoint handles setting it).
		if *req.Status == "approved" {
			now := time.Now()
			fields["published_at"] = &now
			fields["rejection_note"] = ""
		}
	}

	if len(fields) == 0 {
		return prompt, nil // nothing to update
	}

	if err := s.promptRepo.UpdateFields(ctx, promptID, fields); err != nil {
		return nil, err
	}

	// Re-fetch so the response reflects exactly what's in the DB.
	return s.promptRepo.FindByIDForAdmin(ctx, promptID)
}

// RemovePromptImage deletes a single gallery image by index. Returns the
// updated prompt. Index is validated against the current slice length.
// The physical file on disk is left untouched so order history and audit
// trails keep a stable reference — only the DB row loses the reference.
func (s *AdminService) RemovePromptImage(ctx context.Context, promptID string, index int) (*domain.Prompt, error) {
	prompt, err := s.promptRepo.FindByIDForAdmin(ctx, promptID)
	if err != nil {
		return nil, err
	}
	if prompt == nil {
		return nil, errors.New("prompt not found")
	}
	if prompt.DeletedAt != nil {
		return nil, errors.New("cannot edit a deleted prompt")
	}

	if index < 0 || index >= len(prompt.Images) {
		return nil, errors.New("image index out of range")
	}

	// Copy into a new slice so we don't mutate the loaded model in place
	// before persistence — makes the intent explicit and avoids surprises
	// if the slice were ever reused.
	newImages := make([]string, 0, len(prompt.Images)-1)
	for i, img := range prompt.Images {
		if i == index {
			continue
		}
		newImages = append(newImages, img)
	}

	if err := s.promptRepo.UpdateFields(ctx, promptID, map[string]interface{}{
		"images": pqStringArray(newImages),
	}); err != nil {
		return nil, err
	}

	return s.promptRepo.FindByIDForAdmin(ctx, promptID)
}

// RemovePromptCover clears the cover image reference on a prompt.
func (s *AdminService) RemovePromptCover(ctx context.Context, promptID string) (*domain.Prompt, error) {
	prompt, err := s.promptRepo.FindByIDForAdmin(ctx, promptID)
	if err != nil {
		return nil, err
	}
	if prompt == nil {
		return nil, errors.New("prompt not found")
	}
	if prompt.DeletedAt != nil {
		return nil, errors.New("cannot edit a deleted prompt")
	}

	if err := s.promptRepo.UpdateFields(ctx, promptID, map[string]interface{}{
		"cover_image": "",
	}); err != nil {
		return nil, err
	}

	return s.promptRepo.FindByIDForAdmin(ctx, promptID)
}

// ============================================================
//  ORDER MANAGEMENT
// ============================================================

func (s *AdminService) GetAllOrders(ctx context.Context, page, limit int) ([]domain.Order, int64, error) {
	return s.adminRepo.GetAllOrders(ctx, page, limit)
}

func (s *AdminService) GetOrderByID(ctx context.Context, orderID string) (*domain.Order, error) {
	return s.orderRepo.FindByID(ctx, orderID)
}

func (s *AdminService) UpdateOrderStatus(ctx context.Context, orderID, status string) error {
	order, err := s.orderRepo.FindByID(ctx, orderID)
	if err != nil {
		return err
	}
	if order == nil {
		return errors.New("order not found")
	}

	order.Status = status
	return s.orderRepo.Update(ctx, order)
}

// ============================================================
//  DASHBOARD & LOGS
// ============================================================

func (s *AdminService) GetDashboardStats(ctx context.Context) (map[string]interface{}, error) {
	return s.adminRepo.GetDashboardStats(ctx)
}

func (s *AdminService) GetAdminLogs(ctx context.Context, page, limit int) ([]domain.AdminLog, int64, error) {
	return s.adminRepo.GetAdminLogs(ctx, page, limit)
}

func (s *AdminService) LogAdminAction(ctx context.Context, adminID, action, targetType, targetID string, details domain.JSONMap) error {
	log := &domain.AdminLog{
		AdminID:    adminID,
		Action:     action,
		TargetType: targetType,
		TargetID:   targetID,
		Details:    details,
	}
	return s.adminRepo.CreateLog(ctx, log)
}

// ============================================================
//  AUTHOR APPLICATION REVIEW
// ============================================================

func (s *AdminService) GetAuthorApplications(ctx context.Context, status string, page, limit int) ([]domain.AuthorApplication, int64, error) {
	return s.appService.GetApplicationsByStatus(ctx, status, page, limit)
}

func (s *AdminService) GetAuthorApplicationByID(ctx context.Context, id string) (*domain.AuthorApplication, error) {
	return s.appService.GetApplicationByID(ctx, id)
}

func (s *AdminService) ApproveAuthorApplication(ctx context.Context, appID, adminID string) (*domain.AuthorApplication, error) {
	return s.appService.ApproveApplication(ctx, appID, adminID)
}

func (s *AdminService) RejectAuthorApplication(ctx context.Context, appID, adminID, reason string) (*domain.AuthorApplication, error) {
	return s.appService.RejectApplication(ctx, appID, adminID, reason)
}


// ============================================================
//  REVIEW MODERATION
// ============================================================

// ListReviewsByStatus returns reviews for admin moderation. Delegates to
// ReviewService so AdminService remains the single entry point for the
// admin handler.
func (s *AdminService) ListReviewsByStatus(ctx context.Context, status string, page, limit int) ([]domain.Review, int64, error) {
	return s.reviewService.ListByStatus(ctx, status, page, limit)
}

func (s *AdminService) ApproveReview(ctx context.Context, reviewID string) error {
	return s.reviewService.ApproveReview(ctx, reviewID)
}

func (s *AdminService) RejectReview(ctx context.Context, reviewID, reason string) error {
	return s.reviewService.RejectReview(ctx, reviewID, reason)
}

func (s *AdminService) DeleteReview(ctx context.Context, reviewID string) (*domain.Review, error) {
	return s.reviewService.DeleteReview(ctx, reviewID)
}