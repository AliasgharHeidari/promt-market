package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"

	"promt-market/pkg/jwt"
)

// JWTProtected validates the JWT token and adds user info to context.
// Rejects the request when no valid token is present.
func JWTProtected() fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims, err := extractClaims(c)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		setLocals(c, claims)
		return c.Next()
	}
}

// OptionalJWT is like JWTProtected but does not reject the request when no
// token is present. If a valid token is supplied, the viewer's identity is
// available in Locals; otherwise, the request proceeds as anonymous.
//
// Used on endpoints that behave differently for logged-in vs anonymous
// users but must remain publicly reachable (e.g. /prompts/:slug, where
// content visibility depends on purchase state).
func OptionalJWT() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// If there's no Authorization header at all, just skip — this is
		// an anonymous request.
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Next()
		}

		// If a header is present, we do enforce validity: a malformed or
		// expired token is treated as "not authenticated" and the request
		// proceeds anonymously. This avoids surprising 401s on the public
		// storefront while still not leaking any protected data.
		claims, err := extractClaims(c)
		if err == nil {
			setLocals(c, claims)
		}
		return c.Next()
	}
}

// extractClaims parses the Authorization header and returns the JWT claims.
// Returns an error message suitable for a 401 response.
func extractClaims(c *fiber.Ctx) (*jwt.Claims, error) {
	authHeader := c.Get("Authorization")
	if authHeader == "" {
		return nil, fiber.ErrUnauthorized
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return nil, fiber.ErrUnauthorized
	}

	claims, err := jwt.ValidateAccessToken(parts[1])
	if err != nil {
		return nil, fiber.ErrUnauthorized
	}
	return claims, nil
}

// setLocals copies the JWT claims into Fiber's per-request locals so
// downstream handlers can read userID, email, and role.
func setLocals(c *fiber.Ctx, claims *jwt.Claims) {
	c.Locals("userID", claims.UserID)
	c.Locals("email", claims.Email)
	c.Locals("role", claims.Role)
}