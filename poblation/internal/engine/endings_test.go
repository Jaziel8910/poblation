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

func TestPlagueEndingBeatsGenericWarCollapse(t *testing.T) {
	gameWorld := world.NewWorld(91)
	gameWorld.EventHistory = append(gameWorld.EventHistory, ai.GameEvent{
		ID:          "plague_final",
		Type:        ai.GameEventDeath,
		Time:        entities.NewGameTime(50, 0, 0),
		Description: "plague took the last breath from the settlement",
		Tags:        []string{"plague", "illness"},
	})

	ending := CheckEndingConditions(gameWorld)
	if ending == nil || ending.Type != END_PLAGUE {
		t.Fatalf("expected plague ending, got %+v", ending)
	}
}

func TestMonopolyEndingDetectsCapturedEconomy(t *testing.T) {
	gameWorld := world.NewWorld(92)
	gameWorld.Era = entities.EraTwo
	gameWorld.TechTree.Unlocked[world.TechCurrency] = true
	for i := 0; i < 5; i++ {
		poble := entities.NewPoble("m"+string(rune('a'+i)), "Market", 30, entities.Female)
		poble.IsAlive = true
		poble.Money = 10
		if i == 0 {
			poble.Money = 180
		}
		gameWorld.AddPoble(&poble, world.Location{IslandID: "island_0"})
	}

	ending := CheckEndingConditions(gameWorld)
	if ending == nil || ending.Type != END_MONOPOLY {
		t.Fatalf("expected monopoly ending, got %+v", ending)
	}
}

func TestQuarantineEndingDetectsLivingHealthCollapse(t *testing.T) {
	gameWorld := world.NewWorld(93)
	for i := 0; i < 5; i++ {
		poble := entities.NewPoble("q"+string(rune('a'+i)), "Sick", 30, entities.Female)
		poble.IsAlive = true
		poble.Health.HP = 20
		poble.Health.Conditions = []entities.ConditionID{entities.ConditionSick}
		gameWorld.AddPoble(&poble, world.Location{IslandID: "island_0"})
	}
	gameWorld.EventHistory = append(gameWorld.EventHistory, ai.GameEvent{
		ID:          "sti_crisis",
		Type:        ai.GameEventSocialNegative,
		Time:        entities.NewGameTime(20, 0, 0),
		Description: "sti and illness spread through the settlement",
		Tags:        []string{"sti", "illness"},
	})

	ending := CheckEndingConditions(gameWorld)
	if ending == nil || ending.Type != END_QUARANTINE {
		t.Fatalf("expected quarantine ending, got %+v", ending)
	}
}

func TestScandalEndingDetectsSecretEncounterCollapse(t *testing.T) {
	gameWorld := world.NewWorld(94)
	for i := 0; i < 6; i++ {
		poble := entities.NewPoble("s"+string(rune('a'+i)), "Rumor", 30, entities.Female)
		poble.IsAlive = true
		gameWorld.AddPoble(&poble, world.Location{IslandID: "island_0"})
	}
	for i := 0; i < 4; i++ {
		gameWorld.EventHistory = append(gameWorld.EventHistory, ai.GameEvent{
			ID:          "affair_secret_" + string(rune('a'+i)),
			Type:        ai.GameEventSocialNegative,
			Time:        entities.NewGameTime(10+i, 0, 0),
			Description: "secret encounter became a public revelation",
			Tags:        []string{"affair", "secret", "revelation"},
		})
	}

	ending := CheckEndingConditions(gameWorld)
	if ending == nil || ending.Type != END_SCANDAL {
		t.Fatalf("expected scandal ending, got %+v", ending)
	}
}
