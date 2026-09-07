// internal/middleware/rate_limit.go (نسخه جایگزین)
package middleware

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	fiberredis "github.com/gofiber/storage/redis/v3"
	"github.com/redis/go-redis/v9"
)

// newRedisStorage creates a fiber.Storage from redis client
func newRedisStorage(redisClient *redis.Client) fiber.Storage {
	return fiberredis.New(fiberredis.Config{
		Host:     "localhost",  
		Port:     6379,         
		Password: "",
		Database: 0,
	})
}

// RateLimit returns a rate limiter middleware
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

// AdminRateLimit stricter rate limit for admin routes
func AdminRateLimit(redisClient *redis.Client) fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        50,
		Expiration: 1 * time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string {
			userID := c.Locals("userID")
			if userID != nil {
				return userID.(string) + ":" + c.IP()
			}
			return c.IP()
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": "Admin rate limit exceeded, please try again later",
			})
		},
		Storage: newRedisStorage(redisClient),
	})
}