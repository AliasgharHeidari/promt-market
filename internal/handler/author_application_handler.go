package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"promt-market/internal/repository"
	"promt-market/internal/service"
)

type AuthorApplicationHandler struct {
	service *service.AuthorApplicationService
}

func NewAuthorApplicationHandler(db *gorm.DB, redisClient *redis.Client) *AuthorApplicationHandler {
	appRepo := repository.NewAuthorApplicationRepository(db)
	userRepo := repository.NewUserRepository(db)
	verifyRepo := repository.NewVerificationRepository(redisClient)
	svc := service.NewAuthorApplicationService(appRepo, userRepo, verifyRepo)

	return &AuthorApplicationHandler{service: svc}
}

// SendPhoneOTP: POST /author-application/send-phone-otp
// body: { "phone": "+989123456789" }
func (h *AuthorApplicationHandler) SendPhoneOTP(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)

	var req struct {
		Phone string `json:"phone" validate:"required"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}
	if req.Phone == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "phone is required"})
	}

	if err := h.service.SendPhoneOTP(c.Context(), userID, req.Phone); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		// Reminder: no real SMS provider is wired up yet — check the
		// server terminal logs for the OTP code during testing.
		"message": "OTP sent. For now, check the server terminal logs for the code.",
	})
}

// VerifyPhone: POST /author-application/verify-phone
// body: { "code": "123456" }
func (h *AuthorApplicationHandler) VerifyPhone(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)

	var req struct {
		Code string `json:"code" validate:"required,len=6"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if err := h.service.VerifyPhoneOTP(c.Context(), userID, req.Code); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"message": "Phone verified successfully"})
}

// SubmitApplication: POST /author-application  (multipart/form-data)
// fields: expertise, national_id, id_document (file)
func (h *AuthorApplicationHandler) SubmitApplication(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)

	expertise := c.FormValue("expertise")
	nationalID := c.FormValue("national_id")

	fileHeader, err := c.FormFile("id_document")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "id_document file is required",
		})
	}

	app, err := h.service.SubmitApplication(c.Context(), userID, expertise, nationalID, fileHeader)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Application submitted successfully. Please wait for admin review.",
		"data": fiber.Map{
			"id":         app.ID,
			"status":     app.Status,
			"expertise":  app.Expertise,
			"created_at": app.CreatedAt,
		},
	})
}

// GetMyApplication: GET /author-application/me
func (h *AuthorApplicationHandler) GetMyApplication(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)

	app, err := h.service.GetMyApplication(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	if app == nil {
		return c.JSON(fiber.Map{"data": nil, "message": "No application submitted yet"})
	}

	return c.JSON(fiber.Map{
		"data": fiber.Map{
			"id":               app.ID,
			"status":           app.Status,
			"expertise":        app.Expertise,
			"rejection_reason": app.RejectionReason,
			"created_at":       app.CreatedAt,
			"reviewed_at":      app.ReviewedAt,
		},
	})
}