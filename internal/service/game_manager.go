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

	// If there's an existing channel, we need to handle it properly
	if existingCh, exists := gm.matchingChannels[playerID]; exists {
		fmt.Println("Found existing channel for player", playerID)
		// Remove from map first to prevent any new writes
		delete(gm.matchingChannels, playerID)
		// Then close the channel
		close(existingCh)
	}

	// Register the new channel
	gm.matchingChannels[playerID] = ch
	return nil
}

// Now let's modify processMatchmaking to handle channel cleanup after sending events
func (gm *GameManager) processMatchmaking() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		gm.mu.Lock()
		if gm.queue.Size() >= 2 {
			player1, player2 := gm.queue.GetNextPair()

			// Create and set up the game as before...
			// Create new game
			gameID := uuid.New().String()
			game := model.NewGame(gameID)

			// Add players to game
			p1Color, err := game.AddPlayer(player1.ID) // Assuming this returns the assigned color
			if err != nil {
				fmt.Println("Error adding player to game", err)
				continue
			}
			p2Color, err := game.AddPlayer(player2.ID)
			if err != nil {
				fmt.Println("Error adding player to game", err)
				continue
			}
			gm.games[gameID] = game

			// Create match events for each player
			player1Event := model.MatchFoundEvent{
				GameID: gameID,
				Color:  p1Color, // Use actual color assigned by AddPlayer
			}
			player2Event := model.MatchFoundEvent{
				GameID: gameID,
				Color:  p2Color, // Use actual color assigned by AddPlayer
			}

			// Send events and clean up channels
			successfullySentBoth := true

			// Helper function to send event and clean up channel
			sendEventAndCleanup := func(playerID string, event model.MatchFoundEvent) bool {
				if ch, ok := gm.matchingChannels[playerID]; ok {
					select {
					case ch <- mustJSON(event):
						fmt.Printf("Sent match found event to player %s\n", playerID)
						// Remove the channel from the map
						delete(gm.matchingChannels, playerID)
						// Close the channel
						close(ch)
						return true
					default:
						fmt.Printf("Failed to send event to player %s\n", playerID)
						return false
					}
				}
				return false
			}

			// Send to both players
			if !sendEventAndCleanup(player1.ID, player1Event) {
				successfullySentBoth = false
			}
			if !sendEventAndCleanup(player2.ID, player2Event) {
				successfullySentBoth = false
			}

			// If we failed to notify both players, we might want to handle that
			if !successfullySentBoth {
				// Maybe add them back to queue or implement retry logic
				fmt.Println("Failed to notify all players of match")
			}
		}
		gm.mu.Unlock()
	}
}

func (gm *GameManager) UnregisterMatchmakingChannel(playerID string) {
	gm.mu.Lock()
	defer gm.mu.Unlock()
	fmt.Println("Unregistering matchmaking channel for player", playerID, "with channel", gm.matchingChannels[playerID])

	// We don't close the channel here because it might be used by other goroutines
	// The creator of the channel (HandleMatchmakingEvents) is responsible for closing it
	delete(gm.matchingChannels, playerID)
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

func (gm *GameManager) AddPlayerToGame(gameID string, playerID string) (model.PlayerColor, error) {
	fmt.Println("Adding player to game", gameID, playerID)
	gm.mu.Lock()
	defer gm.mu.Unlock()

	game, exists := gm.games[gameID]
	if !exists {
		return model.PlayerColor(""), errors.New("game not found")
	}

	return game.AddPlayer(playerID)
}

func (gm *GameManager) JoinMatchmaking(playerID string) error {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	err := gm.queue.AddPlayer(model.Player{ID: playerID})
	if err != nil {
		fmt.Println("Error adding player to matchmaking queue:", err)
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
