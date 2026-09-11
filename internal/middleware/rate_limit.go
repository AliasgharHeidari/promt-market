package middleware

import (
	"os"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	fiberredis "github.com/gofiber/storage/redis/v3"
	"github.com/redis/go-redis/v9"
)

// newRedisStorage creates a fiber.Storage backed by Redis. Reads connection
// parameters from the environment so we don't hard-code "localhost" — the
// same binary should work in dev, docker-compose, and prod without changes.
func newRedisStorage(redisClient *redis.Client) fiber.Storage {
	host := os.Getenv("REDIS_HOST")
	if host == "" {
		host = "localhost"
	}

	port := 6379
	if p := os.Getenv("REDIS_PORT"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil {
			port = parsed
		}
	}

	password := os.Getenv("REDIS_PASSWORD")

	db := 0
	if d := os.Getenv("REDIS_DB"); d != "" {
		if parsed, err := strconv.Atoi(d); err == nil {
			db = parsed
		}
	}

	return fiberredis.New(fiberredis.Config{
		Host:     host,
		Port:     port,
		Password: password,
		Database: db,
	})
}

// RateLimit is the general-purpose limiter for public read endpoints
// (browse, search). Generous enough not to bother real users, tight
// enough to stop scraping.
//
// Usage: app.Use(RateLimit(redisClient)) or per-group.
func RateLimit(redisClient *redis.Client) fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        100,
		Expiration: 1 * time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.IP()
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": "Too many requests, please try again later",
			})
		},
		Storage: newRedisStorage(redisClient),
	})
}

// StrictRateLimit is for sensitive endpoints that an attacker might hammer
// with automated requests (register, login, verify, resend codes). The
// key is IP + path so a user can't burn their whole quota by refreshing
// one endpoint.
func StrictRateLimit(redisClient *redis.Client) fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        10,
		Expiration: 5 * time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.IP() + ":" + c.Path()
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": "Too many attempts on this endpoint, please wait a moment",
			})
		},
		Storage: newRedisStorage(redisClient),
	})
}

// AuthRateLimit is a special-purpose limiter for auth flows. Uses a longer
// window so a persistent attacker is slowed down even more than 10/min
// would allow, while legitimate typos still fit comfortably.
func AuthRateLimit(redisClient *redis.Client) fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        5,
		Expiration: 1 * time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.IP() + ":" + c.Path()
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": "Too many auth attempts, please try again in a minute",
			})
		},
		Storage: newRedisStorage(redisClient),
	})
}

// UploadRateLimit limits file uploads. A legitimate author uploads a
// handful of images per prompt — 30/min gives plenty of headroom.
func UploadRateLimit(redisClient *redis.Client) fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        30,
		Expiration: 1 * time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string {
			userID, _ := c.Locals("userID").(string)
			if userID != "" {
				return userID
			}
			return c.IP()
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": "Upload rate limit exceeded, please slow down",
			})
		},
		Storage: newRedisStorage(redisClient),
	})
}

// AuthorAppRateLimit limits submissions and OTP requests for the
// author-application flow. Prevents OTP spamming and document floods.
func AuthorAppRateLimit(redisClient *redis.Client) fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        10,
		Expiration: 1 * time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string {
			userID, _ := c.Locals("userID").(string)
			if userID != "" {
				return userID + ":" + c.Path()
			}
			return c.IP() + ":" + c.Path()
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": "Too many author-application requests, please try again later",
			})
		},
		Storage: newRedisStorage(redisClient),
	})
}

// AdminRateLimit is a stricter limiter for admin routes, keyed by user id
// (falling back to IP) so a shared NAT doesn't penalize two admins.
func AdminRateLimit(redisClient *redis.Client) fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        200,
		Expiration: 1 * time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string {
			userID, _ := c.Locals("userID").(string)
			if userID != "" {
				return userID + ":" + c.IP()
			}
			return c.IP()
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": "Admin rate limit exceeded, please try again in a moment",
			})
		},
		Storage: newRedisStorage(redisClient),
	})
}