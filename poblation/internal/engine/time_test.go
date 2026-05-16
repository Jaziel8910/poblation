package engine

import (
	"context"
	"testing"
	"time"
)

func TestOneTickAdvancesOneGameHour(t *testing.T) {
	engine := NewTimeEngine()

	engine.AdvanceBy(1)

	current := engine.GetCurrentTime()
	if current.Day != 0 || current.Hour != 1 || current.Minute != 0 {
		t.Fatalf("expected day 0 01:00, got %s", current.String())
	}
}

func TestDayChange(t *testing.T) {
	engine := NewTimeEngine()
	subscription := engine.Subscribe()

	engine.AdvanceBy(24)

	current := engine.GetCurrentTime()
	if current.Day != 1 || current.Hour != 0 || current.Minute != 0 {
		t.Fatalf("expected day 1 00:00, got %s", current.String())
	}

	select {
	case tick := <-subscription:
		if !tick.IsNewDay {
			t.Fatal("expected IsNewDay true")
		}
	case <-time.After(time.Second):
		t.Fatal("expected game tick")
	}
}

func TestPauseResume(t *testing.T) {
	engine := NewTimeEngine()

	engine.Pause()
	engine.AdvanceBy(1)
	if current := engine.GetCurrentTime(); current.Hour != 0 {
		t.Fatalf("expected paused time to stay at 00:00, got %s", current.String())
	}

	engine.Resume()
	engine.AdvanceBy(1)
	if current := engine.GetCurrentTime(); current.Hour != 1 {
		t.Fatalf("expected resumed time to advance to 01:00, got %s", current.String())
	}
}

func TestSubscription(t *testing.T) {
	engine := NewTimeEngine()
	subscription := engine.Subscribe()

	engine.AdvanceBy(1)

	select {
	case tick := <-subscription:
		if tick.DeltaHours != 1 {
			t.Fatalf("expected DeltaHours 1, got %d", tick.DeltaHours)
		}
		if tick.CurrentTime.Hour != 1 {
			t.Fatalf("expected current hour 1, got %d", tick.CurrentTime.Hour)
		}
	case <-time.After(time.Second):
		t.Fatal("expected subscription tick")
	}

	engine.Unsubscribe(subscription)
	if _, ok := <-subscription; ok {
		t.Fatal("expected subscription channel to close")
	}
}

func TestStartStopsWithContext(t *testing.T) {
	engine := NewTimeEngine()
	engine.SetSpeed(4)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	engine.Start(ctx)
	cancel()
	engine.Stop()
}
