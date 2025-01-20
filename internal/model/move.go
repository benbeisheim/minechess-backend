package model

type WSMove struct {
	From      Position
	To        Position
	Promotion PieceType
	Mine      Position
}

type CastleRookMove struct {
	From Position `json:"from"`
	To   Position `json:"to"`
}

type Ply struct {
	Piece          *Piece          `json:"piece"`
	From           Position        `json:"from"`
	To             Position        `json:"to"`
	CapturedPiece  *Piece          `json:"capturedPiece"`
	CastleRookMove *CastleRookMove `json:"castleRookMove"`
	Promotion      PieceType       `json:"promotion"`
	Notation       string          `json:"notation"`
}

type Move struct {
	WhitePly Ply `json:"whitePly"`
	BlackPly Ply `json:"blackPly"`
}

type SimpleMove struct {
	From Position `json:"from"`
	To   Position `json:"to"`
}
