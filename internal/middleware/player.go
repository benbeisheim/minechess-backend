package middleware

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
)

func EnsurePlayerID() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Check if playerID is already set
		if c.Locals("playerID") != nil {
			fmt.Println("Player ID already set:", c.Locals("playerID"))
			return c.Next()
		}

		var playerID string
		// Check header first
		playerID = c.Get("X-Player-ID")
		fmt.Println("Player ID from header:", playerID)
		if playerID == "" {
			playerID = c.Query("playerId")
			fmt.Println("Player ID from query:", playerID)
		}

		if playerID == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Player ID is required. Please ensure client is properly initialized.",
			})
		}

		// Store in context for this request
		c.Locals("playerID", playerID)
		return c.Next()
	}
}
