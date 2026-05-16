package world

import (
	"testing"

	"github.com/user/poblation/internal/entities"
)

func TestNewWorldCreatesOriginAndLockedFutureIslands(t *testing.T) {
	world := NewWorld(7)

	origin := world.GetIsland("island_0")
	if origin == nil {
		t.Fatal("expected origin island")
	}
	if origin.Name != "El Origen" {
		t.Fatalf("expected El Origen, got %q", origin.Name)
	}
	if origin.Size.Width != 60 || origin.Size.Height != 40 {
		t.Fatalf("expected 60x40 origin, got %+v", origin.Size)
	}
	if !origin.IsDiscovered {
		t.Fatal("expected origin discovered")
	}

	for i := 1; i <= 4; i++ {
		island := world.GetIsland("island_" + string(rune('0'+i)))
		if island == nil {
			t.Fatalf("expected island_%d", i)
		}
		if island.IsDiscovered {
			t.Fatalf("expected island_%d locked", i)
		}
	}
}

func TestDiscoverIslandRequiresNavigation(t *testing.T) {
	world := NewWorld(3)

	if island, ok := world.DiscoverIsland("explorer"); ok || island != nil {
		t.Fatal("expected discovery to fail without navigation")
	}

	world.TechTree.Unlocked[techNavigation] = true
	world.Era = entities.EraFour

	discovered := false
	for i := 0; i < 10; i++ {
		if island, ok := world.DiscoverIsland("explorer"); ok {
			discovered = true
			if island == nil || !island.IsDiscovered {
				t.Fatal("expected discovered island to be marked discovered")
			}
			break
		}
	}

	if !discovered {
		t.Fatal("expected discovery eventually with navigation and high era")
	}
}

func TestPopulationAndLocationLookup(t *testing.T) {
	world := NewWorld(11)
	poble := entities.NewPoble("p1", "Ami", 24, entities.Female)
	location := Location{IslandID: "island_0", BuildingID: "island_0_home_0", X: 3, Y: 4}

	if !world.AddPoble(&poble, location) {
		t.Fatal("expected AddPoble to succeed")
	}
	if world.GetPopulation() != 1 {
		t.Fatalf("expected population 1, got %d", world.GetPopulation())
	}
	if got := world.GetPoblAt(location); got == nil || got.ID != "p1" {
		t.Fatalf("expected to find p1 at location, got %+v", got)
	}
}
