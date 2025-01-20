package service

import (
	"fmt"

	"github.com/benbeisheim/minechess-backend/internal/model"
	"github.com/gofiber/websocket/v2"
	"github.com/google/uuid"
)

type GameService struct {
	gameManager *GameManager
}

func NewGameService(gameManager *GameManager) *GameService {
	return &GameService{
		gameManager: gameManager,
	}
}

func (gs *GameService) JoinGame(gameID string, playerID string) (string, error) {
	return gs.gameManager.AddPlayerToGame(gameID, playerID)
}

func (gs *GameService) CreateGame() (string, error) {
	gameID := uuid.New().String()

	if err := gs.gameManager.CreateGame(gameID); err != nil {
		return "", fmt.Errorf("failed to create game: %w", err)
	}

	return gameID, nil
}

func (gs *GameService) JoinMatchmaking(playerID string) error {
	return gs.gameManager.JoinMatchmaking(playerID)
}

func (gs *GameService) GetGameState(gameID string) (model.GameState, error) {
	return gs.gameManager.GetGameState(gameID)
}

// HandleMove processes a move from a player and broadcasts the update
func (gs *GameService) HandleMove(gameID string, playerID string, move model.WSMove) error {
	// Validate and execute the move using game manager
	if err := gs.gameManager.MakeMove(gameID, playerID, move); err != nil {
		return err
	}

	return nil
}

func (gs *GameService) RegisterConnection(gameID string, playerID string, conn *websocket.Conn) error {
	fmt.Println("Registering connection in game service")
	return gs.gameManager.RegisterConnection(gameID, playerID, conn)
}

func (gs *GameService) UnregisterConnection(gameID string, playerID string) {
	fmt.Println("Unregistering connection in game service")
	gs.gameManager.UnregisterConnection(gameID, playerID)
}
