package crown

import (
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
		t.Log("Sleeper 2 returned.", clock.Now())
	case <-time.After(1 * time.Second):
		t.Fatalf("Sleeper 2 did not return after clock has reached t+4s. t=%q", clock.Now())
	}
	clock.Forward(1 * time.Second)
	clock.Forward(1 * time.Second) // +6 secs
	select {
	case <-done1:
		t.Log("Sleeper 1 returned.", clock.Now())
	case <-time.After(1 * time.Second):
		t.Fatalf("Sleeper 1 did not return after clock has reached t+6s. t=%q", clock.Now())
	}
}
