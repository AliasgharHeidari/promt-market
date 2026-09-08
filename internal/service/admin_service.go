package service

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"promt-market/internal/domain"
	"promt-market/internal/repository"
)

type AdminService struct {
	adminRepo  *repository.AdminRepository
	userRepo   *repository.UserRepository
	promptRepo *repository.PromptRepository
	orderRepo  *repository.OrderRepository
	db         *gorm.DB
}

func NewAdminService(
	adminRepo *repository.AdminRepository,
	userRepo *repository.UserRepository,
	promptRepo *repository.PromptRepository,
	orderRepo *repository.OrderRepository,
	db *gorm.DB,
) *AdminService {
	return &AdminService{
		adminRepo:  adminRepo,
		userRepo:   userRepo,
		promptRepo: promptRepo,
		orderRepo:  orderRepo,
		db:         db,
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

func (s *AdminService) ApprovePrompt(ctx context.Context, promptID string) error {
	prompt, err := s.promptRepo.FindByID(ctx, promptID)
	if err != nil {
		return err
	}
	if prompt == nil {
		return errors.New("prompt not found")
	}

	prompt.Status = "approved"
	now := time.Now()
	prompt.PublishedAt = &now

	return s.promptRepo.Update(ctx, prompt)
}

func (s *AdminService) RejectPrompt(ctx context.Context, promptID string) error {
	prompt, err := s.promptRepo.FindByID(ctx, promptID)
	if err != nil {
		return err
	}
	if prompt == nil {
		return errors.New("prompt not found")
	}

	prompt.Status = "rejected"
	return s.promptRepo.Update(ctx, prompt)
}

// ✅ DeletePromptAdmin - Admin can delete any prompt without ownership check.
// Uses FindByIDForAdmin (not FindByID) so that re-running this on an
// already-deleted prompt returns a clear "already deleted" error instead of
// a misleading "prompt not found" — the row still exists, just soft-deleted.
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

// LogAdminAction persists a structured audit-log entry for a sensitive admin
// action (e.g. deleting a prompt that belongs to another user). Kept separate
// from the generic AuditLog middleware so handlers can attach domain-specific
// details (like the prompt title) that the middleware has no way to know.
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

// GetByIDForAdmin returns a prompt by ID (includes soft-deleted)
func (s *PromptService) GetByIDForAdmin(ctx context.Context, id string) (*domain.Prompt, error) {
	return s.repo.FindByIDForAdmin(ctx, id)
}