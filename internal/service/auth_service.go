package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	"promt-market/internal/domain"
	"promt-market/internal/repository"
	"promt-market/pkg/jwt"
)

// AuthService handles user authentication and registration logic.
type AuthService struct {
	userRepo   *repository.UserRepository
	verifyRepo *repository.VerificationRepository
	emailSvc   *EmailService
}

// NewAuthService creates a new instance of AuthService.
func NewAuthService(
	userRepo *repository.UserRepository,
	verifyRepo *repository.VerificationRepository,
	emailSvc *EmailService,
) *AuthService {
	return &AuthService{
		userRepo:   userRepo,
		verifyRepo: verifyRepo,
		emailSvc:   emailSvc,
	}
}

// RegisterRequest represents the payload for user registration.
type RegisterRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
	FullName string `json:"full_name" validate:"required"`
}

// LoginRequest represents the payload for user login.
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// RefreshRequest represents the payload for refreshing tokens.
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// VerifyEmailRequest represents the payload for email verification.
type VerifyEmailRequest struct {
	Email string `json:"email" validate:"required,email"`
	Code  string `json:"code" validate:"required,len=6"`
}

// ResendVerificationRequest represents the payload for resending a verification code.
type ResendVerificationRequest struct {
	Email string `json:"email" validate:"required,email"`
}

func (s *AuthService) Register(ctx context.Context, req *RegisterRequest) (*domain.User, error) {
	// ---- Manual validation ----
	// We validate inside the service (not the handler) so the rules are
	// colocated with business logic and can reuse the same errors for any
	// transport layer (HTTP, gRPC, CLI...).

	email := strings.ToLower(strings.TrimSpace(req.Email))
	fullName := strings.TrimSpace(req.FullName)

	if email == "" {
		return nil, errors.New("ایمیل الزامی است")
	}
	if !strings.Contains(email, "@") || !strings.Contains(email, ".") {
		return nil, errors.New("ایمیل معتبر نیست")
	}
	if req.Password == "" {
		return nil, errors.New("رمز عبور الزامی است")
	}
	if len(req.Password) < 8 {
		return nil, errors.New("رمز عبور باید حداقل ۸ کاراکتر باشد")
	}
	if fullName == "" {
		return nil, errors.New("نام کامل الزامی است")
	}
	if len(fullName) < 2 {
		return nil, errors.New("نام کامل باید حداقل ۲ کاراکتر باشد")
	}

	// ---- Business logic ----

	// 1. Check if the user already exists
	existingUser, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if existingUser != nil {
		if existingUser.IsVerified {
			return nil, errors.New("user already exists with this email")
		}
		return nil, errors.New("user already registered but not verified. Please use the resend verification endpoint")
	}

	// 2. Hash the user's password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, errors.New("failed to hash password")
	}

	// 3. Create the new user object
	user := &domain.User{
		Email:        email,
		PasswordHash: string(hashedPassword),
		FullName:     fullName,
		Role:         "user",
		IsActive:     true,
		IsVerified:   false,
	}

	// 4. Save the user to the database
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	// 5. Generate and save a secure verification code
	code := generateVerificationCode()
	if err := s.verifyRepo.SaveVerificationCode(ctx, user.Email, code, 10*time.Minute); err != nil {
		return nil, fmt.Errorf("failed to save verification code: %w", err)
	}

	// 6. Send the verification email asynchronously
	go func() {
		err := s.emailSvc.SendVerificationEmail(user.Email, user.FullName, code)
		if err != nil {
			log.Printf("❌ [EMAIL ERROR] Failed to send verification email to %s: %v\n", user.Email, err)
		} else {
			log.Printf("✅ [EMAIL SUCCESS] Verification email sent successfully to %s\n", user.Email)
		}
	}()

	return user, nil
}
// ResendVerification generates a new verification code for an existing, unverified
// user and sends it via email. This lets users who never received or who lost
// their original code get a new one without being stuck behind "already exists".
func (s *AuthService) ResendVerification(ctx context.Context, req *ResendVerificationRequest) error {
	email := strings.ToLower(req.Email)

	user, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		return err
	}
	if user == nil {
		return errors.New("user not found")
	}
	if user.IsVerified {
		return errors.New("email already verified")
	}

	code := generateVerificationCode()
	if err := s.verifyRepo.SaveVerificationCode(ctx, user.Email, code, 10*time.Minute); err != nil {
		return fmt.Errorf("failed to save verification code: %w", err)
	}

	go func() {
		err := s.emailSvc.SendVerificationEmail(user.Email, user.FullName, code)
		if err != nil {
			log.Printf("❌ [EMAIL ERROR] Failed to resend verification email to %s: %v\n", user.Email, err)
		} else {
			log.Printf("✅ [EMAIL SUCCESS] Verification email resent successfully to %s\n", user.Email)
		}
	}()

	return nil
}

// VerifyEmail checks the provided code and marks the user as verified.
func (s *AuthService) VerifyEmail(ctx context.Context, req *VerifyEmailRequest) error {
	email := strings.ToLower(req.Email)

	user, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		return err
	}
	if user == nil {
		return errors.New("user not found")
	}
	if user.IsVerified {
		return errors.New("email already verified")
	}

	storedCode, err := s.verifyRepo.GetVerificationCode(ctx, email)
	if err != nil {
		return errors.New("verification code expired or not found")
	}
	if storedCode != req.Code {
		return errors.New("invalid verification code")
	}

	// Update user status and delete the used code
	user.IsVerified = true
	if err := s.userRepo.Update(ctx, user); err != nil {
		return err
	}

	_ = s.verifyRepo.DeleteVerificationCode(ctx, email)
	return nil
}

// Login authenticates the user and returns a JWT token pair.
func (s *AuthService) Login(ctx context.Context, req *LoginRequest) (*jwt.TokenPair, *domain.User, error) {
	user, err := s.userRepo.FindByEmail(ctx, strings.ToLower(req.Email))
	if err != nil {
		return nil, nil, err
	}
	if user == nil {
		return nil, nil, errors.New("invalid credentials")
	}
	if !user.IsVerified {
		return nil, nil, errors.New("email not verified. Please verify your email first")
	}
	if !user.IsActive {
		return nil, nil, errors.New("account is deactivated")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, nil, errors.New("invalid credentials")
	}

	tokenPair, err := jwt.GenerateTokenPair(user.ID, user.Email, user.Role)
	if err != nil {
		return nil, nil, err
	}

	return tokenPair, user, nil
}

// RefreshTokens validates the refresh token and returns a new token pair.
func (s *AuthService) RefreshTokens(ctx context.Context, refreshToken string) (*jwt.TokenPair, error) {
	claims, err := jwt.ValidateRefreshToken(refreshToken)
	if err != nil {
		return nil, errors.New("invalid refresh token")
	}

	user, err := s.userRepo.FindByID(ctx, claims.UserID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.New("user not found")
	}
	if !user.IsActive {
		return nil, errors.New("account is deactivated")
	}

	tokenPair, err := jwt.GenerateTokenPair(user.ID, user.Email, user.Role)
	if err != nil {
		return nil, err
	}

	return tokenPair, nil
}

// GetUserByID retrieves a user by their ID.
func (s *AuthService) GetUserByID(ctx context.Context, id string) (*domain.User, error) {
	return s.userRepo.FindByID(ctx, id)
}

// generateVerificationCode generates a secure 6-digit code using crypto/rand.
func generateVerificationCode() string {
	b := make([]byte, 3) // 3 bytes are sufficient for a 6-digit number

	// Read cryptographically secure random bytes
	if _, err := rand.Read(b); err != nil {
		// Fallback in the extremely rare case of a system random generator failure
		return "000000"
	}

	// Convert bytes to an integer and format as a 6-digit string
	number := int(b[0])<<16 | int(b[1])<<8 | int(b[2])
	return fmt.Sprintf("%06d", number%1000000)
}