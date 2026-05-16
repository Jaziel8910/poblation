package ai

import (
	"testing"

	"github.com/user/poblation/internal/entities"
)

func TestRulerPrioritizesPowerOverHunger(t *testing.T) {
	poble := entities.NewPoble("ruler", "Ruler", 30, entities.Female)
	poble.Archetype = entities.ArchetypeRuler
	poble.Personality.Conscientiousness = 80
	poble.Personality.Ambition = 92
	poble.Needs.Hunger = 82
	poble.Needs.Power = 74

	system := NewNeedsSystem(&poble)
	urgent := system.GetUrgentNeeds()
	if len(urgent) == 0 {
		t.Fatal("expected urgent needs")
	}
	if urgent[0].Need != NeedPower {
		t.Fatalf("expected power first, got %s", urgent[0].Need)
	}
}

func TestNeedsUpdateUsesContext(t *testing.T) {
	poble := entities.NewPoble("care", "Care", 28, entities.Male)
	poble.Personality.Horniness = 80
	system := NewNeedsSystem(&poble)

	system.Update(4, WorldContext{
		IsSleeping:                 false,
		ConflictActive:             true,
		IsAlone:                    true,
		PositiveSocialInteractions: 0,
		NegativeSocialInteractions: 1,
		EnvironmentalSexualStimuli: 50,
		HoursSinceSex:              48,
		HasControl:                 false,
		ActiveGoals:                0,
	})

	if poble.Needs.Hunger <= 0 || poble.Needs.Thirst <= 0 || poble.Needs.Sleep <= 0 {
		t.Fatalf("expected physical needs to rise, got %+v", poble.Needs)
	}
	if poble.Needs.Safety <= 0 {
		t.Fatalf("expected safety need to rise in conflict, got %.2f", poble.Needs.Safety)
	}
	if poble.Needs.Belonging <= 50 {
		t.Fatalf("expected belonging need to rise while alone, got %.2f", poble.Needs.Belonging)
	}
	if poble.Needs.Sex <= 0 {
		t.Fatalf("expected sex need to rise, got %.2f", poble.Needs.Sex)
	}
	if poble.Needs.Purpose <= 50 {
		t.Fatalf("expected purpose need to rise without goals, got %.2f", poble.Needs.Purpose)
	}
}
