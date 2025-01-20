// service/game_manager.go
package service

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/benbeisheim/minechess-backend/internal/model"
	"github.com/gofiber/websocket/v2"
	"github.com/google/uuid"
)

type GameManager struct {
	// Map to store all active games, keyed by game ID
	games map[string]*model.Game
	queue *model.Queue

	// RWMutex for safe concurrent access
	mu sync.RWMutex
}

func NewGameManager() *GameManager {
	gm := &GameManager{
		games: make(map[string]*model.Game),
		queue: model.NewQueue(),
	}

	// Start matchmaking processor
	go gm.processMatchmaking()

	return gm
}

func (gm *GameManager) processMatchmaking() {
	ticker := time.NewTicker(1 * time.Second)
	for range ticker.C {
		gm.mu.Lock()
		if gm.queue.Size() >= 2 {
			player1, player2 := gm.queue.GetNextPair()

			// Create a game for these players
			gameID := uuid.New().String()
			game := model.NewGame(gameID)

			// Add players to game
			game.AddPlayer(player1.ID)
			game.AddPlayer(player2.ID)

			gm.games[gameID] = game

			// Notify players they've been matched (we'll implement this with WebSockets)
		}
		gm.mu.Unlock()
	}
}

// CreateGame instantiates a new game instance
func (gm *GameManager) CreateGame(gameID string) error {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	if _, exists := gm.games[gameID]; exists {
		return errors.New("game already exists")
	}

	gm.games[gameID] = model.NewGame(gameID)
	return nil
}

// GetGame retrieves a game by ID
func (gm *GameManager) GetGame(gameID string) (*model.Game, error) {
	gm.mu.RLock()
	defer gm.mu.RUnlock()

	game, exists := gm.games[gameID]
	if !exists {
		return nil, errors.New("game not found")
	}

	return game, nil
}

func (gm *GameManager) AddPlayerToGame(gameID string, playerID string) (string, error) {
	fmt.Println("Adding player to game", gameID, playerID)
	gm.mu.Lock()
	defer gm.mu.Unlock()

	game, exists := gm.games[gameID]
	if !exists {
		return "", errors.New("game not found")
	}

	return game.AddPlayer(playerID)
}

func (gm *GameManager) JoinMatchmaking(playerID string) error {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	err := gm.queue.AddPlayer(model.Player{ID: playerID})
	if err != nil {
		return err
	}

	return nil
}

func (gm *GameManager) GetGameState(gameID string) (model.GameState, error) {
	gm.mu.RLock()
	defer gm.mu.RUnlock()
	game, exists := gm.games[gameID]
	if !exists {
		return model.GameState{}, errors.New("game not found")
	}

	return game.GetState(), nil
}

func (gm *GameManager) MakeMove(gameID string, playerID string, move model.WSMove) error {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	game, exists := gm.games[gameID]
	if !exists {
		return errors.New("game not found")
	}

	return game.MakeMove(move)
}

func (gm *GameManager) RegisterConnection(gameID string, playerID string, conn *websocket.Conn) error {
	fmt.Println("Registering connection in game manager")
	gm.mu.Lock()
	defer gm.mu.Unlock()

	game, exists := gm.games[gameID]
	if !exists {
		return errors.New("game not found")
	}

	return game.RegisterConnection(playerID, conn)
}

func (gm *GameManager) UnregisterConnection(gameID string, playerID string) {
	fmt.Println("Unregistering connection in game manager")
	gm.mu.Lock()
	defer gm.mu.Unlock()
	game, exists := gm.games[gameID]
	if !exists {
		return
	}

	game.UnregisterConnection(playerID)
}
