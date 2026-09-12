package handler

import (
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"promt-market/internal/utils"
)

type UploadHandler struct {
	uploadDir string
	baseURL   string
}

func NewUploadHandler(uploadDir, baseURL string) *UploadHandler {
	return &UploadHandler{
		uploadDir: uploadDir,
		baseURL:   baseURL,
	}
}

var allowedImageExts = map[string]bool{
	".jpg":  true,
	".jpeg": true,
	".png":  true,
	".webp": true,
	".gif":  true,
}

const maxImageSize = 5 * 1024 * 1024 // 5MB

// ensureDir creates path recursively if it doesn't exist.
func ensureDir(path string) error {
	return os.MkdirAll(path, 0o755)
}

// saveImageAs stores a single uploaded image into uploads/<subDir>/<ownerID>/
// and returns the public URL. Shared by prompt-image and avatar uploads so
// validation rules stay consistent.
//
// After the file is written to disk, it is passed through CompressImage
// which may:
//   - downscale oversized images (longest side > 1200px)
//   - re-encode JPEG at quality 85
//   - re-encode PNG losslessly at max compression
//
// Small files (<2 MB) and non-compressible formats (WebP/GIF) are left as-is.
func (h *UploadHandler) saveImageAs(file *multipart.FileHeader, subDir, ownerID string) (string, error) {
	if file.Size > maxImageSize {
		return "", fmt.Errorf("حجم فایل نباید بیشتر از 5 مگابایت باشد")
	}
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if !allowedImageExts[ext] {
		return "", fmt.Errorf("فرمت فایل مجاز نیست (jpg, jpeg, png, webp, gif)")
	}

	dir := filepath.Join(h.uploadDir, subDir, ownerID)
	if err := ensureDir(dir); err != nil {
		return "", fmt.Errorf("ساخت پوشه ناموفق بود: %w", err)
	}

	filename := fmt.Sprintf("%d_%s%s", time.Now().UnixNano(), uuid.New().String()[:8], ext)
	fullPath := filepath.Join(dir, filename)

	src, err := file.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()

	dst, err := os.Create(fullPath)
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(dst, src); err != nil {
		dst.Close()
		return "", err
	}
	if err := dst.Close(); err != nil {
		return "", err
	}

	// Best-effort compression. If it fails, we still keep the original file
	// — the upload is not rejected because of a compression error.
	if _, err := utils.CompressImage(fullPath); err != nil {
		// Log via fmt for now; replace with your logger if you have one.
		fmt.Printf("⚠️ compress failed for %s: %v\n", fullPath, err)
	}

	urlPath := "/uploads/" + filepath.ToSlash(filepath.Join(subDir, ownerID, filename))
	return strings.TrimRight(h.baseURL, "/") + urlPath, nil
}

// UploadPromptImage - POST /api/v1/upload/prompt-image
// Form field: file
func (h *UploadHandler) UploadPromptImage(c *fiber.Ctx) error {
	userID, _ := c.Locals("userID").(string)
	if userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "unauthorized",
		})
	}

	file, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "فایل ارسال نشده است (فیلد file الزامی است)",
		})
	}

	url, err := h.saveImageAs(file, "prompts", userID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"data": fiber.Map{
			"url": url,
		},
	})
}

// UploadPromptImages - POST /api/v1/upload/prompt-images (multiple)
// Form field: files
func (h *UploadHandler) UploadPromptImages(c *fiber.Ctx) error {
	userID, _ := c.Locals("userID").(string)
	if userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "unauthorized",
		})
	}

	form, err := c.MultipartForm()
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "فرم نامعتبر است",
		})
	}

	files := form.File["files"]
	if len(files) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "هیچ فایلی ارسال نشده است",
		})
	}
	if len(files) > 10 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "حداکثر 10 تصویر مجاز است",
		})
	}

	type uploaded struct {
		URL string `json:"url"`
	}
	results := make([]uploaded, 0, len(files))

	for _, file := range files {
		url, err := h.saveImageAs(file, "prompts", userID)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": fmt.Sprintf("%s: %s", file.Filename, err.Error()),
			})
		}
		results = append(results, uploaded{URL: url})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"data": results,
	})
}