package crown

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// Clock represents a controllable clock. The function NewClock returns a new
// one, so there is no need to initialize Clock directly. A Clock object must
// not be copied.
type Clock struct {
	mu         sync.RWMutex
	current    time.Time
	handlers   sync.Map
	sleepCount int32
}

type Timer struct {
	C      <-chan time.Time
	cancel func()
}

type sleepHandler struct {
	deadline time.Time
	c        chan struct{}
}

// NewClock initializes and returns a new Clock object which starts at time t.
func NewClock(t time.Time) *Clock {
	clock := new(Clock)
	clock.current = t
	return clock
}

func (c *Clock) GetSleepCount() int32 {
	return atomic.LoadInt32(&c.sleepCount)
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

	// Broadcast
	c.mu.RLock()
	defer c.mu.RUnlock()
	c.handlers.Range(func(key, val any) bool {
		handler := val.(*sleepHandler)
		if c.current.Before(handler.deadline) {
			return true
		}
		close(handler.c)
		c.handlers.Delete(key)
		return true
	})
}

// Sleep returns when the clock has reached its curent time + the specified
// duration d.
func (c *Clock) Sleep(d time.Duration) {
	c.SleepWithContext(context.Background(), d)
}

func (c *Clock) SleepWithContext(ctx context.Context, d time.Duration) error {
	deadline := c.Now().Add(d)
	handlerID := atomic.AddInt32(&c.sleepCount, 1)
	ch := make(chan struct{})
	handler := &sleepHandler{
		c:        ch,
		deadline: deadline,
	}
	c.handlers.Store(handlerID, handler)
	select {
	case <-ctx.Done():
		//fmt.Println("cancel", c.Now(), "/", deadline)
		return ctx.Err()
	case <-ch:
		//fmt.Println("bye", c.Now(), "/", deadline)
	}
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
