package engine

import (
	"math/rand"
	"strings"
	"testing"

	"github.com/user/poblation/internal/entities"
	"github.com/user/poblation/internal/world"
)

func TestConsoleHelpIncludesSpanishFriendlyListing(t *testing.T) {
	console := NewConsoleSystem(world.NewWorld(11), NewTimeEngine(), rand.New(rand.NewSource(11)))

	result := console.Execute("ayuda")

	if !strings.Contains(result.Feedback, "god mode") {
		t.Fatalf("expected help to include god mode, got %q", result.Feedback)
	}
	if !strings.Contains(result.Feedback, "relations") {
		t.Fatalf("expected help to include relations, got %q", result.Feedback)
	}
}

func TestConsoleKillMarksPobleDead(t *testing.T) {
	gameWorld, clock, alice, _ := seededConsoleWorld(t)
	console := NewConsoleSystem(gameWorld, clock, rand.New(rand.NewSource(22)))

	result := console.Execute("kill Alice murder")

	if !alice.IsAlive {
		if result.Event == nil || !strings.Contains(result.Feedback, "Alice") {
			t.Fatalf("expected kill feedback for Alice, got %+v", result)
		}
		return
	}
	t.Fatalf("expected Alice to die")
}

func TestConsoleSpeedPauseAndResume(t *testing.T) {
	gameWorld, clock, _, _ := seededConsoleWorld(t)
	console := NewConsoleSystem(gameWorld, clock, rand.New(rand.NewSource(33)))

	console.Execute("velocidad 2")
	if clock.Speed != 2 {
		t.Fatalf("expected speed 2, got %v", clock.Speed)
	}

	console.Execute("pausa")
	if !clock.IsPaused {
		t.Fatal("expected paused clock")
	}

	console.Execute("continuar")
	if clock.IsPaused {
		t.Fatal("expected resumed clock")
	}
}

func TestConsoleTechUnlocksImmediately(t *testing.T) {
	gameWorld, clock, _, _ := seededConsoleWorld(t)
	console := NewConsoleSystem(gameWorld, clock, rand.New(rand.NewSource(44)))

	console.Execute("tech navigation")

	if !gameWorld.TechTree.Unlocked[world.TechNavigation] {
		t.Fatal("expected NAVIGATION unlocked")
	}
}

func TestConsoleSpawnAddsPopulation(t *testing.T) {
	gameWorld, clock, _, _ := seededConsoleWorld(t)
	console := NewConsoleSystem(gameWorld, clock, rand.New(rand.NewSource(55)))
	before := gameWorld.GetPopulation()

	result := console.Execute("spawn sage")

	if gameWorld.GetPopulation() != before+1 {
		t.Fatalf("expected population %d, got %d", before+1, gameWorld.GetPopulation())
	}
	if !strings.Contains(result.Feedback, "Arquetipo") {
		t.Fatalf("unexpected spawn feedback: %q", result.Feedback)
	}
}

func TestConsoleBabyRequiresBiologicalViability(t *testing.T) {
	gameWorld, clock, alice, bob := seededConsoleWorld(t)
	console := NewConsoleSystem(gameWorld, clock, rand.New(rand.NewSource(66)))

	result := console.Execute("baby Alice Bob")

	if result.Event == nil {
		t.Fatalf("expected pregnancy event, got %+v", result)
	}
	if !hasPregnancyCondition(alice) && !hasPregnancyCondition(bob) {
		t.Fatal("expected one parent to become pregnant")
	}
}

func seededConsoleWorld(t *testing.T) (*world.World, *TimeEngine, *entities.Poble, *entities.Poble) {
	t.Helper()

	gameWorld := world.NewWorld(7)
	clock := NewTimeEngine()

	female := entities.Female
	male := entities.Male
	lover := entities.ArchetypeLover
	warrior := entities.ArchetypeWarrior

	alice, err := entities.GeneratePople(entities.PoblConfig{
		Name:      "Alice",
		Sex:       &female,
		Archetype: &lover,
		AgeRange:  [2]int{24, 24},
	}, rand.New(rand.NewSource(1)))
	if err != nil {
		t.Fatalf("generate Alice: %v", err)
	}
	bob, err := entities.GeneratePople(entities.PoblConfig{
		Name:      "Bob",
		Sex:       &male,
		Archetype: &warrior,
		AgeRange:  [2]int{28, 28},
	}, rand.New(rand.NewSource(2)))
	if err != nil {
		t.Fatalf("generate Bob: %v", err)
	}

	alice.Health.Fertility = 1
	bob.Health.Fertility = 1
	gameWorld.AddPoble(alice, world.Location{IslandID: "island_0"})
	gameWorld.AddPoble(bob, world.Location{IslandID: "island_0"})
	return gameWorld, clock, alice, bob
}

func hasPregnancyCondition(poble *entities.Poble) bool {
	if poble == nil {
		return false
	}
	for _, condition := range poble.Health.Conditions {
		if condition == entities.ConditionPregnant {
			return true
		}
	}
	return false
}
