package handler

import (
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"promt-market/internal/repository"
	"promt-market/internal/service"
)

type AuthHandler struct {
	service   *service.AuthService
	validator *validator.Validate
}

func NewAuthHandler(db *gorm.DB, redisClient *redis.Client, emailService *service.EmailService) *AuthHandler {
	userRepo := repository.NewUserRepository(db)
	verifyRepo := repository.NewVerificationRepository(redisClient)
	authService := service.NewAuthService(userRepo, verifyRepo, emailService)

	return &AuthHandler{
		service:   authService,
		validator: validator.New(),
	}
}

func (h *AuthHandler) Register(c *fiber.Ctx) error {
	var req service.RegisterRequest

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	user, err := h.service.Register(c.Context(), &req)
	if err != nil {
		msg := err.Error()

		// Validation errors from the service start with Persian letters.
		// We can distinguish them from internal errors by message prefix —
		// a simpler alternative is a typed error, but this keeps the
		// service layer dependency-free.
		if isValidationError(msg) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": msg,
			})
		}

		switch msg {
		case "user already exists with this email":
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error": msg,
			})
		case "user already registered but not verified. Please use the resend verification endpoint":
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error": msg,
			})
		}

		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to register user",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "User registered successfully. Please check your email for verification code.",
		"data": fiber.Map{
			"id":        user.ID,
			"email":     user.Email,
			"full_name": user.FullName,
			"role":      user.Role,
		},
	})
}

// isValidationError returns true when msg is a user-facing validation error
// (written in Persian) rather than an internal/English error. This is a
// pragmatic shortcut — for larger codebases, define a typed ValidationError
// in the service package instead.
func isValidationError(msg string) bool {
	if msg == "" {
		return false
	}
	// Persian Unicode range starts at U+0600. If the first rune is Persian,
	// it's a validation error we authored ourselves.
	r := []rune(msg)[0]
	return r >= 0x0600 && r <= 0x06FF
}

func (h *AuthHandler) VerifyEmail(c *fiber.Ctx) error {
	var req service.VerifyEmailRequest

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if err := h.validator.Struct(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	if err := h.service.VerifyEmail(c.Context(), &req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Email verified successfully. You can now login.",
	})
}

// ResendVerification issues a fresh verification code for a user who registered
// but hasn't verified their email yet (e.g. the original email was lost, expired,
// or never arrived).
func (h *AuthHandler) ResendVerification(c *fiber.Ctx) error {
	var req service.ResendVerificationRequest

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if err := h.validator.Struct(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	if err := h.service.ResendVerification(c.Context(), &req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Verification code resent. Please check your email.",
	})
}

func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var req service.LoginRequest

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if err := h.validator.Struct(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	tokenPair, user, err := h.service.Login(c.Context(), &req)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Login successful",
		"data": fiber.Map{
			"user": fiber.Map{
				"id":        user.ID,
				"email":     user.Email,
				"full_name": user.FullName,
				"role":      user.Role,
				"rating":    user.Rating,
			},
			"tokens": tokenPair,
		},
	})
}

func (h *AuthHandler) RefreshToken(c *fiber.Ctx) error {
	var req service.RefreshRequest

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if err := h.validator.Struct(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	tokenPair, err := h.service.RefreshTokens(c.Context(), req.RefreshToken)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Tokens refreshed successfully",
		"data":    tokenPair,
	})
}

func (h *AuthHandler) GetProfile(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)

	user, err := h.service.GetUserByID(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get user profile",
		})
	}
	if user == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"data": fiber.Map{
			"id":             user.ID,
			"email":          user.Email,
			"full_name":      user.FullName,
			"expertise":      user.Expertise,
			"bio":            user.Bio,
			"rating":         user.Rating,
			"total_sales":    user.TotalSales,
			"wallet_balance": user.WalletBalance,
			"role":           user.Role,
		},
	})
}