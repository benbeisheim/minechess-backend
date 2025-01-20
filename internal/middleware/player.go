package middleware

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// EnsurePlayerID middleware checks for a player ID cookie and creates one if it doesn't exist
func EnsurePlayerID() fiber.Handler {
	return func(c *fiber.Ctx) error {
		playerID := c.Cookies("player_id")
		oldID := playerID

		if playerID == "" {
			playerID = uuid.New().String()
			fmt.Printf("No existing player ID found, creating new one: %s\n", playerID)
		} else {
			fmt.Printf("Found existing player ID: %s\n", playerID)
			// If you want to force a new ID even when one exists:
			// playerID = uuid.New().String()
			// fmt.Printf("Replacing with new player ID: %s\n", playerID)
		}

		// Always set the cookie with consistent attributes
		cookie := &fiber.Cookie{
			Name:     "player_id",
			Value:    playerID,
			Expires:  time.Now().Add(24 * time.Hour),
			HTTPOnly: true,
			SameSite: "Lax",
			Path:     "/", // Important: Use consistent path
			Domain:   "",  // Let browser set this automatically
		}

		// If we're replacing an old ID, explicitly expire the old cookie first
		if oldID != "" && oldID != playerID {
			expiredCookie := &fiber.Cookie{
				Name:     "player_id",
				Value:    "",
				Expires:  time.Now().Add(-24 * time.Hour), // Past expiration
				HTTPOnly: true,
				SameSite: "Lax",
				Path:     "/",
			}
			c.Cookie(expiredCookie)
		}

		// Set the new cookie
		c.Cookie(cookie)

		// Store in context for this request
		c.Locals("playerID", playerID)
		return c.Next()
	}
}
