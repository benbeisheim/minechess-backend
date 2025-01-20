package model

import (
	"github.com/gofiber/websocket/v2"
)

type Player struct {
	ID       string
	Color    string
	Conn     *websocket.Conn
	TimeLeft int
}

type ClientPlayer struct {
	ID       string `json:"name"`
	Color    string `json:"color"`
	TimeLeft int    `json:"timeLeft"`
}

type PlayerColor string

const (
	PlayerColorWhite PlayerColor = "white"
	PlayerColorBlack PlayerColor = "black"
)
