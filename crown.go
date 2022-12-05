package crown

import (
	"context"
	"errors"
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

type Timer struct {
	C      <-chan time.Time
	cancel func()
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
	c.SleepWithContext(context.Background(), d)
}

func (c *Clock) SleepWithContext(ctx context.Context, d time.Duration) error {
	c.mu.Lock()
	deadline := c.current.Add(d)
	condVarWait := make(chan struct{})
	defer close(condVarWait)
	atomic.AddInt32(&c.sleepCount, 1)
	for c.current.Before(deadline) { // c.current is locked here
		go func() {
			c.tick.Wait()
			select {
			case <-ctx.Done():
				return // Return before attempting sending on closed channel
			default:
			}
			condVarWait <- struct{}{}
		}()
		select {
		case <-condVarWait:
		case <-ctx.Done():
			// Bypass c.mu.Unlock() here
			return errors.New("Canceled")
		}
	}
	c.mu.Unlock()
	return nil
}

// NewTimer creates a new clock-associated Timer that will send the current
// time on its channel after at least duration d.
func (c *Clock) NewTimer(d time.Duration) *Timer {
	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan time.Time, 1)
	go func() {
		defer close(ch)
		err := c.SleepWithContext(ctx, d)
		if err != nil {
			return
		}
		ch <- c.Now()
	}()
	return &Timer{
		C:      ch,
		cancel: cancel,
	}
}

func (t *Timer) Stop() bool {
	t.cancel()
	return true
}
