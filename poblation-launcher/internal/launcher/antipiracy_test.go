package launcher

import (
	"math/rand"
	"strings"
	"testing"
	"time"
)

func TestRandomAntiPiracyProbabilityGate(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	for i := 0; i < 1000; i++ {
		_ = RandomAntiPiracyTriggered(rng)
	}
}

func TestAntiPiracySequenceUsesSaveData(t *testing.T) {
	save := SaveSummary{
		WorldName:  "Nooblandia",
		Day:        847,
		Population: 234,
		LastPlayed: time.Date(2026, 5, 15, 12, 0, 0, 0, time.UTC),
	}

	sequence := BuildAntiPiracySequence(APRandom, save)
	joined := strings.Join(sequence.Lines, "\n")
	for _, want := range []string{"Nooblandia", "847", "234"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("sequence does not include %q: %s", want, joined)
		}
	}
}
