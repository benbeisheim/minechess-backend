package main

import (
	"fmt"
	"log"

	"github.com/benbeisheim/MineChess/backend/internal/controller"
	"github.com/benbeisheim/MineChess/backend/internal/middleware"
	"github.com/benbeisheim/MineChess/backend/internal/service"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/websocket/v2"
)

func main() {
	// Initialize the application

	app := fiber.New()

	// Then add the CORS middleware
	app.Use(cors.New(cors.Config{
		AllowOrigins:     "http://localhost:5173", // Your React app's exact origin
		AllowHeaders:     "Origin, Content-Type, Accept",
		AllowMethods:     "GET, POST, OPTIONS",
		AllowCredentials: true,
	}))
	// Debugging middleware
	app.Use(func(c *fiber.Ctx) error {
		fmt.Println("--------------------------------")
		fmt.Printf("Incoming request to path: %s\n", c.Path())
		fmt.Printf("Method: %s\n", c.Method())
		fmt.Printf("Headers: %v\n", c.GetReqHeaders())
		fmt.Println("--------------------------------")
		return c.Next()
	})

	// Initialize services
	gameManager := service.NewGameManager()
	gameService := service.NewGameService(gameManager)

	// Initialize controllers
	gameController := controller.NewGameController(gameService)
	wsController := controller.NewWebSocketController(gameService)

	// Set up WebSocket routes
	app.Use("/ws/*", middleware.EnsurePlayerID())
	app.Get("/ws/game/:gameId", websocket.New(func(c *websocket.Conn) {
		fmt.Printf("WebSocket connection established for game: %s\n", c.Params("gameId"))
		wsController.HandleConnection(c)
	}, websocket.Config{
		// Add some WebSocket-specific configuration
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		// Enable all origins during development
		Origins: []string{"http://localhost:5173"},
	}))

	// Set up REST routes
	api := app.Group("/api", middleware.EnsurePlayerID())

	// Game routes
	gameRoutes := api.Group("/game")
	gameRoutes.Post("/matchmaking/join", gameController.JoinMatchmaking)
	gameRoutes.Post("/create", gameController.CreateGame)
	gameRoutes.Post("/join/:gameId", gameController.JoinGame)
	gameRoutes.Get("/:gameId", gameController.GetGameState)

	log.Fatal(app.Listen(":3000"))
}
