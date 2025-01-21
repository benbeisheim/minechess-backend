package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func EnsurePlayerID() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Check header first
		playerID := c.Get("X-Player-ID")
		if playerID == "" {
			// Generate new ID if none exists
			playerID = uuid.New().String()
		}

		// Store in context for this request
		c.Locals("playerID", playerID)
		return c.Next()
	}
}
