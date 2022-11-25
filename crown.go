package horloger

import (
	"sync"
	"time"
)

// Clock represents a controllable clock. The function NewClock returns a new
// one, so there is no need to initialize Clock directly.
type Clock struct {
	mu      sync.Mutex
	tick    sync.Cond
	current time.Time
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
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.current
}

// Forward makes a forward time travel according to the specified duration d.
func (c *Clock) Forward(d time.Duration) {
	c.mu.Lock()
	c.current = c.current.Add(d)
	c.mu.Unlock()
	c.Tick()
}

// Tick lets all listeners know that the current time has changed. Tick is
// automatically called when Forward is called.
func (c *Clock) Tick() {
	time.Sleep(time.Millisecond) // rate-limit Broadcast call
	c.tick.Broadcast()
}

// Sleep returns when the clock has reached its curent time + the specified
// duration d.
func (c *Clock) Sleep(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	deadline := c.current.Add(d)
	for c.current.Before(deadline) { // c.current is locked here
		c.tick.Wait()
	}
}
