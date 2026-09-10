package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"promt-market/internal/domain"
	"promt-market/internal/repository"
)

const (
	phoneOTPTTL      = 5 * time.Minute
	phoneOTPCooldown = 60 * time.Second
	idDocumentDir    = "./uploads/id_documents" // server-local disk, per user's choice
)

type AuthorApplicationService struct {
	appRepo    *repository.AuthorApplicationRepository
	userRepo   *repository.UserRepository
	verifyRepo *repository.VerificationRepository
}

func NewAuthorApplicationService(
	appRepo *repository.AuthorApplicationRepository,
	userRepo *repository.UserRepository,
	verifyRepo *repository.VerificationRepository,
) *AuthorApplicationService {
	return &AuthorApplicationService{
		appRepo:    appRepo,
		userRepo:   userRepo,
		verifyRepo: verifyRepo,
	}
}

// ============================================================
//  PHONE OTP
// ============================================================

// SendPhoneOTP generates a 6-digit code for the given phone number and
// "sends" it. There is no SMS provider wired up yet, so for now the code
// is printed to the server terminal — swap the log.Printf below for a real
// SMS client call once one is purchased/configured.
func (s *AuthorApplicationService) SendPhoneOTP(ctx context.Context, userID, phone string) error {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return err
	}
	if user == nil {
		return errors.New("user not found")
	}

	phone = strings.TrimSpace(phone)
	if phone == "" {
		return errors.New("phone number is required")
	}

	onCooldown, err := s.verifyRepo.IsPhoneOTPOnCooldown(ctx, userID)
	if err != nil {
		return err
	}
	if onCooldown {
		return errors.New("please wait before requesting another code")
	}

	code := generatePhoneOTP()
	if err := s.verifyRepo.SavePhoneOTP(ctx, userID, code, phoneOTPTTL); err != nil {
		return fmt.Errorf("failed to save phone otp: %w", err)
	}
	_ = s.verifyRepo.SetPhoneOTPCooldown(ctx, userID, phoneOTPCooldown)

	// Stash the phone number on the (not-yet-verified) user row so
	// VerifyPhoneOTP knows which number is being confirmed, without
	// trusting a phone number passed again on the verify call.
	user.Phone = phone
	user.PhoneVerified = false
	if err := s.userRepo.Update(ctx, user); err != nil {
		return err
	}

	// TODO: replace with real SMS provider once purchased.
	log.Printf("📱 [SMS SIMULATION] OTP for user %s (phone %s): %s (expires in %s)\n",
		userID, phone, code, phoneOTPTTL)

	return nil
}

// VerifyPhoneOTP checks the code and marks the user's phone as verified.
func (s *AuthorApplicationService) VerifyPhoneOTP(ctx context.Context, userID, code string) error {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return err
	}
	if user == nil {
		return errors.New("user not found")
	}
	if user.Phone == "" {
		return errors.New("no phone number on file, request an OTP first")
	}
	if user.PhoneVerified {
		return errors.New("phone already verified")
	}

	storedCode, err := s.verifyRepo.GetPhoneOTP(ctx, userID)
	if err != nil {
		return errors.New("verification code expired or not found")
	}
	if storedCode != code {
		return errors.New("invalid verification code")
	}

	user.PhoneVerified = true
	if err := s.userRepo.Update(ctx, user); err != nil {
		return err
	}

	_ = s.verifyRepo.DeletePhoneOTP(ctx, userID)
	return nil
}

// ============================================================
//  AUTHOR APPLICATION
// ============================================================

// SubmitApplication validates preconditions (verified phone) and stores
// the KYC application (expertise, national ID, ID document image) with
// status "pending" for admin review. The ID document is saved to local
// disk under idDocumentDir.
func (s *AuthorApplicationService) SubmitApplication(
	ctx context.Context,
	userID string,
	expertise string,
	nationalID string,
	file *multipart.FileHeader,
) (*domain.AuthorApplication, error) {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.New("user not found")
	}
	if user.Role == "prompt_author" || user.Role == "admin" {
		return nil, errors.New("user is already a prompt author")
	}
	if !user.PhoneVerified {
		return nil, errors.New("phone number must be verified before applying")
	}

	expertise = strings.TrimSpace(expertise)
	nationalID = strings.TrimSpace(nationalID)
	if expertise == "" {
		return nil, errors.New("expertise is required")
	}
	if err := validateNationalID(nationalID); err != nil {
		return nil, err
	}
	if file == nil {
		return nil, errors.New("ID document image is required")
	}

	// Block duplicate applications while one is pending or already approved.
	latest, err := s.appRepo.FindLatestByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if latest != nil {
		switch latest.Status {
		case "pending":
			return nil, errors.New("you already have a pending application")
		case "approved":
			return nil, errors.New("your application was already approved")
		}
		// "rejected" -> allowed to re-apply
	}

	savedPath, err := saveIDDocument(userID, file)
	if err != nil {
		return nil, fmt.Errorf("failed to save ID document: %w", err)
	}

	app := &domain.AuthorApplication{
		UserID:         userID,
		Expertise:      expertise,
		NationalID:     nationalID,
		IDDocumentPath: savedPath,
		Status:         "pending",
	}

	if err := s.appRepo.Create(ctx, app); err != nil {
		return nil, err
	}

	// Keep expertise/national ID visible on the user record too, for
	// convenience, even before the application is approved.
	user.Expertise = expertise
	user.NationalID = nationalID
	_ = s.userRepo.Update(ctx, user)

	return app, nil
}

// GetMyApplication returns the caller's most recent application, if any.
func (s *AuthorApplicationService) GetMyApplication(ctx context.Context, userID string) (*domain.AuthorApplication, error) {
	return s.appRepo.FindLatestByUserID(ctx, userID)
}

// ============================================================
//  ADMIN REVIEW
// ============================================================

func (s *AuthorApplicationService) GetApplicationsByStatus(ctx context.Context, status string, page, limit int) ([]domain.AuthorApplication, int64, error) {
	return s.appRepo.GetByStatus(ctx, status, page, limit)
}

func (s *AuthorApplicationService) GetApplicationByID(ctx context.Context, id string) (*domain.AuthorApplication, error) {
	return s.appRepo.FindByID(ctx, id)
}

// ApproveApplication marks the application approved and promotes the
// user's role to "prompt_author".
func (s *AuthorApplicationService) ApproveApplication(ctx context.Context, appID, adminID string) (*domain.AuthorApplication, error) {
	app, err := s.appRepo.FindByID(ctx, appID)
	if err != nil {
		return nil, err
	}
	if app == nil {
		return nil, errors.New("application not found")
	}
	if app.Status != "pending" {
		return nil, fmt.Errorf("application is already %s", app.Status)
	}

	user, err := s.userRepo.FindByID(ctx, app.UserID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.New("applicant user not found")
	}

	now := time.Now()
	app.Status = "approved"
	app.ReviewedBy = &adminID
	app.ReviewedAt = &now
	if err := s.appRepo.Update(ctx, app); err != nil {
		return nil, err
	}

	user.Role = "prompt_author"
	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, err
	}

	return app, nil
}

// RejectApplication marks the application rejected with a reason. The user
// keeps their current role and may submit a new application afterwards.
func (s *AuthorApplicationService) RejectApplication(ctx context.Context, appID, adminID, reason string) (*domain.AuthorApplication, error) {
	app, err := s.appRepo.FindByID(ctx, appID)
	if err != nil {
		return nil, err
	}
	if app == nil {
		return nil, errors.New("application not found")
	}
	if app.Status != "pending" {
		return nil, fmt.Errorf("application is already %s", app.Status)
	}

	now := time.Now()
	app.Status = "rejected"
	app.RejectionReason = strings.TrimSpace(reason)
	app.ReviewedBy = &adminID
	app.ReviewedAt = &now

	if err := s.appRepo.Update(ctx, app); err != nil {
		return nil, err
	}

	return app, nil
}

// ============================================================
//  HELPERS
// ============================================================

func generatePhoneOTP() string {
	b := make([]byte, 3)
	if _, err := rand.Read(b); err != nil {
		return "000000"
	}
	number := int(b[0])<<16 | int(b[1])<<8 | int(b[2])
	return fmt.Sprintf("%06d", number%1000000)
}

// validateNationalID does a light sanity check (Iranian national ID is
// 10 digits). Adjust/replace with a proper checksum validator if needed.
func validateNationalID(id string) error {
	if len(id) != 10 {
		return errors.New("national ID must be 10 digits")
	}
	for _, r := range id {
		if r < '0' || r > '9' {
			return errors.New("national ID must contain only digits")
		}
	}
	return nil
}

// saveIDDocument writes the uploaded file to local disk under
// idDocumentDir/<userID>/<timestamp>_<originalname> and returns the
// stored path. Only image-ish extensions are accepted.
func saveIDDocument(userID string, file *multipart.FileHeader) (string, error) {
	ext := strings.ToLower(filepath.Ext(file.Filename))
	allowed := map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".webp": true, ".pdf": true}
	if !allowed[ext] {
		return "", errors.New("unsupported file type, use jpg/png/webp/pdf")
	}
	const maxSize = 8 << 20 // 8MB
	if file.Size > maxSize {
		return "", errors.New("file too large (max 8MB)")
	}

	userDir := filepath.Join(idDocumentDir, userID)
	if err := os.MkdirAll(userDir, 0o750); err != nil {
		return "", err
	}

	filename := fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)
	destPath := filepath.Join(userDir, filename)

	src, err := file.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()

	dst, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o640)
	if err != nil {
		return "", err
	}
	defer dst.Close()

	buf := make([]byte, 32*1024)
	for {
		n, readErr := src.Read(buf)
		if n > 0 {
			if _, writeErr := dst.Write(buf[:n]); writeErr != nil {
				return "", writeErr
			}
		}
		if readErr != nil {
			break
		}
	}

	return destPath, nil
}