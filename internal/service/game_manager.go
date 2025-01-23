// service/game_manager.go
package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/benbeisheim/minechess-backend/internal/model"
	"github.com/gofiber/websocket/v2"
	"github.com/google/uuid"
)

type GameManager struct {
	games            map[string]*model.Game
	queue            *model.Queue
	matchingChannels map[string]chan string
	mu               sync.RWMutex
}

func (gm *GameManager) RegisterMatchmakingChannel(playerID string, ch chan string) error {
	gm.mu.Lock()
	defer gm.mu.Unlock()
	fmt.Println("Registering matchmaking channel for player", playerID)

	// If there's an existing channel, close it first
	if existingCh, exists := gm.matchingChannels[playerID]; exists {
		close(existingCh)
	}

	gm.matchingChannels[playerID] = ch
	return nil
}

func (gm *GameManager) UnregisterMatchmakingChannel(playerID string) {
	gm.mu.Lock()
	defer gm.mu.Unlock()
	fmt.Println("Unregistering matchmaking channel for player", playerID)

	if _, exists := gm.matchingChannels[playerID]; exists {
		// We don't close the channel here because it might be used by other goroutines
		// The creator of the channel (HandleMatchmakingEvents) is responsible for closing it
		delete(gm.matchingChannels, playerID)
	}
}

func (gm *GameManager) processMatchmaking() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		gm.mu.Lock()
		if gm.queue.Size() >= 2 {
			player1, player2 := gm.queue.GetNextPair()

			// Create new game
			gameID := uuid.New().String()
			game := model.NewGame(gameID)

			// Add players to game
			game.AddPlayer(player1.ID) // Assuming this returns the assigned color
			game.AddPlayer(player2.ID)
			gm.games[gameID] = game

			// Create match events for each player
			player1Event := model.MatchFoundEvent{
				GameID: gameID,
				Color:  "white", // Use actual color assigned by AddPlayer
			}
			player2Event := model.MatchFoundEvent{
				GameID: gameID,
				Color:  "black", // Use actual color assigned by AddPlayer
			}

			// Send events to players
			fmt.Println("Sending match found event to player", player1.ID)
			if ch1, ok := gm.matchingChannels[player1.ID]; ok {
				select {
				case ch1 <- mustJSON(player1Event): // Helper function to handle JSON marshaling
					fmt.Println("Sent match found event to player", player1.ID)
				default:
					// Channel is blocked or closed, skip this notification
					fmt.Println("Channel is blocked or closed, skipping match found event to player", player1.ID)
				}
			}
			fmt.Println("Sending match found event to player", player2.ID)
			if ch2, ok := gm.matchingChannels[player2.ID]; ok {
				select {
				case ch2 <- mustJSON(player2Event):
					fmt.Println("Sent match found event to player", player2.ID)
				default:
					// Channel is blocked or closed, skip this notification
					fmt.Println("Channel is blocked or closed, skipping match found event to player", player2.ID)
				}
			}
		}
		gm.mu.Unlock()
	}
}

// Helper function for JSON marshaling
func mustJSON(v interface{}) string {
	bytes, err := json.Marshal(v)
	if err != nil {
		// In production, you'd want to handle this error more gracefully
		panic(err)
	}
	fmt.Println("Marshalled JSON:", string(bytes))
	return string(bytes)
}

func NewGameManager() *GameManager {
	gm := &GameManager{
		games:            make(map[string]*model.Game),
		queue:            model.NewQueue(),
		matchingChannels: make(map[string]chan string),
	}

	// Start matchmaking processor
	go gm.processMatchmaking()

	return gm
}

func (gm *GameManager) CreateGame(gameID string) error {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	if _, exists := gm.games[gameID]; exists {
		return errors.New("game already exists")
	}

	gm.games[gameID] = model.NewGame(gameID)
	return nil
}

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
