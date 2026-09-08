package repository

import (
	"context"

	"gorm.io/gorm"

	"promt-market/internal/domain"
)

type AdminRepository struct {
	db *gorm.DB
}

func NewAdminRepository(db *gorm.DB) *AdminRepository {
	return &AdminRepository{db: db}
}

// ============================================================
//  USER
// ============================================================

func (r *AdminRepository) GetAllUsers(ctx context.Context, page, limit int) ([]domain.User, int64, error) {
	var users []domain.User
	var total int64

	query := r.db.WithContext(ctx).Model(&domain.User{})

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.
		Order("created_at DESC").
		Limit(limit).
		Offset((page-1)*limit).
		Find(&users).Error

	return users, total, err
}

// ============================================================
//  PROMPT
// ============================================================

func (r *AdminRepository) GetPromptsByStatus(ctx context.Context, status string, page, limit int) ([]domain.Prompt, int64, error) {
	var prompts []domain.Prompt
	var total int64

	query := r.db.WithContext(ctx).Model(&domain.Prompt{}).Where("status = ?", status)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.
		Preload("Seller").
		Order("created_at DESC").
		Limit(limit).
		Offset((page-1)*limit).
		Find(&prompts).Error

	return prompts, total, err
}

// ✅ GetAllPromptsAdmin - returns all prompts regardless of status with Seller preloaded
func (r *AdminRepository) GetAllPromptsAdmin(ctx context.Context, page, limit int) ([]domain.Prompt, int64, error) {
	var prompts []domain.Prompt
	var total int64

	query := r.db.WithContext(ctx).Model(&domain.Prompt{})

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.
		Preload("Seller").
		Order("created_at DESC").
		Limit(limit).
		Offset((page-1)*limit).
		Find(&prompts).Error

	return prompts, total, err
}

// ============================================================
//  ORDER
// ============================================================

func (r *AdminRepository) GetAllOrders(ctx context.Context, page, limit int) ([]domain.Order, int64, error) {
	var orders []domain.Order
	var total int64

	query := r.db.WithContext(ctx).Model(&domain.Order{})

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.
		Preload("Buyer").
		Preload("Prompt").
		Order("created_at DESC").
		Limit(limit).
		Offset((page-1)*limit).
		Find(&orders).Error

	return orders, total, err
}

// ============================================================
//  DASHBOARD
// ============================================================

func (r *AdminRepository) GetDashboardStats(ctx context.Context) (map[string]interface{}, error) {
	var totalUsers int64
	var totalPrompts int64
	var totalOrders int64
	var totalRevenue int64
	var totalPending int64

	r.db.Model(&domain.User{}).Count(&totalUsers)
	r.db.Model(&domain.Prompt{}).Where("status = ?", "approved").Count(&totalPrompts)
	r.db.Model(&domain.Order{}).Count(&totalOrders)
	r.db.Model(&domain.Order{}).Select("COALESCE(SUM(amount), 0)").Scan(&totalRevenue)
	r.db.Model(&domain.Prompt{}).Where("status = ?", "pending").Count(&totalPending)

	return map[string]interface{}{
		"total_users":   totalUsers,
		"total_prompts": totalPrompts,
		"total_orders":  totalOrders,
		"total_revenue": totalRevenue,
		"total_pending": totalPending,
	}, nil
}

// ============================================================
//  LOGS
// ============================================================

func (r *AdminRepository) GetAdminLogs(ctx context.Context, page, limit int) ([]domain.AdminLog, int64, error) {
	var logs []domain.AdminLog
	var total int64

	query := r.db.WithContext(ctx).Model(&domain.AdminLog{})

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.
		Order("created_at DESC").
		Limit(limit).
		Offset((page-1)*limit).
		Find(&logs).Error

	return logs, total, err
}

// CreateLog inserts a single audit-log entry. Used for admin actions that
// need domain-specific context (e.g. what was deleted) beyond what the
// generic AuditLog middleware captures.
func (r *AdminRepository) CreateLog(ctx context.Context, log *domain.AdminLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}