package middleware

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
)

// WebSocketUpgrade ensures that requests to WebSocket endpoints are valid WebSocket connection attempts.
// It also checks that necessary game and player information is present before allowing the upgrade.
func WebSocketUpgrade() fiber.Handler {
	fmt.Println("WebSocketUpgrade middleware")
	return func(c *fiber.Ctx) error {
		// First, check if this is a WebSocket upgrade request
		if !websocket.IsWebSocketUpgrade(c) {
			return fiber.ErrUpgradeRequired
		}

		// Ensure we have a game ID
		gameID := c.Params("gameId")
		if gameID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "game ID is required",
			})
		}

		// Ensure we have a player ID (this would have been set by our EnsurePlayerID middleware)
		playerID := c.Locals("playerID")
		if playerID == nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "player ID is required",
			})
		}

		// Store these IDs in locals so they're available after the WebSocket upgrade
		// This is important because the connection context is different from the upgrade context
		c.Locals("wsGameID", gameID)
		c.Locals("wsPlayerID", playerID)

		// Allow the upgrade to proceed
		return c.Next()
	}
}
