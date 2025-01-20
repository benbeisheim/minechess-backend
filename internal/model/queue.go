package model

import (
	"fmt"
	"sync"
	"time"
)

type QueuedPlayer struct {
	Player   Player
	JoinedAt time.Time
	// We might want to add fields like:
	// PreferredTimeControl string
	// Rating int
}

type Queue struct {
	players []QueuedPlayer
	mu      sync.Mutex
}

func NewQueue() *Queue {
	return &Queue{
		players: []QueuedPlayer{},
	}
}

func (q *Queue) AddPlayer(player Player) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	for _, p := range q.players {
		if p.Player.ID == player.ID {
			return fmt.Errorf("player already in queue")
		}
	}

	qp := QueuedPlayer{
		Player:   player,
		JoinedAt: time.Now(),
	}
	q.players = append(q.players, qp)
	return nil
}

// GetNextPair finds two players to match together
func (q *Queue) GetNextPair() (Player, Player) {
	q.mu.Lock()
	defer q.mu.Unlock()

	// For now, just match the two players who have been waiting longest
	player1 := q.players[0].Player
	player2 := q.players[1].Player

	// Remove these players from the queue
	q.players = q.players[2:]

	return player1, player2
}

func (q *Queue) Size() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.players)
}
