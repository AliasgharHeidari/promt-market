package service

import (
	"context"
	"errors"
	"strings"

	"github.com/microcosm-cc/bluemonday"

	"promt-market/internal/domain"
	"promt-market/internal/repository"
)

// MaxReviewCommentLen caps the comment length. Enforced both here and
// (optionally) in the DB check for defense in depth.
const MaxReviewCommentLen = 250

type ReviewService struct {
	reviewRepo *repository.ReviewRepository
	promptRepo *repository.PromptRepository
	orderRepo  *repository.OrderRepository

	// sanitizer strips any HTML/JS from user-submitted text so comments
	// are safe to render even if a future frontend forgets to escape.
	sanitizer *bluemonday.Policy
}

func NewReviewService(
	reviewRepo *repository.ReviewRepository,
	promptRepo *repository.PromptRepository,
	orderRepo *repository.OrderRepository,
) *ReviewService {
	// StrictPolicy strips ALL HTML tags. Review comments are plain text,
	// so we don't need any formatting — this is the safest choice.
	p := bluemonday.StrictPolicy()
	return &ReviewService{
		reviewRepo: reviewRepo,
		promptRepo: promptRepo,
		orderRepo:  orderRepo,
		sanitizer:  p,
	}
}

// Create adds a review for the given prompt. Rules:
//   - Rating must be 1–5.
//   - Comment is trimmed, sanitized, and capped at MaxReviewCommentLen runes.
//   - One review per (user, prompt).
//   - For paid prompts, the user must have a paid order.
//   - Reviews start as "pending" and require admin approval.
func (s *ReviewService) Create(ctx context.Context, userID, promptID string, req *domain.CreateReviewRequest) (*domain.Review, error) {
	if req.Rating < 1 || req.Rating > 5 {
		return nil, errors.New("امتیاز باید بین ۱ تا ۵ ستاره باشد")
	}

	comment := strings.TrimSpace(req.Comment)
	if comment == "" {
		return nil, errors.New("متن نظر الزامی است")
	}
	if len([]rune(comment)) > MaxReviewCommentLen {
		return nil, errors.New("متن نظر نباید بیشتر از ۲۵۰ کاراکتر باشد")
	}

	// Strip any HTML/script tags. Even though the frontend escapes text,
	// we never trust the client.
	comment = s.sanitizer.Sanitize(comment)
	comment = strings.TrimSpace(comment)
	if comment == "" {
		return nil, errors.New("متن نظر پس از پاک‌سازی خالی است")
	}

	// Prompt must exist and be visible (approved).
	prompt, err := s.promptRepo.FindByID(ctx, promptID)
	if err != nil {
		return nil, err
	}
	if prompt == nil {
		return nil, errors.New("پرامپت یافت نشد")
	}
	if prompt.Status != "approved" {
		return nil, errors.New("امکان ثبت نظر برای این پرامپت وجود ندارد")
	}

	// One review per user per prompt — regardless of status.
	existing, err := s.reviewRepo.FindByUserAndPrompt(ctx, userID, promptID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, errors.New("شما قبلاً برای این پرامپت نظر ثبت کرده‌اید")
	}

	// Paid prompts require a completed purchase.
	var orderID string
	if prompt.Price > 0 {
		order, err := s.orderRepo.FindPaidOrder(ctx, userID, promptID)
		if err != nil {
			return nil, err
		}
		if order == nil {
			return nil, errors.New("برای ثبت نظر باید ابتدا این پرامپت را خریداری کنید")
		}
		orderID = order.ID
	}

	review := &domain.Review{
		UserID:   userID,
		PromptID: promptID,
		OrderID:  orderID,
		Rating:   req.Rating,
		Comment:  comment,
		Status:   domain.ReviewStatusPending,
	}

	if err := s.reviewRepo.Create(ctx, review); err != nil {
		return nil, err
	}

	return review, nil
}

// ListApproved returns approved reviews for a prompt (public).
func (s *ReviewService) ListApproved(ctx context.Context, promptID string, page, limit int) ([]domain.Review, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 50 {
		limit = 10
	}
	offset := (page - 1) * limit
	return s.reviewRepo.FindApprovedByPrompt(ctx, promptID, limit, offset)
}

// DeleteOwn removes a review, but only if it belongs to userID AND is still
// pending (before admin approval). After approval, only an admin can remove.
func (s *ReviewService) DeleteOwn(ctx context.Context, reviewID, userID string) error {
	review, err := s.reviewRepo.FindByID(ctx, reviewID)
	if err != nil {
		return err
	}
	if review == nil {
		return errors.New("نظر یافت نشد")
	}
	if review.UserID != userID {
		return errors.New("شما مالک این نظر نیستید")
	}
	if review.Status != domain.ReviewStatusPending {
		return errors.New("فقط نظرات در انتظار تایید قابل حذف هستند")
	}
	return s.reviewRepo.Delete(ctx, reviewID)
}

// ============================================================
//  ADMIN
// ============================================================

// ListByStatus returns reviews for admin moderation (any status).
func (s *ReviewService) ListByStatus(ctx context.Context, status string, page, limit int) ([]domain.Review, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit
	return s.reviewRepo.FindByStatus(ctx, status, limit, offset)
}

// ApproveReview flips a review to approved and recalculates the prompt's
// aggregate rating/count in a single logical operation.
func (s *ReviewService) ApproveReview(ctx context.Context, reviewID string) error {
	review, err := s.reviewRepo.FindByID(ctx, reviewID)
	if err != nil {
		return err
	}
	if review == nil {
		return errors.New("نظر یافت نشد")
	}

	if err := s.reviewRepo.UpdateFields(ctx, reviewID, map[string]interface{}{
		"status":           domain.ReviewStatusApproved,
		"rejection_reason": "",
	}); err != nil {
		return err
	}

	return s.recalcPromptRating(ctx, review.PromptID)
}

// RejectReview marks a review as rejected with an optional reason.
func (s *ReviewService) RejectReview(ctx context.Context, reviewID, reason string) error {
	review, err := s.reviewRepo.FindByID(ctx, reviewID)
	if err != nil {
		return err
	}
	if review == nil {
		return errors.New("نظر یافت نشد")
	}

	reason = strings.TrimSpace(reason)
	reason = s.sanitizer.Sanitize(reason)

	if err := s.reviewRepo.UpdateFields(ctx, reviewID, map[string]interface{}{
		"status":           domain.ReviewStatusRejected,
		"rejection_reason": reason,
	}); err != nil {
		return err
	}

	// If the review had previously been approved (edge case), recalc.
	return s.recalcPromptRating(ctx, review.PromptID)
}

// DeleteReview is the admin hard-delete path. Also recalcs the prompt's
// rating aggregate so a deleted approved review stops counting.
func (s *ReviewService) DeleteReview(ctx context.Context, reviewID string) (*domain.Review, error) {
	review, err := s.reviewRepo.FindByID(ctx, reviewID)
	if err != nil {
		return nil, err
	}
	if review == nil {
		return nil, errors.New("نظر یافت نشد")
	}

	if err := s.reviewRepo.Delete(ctx, reviewID); err != nil {
		return nil, err
	}

	_ = s.recalcPromptRating(ctx, review.PromptID)
	return review, nil
}

// recalcPromptRating recomputes Prompt.Rating and Prompt.ReviewCount from
// the approved review set. Best-effort: a failure here does not roll back
// the caller's action, but it will be visible in logs.
func (s *ReviewService) recalcPromptRating(ctx context.Context, promptID string) error {
	avg, count, err := s.reviewRepo.AggregateRating(ctx, promptID)
	if err != nil {
		return err
	}
	return s.promptRepo.UpdateFields(ctx, promptID, map[string]interface{}{
		"rating":       avg,
		"review_count": count,
	})
}