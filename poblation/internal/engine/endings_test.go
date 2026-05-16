package engine

import (
	"strings"
	"testing"

	"github.com/user/poblation/internal/ai"
	"github.com/user/poblation/internal/entities"
	"github.com/user/poblation/internal/world"
)

func TestLoveEndingRequiresDeathAndReproductionRejection(t *testing.T) {
	gameWorld := world.NewWorld(7)
	alba := entities.NewPoble("p1", "Alba", 32, entities.Female)
	noe := entities.NewPoble("p2", "Noe", 34, entities.Male)

	if !gameWorld.AddPoble(&alba, world.Location{IslandID: "island_0"}) {
		t.Fatal("expected Alba to be added")
	}
	if !gameWorld.AddPoble(&noe, world.Location{IslandID: "island_0"}) {
		t.Fatal("expected Noe to be added")
	}
	noe.IsAlive = false
	gameWorld.EventHistory = append(gameWorld.EventHistory,
		ai.GameEvent{
			ID:          "death_noe",
			Type:        ai.GameEventDeath,
			Time:        entities.NewGameTime(12, 0, 0),
			Description: "Noe died after the last storm.",
			Tags:        []string{"death"},
			Severity:    0.9,
		},
		ai.GameEvent{
			ID:          "decision_point_reproduction",
			Type:        ai.GameEventGeneric,
			Time:        entities.NewGameTime(13, 0, 0),
			Description: "Alba rejected reproduction after Noe died.",
			Tags:        []string{"decision_point", "no_reproduction"},
			Severity:    0.8,
		},
	)

	ending := CheckEndingConditions(gameWorld)
	if ending == nil {
		t.Fatal("expected love ending")
	}
	if ending.Type != END_LOVE {
		t.Fatalf("expected END_LOVE, got %s", ending.Type)
	}
	if len(ending.NarrativeChapters) < 3 {
		t.Fatalf("expected narrative chapters, got %d", len(ending.NarrativeChapters))
	}
	if !strings.Contains(strings.Join(ending.NarrativeChapters, " "), "Alba") {
		t.Fatalf("expected generated ending to mention the surviving poble")
	}
}

func TestResetEndingUsesStatisticsFromWorld(t *testing.T) {
	gameWorld := world.NewWorld(9)
	first := entities.NewPoble("p1", "Iris", 44, entities.Female)
	second := entities.NewPoble("p2", "Luca", 42, entities.Male)
	third := entities.NewPoble("p3", "Mara", 80, entities.Female)
	third.IsAlive = false
	third.Money = 100

	_ = gameWorld.AddPoble(&first, world.Location{IslandID: "island_0"})
	_ = gameWorld.AddPoble(&second, world.Location{IslandID: "island_0"})
	_ = gameWorld.AddPoble(&third, world.Location{IslandID: "island_0"})
	gameWorld.EventHistory = append(gameWorld.EventHistory, ai.GameEvent{
		ID:          "collapse_final",
		Type:        ai.GameEventThreat,
		Time:        entities.NewGameTime(30, 0, 0),
		Description: "The civilization collapsed and left two survivors.",
		Tags:        []string{"collapse"},
		Severity:    0.95,
	})

	ending := CheckEndingConditions(gameWorld)
	if ending == nil {
		t.Fatal("expected reset ending")
	}
	if ending.Type != END_RESET {
		t.Fatalf("expected END_RESET, got %s", ending.Type)
	}
	if ending.Statistics.TotalPobles != 3 {
		t.Fatalf("expected 3 historical pobles, got %d", ending.Statistics.TotalPobles)
	}
	if ending.Statistics.LongestLivedPople != "Mara" {
		t.Fatalf("expected Mara as longest lived, got %s", ending.Statistics.LongestLivedPople)
	}
}
