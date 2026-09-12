package repository

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"promt-market/internal/domain"
)

type OrderRepository struct {
	db *gorm.DB
}

func NewOrderRepository(db *gorm.DB) *OrderRepository {
	return &OrderRepository{db: db}
}

func (r *OrderRepository) FindByID(ctx context.Context, id string) (*domain.Order, error) {
	var order domain.Order
	err := r.db.WithContext(ctx).
		Preload("Buyer").
		Preload("Prompt").
		Where("id = ?", id).
		First(&order).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &order, nil
}

func (r *OrderRepository) Update(ctx context.Context, order *domain.Order) error {
	return r.db.WithContext(ctx).Save(order).Error
}

func (r *OrderRepository) FindPaidOrder(ctx context.Context, buyerID, promptID string) (*domain.Order, error) {
	var order domain.Order
	err := r.db.WithContext(ctx).
		Where("buyer_id = ? AND prompt_id = ? AND status = ?", buyerID, promptID, "paid").
		First(&order).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &order, nil
}

// SellerStats is a small aggregate returned to the author dashboard.
type SellerStats struct {
	TotalSales     int64
	TotalRevenue   int64 // sum of (amount - commission) for paid orders
	MonthlySales   int64
	MonthlyRevenue int64
}

// GetSellerStats computes lifetime and rolling-30-day sales/revenue for a
// seller's prompts. Revenue is what the seller earns (amount - commission).
func (r *OrderRepository) GetSellerStats(ctx context.Context, sellerID string, since time.Time) (*SellerStats, error) {
	var stats SellerStats

	// Lifetime totals
	err := r.db.WithContext(ctx).
		Model(&domain.Order{}).
		Joins("JOIN prompts ON prompts.id = orders.prompt_id").
		Where("prompts.seller_id = ? AND orders.status = ?", sellerID, "paid").
		Select("COUNT(*) AS total_sales, COALESCE(SUM(orders.amount - orders.commission), 0) AS total_revenue").
		Scan(&stats).Error
	if err != nil {
		return nil, err
	}

	// Rolling window totals
	var monthly struct {
		Sales   int64
		Revenue int64
	}
	err = r.db.WithContext(ctx).
		Model(&domain.Order{}).
		Joins("JOIN prompts ON prompts.id = orders.prompt_id").
		Where("prompts.seller_id = ? AND orders.status = ? AND orders.created_at >= ?", sellerID, "paid", since).
		Select("COUNT(*) AS sales, COALESCE(SUM(orders.amount - orders.commission), 0) AS revenue").
		Scan(&monthly).Error
	if err != nil {
		return nil, err
	}

	stats.MonthlySales = monthly.Sales
	stats.MonthlyRevenue = monthly.Revenue
	return &stats, nil
}

// DailySalesSeries returns (date, count, revenue) for a seller over a period.
// Used by the dashboard revenue chart.
func (r *OrderRepository) DailySalesSeries(ctx context.Context, sellerID string, from, to time.Time) ([]struct {
	Date    time.Time
	Sales   int64
	Revenue int64
}, error) {
	var rows []struct {
		Date    time.Time
		Sales   int64
		Revenue int64
	}
	err := r.db.WithContext(ctx).
		Model(&domain.Order{}).
		Joins("JOIN prompts ON prompts.id = orders.prompt_id").
		Where("prompts.seller_id = ? AND orders.status = ? AND orders.created_at >= ? AND orders.created_at < ?",
			sellerID, "paid", from, to).
		Select("DATE(orders.created_at) AS date, COUNT(*) AS sales, COALESCE(SUM(orders.amount - orders.commission), 0) AS revenue").
		Group("DATE(orders.created_at)").
		Order("DATE(orders.created_at) ASC").
		Scan(&rows).Error
	return rows, err
}