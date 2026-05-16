package world

import (
	"testing"

	"github.com/user/poblation/internal/entities"
)

func TestNewWorldSeedsEraZeroLocations(t *testing.T) {
	gameWorld := NewWorld(42)

	if len(gameWorld.Locations) < 4 {
		t.Fatalf("expected era zero locations, got %d", len(gameWorld.Locations))
	}
	if !gameWorld.hasLocationType(LocationCampfire) {
		t.Fatalf("expected campfire location to exist")
	}
	if !gameWorld.hasLocationType(LocationLeafShelter) {
		t.Fatalf("expected leaf shelter location to exist")
	}
}

func TestMovePobleUpdatesOccupancyAndDrama(t *testing.T) {
	gameWorld := NewWorld(99)
	first := entities.NewPoble("a", "A", 24, entities.Female)
	second := entities.NewPoble("b", "B", 25, entities.Male)
	firstRel := entities.NewRelationship(second.ID, entities.RelationshipComplicated)
	firstRel.Attraction = 82
	firstRel.Resentment = 61
	first.Relationships[second.ID] = firstRel
	secondRel := entities.NewRelationship(first.ID, entities.RelationshipComplicated)
	secondRel.Attraction = 77
	secondRel.Resentment = 64
	second.Relationships[first.ID] = secondRel

	from := Location{ID: "spawn_a", IslandID: "island_0", X: 1, Y: 1}
	to := Location{ID: "spawn_b", IslandID: "island_0", X: 2, Y: 2}

	if !gameWorld.AddPoble(&first, from) {
		t.Fatalf("failed to add first poble")
	}
	if !gameWorld.AddPoble(&second, to) {
		t.Fatalf("failed to add second poble")
	}

	targetID := ""
	for _, loc := range gameWorld.Locations {
		if loc != nil && loc.Type == LocationCampfire {
			targetID = loc.ID
			break
		}
	}
	if targetID == "" {
		t.Fatalf("missing campfire target location")
	}

	gameWorld.MovePoble(first.ID, from.ID, targetID)
	gameWorld.MovePoble(second.ID, to.ID, targetID)

	target, ok := gameWorld.locationByID(targetID)
	if !ok {
		t.Fatalf("target location missing after move")
	}
	if len(target.CurrentOccupants) != 2 {
		t.Fatalf("expected 2 occupants at target, got %d", len(target.CurrentOccupants))
	}
	if score := gameWorld.GetDramaScore(targetID); score <= 0 {
		t.Fatalf("expected positive drama score, got %d", score)
	}
}
