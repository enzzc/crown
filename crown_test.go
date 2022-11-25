package horloger

import (
	"sync"
	"sync/atomic"
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

	var returned int32 = 0
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		clock.Sleep(9 * time.Second)
		atomic.AddInt32(&returned, 1)
	}()

	clock.Tick()
	clock.Forward(2 * time.Second)
	clock.Forward(2 * time.Second)
	if atomic.LoadInt32(&returned) == 1 {
		t.Errorf("Sleep returned too early after 4 secs")
	}
	clock.Forward(2 * time.Second)
	clock.Forward(2 * time.Second)
	clock.Forward(2 * time.Second)
	if atomic.LoadInt32(&returned) == 0 {
		t.Errorf("Sleep did not return after 10 secs")
		return
	}
	wg.Wait()
}
