package crown

import (
	"context"
	"sync"
	"testing"
	"time"
)

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

	// Wait for sleepers to be ready (retry=5)
	for try := 1; ; try++ {
		c := clock.GetSleepCount()
		if c == 2 {
			break
		}
		if c != 2 && try == 5 {
			t.Fatalf("Retry=5 times to wait for sleepers")
		}
		time.Sleep(time.Millisecond)
	}

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

	// Wait for timer to be ready (retry=5)
	for try := 1; ; try++ {
		c := clock.GetSleepCount()
		if c == 1 {
			break
		}
		if c != 1 && try == 5 {
			t.Fatalf("Retry=5 times to wait for timer to be ready")
		}
		time.Sleep(time.Millisecond)
	}

	clock.Forward(delta)
	wg.Wait()
}

func TestTimerStop(t *testing.T) {
	refT, _ := time.Parse(time.RFC3339, "2022-12-06T09:00:00Z")
	clock := NewClock(refT)
	timer := clock.NewTimer(42 * time.Second)

	done := make(chan struct{}, 1)
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

	// Wait for timer to be ready (retry=5)
	for try := 1; ; try++ {
		c := clock.GetSleepCount()
		if c == 1 {
			break
		}
		if c != 1 && try == 5 {
			t.Fatalf("Retry=5 times to wait for timer to be ready")
		}
		time.Sleep(time.Millisecond)
	}

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
		defer func() { done <- struct{}{} }()
		err := clock.SleepWithContext(ctx, 42*time.Second)
		if err == nil {
			t.Errorf("Expected Canceled error here")
			return
		}
		now := clock.Now()
		if !now.Equal(refT) {
			t.Errorf("SleepWithContext must return with %q=%q", now, refT)
		}
	}()

	// Wait for sleeper to be ready (retry=5)
	for try := 1; ; try++ {
		c := clock.GetSleepCount()
		if c == 1 {
			break
		}
		if c != 1 && try == 5 {
			t.Fatalf("Retry=5 times to wait for timer to be ready")
		}
		time.Sleep(time.Millisecond)
	}

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
			clock.Sleep(42 * time.Second)
		}()
	}

	// Wait for sleepers to be ready (retry=50)
	for try := 1; ; try++ {
		c := int(clock.GetSleepCount())
		if c == N {
			break
		}
		if c != N && try == 50 {
			t.Fatalf("Retry=50 times to wait for timer to be ready")
		}
		time.Sleep(time.Millisecond)
	}

	clock.Forward(1 * time.Second)
	clock.Forward(20 * time.Second)
	clock.Forward(20 * time.Second)
	clock.Forward(1 * time.Second) // Total = 42 secs

	wg.Wait()
}
