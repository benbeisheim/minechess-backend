package model

import (
	"fmt"
	"sync"
	"time"
)

type Clock struct {
	mu          sync.Mutex
	timeLeft    time.Duration
	lastStarted time.Time // When the clock was last started
	isRunning   bool
}

type ClientClock struct {
	TimeLeft int `json:"timeLeft"`
}

func NewClientClock(initialTime int) *ClientClock {
	return &ClientClock{
		TimeLeft: initialTime,
	}
}

func NewClock(initialTime time.Duration) *Clock {
	return &Clock{
		timeLeft:  initialTime,
		isRunning: false,
	}
}

func (c *Clock) Start() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.isRunning {
		c.lastStarted = time.Now()
		fmt.Println("clock started", c.lastStarted)
		c.isRunning = true
	}
}

func (c *Clock) Stop() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.isRunning {
		c.timeLeft -= time.Since(c.lastStarted)
		fmt.Println("clock stopped", c.timeLeft)
		c.isRunning = false
	}
}

func (c *Clock) GetTimeLeft() time.Duration {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.isRunning {
		return c.timeLeft - time.Since(c.lastStarted)
	}
	return c.timeLeft
}
