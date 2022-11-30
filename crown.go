package crown

import (
	"sync"
	"sync/atomic"
	"time"
)

// Clock represents a controllable clock. The function NewClock returns a new
// one, so there is no need to initialize Clock directly.
type Clock struct {
	mu         sync.RWMutex
	tick       sync.Cond
	current    time.Time
	sleepCount int32 // For testing purposes
}

func (c *Clock) GetSleepCount() int32 {
	return atomic.LoadInt32(&c.sleepCount)
}

// NewClock initializes and returns a new Clock object which starts at time t.
func NewClock(t time.Time) *Clock {
	clock := new(Clock)
	clock.tick.L = &clock.mu
	clock.current = t
	return clock
}

// Now returns the current clock time.
func (c *Clock) Now() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.current
}

// Forward makes a forward time travel according to the specified duration d.
func (c *Clock) Forward(d time.Duration) {
	c.mu.Lock()
	c.current = c.current.Add(d)
	c.mu.Unlock()
	c.tick.Broadcast()
}

// Sleep returns when the clock has reached its curent time + the specified
// duration d.
func (c *Clock) Sleep(d time.Duration) {
	atomic.AddInt32(&c.sleepCount, 1)
	c.mu.Lock()
	defer c.mu.Unlock()
	deadline := c.current.Add(d)
	for c.current.Before(deadline) { // c.current is locked here
		c.tick.Wait()
	}
}
