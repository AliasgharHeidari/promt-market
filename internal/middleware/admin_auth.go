// internal/middleware/admin_auth.go
package middleware

import (
	"crypto/rand"
	"encoding/base64"
	"os"

	"github.com/gofiber/fiber/v2"
)

// AdminOnly ensures the user has admin role
func AdminOnly() fiber.Handler {
	return func(c *fiber.Ctx) error {
		role, ok := c.Locals("role").(string)
		if !ok || role != "admin" {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Admin access required",
			})
		}
		return c.Next()
	}
}

// GenerateCSRFToken generates a new CSRF token
func GenerateCSRFToken() string {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return base64.StdEncoding.EncodeToString(bytes)
}

// CSRFProtection validates CSRF token
func CSRFProtection() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Skip CSRF for GET, HEAD, OPTIONS
		if c.Method() == "GET" || c.Method() == "HEAD" || c.Method() == "OPTIONS" {
			return c.Next()
		}

		// Get token from header
		token := c.Get("X-CSRF-Token")
		if token == "" {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "CSRF token required",
			})
		}

		// Get token from cookie
		cookieToken := c.Cookies("csrf_token")
		if cookieToken == "" {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "CSRF cookie missing",
			})
		}

		// Compare tokens
		if token != cookieToken {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "CSRF token mismatch",
			})
		}

		return c.Next()
	}
}

// IPWhitelist checks if IP is allowed
func IPWhitelist() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get whitelist from env
		whitelist := os.Getenv("ADMIN_IP_WHITELIST")
		if whitelist == "" {
			return c.Next() // Skip if no whitelist
		}

		// TODO: Implement IP whitelist check
		// clientIP := c.IP()
		// if !isIPInWhitelist(clientIP, whitelist) {
		//     return c.Status(403).JSON(fiber.Map{"error": "Access denied"})
		// }

		return c.Next()
	}
}