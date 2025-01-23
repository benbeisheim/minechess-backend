package controller

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/benbeisheim/minechess-backend/internal/model"
	"github.com/benbeisheim/minechess-backend/internal/service"
	"github.com/benbeisheim/minechess-backend/internal/ws"
	"github.com/gofiber/websocket/v2"
)

type WebSocketController struct {
	gameService *service.GameService
}

func NewWebSocketController(gameService *service.GameService) *WebSocketController {
	return &WebSocketController{
		gameService: gameService,
	}
}

// HandleConnection is called when a new WebSocket connection is established
func (wsc *WebSocketController) HandleConnection(c *websocket.Conn) {
	fmt.Println("Handling connection")
	// Extract game ID and player ID from context
	gameID := c.Params("gameId")
	playerID := c.Locals("playerID").(string)

	if playerID == "" {
		wsc.sendError(c, "playerId is required")
		c.Close()
		return
	}

	// Register this connection with the game
	if err := wsc.gameService.RegisterConnection(gameID, playerID, c); err != nil {
		log.Printf("Failed to register connection: %v", err)
		c.Close()
		return
	}

	// Start message handling loop
	for {
		messageType, message, err := c.ReadMessage()
		if err != nil {
			log.Printf("read error: %v", err)
			break
		}

		if messageType == websocket.TextMessage {
			var msg ws.Message
			if err := json.Unmarshal(message, &msg); err != nil {
				log.Printf("parse error: %v", err)
				continue
			}

			if err := wsc.handleMessage(gameID, playerID, msg); err != nil {
				log.Printf("handle error: %v", err)
				wsc.sendError(c, err.Error())
			}
		}
	}

	wsc.gameService.UnregisterConnection(gameID, playerID)
}

func (wsc *WebSocketController) handleMessage(gameID, playerID string, msg ws.Message) error {
	fmt.Println("Handling message:", msg.Type)
	switch msg.Type {
	case ws.MessageTypeMove:
		var move model.WSMove
		if err := json.Unmarshal(msg.Payload, &move); err != nil {
			return err
		}
		return wsc.gameService.HandleMove(gameID, playerID, move)

	default:
		return fmt.Errorf("unknown message type: %s", msg.Type)
	}
}

func (wsc *WebSocketController) sendError(c *websocket.Conn, errorMsg string) {
	c.WriteJSON(ws.Message{
		Type:    ws.MessageTypeError,
		Payload: json.RawMessage(errorMsg),
	})
}
