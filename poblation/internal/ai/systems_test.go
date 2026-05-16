package ai

import (
	"math/rand"
	"testing"

	"github.com/user/poblation/internal/entities"
)

// --- Teaching System Tests ---

type mockTeachingWorld struct {
	pobles []*entities.Poble
}

func (m *mockTeachingWorld) GetAllPobles() []*entities.Poble { return m.pobles }
func (m *mockTeachingWorld) GetPoblCount() int               { return len(m.pobles) }

func TestTeachingSystem_SageFindsOpportunity(t *testing.T) {
	teacher := &entities.Poble{
		ID:        "teacher_1",
		Archetype: entities.ArchetypeSage,
		Age:       35,
		IsAlive:   true,
		Personality: entities.Personality{
			Openness:          80,
			Conscientiousness: 70,
		},
		Needs:         entities.Needs{Purpose: 70},
		Relationships: map[string]entities.Relationship{},
	}
	student := &entities.Poble{
		ID:        "student_1",
		Age:       20,
		IsAlive:   true,
		Personality: entities.Personality{Openness: 60},
	}
	teacher.Relationships["student_1"] = entities.Relationship{
		TargetID: "student_1",
		Trust:    60,
		Respect:  50,
	}

	world := &mockTeachingWorld{pobles: []*entities.Poble{teacher, student}}
	rng := rand.New(rand.NewSource(42))
	ts := NewTeachingSystem(teacher, world, rng)
	result := ts.FindTeachingOpportunity()
	if result == nil {
		t.Fatal("sage should find teaching opportunity")
	}
	if result.TeacherID != "teacher_1" || result.StudentID != "student_1" {
		t.Errorf("wrong IDs: teacher=%s student=%s", result.TeacherID, result.StudentID)
	}
	if result.Skill == "" {
		t.Error("teaching result should have a skill")
	}
}

func TestTeachingSystem_NoStudentsAvailable(t *testing.T) {
	teacher := &entities.Poble{
		ID:            "teacher_1",
		Archetype:     entities.ArchetypeSage,
		Age:           35,
		IsAlive:       true,
		Personality:   entities.Personality{Openness: 80, Conscientiousness: 70},
		Needs:         entities.Needs{Purpose: 70},
		Relationships: map[string]entities.Relationship{},
	}
	world := &mockTeachingWorld{pobles: []*entities.Poble{teacher}}
	rng := rand.New(rand.NewSource(42))
	ts := NewTeachingSystem(teacher, world, rng)
	result := ts.FindTeachingOpportunity()
	if result != nil {
		t.Error("should not find opportunity without students")
	}
}

func TestTeachingSystem_NonTeacherArchetype(t *testing.T) {
	drifter := &entities.Poble{
		ID:          "drifter_1",
		Archetype:   entities.ArchetypeDrifter,
		Age:         30,
		IsAlive:     true,
		Personality: entities.Personality{Openness: 40, Conscientiousness: 30},
		Needs:       entities.Needs{Purpose: 40},
	}
	world := &mockTeachingWorld{pobles: []*entities.Poble{drifter}}
	rng := rand.New(rand.NewSource(42))
	ts := NewTeachingSystem(drifter, world, rng)
	result := ts.FindTeachingOpportunity()
	if result != nil {
		t.Error("drifter with low stats should not be a teacher")
	}
}

// --- Defense Mechanism Tests ---

func TestCheckDefenseMechanism_ActivatesAboveThreshold(t *testing.T) {
	poble := &entities.Poble{
		Defense: entities.DefenseMechanism{
			Primary:   entities.DefenseHumor,
			Secondary: entities.DefenseRepression,
			Threshold: 50,
			Strength:  60,
		},
	}
	result := CheckDefenseMechanism(poble, 70)
	if !result.Activated {
		t.Error("defense should activate when event severity > threshold")
	}
	if result.Mechanism != entities.DefenseHumor {
		t.Errorf("expected HUMOR, got %s", result.Mechanism)
	}
	if result.ThoughtOverride == "" {
		t.Error("defense should produce thought override")
	}
}

func TestCheckDefenseMechanism_DoesNotActivateBelowThreshold(t *testing.T) {
	poble := &entities.Poble{
		Defense: entities.DefenseMechanism{
			Primary:   entities.DefenseDenial,
			Secondary: entities.DefenseRationalization,
			Threshold: 60,
			Strength:  50,
		},
	}
	result := CheckDefenseMechanism(poble, 40)
	if result.Activated {
		t.Error("defense should not activate when event severity < threshold")
	}
}

func TestCheckDefenseMechanism_NilPoble(t *testing.T) {
	result := CheckDefenseMechanism(nil, 80)
	if result.Activated {
		t.Error("nil poble should not activate defense")
	}
}
