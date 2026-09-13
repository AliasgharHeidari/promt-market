// internal/middleware/prompt_author_only.go
package middleware

import (
	"github.com/gofiber/fiber/v2"
)

// PromptAuthorOnly ensures the caller's role is "prompt_author" or "admin"
// before allowing prompt-creation endpoints. Must run after JWTProtected(),
// which is what populates c.Locals("role").
func PromptAuthorOnly() fiber.Handler {
	return func(c *fiber.Ctx) error {
		role, ok := c.Locals("role").(string)
		if !ok || (role != "prompt_author" && role != "admin") {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "You must be a verified prompt author to do this. Apply at /api/v1/author-application.",
			})
		}
		return c.Next()
	}
}