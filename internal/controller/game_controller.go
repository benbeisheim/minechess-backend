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
	fmt.Println("Adding player to matchmaking queue:", playerID)

	if err := gc.gameService.JoinMatchmaking(playerID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to join matchmaking",
		})
	}
	fmt.Println("Player added to matchmaking queue:", playerID)

	return c.JSON(fiber.Map{
		"status": "queued",
	})
}
func (gc *GameController) HandleMatchmakingEvents(c *fiber.Ctx) error {
	// ... existing header setup code ...
	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("Transfer-Encoding", "chunked")

	playerID := c.Query("playerId")
	matchChan := make(chan string)

	if err := gc.gameService.RegisterMatchmakingChannel(playerID, matchChan); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to register for matchmaking events",
		})
	}

	// Set up cleanup for when the client disconnects
	c.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
		defer func() {
			// The channel will already be closed if a match was found
			// Otherwise, we need to clean up here
			gc.gameService.UnregisterMatchmakingChannel(playerID)
		}()

		for {
			select {
			case msg, ok := <-matchChan:
				if !ok {
					// Channel was closed, exit the stream
					return
				}
				fmt.Fprintf(w, "data: %s\n\n", msg)
				err := w.Flush()
				if err != nil {
					return
				}
			}
		}
	})

	return nil
}
