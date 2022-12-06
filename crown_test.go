package crown

import (
	"context"
	"math/rand"
	"sync"
	"testing"
	"time"
)

func waitForSleepers(t *testing.T, clock *Clock, target, maxretry int) {
	for try := 1; ; try++ {
		time.Sleep(333 * time.Microsecond)
		c := int(clock.GetSleepCount())
		if c == target {
			break
		}
		if c != target && try == maxretry {
			t.Fatalf("Retry=%d times to wait for timer to be ready", maxretry)
		}
	}
}

func TestClockSet(t *testing.T) {
	refT, _ := time.Parse(time.RFC3339, "2006-01-02T15:04:05Z")
	clock := NewClock(refT)
	got := clock.Now()
	if got != refT {
		t.Errorf("Should be %q, got %q instead", refT, got)
	}
}

func TestClockForward(t *testing.T) {
	refT, _ := time.Parse(time.RFC3339, "2006-01-02T15:04:05Z")
	clock := NewClock(refT)
	timeForward := 2 * time.Second
	clock.Forward(timeForward)
	want, _ := time.Parse(time.RFC3339, "2006-01-02T15:04:07Z") // refT + 2 secs
	got := clock.Now()
	if got != want {
		t.Errorf("Should be %q, got %q instead", want, got)
	}
}

func TestSleep(t *testing.T) {
	refT, _ := time.Parse(time.RFC3339, "2022-11-25T01:00:00Z")
	clock := NewClock(refT)

	done1 := make(chan struct{})
	done2 := make(chan struct{})

	// Sleeper 1 (6 secs)
	go func() {
		clock.Sleep(6 * time.Second)
		done1 <- struct{}{}
	}()

	// Sleeper 2 (4 secs)
	go func() {
		clock.Sleep(4 * time.Second)
		done2 <- struct{}{}
	}()

	waitForSleepers(t, clock, 2, 10)

	clock.Forward(1 * time.Second)
	clock.Forward(1 * time.Second) // +2 secs
	select {
	case <-done1:
		t.Fatalf("Sleeper 1 returned prematuraly at t+2s. t=%q", clock.Now())
	case <-done2:
		t.Fatalf("Sleeper 2 returned prematuraly at t+2s. t=%q", clock.Now())
	default:
	}
	clock.Forward(1 * time.Second)
	clock.Forward(1 * time.Second) // +4 secs
	select {
	case <-done2:
	case <-time.After(1 * time.Second):
		t.Fatalf("Sleeper 2 did not return after clock has reached t+4s. t=%q", clock.Now())
	}
	clock.Forward(1 * time.Second)
	clock.Forward(1 * time.Second) // +6 secs
	select {
	case <-done1:
	case <-time.After(1 * time.Second):
		t.Fatalf("Sleeper 1 did not return after clock has reached t+6s. t=%q", clock.Now())
	}
}

func TestTimer(t *testing.T) {
	refT, _ := time.Parse(time.RFC3339, "2022-12-01T09:00:00Z")
	clock := NewClock(refT)
	delta := 42 * time.Second
	target := refT.Add(delta)

	timer := clock.NewTimer(delta)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		select {
		case <-timer.C:
		case <-time.After(time.Second):
			t.Fatalf("Did not return. t=%q", clock.Now())
		}
		now := clock.Now()
		if now.Before(target) {
			t.Errorf("Timer returned before target time %q. t=%q", target, now)
		}
	}()

	waitForSleepers(t, clock, 1, 10)

	clock.Forward(delta)
	wg.Wait()
}

func TestTimerStop(t *testing.T) {
	refT, _ := time.Parse(time.RFC3339, "2022-12-06T09:00:00Z")
	clock := NewClock(refT)
	timer := clock.NewTimer(42 * time.Second)

	done := make(chan struct{})
	defer close(done)
	go func() {
		var ok bool
		select {
		case _, ok = <-timer.C:
		case <-time.After(time.Second):
			t.Fatalf("Did not return after 1 sec")
		}
		if ok {
			t.Error("Timer has been fired despite Stop() call.", ok)
		}
		done <- struct{}{}
	}()

	waitForSleepers(t, clock, 1, 10)

	timer.Stop()
	clock.Forward(43 * time.Second)
	<-done
}

func TestSleepWithContext(t *testing.T) {
	refT, _ := time.Parse(time.RFC3339, "2022-12-01T09:00:00Z")
	clock := NewClock(refT)
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})

	// Sleeper
	go func() {
		err := clock.SleepWithContext(ctx, 42*time.Second)
		if err == nil {
			t.Errorf("Expected Canceled error here")
			return
		}
		close(done)
	}()

	waitForSleepers(t, clock, 1, 10)

	clock.Forward(20 * time.Second)
	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Errorf("Did not return after 1 sec")
	}
}

func TestConcurrentSleepers(t *testing.T) {
	refT, _ := time.Parse(time.RFC3339, "2022-12-05T09:00:00Z")
	clock := NewClock(refT)

	N := 100

	var wg sync.WaitGroup
	wg.Add(N)
	for i := 0; i < N; i++ {
		go func() {
			defer wg.Done()
			rsecs := time.Duration(rand.Intn(41))
			clock.Sleep(rsecs * time.Second)
		}()
	}

	waitForSleepers(t, clock, N, 10)

	clock.Forward(1 * time.Second)
	clock.Forward(5 * time.Second)
	clock.Forward(5 * time.Second)
	clock.Forward(10 * time.Second)
	clock.Forward(15 * time.Second)
	clock.Forward(5 * time.Second)
	clock.Forward(1 * time.Second) // Total = 42 secs

	wg.Wait()
}
