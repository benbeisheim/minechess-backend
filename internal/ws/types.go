package ws

import (
	"encoding/json"
)

// MessageType represents the different kinds of messages our system can handle
type MessageType string

const (
	MessageTypeMove      MessageType = "move"
	MessageTypeGameState MessageType = "gameState"
	MessageTypeDrawOffer MessageType = "drawOffer"
	MessageTypeResign    MessageType = "resign"
	MessageTypeDraw      MessageType = "draw"
	MessageTypeError     MessageType = "error"
)

// Message represents a WebSocket message in our system
type Message struct {
	Type    MessageType     `json:"type"`
	Payload json.RawMessage `json:"payload"`
}
