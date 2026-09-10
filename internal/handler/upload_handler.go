package handler

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
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

// ensureDir - پوشه رو (به صورت بازگشتی) می‌سازه اگه وجود نداشته باشه
func ensureDir(path string) error {
	return os.MkdirAll(path, 0o755)
}

// UploadPromptImage - POST /api/v1/upload/prompt-image
// فیلد فرم: file
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

	if file.Size > maxImageSize {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "حجم فایل نباید بیشتر از 5 مگابایت باشد",
		})
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	if !allowedImageExts[ext] {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "فرمت فایل مجاز نیست (jpg, jpeg, png, webp, gif)",
		})
	}

	// مسیر: uploads/prompts/<userID>/
	subDir := filepath.Join("prompts", userID)
	fullDir := filepath.Join(h.uploadDir, subDir)

	// ✅ این خط حیاتی است: پوشه رو بساز (اگه نباشه)
	if err := ensureDir(fullDir); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "ساخت پوشه ناموفق بود: " + err.Error(),
		})
	}

	filename := fmt.Sprintf("%d_%s%s", time.Now().UnixNano(), uuid.New().String()[:8], ext)
	fullPath := filepath.Join(fullDir, filename)

	if err := c.SaveFile(file, fullPath); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "ذخیره فایل ناموفق بود: " + err.Error(),
		})
	}

	urlPath := "/uploads/" + filepath.ToSlash(filepath.Join(subDir, filename))
	publicURL := strings.TrimRight(h.baseURL, "/") + urlPath

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"data": fiber.Map{
			"url":      publicURL,
			"path":     urlPath,
			"filename": filename,
			"size":     file.Size,
		},
	})
}

// UploadPromptImages - POST /api/v1/upload/prompt-images (چندتایی)
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

	subDir := filepath.Join("prompts", userID)
	fullDir := filepath.Join(h.uploadDir, subDir)

	// ✅ این خط حیاتی است
	if err := ensureDir(fullDir); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "ساخت پوشه ناموفق بود: " + err.Error(),
		})
	}

	type uploaded struct {
		URL      string `json:"url"`
		Path     string `json:"path"`
		Filename string `json:"filename"`
		Size     int64  `json:"size"`
	}
	results := make([]uploaded, 0, len(files))

	for _, file := range files {
		if file.Size > maxImageSize {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": fmt.Sprintf("حجم فایل %s بیشتر از 5 مگابایت است", file.Filename),
			})
		}
		ext := strings.ToLower(filepath.Ext(file.Filename))
		if !allowedImageExts[ext] {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": fmt.Sprintf("فرمت %s مجاز نیست", file.Filename),
			})
		}

		filename := fmt.Sprintf("%d_%s%s", time.Now().UnixNano(), uuid.New().String()[:8], ext)
		fullPath := filepath.Join(fullDir, filename)

		if err := c.SaveFile(file, fullPath); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "ذخیره فایل ناموفق بود: " + err.Error(),
			})
		}

		urlPath := "/uploads/" + filepath.ToSlash(filepath.Join(subDir, filename))
		publicURL := strings.TrimRight(h.baseURL, "/") + urlPath

		results = append(results, uploaded{
			URL:      publicURL,
			Path:     urlPath,
			Filename: filename,
			Size:     file.Size,
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"data": results,
	})
}