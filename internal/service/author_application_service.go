package service

import (
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"promt-market/internal/domain"
	"promt-market/internal/repository"
	"promt-market/internal/utils"
)

const (
	phoneOTPTTL      = 5 * time.Minute
	phoneOTPCooldown = 60 * time.Second

	// idDocumentDir is DELIBERATELY outside ./uploads. ./uploads is served
	// publicly via app.Static("/uploads", "./uploads") in routes.go, so
	// anything placed under it is reachable by anyone who guesses/leaks a
	// URL — completely wrong for a national ID document. Keeping ID
	// documents in a separate, non-served directory means the only way to
	// read one back is through a dedicated, authenticated handler (admin
	// review), which does not exist yet as an HTTP-exposed route.
	idDocumentDir = "./private/id_documents"
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

// idDocumentAllowedExts maps an accepted extension to how its content
// should be verified. Images are verified via utils.ValidateImageContent
// (decodes the real header); PDF is verified via its magic bytes below.
var idDocumentAllowedExts = map[string]bool{
	".jpg": true, ".jpeg": true, ".png": true, ".webp": true, ".pdf": true,
}

// pdfMagicBytes is the fixed 5-byte signature every valid PDF file starts
// with ("%PDF-"). Checking this is the same class of validation as
// image.DecodeConfig for images: it confirms the bytes are actually what
// the extension claims, rather than trusting the filename.
var pdfMagicBytes = []byte("%PDF-")

// validateIDDocumentContent checks that data's real content matches the
// claimed extension. This is a KYC document (national ID proof) — treating
// its content with more suspicion than a decorative prompt-cover image is
// warranted, since a mismatched/malicious file here could later be opened
// by an admin reviewer.
func validateIDDocumentContent(data []byte, ext string) error {
	switch ext {
	case ".jpg", ".jpeg", ".png", ".webp":
		if _, err := utils.ValidateImageContent(data, ext); err != nil {
			return err
		}
		return nil
	case ".pdf":
		if !bytes.HasPrefix(data, pdfMagicBytes) {
			return errors.New("محتوای فایل یک PDF معتبر نیست")
		}
		return nil
	default:
		return fmt.Errorf("پسوند فایل مجاز نیست: %s", ext)
	}
}

// saveIDDocument writes the uploaded file to local disk under
// idDocumentDir/<userID>/<timestamp><ext> and returns the stored path.
// The file's actual content is validated against its claimed extension
// before anything is written — see validateIDDocumentContent.
func saveIDDocument(userID string, file *multipart.FileHeader) (string, error) {
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if !idDocumentAllowedExts[ext] {
		return "", errors.New("unsupported file type, use jpg/png/webp/pdf")
	}
	const maxSize = 8 << 20 // 8MB
	if file.Size > maxSize {
		return "", errors.New("file too large (max 8MB)")
	}

	src, err := file.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(io.LimitReader(src, maxSize+1)); err != nil {
		return "", err
	}
	data := buf.Bytes()
	if int64(len(data)) > maxSize {
		// Defense in depth: FileHeader.Size can be client-reported in some
		// multipart implementations, so re-check the bytes actually read.
		return "", errors.New("file too large (max 8MB)")
	}

	if err := validateIDDocumentContent(data, ext); err != nil {
		return "", fmt.Errorf("فایل نامعتبر است: %w", err)
	}

	userDir := filepath.Join(idDocumentDir, userID)
	if err := os.MkdirAll(userDir, 0o750); err != nil {
		return "", err
	}

	filename := fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)
	destPath := filepath.Join(userDir, filename)

	if err := os.WriteFile(destPath, data, 0o640); err != nil {
		return "", err
	}

	return destPath, nil
}