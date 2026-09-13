// internal/middleware/admin_auth.go
package middleware

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"net"
	"os"
	"strings"

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
	if _, err := rand.Read(bytes); err != nil {
		// crypto/rand failing is effectively fatal for anything security
		// sensitive — better to panic loudly at token-generation time than
		// to hand out a predictable/zeroed token silently.
		panic("middleware: failed to generate CSRF token: " + err.Error())
	}
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

		// Constant-time compare so response timing can't be used to guess
		// the token byte by byte. subtle.ConstantTimeCompare requires equal
		// length inputs, so check length first (this leak is harmless: an
		// attacker already knows the token length from GenerateCSRFToken).
		if len(token) != len(cookieToken) ||
			subtle.ConstantTimeCompare([]byte(token), []byte(cookieToken)) != 1 {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "CSRF token mismatch",
			})
		}

		return c.Next()
	}
}

// IPWhitelist restricts access to a set of allowed client IPs/CIDR ranges,
// read from the ADMIN_IP_WHITELIST env var as a comma-separated list, e.g.:
//
//	ADMIN_IP_WHITELIST=203.0.113.10,10.0.0.0/8,2001:db8::/32
//
// Behavior:
//   - If ADMIN_IP_WHITELIST is unset or empty, the whitelist is considered
//     disabled and every request passes (opt-in feature).
//   - If it IS set, every entry must parse as either a plain IP or a CIDR;
//     a malformed entry is treated as a configuration error and the
//     middleware fails CLOSED (denies all requests) rather than silently
//     allowing everyone through. This surfaces misconfiguration immediately
//     instead of leaving admin routes unintentionally open.
//   - The whitelist is parsed on every request from the env var directly
//     (cheap: a handful of entries), so changing it via env/redeploy takes
//     effect without a code change.
func IPWhitelist() fiber.Handler {
	return func(c *fiber.Ctx) error {
		whitelist := os.Getenv("ADMIN_IP_WHITELIST")
		if strings.TrimSpace(whitelist) == "" {
			return c.Next() // feature disabled
		}

		allowedNets, allowedIPs, err := parseIPWhitelist(whitelist)
		if err != nil {
			// Fail closed: a broken whitelist config must not silently
			// become "allow everyone".
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "server misconfiguration: invalid ADMIN_IP_WHITELIST",
			})
		}

		clientIP := net.ParseIP(c.IP())
		if clientIP == nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Access denied",
			})
		}

		for _, ip := range allowedIPs {
			if ip.Equal(clientIP) {
				return c.Next()
			}
		}
		for _, ipNet := range allowedNets {
			if ipNet.Contains(clientIP) {
				return c.Next()
			}
		}

		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Access denied",
		})
	}
}

// parseIPWhitelist splits a comma-separated whitelist string into parsed
// CIDR networks and plain IPs. Returns an error if any entry is neither.
func parseIPWhitelist(whitelist string) ([]*net.IPNet, []net.IP, error) {
	var nets []*net.IPNet
	var ips []net.IP

	for _, raw := range strings.Split(whitelist, ",") {
		entry := strings.TrimSpace(raw)
		if entry == "" {
			continue
		}

		if strings.Contains(entry, "/") {
			_, ipNet, err := net.ParseCIDR(entry)
			if err != nil {
				return nil, nil, err
			}
			nets = append(nets, ipNet)
			continue
		}

		ip := net.ParseIP(entry)
		if ip == nil {
			return nil, nil, &net.ParseError{Type: "IP address", Text: entry}
		}
		ips = append(ips, ip)
	}

	return nets, ips, nil
}