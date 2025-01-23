package controller

import (
	"bufio"
	"fmt"

	"github.com/benbeisheim/minechess-backend/internal/service"
	"github.com/gofiber/fiber/v2"
)

type GameController struct {
	gameService *service.GameService
}

func NewGameController(gameService *service.GameService) *GameController {
	return &GameController{gameService: gameService}
}

func (gc *GameController) CreateGame(c *fiber.Ctx) error {

	gameID, err := gc.gameService.CreateGame()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	return c.JSON(fiber.Map{
		"message": "Game created",
		"game_id": gameID,
	})
}

func (gc *GameController) JoinGame(c *fiber.Ctx) error {
	gameID := c.Params("gameId")
	fmt.Println("Game ID:", gameID)
	playerID := c.Locals("playerID").(string)
	fmt.Println("Player ID:", playerID)

	color, err := gc.gameService.JoinGame(gameID, playerID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Game joined",
		"color":   color,
	})
}

func (gc *GameController) GetGameState(c *fiber.Ctx) error {
	gameID := c.Params("gameId")

	gameState, err := gc.gameService.GetGameState(gameID)
	if err != nil {
		if err.Error() == "game not found" {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch game state",
		})
	}

	return c.JSON(gameState)
}

func (gc *GameController) JoinMatchmaking(c *fiber.Ctx) error {
	playerID := c.Locals("playerID").(string)

	if err := gc.gameService.JoinMatchmaking(playerID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to join matchmaking",
		})
	}

	return c.JSON(fiber.Map{
		"status": "queued",
	})
}

func (gc *GameController) HandleMatchmakingEvents(c *fiber.Ctx) error {
	// Set required headers for SSE
	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("Transfer-Encoding", "chunked")

	// Get player ID from context
	playerID := c.Locals("playerID").(string)

	// Create channel for this client
	matchChan := make(chan string)
	// Create a done channel to signal connection closure
	done := make(chan struct{})

	// Register the channel with the game service
	if err := gc.gameService.RegisterMatchmakingChannel(playerID, matchChan); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to register for matchmaking events",
		})
	}

	// Use Fiber's streaming capability
	c.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
		defer func() {
			gc.gameService.UnregisterMatchmakingChannel(playerID)
			close(done)
		}()

		// Instead of trying to use context values, we'll use our done channel
		for {
			select {
			case gameID, ok := <-matchChan:
				if !ok {
					// Channel was closed by the service
					return
				}

				// Send the game ID as an SSE event
				_, err := fmt.Fprintf(w, "data: %s\n\n", gameID)
				if err != nil {
					return
				}

				// Ensure the data is sent immediately
				err = w.Flush()
				if err != nil {
					return
				}

			case <-done:
				// Connection was closed
				return
			}
		}
	})

	return nil
}
