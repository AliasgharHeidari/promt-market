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

		log := domain.AdminLog{
			AdminID:    c.Locals("userID").(string),
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