package server

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

// recoveryMiddleware creates a recovery middleware.
func recoveryMiddleware() fiber.Handler {
	return recover.New(recover.Config{
		EnableStackTrace: true,
	})
}

// loggerMiddleware creates a logger middleware.
func loggerMiddleware() fiber.Handler {
	return logger.New(logger.Config{
		Format:     "[${time}] ${status} - ${method} ${path} ${latency}\n",
		TimeFormat: "2006-01-02 15:04:05",
		Output:     nil, // Use stdout
	})
}

// corsMiddleware creates a CORS middleware.
func corsMiddleware(origins []string) fiber.Handler {
	return cors.New(cors.Config{
		AllowOrigins: strings.Join(origins, ","),
		AllowMethods: strings.Join([]string{
			"GET",
			"POST",
			"PUT",
			"DELETE",
			"OPTIONS",
		}, ","),
		AllowHeaders: strings.Join([]string{
			"Origin",
			"Content-Type",
			"Accept",
			"Authorization",
			"X-Requested-With",
		}, ","),
		AllowCredentials: true,
		MaxAge:           86400, // 24 hours
	})
}

// compressMiddleware creates a compression middleware.
func compressMiddleware() fiber.Handler {
	return compress.New(compress.Config{
		Level: compress.LevelBestSpeed,
	})
}

// errorHandler creates a custom error handler.
func errorHandler(c *fiber.Ctx, err error) error {
	// Default 500 status code
	code := fiber.StatusInternalServerError

	// Retrieve the custom status code if it's a fiber.Error
	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
	}

	// Set Content-Type to JSON
	c.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)

	// Return JSON error
	return c.Status(code).JSON(fiber.Map{
		"error":   true,
		"message": err.Error(),
		"code":    code,
	})
}

// requestIDMiddleware adds a unique request ID to each request.
func requestIDMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Generate request ID
		requestID := generateRequestID()
		c.Set("X-Request-ID", requestID)
		c.Locals("requestID", requestID)
		return c.Next()
	}
}

// timingMiddleware adds timing information to requests.
func timingMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		err := c.Next()
		elapsed := time.Since(start)

		c.Set("X-Response-Time", elapsed.String())
		return err
	}
}

// generateRequestID generates a unique request ID.
func generateRequestID() string {
	return fmt.Sprintf("req-%d-%s", time.Now().UnixNano(), randomString(8))
}

// randomString generates a cryptographically secure random string of given length.
func randomString(n int) string {
	bytes := make([]byte, n)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to time-based if crypto fails (shouldn't happen)
		const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
		b := make([]byte, n)
		for i := range b {
			b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
		}
		return string(b)
	}
	return hex.EncodeToString(bytes)[:n]
}

// authMiddleware creates an authentication middleware.
func authMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get API key from environment
		apiKey := os.Getenv("MAGICODE_API_KEY")
		
		// If no API key is configured, allow all requests (development mode)
		if apiKey == "" {
			return c.Next()
		}
		
		// Check Authorization header
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   true,
				"message": "Authorization header required",
				"code":    fiber.StatusUnauthorized,
			})
		}
		
		// Extract token from "Bearer <token>" format
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   true,
				"message": "Invalid authorization format. Use: Bearer <token>",
				"code":    fiber.StatusUnauthorized,
			})
		}
		
		token := parts[1]
		
		// Validate token
		if token != apiKey {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   true,
				"message": "Invalid API key",
				"code":    fiber.StatusUnauthorized,
			})
		}
		
		return c.Next()
	}
}

// instanceMiddleware creates an instance context middleware.
func instanceMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Set instance directory in context
		c.Locals("directory", c.Path())
		return c.Next()
	}
}

// cacheControlMiddleware sets cache control headers.
func cacheControlMiddleware(maxAge int) fiber.Handler {
	return func(c *fiber.Ctx) error {
		c.Set("Cache-Control", fmt.Sprintf("public, max-age=%d", maxAge))
		return c.Next()
	}
}

// noCacheMiddleware prevents caching.
func noCacheMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		c.Set("Cache-Control", "no-cache, no-store, must-revalidate")
		c.Set("Pragma", "no-cache")
		c.Set("Expires", "0")
		return c.Next()
	}
}