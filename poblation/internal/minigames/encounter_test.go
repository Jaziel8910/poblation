package minigames

import (
	"strings"
	"testing"

	"github.com/user/poblation/internal/config"
	"github.com/user/poblation/internal/entities"
	"github.com/user/poblation/internal/world"
)

func TestGenerateAftermathRestrictedUsesSafeSummary(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	first := entities.NewPoble("a", "Ada", 26, entities.Female)
	second := entities.NewPoble("b", "Bruno", 27, entities.Male)
	firstRel := entities.NewRelationship(second.ID, entities.RelationshipLover)
	firstRel.Trust = 78
	firstRel.Attraction = 81
	first.Relationships[second.ID] = firstRel
	secondRel := entities.NewRelationship(first.ID, entities.RelationshipLover)
	secondRel.Trust = 75
	secondRel.Attraction = 79
	second.Relationships[first.ID] = secondRel

	ctx := EncounterContext{
		A:            &first,
		B:            &second,
		StartedAt:    entities.NewGameTime(3, 22, 15),
		Relationship: first.Relationships[second.ID],
		Location:     &world.Location{Name: "Cabana Roja", Type: world.LocationWoodenCabin, PrivacyLevel: 82},
	}
	ctx.Power = derivePowerDynamic(&first, &second, ctx.Relationship)

	aftermath := GenerateAftermath(ctx, ResolvePreferenceMatch(&first, &second))
	if aftermath.VisibleSummary != "[Ada y Bruno pasaron tiempo juntos]" {
		t.Fatalf("expected restricted summary, got %q", aftermath.VisibleSummary)
	}
	if aftermath.InternalSummary == aftermath.VisibleSummary {
		t.Fatalf("internal summary should stay distinct in restricted mode")
	}
}

func TestResolvePreferenceMatchFindsSharedPreferenceInFullMode(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	if err := config.EnableAdultContent(); err != nil {
		t.Fatalf("enable adult content: %v", err)
	}

	first := entities.NewPoble("a", "Ada", 26, entities.Female)
	second := entities.NewPoble("b", "Bruno", 27, entities.Male)
	first.HiddenPrefs.Preferences = []entities.HiddenPreference{{
		Name:         "morder el labio",
		Category:     entities.PrefPhysical,
		Intensity:    71,
		IsDiscovered: false,
	}}
	first.HiddenPrefs.DominantCategory = entities.PrefPhysical
	second.HiddenPrefs.Preferences = []entities.HiddenPreference{{
		Name:         "morder el labio",
		Category:     entities.PrefPhysical,
		Intensity:    68,
		IsDiscovered: false,
	}}
	second.HiddenPrefs.DominantCategory = entities.PrefPhysical

	match := ResolvePreferenceMatch(&first, &second)
	if len(match.SharedNames) == 0 {
		t.Fatalf("expected shared preference match")
	}
	if len(match.NewlyDiscovered) == 0 {
		t.Fatalf("expected newly discovered preference")
	}
	if match.CompatibilityScore <= 50 {
		t.Fatalf("expected positive compatibility score, got %d", match.CompatibilityScore)
	}
}

func TestDetermineEncounterTypeReadsContext(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	first := entities.NewPoble("a", "Ada", 29, entities.Female)
	second := entities.NewPoble("b", "Bruno", 31, entities.Male)
	firstRel := entities.NewRelationship(second.ID, entities.RelationshipEnemy)
	firstRel.Resentment = 91
	firstRel.Attraction = 48
	first.Relationships[second.ID] = firstRel
	ctx := EncounterContext{
		A:            &first,
		B:            &second,
		StartedAt:    entities.NewGameTime(4, 23, 0),
		TriggerEvent: "fight aftermath",
		Relationship: first.Relationships[second.ID],
		Location:     &world.Location{Name: "Bar Hundido", Type: world.LocationDiveBar, PrivacyLevel: 20},
	}
	ctx.Power = derivePowerDynamic(&first, &second, ctx.Relationship)

	got := DetermineEncounterType(ctx)
	if got != EncounterAngry && !strings.EqualFold(got.String(), "ANGRY") {
		t.Fatalf("expected angry encounter from hostile context, got %s", got)
	}
}

func TestApplyEncounterAftermathRecordsHealthPregnancyAndSTIEvents(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	gameWorld := world.NewWorld(88)
	first := entities.NewPoble("a", "Ada", 26, entities.Female)
	second := entities.NewPoble("b", "Bruno", 27, entities.Male)
	first.Health.Fertility = 1
	second.Health.Fertility = 1
	second.Health.STIs = []entities.STIType{entities.STISyphilis}
	first.Health.HP = 12
	second.Health.HP = 80
	gameWorld.AddPoble(&first, world.Location{IslandID: "island_0"})
	gameWorld.AddPoble(&second, world.Location{IslandID: "island_0"})

	ctx := EncounterContext{
		A:            gameWorld.GetPoble("a"),
		B:            gameWorld.GetPoble("b"),
		StartedAt:    entities.NewGameTime(7, 23, 0),
		Relationship: entities.NewRelationship("b", entities.RelationshipLover),
	}
	ctx.Power = derivePowerDynamic(ctx.A, ctx.B, ctx.Relationship)
	aftermath := GenerateAftermath(ctx, ResolvePreferenceMatch(ctx.A, ctx.B))
	aftermath.PregnancyTriggered = true
	aftermath.STITransmitted = entities.STISyphilis
	aftermath.HealthFallout.HPDelta = [2]int{-9, -2}
	aftermath.HealthFallout.Conditions[0] = []entities.ConditionID{entities.ConditionExhausted}

	ApplyEncounterAftermath(aftermath, gameWorld)

	ada := gameWorld.GetPoble("a")
	if ada == nil {
		t.Fatal("expected Ada to survive this fallout")
	}
	if !hasEncounterCondition(ada, entities.ConditionPregnant) {
		t.Fatal("expected pregnancy condition from encounter")
	}
	if !hasEncounterSTI(ada, entities.STISyphilis) {
		t.Fatal("expected STI transmission from encounter")
	}
	if len(gameWorld.EventHistory) < 3 {
		t.Fatalf("expected encounter events in history, got %d", len(gameWorld.EventHistory))
	}
}

func hasEncounterCondition(poble *entities.Poble, condition entities.ConditionID) bool {
	for _, current := range poble.Health.Conditions {
		if current == condition {
			return true
		}
	}
	return false
}

func hasEncounterSTI(poble *entities.Poble, sti entities.STIType) bool {
	for _, current := range poble.Health.STIs {
		if current == sti {
			return true
		}
	}
	return false
}
