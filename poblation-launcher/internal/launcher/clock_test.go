package launcher

import (
	"strings"
	"testing"
	"time"
)

func TestClockAnomalyFirstOccurrenceGetsBenefitOfDoubt(t *testing.T) {
	now := time.Date(2026, 5, 16, 12, 0, 0, 0, time.UTC)
	settings := Settings{
		Clock: ClockMemory{
			LastSeenUTC: now.Add(6 * time.Hour),
		},
	}

	result := CheckClock(settings, SaveSummary{}, now)
	if result.Kind != ClockHardware {
		t.Fatalf("expected hardware anomaly first, got %s", result.Kind)
	}
	if result.Settings.Clock.FirstAnomalyAccepted != true {
		t.Fatalf("expected benefit of doubt to be recorded")
	}
}

func TestClockAnomalyMultipleSignalsBecomeIntentional(t *testing.T) {
	now := time.Date(2026, 5, 16, 12, 0, 0, 0, time.UTC)
	settings := Settings{
		Clock: ClockMemory{
			LastSeenUTC:          now.Add(6 * time.Hour),
			FirstAnomalyAccepted: true,
			IntentionalSignals:   1,
		},
	}
	save := SaveSummary{LastPlayed: now.Add(24 * time.Hour)}

	result := CheckClock(settings, save, now)
	if result.Kind != ClockIntentional {
		t.Fatalf("expected intentional anomaly, got %s with %#v", result.Kind, result.Signals)
	}
	if !strings.Contains(result.Title, "RAM EATER") {
		t.Fatalf("expected RAM EATER title, got %q", result.Title)
	}
}
