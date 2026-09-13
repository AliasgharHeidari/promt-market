package service

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/lib/pq"
	"gorm.io/datatypes"

	"promt-market/internal/domain"
	"promt-market/internal/repository"
)

// AuthorDashboardService aggregates data the author dashboard needs:
// sales/revenue, view analytics, profile, and edit proposals.
type AuthorDashboardService struct {
	userRepo   *repository.UserRepository
	promptRepo *repository.PromptRepository
	orderRepo  *repository.OrderRepository
	viewRepo   *repository.PromptViewRepository
	editRepo   *repository.PromptEditRepository
}

func NewAuthorDashboardService(
	userRepo *repository.UserRepository,
	promptRepo *repository.PromptRepository,
	orderRepo *repository.OrderRepository,
	viewRepo *repository.PromptViewRepository,
	editRepo *repository.PromptEditRepository,
) *AuthorDashboardService {
	return &AuthorDashboardService{
		userRepo:   userRepo,
		promptRepo: promptRepo,
		orderRepo:  orderRepo,
		viewRepo:   viewRepo,
		editRepo:   editRepo,
	}
}

// ============================================================
//  STATS
// ============================================================

type DashboardStats struct {
	TotalPrompts   int64 `json:"total_prompts"`
	ActivePrompts  int64 `json:"active_prompts"`
	PausedPrompts  int64 `json:"paused_prompts"`
	PendingEdits   int64 `json:"pending_edits"`
	TotalSales     int64 `json:"total_sales"`
	TotalRevenue   int64 `json:"total_revenue"`
	MonthlySales   int64 `json:"monthly_sales"`
	MonthlyRevenue int64 `json:"monthly_revenue"`
	TotalViews30d  int64 `json:"total_views_30d"`
}

func (s *AuthorDashboardService) GetStats(ctx context.Context, sellerID string) (*DashboardStats, error) {
	stats := &DashboardStats{}

	// Prompt counts
	prompts, total, err := s.promptRepo.FindBySeller(ctx, sellerID, 1000, 0)
	if err != nil {
		return nil, err
	}
	stats.TotalPrompts = total
	for _, p := range prompts {
		if p.IsPaused {
			stats.PausedPrompts++
		} else if p.Status == "approved" {
			stats.ActivePrompts++
		}
		if p.HasPendingEdit {
			stats.PendingEdits++
		}
	}

	// Sales & revenue
	now := time.Now()
	since30 := now.AddDate(0, 0, -30)
	sellerStats, err := s.orderRepo.GetSellerStats(ctx, sellerID, since30)
	if err != nil {
		return nil, err
	}
	stats.TotalSales = sellerStats.TotalSales
	stats.TotalRevenue = sellerStats.TotalRevenue
	stats.MonthlySales = sellerStats.MonthlySales
	stats.MonthlyRevenue = sellerStats.MonthlyRevenue

	// Views last 30 days
	views, err := s.viewRepo.SumBetweenForSeller(ctx, sellerID, since30, now)
	if err != nil {
		return nil, err
	}
	stats.TotalViews30d = views

	return stats, nil
}

// ============================================================
//  CHARTS
// ============================================================

type ChartPoint struct {
	Date    string `json:"date"`
	Value   int64  `json:"value"`
	Revenue int64  `json:"revenue,omitempty"`
}

// ViewsChart returns a day-by-day series of views for the last N days.
// Missing days are filled with zero so the chart is continuous.
func (s *AuthorDashboardService) ViewsChart(ctx context.Context, sellerID string, days int) ([]ChartPoint, error) {
	if days < 1 || days > 365 {
		days = 30
	}
	now := time.Now()
	from := now.AddDate(0, 0, -days).Truncate(24 * time.Hour)
	to := now.AddDate(0, 0, 1).Truncate(24 * time.Hour)

	rows, err := s.viewRepo.DailySeries(ctx, sellerID, from, to)
	if err != nil {
		return nil, err
	}

	// Index by date string so we can fill gaps easily.
	byDay := map[string]int64{}
	for _, r := range rows {
		byDay[r.Date.Format("2006-01-02")] = r.Count
	}

	out := make([]ChartPoint, 0, days)
	for d := from; d.Before(to); d = d.AddDate(0, 0, 1) {
		key := d.Format("2006-01-02")
		out = append(out, ChartPoint{Date: key, Value: byDay[key]})
	}
	return out, nil
}

// RevenueChart returns a day-by-day series of paid orders for the last N days.
func (s *AuthorDashboardService) RevenueChart(ctx context.Context, sellerID string, days int) ([]ChartPoint, error) {
	if days < 1 || days > 365 {
		days = 30
	}
	now := time.Now()
	from := now.AddDate(0, 0, -days).Truncate(24 * time.Hour)
	to := now.AddDate(0, 0, 1).Truncate(24 * time.Hour)

	rows, err := s.orderRepo.DailySalesSeries(ctx, sellerID, from, to)
	if err != nil {
		return nil, err
	}

	byDay := map[string]struct {
		Sales   int64
		Revenue int64
	}{}
	for _, r := range rows {
		byDay[r.Date.Format("2006-01-02")] = struct {
			Sales   int64
			Revenue int64
		}{r.Sales, r.Revenue}
	}

	out := make([]ChartPoint, 0, days)
	for d := from; d.Before(to); d = d.AddDate(0, 0, 1) {
		key := d.Format("2006-01-02")
		v := byDay[key]
		out = append(out, ChartPoint{Date: key, Value: v.Sales, Revenue: v.Revenue})
	}
	return out, nil
}

// ============================================================
//  PROFILE
// ============================================================

type UpdateProfileRequest struct {
	FullName  *string `json:"full_name"`
	Bio       *string `json:"bio"`
	Expertise *string `json:"expertise"`
}

func (s *AuthorDashboardService) GetProfile(ctx context.Context, userID string) (*domain.User, error) {
	u, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, errors.New("user not found")
	}
	return u, nil
}

// UpdateProfile saves the fields present in req. Sanitization is done by the
// handler (bluemonday) before calling this.
func (s *AuthorDashboardService) UpdateProfile(ctx context.Context, userID string, req *UpdateProfileRequest) (*domain.User, error) {
	fields := map[string]interface{}{}

	if req.FullName != nil {
		v := strings.TrimSpace(*req.FullName)
		if len(v) < 2 || len(v) > 100 {
			return nil, errors.New("نام باید بین ۲ تا ۱۰۰ کاراکتر باشد")
		}
		fields["full_name"] = v
	}
	if req.Bio != nil {
		v := strings.TrimSpace(*req.Bio)
		if len([]rune(v)) > 500 {
			return nil, errors.New("بیوگرافی نباید بیشتر از ۵۰۰ کاراکتر باشد")
		}
		fields["bio"] = v
	}
	if req.Expertise != nil {
		v := strings.TrimSpace(*req.Expertise)
		if len([]rune(v)) > 100 {
			return nil, errors.New("تخصص نباید بیشتر از ۱۰۰ کاراکتر باشد")
		}
		fields["expertise"] = v
	}

	if len(fields) == 0 {
		return s.userRepo.FindByID(ctx, userID)
	}

	if err := s.userRepo.UpdateFields(ctx, userID, fields); err != nil {
		return nil, err
	}
	return s.userRepo.FindByID(ctx, userID)
}

// UpdateAvatar sets the avatar URL after an upload. Best-effort cleanup of
// the previous avatar file prevents the uploads folder from growing forever
// with orphaned images.
func (s *AuthorDashboardService) UpdateAvatar(ctx context.Context, userID, url string) error {
	// Look up the current avatar so we can delete it after switching.
	prev, _ := s.userRepo.FindByID(ctx, userID)
	if prev != nil && prev.Avatar != "" && prev.Avatar != url {
		go deleteLocalUpload(prev.Avatar)
	}

	return s.userRepo.UpdateFields(ctx, userID, map[string]interface{}{"avatar": url})
}

// ============================================================
//  PAUSE / UNPAUSE
// ============================================================

func (s *AuthorDashboardService) SetPaused(ctx context.Context, sellerID, promptID string, paused bool) (*domain.Prompt, error) {
	prompt, err := s.promptRepo.FindByID(ctx, promptID)
	if err != nil {
		return nil, err
	}
	if prompt == nil {
		return nil, errors.New("prompt not found")
	}
	if prompt.SellerID != sellerID {
		return nil, errors.New("you are not the owner of this prompt")
	}
	if prompt.Status != "approved" {
		return nil, errors.New("فقط پرامپت‌های تایید شده قابل پنهان/نمایش هستند")
	}

	if err := s.promptRepo.SetPaused(ctx, promptID, paused); err != nil {
		return nil, err
	}
	return s.promptRepo.FindByID(ctx, promptID)
}

// ============================================================
//  EDIT PROPOSALS
// ============================================================

// SubmitEdit stores a proposed set of changes for an existing prompt.
// Only fields present in the payload become part of the proposal; all
// others remain untouched on the live prompt. Replaces any pending
// proposal for the same prompt.
func (s *AuthorDashboardService) SubmitEdit(ctx context.Context, sellerID, promptID string, changes map[string]interface{}) (*domain.PromptEditProposal, error) {
	prompt, err := s.promptRepo.FindByID(ctx, promptID)
	if err != nil {
		return nil, err
	}
	if prompt == nil {
		return nil, errors.New("prompt not found")
	}
	if prompt.SellerID != sellerID {
		return nil, errors.New("you are not the owner of this prompt")
	}
	if prompt.Status != "approved" {
		return nil, errors.New("فقط پرامپت‌های تایید شده قابل ویرایش هستند")
	}

	// Whitelist — drop anything the caller isn't supposed to change.
	allowed := map[string]bool{
		"title": true, "description": true, "category": true, "sub_category": true,
		"tags": true, "content": true, "demo_output": true, "instructions": true,
		"cover_image": true, "images": true, "price": true, "discount_price": true,
		"difficulty": true, "language": true,
	}
	filtered := map[string]interface{}{}
	for k, v := range changes {
		if allowed[k] {
			filtered[k] = v
		}
	}
	if len(filtered) == 0 {
		return nil, errors.New("هیچ تغییری ارسال نشده است")
	}

	// Replace any existing pending proposal.
	if err := s.editRepo.DeletePendingByPrompt(ctx, promptID); err != nil {
		return nil, err
	}

	raw, err := json.Marshal(filtered)
	if err != nil {
		return nil, err
	}

	edit := &domain.PromptEditProposal{
		PromptID: promptID,
		SellerID: sellerID,
		Changes:  datatypes.JSON(raw),
		Status:   domain.EditProposalPending,
	}
	if err := s.editRepo.Create(ctx, edit); err != nil {
		return nil, err
	}

	if err := s.promptRepo.SetHasPendingEdit(ctx, promptID, true); err != nil {
		return nil, err
	}

	return edit, nil
}

// ============================================================
//  ADMIN — EDIT PROPOSAL MODERATION
// ============================================================

func (s *AuthorDashboardService) ListEditsByStatus(ctx context.Context, status string, page, limit int) ([]domain.PromptEditProposal, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	return s.editRepo.FindByStatus(ctx, status, limit, (page-1)*limit)
}

func (s *AuthorDashboardService) ApproveEdit(ctx context.Context, editID string) error {
	edit, err := s.editRepo.FindByID(ctx, editID)
	if err != nil {
		return err
	}
	if edit == nil {
		return errors.New("proposal not found")
	}
	if edit.Status != domain.EditProposalPending {
		return errors.New("این پیشنهاد قبلاً بررسی شده است")
	}

	// Decode the sparse changes.
	var raw map[string]interface{}
	if err := json.Unmarshal(edit.Changes, &raw); err != nil {
		return err
	}

	// Normalize types before applying. JSON decoding turns arrays into
	// []interface{}, which GORM can't map to Postgres text[] columns.
	// We convert known array fields back to pq.StringArray.
	changes := map[string]interface{}{}
	for k, v := range raw {
		switch k {
		case "tags", "images":
			// Accept []interface{} or []string.
			if arr, ok := v.([]interface{}); ok {
				strs := make([]string, 0, len(arr))
				for _, item := range arr {
					if s, ok := item.(string); ok {
						strs = append(strs, s)
					}
				}
				changes[k] = pq.StringArray(strs)
			} else if arr, ok := v.([]string); ok {
				changes[k] = pq.StringArray(arr)
			}
			// Skip if the value isn't an array at all.

		default:
			changes[k] = v
		}
	}

	if len(changes) == 0 {
		return errors.New("هیچ تغییری برای اعمال وجود ندارد")
	}

	// Apply to the prompt.
	if err := s.promptRepo.UpdateFields(ctx, edit.PromptID, changes); err != nil {
		return err
	}

	now := time.Now()
	if err := s.editRepo.UpdateFields(ctx, editID, map[string]interface{}{
		"status":      domain.EditProposalApproved,
		"reviewed_at": now,
	}); err != nil {
		return err
	}

	return s.promptRepo.SetHasPendingEdit(ctx, edit.PromptID, false)
}

func (s *AuthorDashboardService) RejectEdit(ctx context.Context, editID, note string) error {
	edit, err := s.editRepo.FindByID(ctx, editID)
	if err != nil {
		return err
	}
	if edit == nil {
		return errors.New("proposal not found")
	}
	if edit.Status != domain.EditProposalPending {
		return errors.New("این پیشنهاد قبلاً بررسی شده است")
	}

	now := time.Now()
	if err := s.editRepo.UpdateFields(ctx, editID, map[string]interface{}{
		"status":      domain.EditProposalRejected,
		"admin_note":  strings.TrimSpace(note),
		"reviewed_at": now,
	}); err != nil {
		return err
	}

	return s.promptRepo.SetHasPendingEdit(ctx, edit.PromptID, false)
}



// HELPER FUNCTION   
// deleteLocalUpload removes a file that lives under ./uploads/. Anything
// outside that directory (external URLs, for example) is ignored — we
// never delete files we don't own.
func deleteLocalUpload(url string) {
	const marker = "/uploads/"
	idx := strings.Index(url, marker)
	if idx == -1 {
		return
	}
	rel := url[idx+len(marker):]
	if strings.Contains(rel, "..") {
		return // refuse path traversal
	}
	_ = os.Remove(filepath.Join("./uploads", filepath.FromSlash(rel)))
}