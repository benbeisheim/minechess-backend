package model

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/benbeisheim/minechess-backend/internal/ws"
	"github.com/gofiber/websocket/v2"
)

// The connections for a specific game
type GameConnections struct {
	connections map[string]*websocket.Conn // playerID -> connection
	mu          sync.RWMutex
}

// The Game struct focuses on a single game's state and its observers
type Game struct {
	ID          string
	mu          sync.Mutex
	state       GameState
	connections *GameConnections // Connections just for this game
	mine        *Position
	whiteClock  *Clock
	blackClock  *Clock
}

type GameState struct {
	Sound           string         `json:"sound"`
	Board           *BoardState    `json:"boardState"`
	ToMove          string         `json:"toMove"`
	MoveHistory     []Move         `json:"moveHistory"`
	CapturedPieces  CapturedPieces `json:"capturedPieces"`
	IsCheck         bool           `json:"isCheck"`
	SelectedSquare  *Position      `json:"selectedSquare"` // Made nullable
	LegalMoves      []Position     `json:"legalMoves"`
	EnPassantTarget *Position      `json:"enPassantTarget"` // Made nullable
	Resolve         *string        `json:"resolve"`         // Made nullable
	Players         struct {
		White ClientPlayer `json:"white"`
		Black ClientPlayer `json:"black"`
	} `json:"players"`
	PromotionSquare        *Position   `json:"promotionSquare"`        // Made nullable
	PromotionPiece         *PieceType  `json:"promotionPiece"`         // Made nullable
	Mine                   *Position   `json:"mine"`                   // Made nullable
	PendingMoveDestination *Position   `json:"pendingMoveDestination"` // Made nullable
	LastMove               *SimpleMove `json:"lastMove"`               // Made nullable
}

type CapturedPieces struct {
	White []Piece `json:"white"`
	Black []Piece `json:"black"`
}

func NewGame(id string) *Game {
	return &Game{
		ID:          id,
		mu:          sync.Mutex{},
		state:       newGameState(),
		connections: NewGameConnections(),
		whiteClock:  NewClock(time.Duration(600) * time.Second),
		blackClock:  NewClock(time.Duration(600) * time.Second),
	}
}

func NewGameConnections() *GameConnections {
	return &GameConnections{
		connections: make(map[string]*websocket.Conn),
	}
}

func newGameState() GameState {
	return GameState{
		Sound:           "",
		Board:           newBoard(),
		ToMove:          "white",
		MoveHistory:     make([]Move, 0),
		CapturedPieces:  newCapturedPieces(),
		IsCheck:         false,
		SelectedSquare:  nil,
		LegalMoves:      make([]Position, 0),
		EnPassantTarget: nil,
		Resolve:         nil,
		Players: struct {
			White ClientPlayer `json:"white"`
			Black ClientPlayer `json:"black"`
		}{
			White: ClientPlayer{
				ID:       "",
				Color:    "",
				TimeLeft: 6000,
			},
			Black: ClientPlayer{
				ID:       "",
				Color:    "",
				TimeLeft: 6000,
			},
		},
		PromotionSquare: nil,
		PromotionPiece:  nil,
		Mine:            nil,
		LastMove:        nil,
	}
}

func newCapturedPieces() CapturedPieces {
	return CapturedPieces{
		White: make([]Piece, 0),
		Black: make([]Piece, 0),
	}
}

func (g *Game) AddPlayer(playerID string) (string, error) {
	fmt.Println("Adding player to game in model/game", playerID, g.state.Players)
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.state.Players.White.ID == "" {
		g.state.Players.White = ClientPlayer{
			ID:       playerID,
			Color:    "white",
			TimeLeft: 6000,
		}
		return "white", nil
	}
	if g.state.Players.Black.ID == "" {
		g.state.Players.Black = ClientPlayer{
			ID:       playerID,
			Color:    "black",
			TimeLeft: 6000,
		}
		return "black", nil
	}
	return "", errors.New("game is full")
}

func (g *Game) GetState() GameState {
	g.mu.Lock()
	defer g.mu.Unlock()

	return g.state
}

func (g *Game) IsPlayerInGame(playerID string) bool {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.state.Players.White.ID != "" && g.state.Players.White.ID == playerID {
		return true
	}
	if g.state.Players.Black.ID != "" && g.state.Players.Black.ID == playerID {
		return true
	}
	return false
}

func (g *Game) isPlayerInGame(playerID string) bool {
	if g.state.Players.White.ID != "" && g.state.Players.White.ID == playerID {
		return true
	}
	if g.state.Players.Black.ID != "" && g.state.Players.Black.ID == playerID {
		return true
	}
	return false
}

func (g *Game) CanSpectate() bool {
	g.mu.Lock()
	defer g.mu.Unlock()

	return g.state.Players.White.ID == "" || g.state.Players.Black.ID == ""
}

func (g *Game) canSpectate() bool {
	return g.state.Players.White.ID == "" || g.state.Players.Black.ID == ""
}

func (g *Game) MakeMove(move WSMove) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	fmt.Println("Making move in model/game", move)

	if g.state.ToMove != g.state.Board.Board[move.From.Y][move.From.X].Color {
		return errors.New("not your turn")
	}

	if g.state.Board.Board[move.From.Y][move.From.X] == nil {
		return errors.New("no piece at from square")
	}

	// Validate and execute the move
	if err := g.validateMove(move); err != nil {
		return err
	}
	// Set opposing players clock
	// Stop current player's clock
	if g.state.ToMove == "white" {
		g.whiteClock.Stop()
	} else {
		g.blackClock.Stop()
	}

	err := g.executeMove(move)
	if err != nil {
		return err
	}
	// Start opposing players clock
	if g.state.ToMove == "white" {
		g.whiteClock.Start()
	} else {
		g.blackClock.Start()
	}

	// update client clock for both players
	g.state.Players.White.TimeLeft = int(g.whiteClock.timeLeft.Milliseconds() / 100)
	g.state.Players.Black.TimeLeft = int(g.blackClock.timeLeft.Milliseconds() / 100)

	return nil
}

/*
	func (g *Game) Resign(playerID string) error {
		g.mu.Lock()
		defer g.mu.Unlock()

		if !g.IsPlayerInGame(playerID) {
			return errors.New("player not in game")
		}

		return nil
	}

	func (g *Game) OfferDraw(playerID string) error {
	    g.mu.Lock()
	    defer g.mu.Unlock()

	    if !g.IsPlayerInGame(playerID) {
	        return errors.New("player not in game")
	    }

	    g.drawOffer = &DrawOffer{
	        OfferedBy: playerID,
	        OfferedAt: time.Now(),
	    }
	    return nil
	}
*/
func (g *Game) validateMove(move WSMove) error {
	fmt.Println("Validating move in model/game", move)
	fmt.Println("legal moves", g.getLegalMovesForPiece(g.state.Board.Board[move.From.Y][move.From.X]))
	// TODO: Implement move validation
	if move.From.X < 0 || move.From.X > 7 || move.From.Y < 0 || move.From.Y > 7 || move.To.X < 0 || move.To.X > 7 || move.To.Y < 0 || move.To.Y > 7 {
		return errors.New("invalid move, out of bounds")
	}
	// check if move is legal
	moveToCheck := SimpleMove{From: move.From, To: move.To}
	isLegal := false
	for _, legalMove := range g.getLegalMovesForPiece(g.state.Board.Board[move.From.Y][move.From.X]) {
		if legalMove.From == moveToCheck.From && legalMove.To == moveToCheck.To {
			isLegal = true
			break
		}
	}
	if !isLegal {
		return errors.New("invalid move, not legal")
	}

	return nil
}

func (g *Game) executeMove(move WSMove) error {
	ply := g.makePly(move)
	// clear last turns sounds
	g.state.Sound = ""
	// add sound to move history
	if g.mine != nil && move.To.X == g.mine.X && move.To.Y == g.mine.Y {
		g.state.Sound = "explosion"
	} else if g.state.Board.Board[move.To.Y][move.To.X] != nil {
		g.state.Sound = "capture"
	} else {
		g.state.Sound = "move"
	}
	// move the piece
	piece := g.state.Board.Board[move.From.Y][move.From.X]
	g.state.Board.Board[move.From.Y][move.From.X] = nil
	g.state.Board.Board[move.To.Y][move.To.X] = piece
	// set hasMoved to true
	g.state.Board.Board[move.To.Y][move.To.X].HasMoved = true
	// if promotion, change piece type
	if move.Promotion != "" {
		g.state.Board.Board[move.To.Y][move.To.X].Type = move.Promotion
	}
	// if king move, handle castle and update king position
	if piece.Type == King {
		ply = g.handleCastle(move, ply)
		switch g.state.ToMove {
		case "white":
			g.state.Board.WhiteKingPosition = move.To
		case "black":
			g.state.Board.BlackKingPosition = move.To
		}
	}
	// if en passant, handle en passant
	if piece.Type == Pawn {
		ply = g.handleEnPassant(move, ply)
	}

	// Add the ply to the move history
	if g.state.ToMove == "white" {
		// If white moved, add Move
		g.state.MoveHistory = append(g.state.MoveHistory, Move{
			WhitePly: ply,
		})
	} else {
		// If black moved, add BlackPly to the last Move
		lastIdx := len(g.state.MoveHistory) - 1
		g.state.MoveHistory[lastIdx].BlackPly = ply
	}

	// update moved pieces position
	g.state.Board.Board[move.To.Y][move.To.X].Position = move.To

	// if piece landed on mine, remove piece and check for bombmate
	if g.mine != nil && move.To.X == g.mine.X && move.To.Y == g.mine.Y {
		switch g.state.ToMove {
		case "white":
			g.state.CapturedPieces.White = append(g.state.CapturedPieces.White, *g.state.Board.Board[move.To.Y][move.To.X])
		case "black":
			g.state.CapturedPieces.Black = append(g.state.CapturedPieces.Black, *g.state.Board.Board[move.To.Y][move.To.X])
		}
		g.state.Board.Board[move.To.Y][move.To.X] = nil
		if isKingInCheck(g.state.Board, g.state.ToMove) {
			result := "bombmate"
			g.state.Resolve = &result
		}
	}

	// set mine
	g.mine = &move.Mine

	// switch turn
	g.switchTurn()
	// check if opponent king is in check after move
	g.state.IsCheck = isKingInCheck(g.state.Board, g.state.ToMove)
	// check if game is over
	if g.isNoLegalMoves(g.state.ToMove) {
		switch g.state.IsCheck {
		case true:
			result := "checkmate"
			g.state.Resolve = &result
		case false:
			result := "stalemate"
			g.state.Resolve = &result
		}
	}
	// if king in check, set check sound
	if g.state.IsCheck {
		g.state.Sound = "check"
	}
	// set lastMove
	g.state.LastMove = &SimpleMove{From: move.From, To: move.To}

	go g.broadcastState()

	return nil
}

func isKingInCheck(boardState *BoardState, color string) bool {
	// TODO: Implement king in check detection
	if color == "white" {
		return isSquareAttacked(boardState, "black", boardState.WhiteKingPosition)
	}
	return isSquareAttacked(boardState, "white", boardState.BlackKingPosition)
}

func isSquareAttacked(boardState *BoardState, attackingColor string, position Position) bool {
	// TODO: Implement square attacked detection
	rookDirs := []Position{{X: 1, Y: 0}, {X: -1, Y: 0}, {X: 0, Y: 1}, {X: 0, Y: -1}}
	bishopDirs := []Position{{X: 1, Y: 1}, {X: 1, Y: -1}, {X: -1, Y: 1}, {X: -1, Y: -1}}
	knightDirs := []Position{{X: 2, Y: 1}, {X: 2, Y: -1}, {X: -2, Y: 1}, {X: -2, Y: -1}, {X: 1, Y: 2}, {X: 1, Y: -2}, {X: -1, Y: 2}, {X: -1, Y: -2}}
	kingDirs := []Position{{X: 1, Y: 0}, {X: -1, Y: 0}, {X: 0, Y: 1}, {X: 0, Y: -1}, {X: 1, Y: 1}, {X: 1, Y: -1}, {X: -1, Y: 1}, {X: -1, Y: -1}}
	pawnDirs := []Position{{X: -1, Y: 1}, {X: 1, Y: 1}}
	for _, dir := range rookDirs {
		targetPos := Position{X: position.X + dir.X, Y: position.Y + dir.Y}
		for boundaryCheck(targetPos) {
			if boardState.Board[targetPos.Y][targetPos.X] != nil {
				if boardState.Board[targetPos.Y][targetPos.X].Color == attackingColor && (boardState.Board[targetPos.Y][targetPos.X].Type == Queen || boardState.Board[targetPos.Y][targetPos.X].Type == Rook) {
					return true
				} else {
					break
				}
			}
			targetPos = Position{X: targetPos.X + dir.X, Y: targetPos.Y + dir.Y}
		}
	}
	for _, dir := range bishopDirs {
		targetPos := Position{X: position.X + dir.X, Y: position.Y + dir.Y}
		for boundaryCheck(targetPos) {
			if boardState.Board[targetPos.Y][targetPos.X] != nil {
				if boardState.Board[targetPos.Y][targetPos.X].Color == attackingColor && (boardState.Board[targetPos.Y][targetPos.X].Type == Queen || boardState.Board[targetPos.Y][targetPos.X].Type == Bishop) {
					return true
				} else {
					break
				}
			}
			targetPos = Position{X: targetPos.X + dir.X, Y: targetPos.Y + dir.Y}
		}
	}
	for _, dir := range knightDirs {
		targetPos := Position{X: position.X + dir.X, Y: position.Y + dir.Y}
		if boundaryCheck(targetPos) && boardState.Board[targetPos.Y][targetPos.X] != nil && boardState.Board[targetPos.Y][targetPos.X].Color == attackingColor && boardState.Board[targetPos.Y][targetPos.X].Type == Knight {
			return true
		}
	}
	for _, dir := range kingDirs {
		targetPos := Position{X: position.X + dir.X, Y: position.Y + dir.Y}
		if boundaryCheck(targetPos) && boardState.Board[targetPos.Y][targetPos.X] != nil && boardState.Board[targetPos.Y][targetPos.X].Color == attackingColor && boardState.Board[targetPos.Y][targetPos.X].Type == King {
			return true
		}
	}
	for _, dir := range pawnDirs {
		targetPos := Position{X: position.X + dir.X, Y: position.Y + dir.Y}
		if boundaryCheck(targetPos) && boardState.Board[targetPos.Y][targetPos.X] != nil && boardState.Board[targetPos.Y][targetPos.X].Color == attackingColor && boardState.Board[targetPos.Y][targetPos.X].Type == Pawn {
			return true
		}
	}
	return false
}

func boundaryCheck(position Position) bool {
	return position.X >= 0 && position.X < 8 && position.Y >= 0 && position.Y < 8
}

func (g *Game) isNoLegalMoves(color string) bool {
	// TODO: Implement no legal moves detection
	return len(g.getLegalMovesForColor(color)) == 0
}

func (g *Game) getLegalMovesForColor(color string) []SimpleMove {
	// TODO: Implement get legal moves for color
	legalMoves := []SimpleMove{}
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			if g.state.Board.Board[y][x] != nil && g.state.Board.Board[y][x].Color == color {
				legalMoves = append(legalMoves, g.getLegalMovesForPiece(g.state.Board.Board[y][x])...)
			}
		}
	}
	return legalMoves
}

func (g *Game) getLegalMovesForPiece(piece *Piece) []SimpleMove {
	// TODO: Implement get legal moves for piece
	switch piece.Type {
	case Pawn:
		psuedoMoves := g.getPsuedoPawnMoves(piece)
		return g.filterLegalMoves(psuedoMoves)
	case Knight:
		psuedoMoves := g.getPsuedoKnightMoves(piece)
		return g.filterLegalMoves(psuedoMoves)
	case Bishop:
		psuedoMoves := g.getPsuedoBishopMoves(piece)
		return g.filterLegalMoves(psuedoMoves)
	case Rook:
		psuedoMoves := g.getPsuedoRookMoves(piece)
		return g.filterLegalMoves(psuedoMoves)
	case Queen:
		psuedoMoves := g.getPsuedoQueenMoves(piece)
		return g.filterLegalMoves(psuedoMoves)
	case King:
		psuedoMoves := g.getPsuedoKingMoves(piece)
		return g.filterLegalMoves(psuedoMoves)
	default:
		return []SimpleMove{}
	}
}

func (g *Game) filterLegalMoves(psuedoMoves []SimpleMove) []SimpleMove {
	// TODO: Implement move filtering
	legalMoves := []SimpleMove{}
	for _, move := range psuedoMoves {
		// record current state
		fromState := g.state.Board.Board[move.From.Y][move.From.X]
		toState := g.state.Board.Board[move.To.Y][move.To.X]
		// if king move, set king position
		if fromState.Type == King {
			switch g.state.ToMove {
			case "white":
				g.state.Board.WhiteKingPosition = move.To
			case "black":
				g.state.Board.BlackKingPosition = move.To
			}
		}
		// execute temp move
		g.state.Board.Board[move.To.Y][move.To.X] = fromState
		g.state.Board.Board[move.From.Y][move.From.X] = nil
		// check if king is in check
		if !isKingInCheck(g.state.Board, g.state.ToMove) {
			legalMoves = append(legalMoves, move)
		}
		// revert temp move
		g.state.Board.Board[move.From.Y][move.From.X] = fromState
		g.state.Board.Board[move.To.Y][move.To.X] = toState
		// if king move, revert king position
		if fromState.Type == King {
			switch g.state.ToMove {
			case "white":
				g.state.Board.WhiteKingPosition = fromState.Position
			case "black":
				g.state.Board.BlackKingPosition = fromState.Position
			}
		}
	}
	return legalMoves
}

func (g *Game) getPsuedoPawnMoves(piece *Piece) []SimpleMove {
	pawnMoves := []SimpleMove{}
	dir := Position{X: 0, Y: -1}
	enPassantDirs := []Position{{X: 1, Y: -1}, {X: -1, Y: -1}}
	if piece.Color == "black" {
		dir = Position{X: 0, Y: 1}
		enPassantDirs = []Position{{X: 1, Y: 1}, {X: -1, Y: 1}}
	}
	// Check move forward 1
	if g.state.Board.Board[piece.Position.Y+dir.Y][piece.Position.X] == nil {
		pawnMoves = append(pawnMoves, SimpleMove{From: piece.Position, To: Position{X: piece.Position.X, Y: piece.Position.Y + dir.Y}})
		// Check move forward 2 if not moved
		if !piece.HasMoved && g.state.Board.Board[piece.Position.Y+dir.Y*2][piece.Position.X] == nil {
			pawnMoves = append(pawnMoves, SimpleMove{From: piece.Position, To: Position{X: piece.Position.X, Y: piece.Position.Y + dir.Y*2}})
		}
	}
	// Check capture left
	if piece.Position.X > 0 && g.state.Board.Board[piece.Position.Y+dir.Y][piece.Position.X-1] != nil && g.state.Board.Board[piece.Position.Y+dir.Y][piece.Position.X-1].Color != piece.Color {
		pawnMoves = append(pawnMoves, SimpleMove{From: piece.Position, To: Position{X: piece.Position.X - 1, Y: piece.Position.Y + dir.Y}})
	}
	// Check capture right
	if piece.Position.X < 7 && g.state.Board.Board[piece.Position.Y+dir.Y][piece.Position.X+1] != nil && g.state.Board.Board[piece.Position.Y+dir.Y][piece.Position.X+1].Color != piece.Color {
		pawnMoves = append(pawnMoves, SimpleMove{From: piece.Position, To: Position{X: piece.Position.X + 1, Y: piece.Position.Y + dir.Y}})
	}
	// Check en passant
	for _, dir := range enPassantDirs {
		if g.state.EnPassantTarget != nil && g.state.EnPassantTarget.X == piece.Position.X+dir.X && g.state.EnPassantTarget.Y == piece.Position.Y+dir.Y {
			pawnMoves = append(pawnMoves, SimpleMove{From: piece.Position, To: Position{X: piece.Position.X + dir.X, Y: piece.Position.Y + dir.Y}})
		}
	}
	return pawnMoves
}

func (g *Game) getPsuedoKnightMoves(piece *Piece) []SimpleMove {
	// TODO: Implement psuedo knight moves
	knightMoves := []SimpleMove{}
	knightDirs := []Position{{X: 2, Y: 1}, {X: 2, Y: -1}, {X: -2, Y: 1}, {X: -2, Y: -1}, {X: 1, Y: 2}, {X: 1, Y: -2}, {X: -1, Y: 2}, {X: -1, Y: -2}}
	for _, dir := range knightDirs {
		targetPos := Position{X: piece.Position.X + dir.X, Y: piece.Position.Y + dir.Y}
		if boundaryCheck(targetPos) && (g.state.Board.Board[targetPos.Y][targetPos.X] == nil || g.state.Board.Board[targetPos.Y][targetPos.X].Color != piece.Color) {
			knightMoves = append(knightMoves, SimpleMove{From: piece.Position, To: targetPos})
		}
	}
	return knightMoves
}

func (g *Game) getPsuedoBishopMoves(piece *Piece) []SimpleMove {
	// TODO: Implement psuedo bishop moves
	bishopMoves := []SimpleMove{}
	bishopDirs := []Position{{X: 1, Y: 1}, {X: 1, Y: -1}, {X: -1, Y: 1}, {X: -1, Y: -1}}
	for _, dir := range bishopDirs {
		targetPos := Position{X: piece.Position.X + dir.X, Y: piece.Position.Y + dir.Y}
		for boundaryCheck(targetPos) {
			if g.state.Board.Board[targetPos.Y][targetPos.X] == nil {
				bishopMoves = append(bishopMoves, SimpleMove{From: piece.Position, To: targetPos})
			} else if g.state.Board.Board[targetPos.Y][targetPos.X].Color != piece.Color {
				bishopMoves = append(bishopMoves, SimpleMove{From: piece.Position, To: targetPos})
				break
			} else {
				break
			}
			targetPos = Position{X: targetPos.X + dir.X, Y: targetPos.Y + dir.Y}
		}
	}
	return bishopMoves
}

func (g *Game) getPsuedoRookMoves(piece *Piece) []SimpleMove {
	// TODO: Implement psuedo rook moves
	rookMoves := []SimpleMove{}
	rookDirs := []Position{{X: 1, Y: 0}, {X: -1, Y: 0}, {X: 0, Y: 1}, {X: 0, Y: -1}}
	for _, dir := range rookDirs {
		targetPos := Position{X: piece.Position.X + dir.X, Y: piece.Position.Y + dir.Y}
		for boundaryCheck(targetPos) {
			if g.state.Board.Board[targetPos.Y][targetPos.X] == nil {
				rookMoves = append(rookMoves, SimpleMove{From: piece.Position, To: targetPos})
			} else if g.state.Board.Board[targetPos.Y][targetPos.X].Color != piece.Color {
				rookMoves = append(rookMoves, SimpleMove{From: piece.Position, To: targetPos})
				break
			} else {
				break
			}
			targetPos = Position{X: targetPos.X + dir.X, Y: targetPos.Y + dir.Y}
		}
	}
	return rookMoves
}

func (g *Game) getPsuedoQueenMoves(piece *Piece) []SimpleMove {
	// TODO: Implement psuedo queen moves
	return append(g.getPsuedoBishopMoves(piece), g.getPsuedoRookMoves(piece)...)
}

func (g *Game) getPsuedoKingMoves(piece *Piece) []SimpleMove {
	// TODO: Implement psuedo king moves
	kingMoves := []SimpleMove{}
	kingDirs := []Position{{X: 1, Y: 0}, {X: -1, Y: 0}, {X: 0, Y: 1}, {X: 0, Y: -1}, {X: 1, Y: 1}, {X: 1, Y: -1}, {X: -1, Y: 1}, {X: -1, Y: -1}}
	for _, dir := range kingDirs {
		targetPos := Position{X: piece.Position.X + dir.X, Y: piece.Position.Y + dir.Y}
		if boundaryCheck(targetPos) && (g.state.Board.Board[targetPos.Y][targetPos.X] == nil || g.state.Board.Board[targetPos.Y][targetPos.X].Color != piece.Color) {
			kingMoves = append(kingMoves, SimpleMove{From: piece.Position, To: targetPos})
		}
	}
	if !piece.HasMoved {
		// check for castle moves
		if g.state.Board.Board[piece.Position.Y][0] != nil && g.state.Board.Board[piece.Position.Y][0].Type == Rook && !g.state.Board.Board[piece.Position.Y][0].HasMoved {
			if g.state.Board.Board[piece.Position.Y][1] == nil && g.state.Board.Board[piece.Position.Y][2] == nil && g.state.Board.Board[piece.Position.Y][3] == nil {
				kingMoves = append(kingMoves, SimpleMove{From: piece.Position, To: Position{X: piece.Position.X - 2, Y: piece.Position.Y}})
			}
		}
		if g.state.Board.Board[piece.Position.Y][7] != nil && g.state.Board.Board[piece.Position.Y][7].Type == Rook && !g.state.Board.Board[piece.Position.Y][7].HasMoved {
			if g.state.Board.Board[piece.Position.Y][5] == nil && g.state.Board.Board[piece.Position.Y][6] == nil {
				kingMoves = append(kingMoves, SimpleMove{From: piece.Position, To: Position{X: piece.Position.X + 2, Y: piece.Position.Y}})
			}
		}
	}
	return kingMoves
}

func (g *Game) handleEnPassant(move WSMove, ply Ply) Ply {
	// if the move is an en passant capture, remove the captured piece and alter ply notation
	if g.state.EnPassantTarget != nil && move.To.X == g.state.EnPassantTarget.X && move.To.Y == g.state.EnPassantTarget.Y {
		switch g.state.ToMove {
		case "white":
			g.state.CapturedPieces.White = append(g.state.CapturedPieces.White, *g.state.Board.Board[move.To.Y+1][move.To.X])
			g.state.Board.Board[move.To.Y+1][move.To.X] = nil
		case "black":
			g.state.CapturedPieces.Black = append(g.state.CapturedPieces.Black, *g.state.Board.Board[move.To.Y-1][move.To.X])
			g.state.Board.Board[move.To.Y-1][move.To.X] = nil
		}
		ply.Notation = "x" + ply.Notation
	}
	// if the move is double pawn move, set en passant target
	switch move.To.Y - move.From.Y {
	case 2:
		g.state.EnPassantTarget = &Position{X: move.To.X, Y: move.To.Y - 1}
	case -2:
		g.state.EnPassantTarget = &Position{X: move.To.X, Y: move.To.Y + 1}
	default:
		g.state.EnPassantTarget = nil
	}

	return ply
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func (g *Game) handleCastle(move WSMove, ply Ply) Ply {
	fmt.Println("Handling castle for move", move)
	// TODO: Implement castle handling
	// assume only called for king move
	if abs(move.From.X-move.To.X) == 2 {
		switch move.To.X {
		case 2:
			rook := g.state.Board.Board[move.From.Y][0]
			g.state.Board.Board[move.From.Y][0] = nil
			g.state.Board.Board[move.From.Y][3] = rook
			rook.HasMoved = true
			ply.CastleRookMove = &CastleRookMove{
				From: Position{X: 0, Y: move.From.Y},
				To:   Position{X: 3, Y: move.From.Y},
			}
			ply.Notation = "O-O-O"
		case 6:
			rook := g.state.Board.Board[move.From.Y][7]
			g.state.Board.Board[move.From.Y][7] = nil
			g.state.Board.Board[move.From.Y][5] = rook
			rook.HasMoved = true
			ply.CastleRookMove = &CastleRookMove{
				From: Position{X: 7, Y: move.From.Y},
				To:   Position{X: 5, Y: move.From.Y},
			}
			ply.Notation = "O-O"
		}
	}
	return ply
}

func (g *Game) makePly(move WSMove) Ply {
	// return ply without rook castle move, add castle rook move in castle detection
	return Ply{
		Piece:          g.state.Board.Board[move.From.Y][move.From.X],
		From:           move.From,
		To:             move.To,
		CapturedPiece:  g.state.Board.Board[move.To.Y][move.To.X],
		CastleRookMove: nil,
		Promotion:      move.Promotion,
		Notation:       g.getNotation(move),
	}
}

func (g *Game) getNotation(move WSMove) string {
	// TODO: Implement notation
	piece := g.state.Board.Board[move.From.Y][move.From.X]
	from := move.From
	to := move.To
	pieceNotationPrefix := piece.Type.getPieceNotation()
	pieceNotationCapture := ""
	if g.state.Board.Board[to.Y][to.X] != nil {
		pieceNotationCapture = "x"
	}
	pieceNotationSuffix := to.getSquareNotation()
	pawnFileSpecifier := ""
	if piece.Type == Pawn && from.X != to.X {
		pawnFileSpecifier = from.getFileNotation()
	}
	if g.mine != nil && to.X == g.mine.X && to.Y == g.mine.Y {
		pieceNotationSuffix += "*"
	}
	return fmt.Sprintf("%s%s%s%s", pieceNotationPrefix, pawnFileSpecifier, pieceNotationCapture, pieceNotationSuffix)
}

func (g *Game) switchTurn() {
	if g.state.ToMove == "white" {
		g.state.ToMove = "black"
	} else {
		g.state.ToMove = "white"
	}
}

func (g *Game) RegisterConnection(playerID string, conn *websocket.Conn) error {
	connID := fmt.Sprintf("%p", conn)
	fmt.Printf("Starting RegisterConnection for player %s, conn %s\n", playerID, connID)

	g.mu.Lock()
	isAuthorized := g.isPlayerInGame(playerID) || g.canSpectate()
	g.mu.Unlock()

	if !isAuthorized {
		return errors.New("not authorized to join this game")
	}

	g.connections.mu.Lock()
	if _, exists := g.connections.connections[playerID]; exists {
		// If we already have a healthy connection, keep it and reject the new one
		g.connections.mu.Unlock()
		conn.WriteMessage(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(
				websocket.CloseNormalClosure,
				"Connection already exists",
			),
		)
		conn.Close()
		return nil // Not really an error, just rejecting duplicate connection
	}

	// Register new connection
	g.connections.connections[playerID] = conn
	g.connections.mu.Unlock()
	fmt.Printf("Registered new connection %s for player %s\n", connID, playerID)

	// Send initial state...
	go g.broadcastState()
	return nil
}

// Modify UnregisterConnection to be more selective
func (g *Game) UnregisterConnection(playerID string) {
	g.connections.mu.Lock()
	defer g.connections.mu.Unlock()

	if conn, exists := g.connections.connections[playerID]; exists {
		connID := fmt.Sprintf("%p", conn)
		// Only unregister if this is still the current connection
		if fmt.Sprintf("%p", g.connections.connections[playerID]) == connID {
			fmt.Printf("Unregistering current connection %s for player %s\n", connID, playerID)
			delete(g.connections.connections, playerID)
		} else {
			fmt.Printf("Ignoring unregister for old connection %s for player %s\n", connID, playerID)
		}
	}
}

// Modify BroadcastState to safely handle concurrent access
func (g *Game) broadcastState() error {
	// Get a snapshot of connections under the connections mutex
	g.connections.mu.RLock()
	defer g.connections.mu.RUnlock()
	// Make a copy of the connections we need to broadcast to
	activeConnections := make(map[string]*websocket.Conn)
	for playerID, conn := range g.connections.connections {
		activeConnections[playerID] = conn
	}

	// Now broadcast to each connection without holding any locks
	for playerID, conn := range activeConnections {
		fmt.Println("Broadcasting state to player", playerID, g.state)
		jsonGameState, err := json.Marshal(g.state)
		if err != nil {
			fmt.Println("Failed to marshal state to JSON", err)
			continue
		}

		if err := conn.WriteJSON(ws.Message{
			Type:    ws.MessageTypeGameState,
			Payload: json.RawMessage(jsonGameState),
		}); err != nil {
			fmt.Println("Failed to send state to player", playerID, err)
			// Consider removing failed connections
			g.connections.mu.Lock()
			delete(g.connections.connections, playerID)
			g.connections.mu.Unlock()
			continue
		}
		fmt.Println("Sent state to player", playerID)
	}
	return nil
}
