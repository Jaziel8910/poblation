package ai

import (
	"testing"

	"github.com/user/poblation/internal/entities"
)

func TestRelationshipTypePrefersLoverOverFriend(t *testing.T) {
	poble := entities.NewPoble("noah", "Noah", 29, entities.Male)
	relationship := entities.NewRelationship("kira", entities.RelationshipFriend)
	relationship.Trust = 84
	relationship.Attraction = 86
	relationship.Respect = 72
	relationship.Affection = 80
	poble.Relationships["kira"] = relationship

	manager := NewRelationshipManager(&poble)
	got := manager.GetRelationshipType("kira")
	if got != entities.RelationshipLover {
		t.Fatalf("expected lover to outrank friend, got %s", got)
	}
}

func TestRelationshipTypeFindsSpecialDrama(t *testing.T) {
	poble := entities.NewPoble("noah", "Noah", 29, entities.Male)
	relationship := entities.NewRelationship("kira", entities.RelationshipAcquaintance)
	relationship.Attraction = 45
	relationship.Resentment = 72
	poble.Relationships["kira"] = relationship

	manager := NewRelationshipManager(&poble)
	if got := manager.GetRelationshipType("kira"); got != entities.RelationshipToxicAttraction {
		t.Fatalf("expected toxic attraction, got %s", got)
	}

	relationship.Resentment = 91
	poble.Relationships["kira"] = relationship
	if got := manager.GetRelationshipType("kira"); got != entities.RelationshipNemesis {
		t.Fatalf("expected nemesis, got %s", got)
	}
}

func TestRelationshipEventAmplifiesThirdPartyNegativesWithJealousy(t *testing.T) {
	poble := entities.NewPoble("noah", "Noah", 29, entities.Male)
	poble.Personality.Jealousy = 100
	poble.Relationships["kira"] = entities.NewRelationship("kira", entities.RelationshipAcquaintance)

	manager := NewRelationshipManager(&poble)
	manager.UpdateRelationship("kira", RelationshipEvent{
		Type:         RelationshipEventBetrayal,
		TrustDelta:   -10,
		ThirdPartyID: "eli",
		Time:         entities.NewGameTime(1, 10, 0),
		Tags:         []string{"third_party"},
	})

	relationship, ok := manager.GetRelationship("kira")
	if !ok {
		t.Fatal("expected relationship to exist")
	}
	if relationship.Trust >= 40 {
		t.Fatalf("expected jealousy to amplify trust loss below 40, got %.2f", relationship.Trust)
	}
	if relationship.Resentment <= 0 {
		t.Fatalf("expected inverse resentment increase, got %.2f", relationship.Resentment)
	}
}

func TestDetectRelationshipEventFindsInfidelityDiscovery(t *testing.T) {
	poble := entities.NewPoble("noah", "Noah", 29, entities.Male)
	relationship := entities.NewRelationship("kira", entities.RelationshipLover)
	relationship.Tags = []string{"discovered_infidelity"}
	poble.Relationships["kira"] = relationship

	manager := NewRelationshipManager(&poble)
	event := manager.DetectRelationshipEvent(entities.NewGameTime(4, 12, 0))
	if event == nil {
		t.Fatal("expected infidelity discovery event")
	}
	if event.Type != RelationshipEventInfidelityDiscovered || event.TrustDelta > -60 {
		t.Fatalf("expected trust collapse infidelity event, got %#v", event)
	}
}
