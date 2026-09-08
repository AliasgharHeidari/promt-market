// internal/handler/auth_handler.go
package handler

import (

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"promt-market/internal/repository"
	"promt-market/internal/service"
)

type AuthHandler struct {
	service   *service.AuthService
	validator *validator.Validate
}

func NewAuthHandler(db *gorm.DB) *AuthHandler {
	userRepo := repository.NewUserRepository(db)
	authService := service.NewAuthService(userRepo)

	return &AuthHandler{
		service:   authService,
		validator: validator.New(),
	}
}

// Register handles user registration
// @Summary Register new user
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body service.RegisterRequest true "Registration data"
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 409 {object} map[string]interface{}
// @Router /auth/register [post]
func (h *AuthHandler) Register(c *fiber.Ctx) error {
	var req service.RegisterRequest

	// Parse body
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate
	if err := h.validator.Struct(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Call service
	user, err := h.service.Register(c.Context(), &req)
	if err != nil {
		if err.Error() == "user already exists with this email" {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to register user",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "User registered successfully",
		"data": fiber.Map{
			"id":        user.ID,
			"email":     user.Email,
			"full_name": user.FullName,
			"role":      user.Role,
		},
	})
}

// Login handles user login
// @Summary Login user
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body service.LoginRequest true "Login credentials"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Router /auth/login [post]
func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var req service.LoginRequest

	// Parse body
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate
	if err := h.validator.Struct(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Call service
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

// RefreshToken handles token refresh
// @Summary Refresh tokens
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body service.RefreshRequest true "Refresh token"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Router /auth/refresh [post]
func (h *AuthHandler) RefreshToken(c *fiber.Ctx) error {
	var req service.RefreshRequest

	// Parse body
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate
	if err := h.validator.Struct(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Call service
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

// GetProfile returns the current user's profile
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