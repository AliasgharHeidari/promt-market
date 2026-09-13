package middleware

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"promt-market/internal/domain"
)

func AuditLog(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		err := c.Next()

		role := c.Locals("role")
		if role != "admin" {
			return err
		}

		// Safe assertion: if userID was somehow never set on this request
		// (e.g. this middleware ends up wired on a route where JWTProtected
		// didn't run first, or a future refactor changes the middleware
		// order), we must not panic mid-response. We just skip audit
		// logging rather than crash the request — a request that already
		// completed shouldn't fail because of a logging concern.
		adminID, ok := c.Locals("userID").(string)
		if !ok || adminID == "" {
			return err
		}

		log := domain.AdminLog{
			AdminID:    adminID,
			Action:     c.Method() + " " + c.Path(),
			TargetType: c.Params("target_type"),
			TargetID:   c.Params("id"),
			Details:    domain.JSONMap{},
			IP:         c.IP(),
			UserAgent:  c.Get("User-Agent"),
			CreatedAt:  time.Now(),
		}

		go func() {
			_ = db.Create(&log).Error
		}()

		return err
	}
}