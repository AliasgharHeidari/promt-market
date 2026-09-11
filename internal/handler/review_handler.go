package handler

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"promt-market/internal/domain"
	"promt-market/internal/repository"
	"promt-market/internal/service"
)

type ReviewHandler struct {
	service *service.ReviewService
}

func NewReviewHandler(db *gorm.DB) *ReviewHandler {
	reviewRepo := repository.NewReviewRepository(db)
	promptRepo := repository.NewPromptRepository(db)
	orderRepo := repository.NewOrderRepository(db)
	reviewService := service.NewReviewService(reviewRepo, promptRepo, orderRepo)

	return &ReviewHandler{service: reviewService}
}

// CreateReview adds a review for a prompt. The user must be authenticated
// (JWTProtected middleware) and — for paid prompts — must have purchased it.
func (h *ReviewHandler) CreateReview(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)
	promptID := c.Params("id")

	var req domain.CreateReviewRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	review, err := h.service.Create(c.Context(), userID, promptID, &req)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "نظر شما ثبت شد و پس از تایید ادمین نمایش داده می‌شود",
		"data":    review,
	})
}

// ListReviews returns approved reviews for a prompt (public).
func (h *ReviewHandler) ListReviews(c *fiber.Ctx) error {
	promptID := c.Params("id")
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "10"))

	reviews, total, err := h.service.ListApproved(c.Context(), promptID, page, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Strip sensitive user fields before returning. Only expose full_name.
	out := make([]domain.ReviewWithUser, 0, len(reviews))
	for _, r := range reviews {
		item := domain.ReviewWithUser{
			ID:        r.ID,
			Rating:    r.Rating,
			Comment:   r.Comment,
			CreatedAt: r.CreatedAt,
		}
		item.User.FullName = r.User.FullName
		out = append(out, item)
	}

	return c.JSON(fiber.Map{
		"data": out,
		"pagination": fiber.Map{
			"page":  page,
			"limit": limit,
			"total": total,
			"pages": (total + int64(limit) - 1) / int64(limit),
		},
	})
}

// DeleteOwnReview lets a user delete their own review before approval.
func (h *ReviewHandler) DeleteOwnReview(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)
	reviewID := c.Params("id")

	if err := h.service.DeleteOwn(c.Context(), reviewID, userID); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "نظر حذف شد",
	})
}