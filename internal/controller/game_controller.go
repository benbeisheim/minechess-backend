package controller

import (
	"fmt"

	"github.com/benbeisheim/MineChess/backend/internal/service"
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
