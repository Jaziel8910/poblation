package ai

import (
	"math/rand"

	"github.com/user/poblation/internal/entities"
)

// TeachingSystem handles knowledge transfer between Pobles.
type TeachingSystem struct {
	poble *entities.Poble
	world World
	rng   *rand.Rand
}

// NewTeachingSystem binds a teaching system to one Poble.
func NewTeachingSystem(poble *entities.Poble, world World, rng *rand.Rand) *TeachingSystem {
	if rng == nil {
		rng = rand.New(rand.NewSource(1))
	}
	return &TeachingSystem{poble: poble, world: world, rng: rng}
}

// FindTeachingOpportunity checks whether this Poble should teach someone.
// SAGE and CARETAKER archetypes actively seek these.
func (t *TeachingSystem) FindTeachingOpportunity() *TeachingResult {
	if t == nil || t.poble == nil || t.world == nil {
		return nil
	}

	if !t.isTeacher() {
		return nil
	}

	candidates := t.findStudents()
	if len(candidates) == 0 {
		return nil
	}

	student := candidates[0]
	skill := t.chooseSkill(student)
	effectiveness := t.calculateEffectiveness(student)

	return &TeachingResult{
		TeacherID:     t.poble.ID,
		StudentID:     student.ID,
		Skill:         skill,
		Effectiveness: effectiveness,
		Tags:          t.teachingTags(),
	}
}

// TeachingResult stores the outcome of a teaching opportunity.
type TeachingResult struct {
	TeacherID     string   `json:"teacher_id"`
	StudentID     string   `json:"student_id"`
	Skill         string   `json:"skill"`
	Effectiveness float32  `json:"effectiveness"`
	Tags          []string `json:"tags"`
}

func (t *TeachingSystem) isTeacher() bool {
	switch t.poble.Archetype {
	case entities.ArchetypeSage, entities.ArchetypeCaretaker:
		return true
	default:
		// Non-archetype teaching: high openness + high conscientiousness + purpose need.
		return t.poble.Personality.Openness > 70 &&
			t.poble.Personality.Conscientiousness > 60 &&
			t.poble.Needs.Purpose > 65
	}
}

func (t *TeachingSystem) findStudents() []*entities.Poble {
	candidates := make([]*entities.Poble, 0, 4)
	for _, other := range t.world.GetAllPobles() {
		if other == nil || !other.IsAlive || other.ID == t.poble.ID {
			continue
		}
		if other.Age > t.poble.Age {
			continue
		}
		relationship, ok := t.poble.Relationships[other.ID]
		if !ok {
			continue
		}
		if relationship.Trust < 35 || relationship.Resentment > 65 {
			continue
		}
		candidates = append(candidates, other)
	}
	return candidates
}

func (t *TeachingSystem) chooseSkill(student *entities.Poble) string {
	skills := []string{
		"fire management", "water purification", "basic construction",
		"conflict resolution", "medicinal plants", "tracking",
		"food preservation", "storytelling", "tool repair",
	}
	return skills[t.rng.Intn(len(skills))]
}

func (t *TeachingSystem) calculateEffectiveness(student *entities.Poble) float32 {
	relationship, ok := t.poble.Relationships[student.ID]
	base := float32(40)
	if ok {
		base += relationship.Trust * 0.2
		base += relationship.Respect * 0.15
		base -= relationship.Resentment * 0.1
	}
	base += t.poble.Personality.Openness * 0.1
	base += student.Personality.Openness * 0.15

	if base < 0 {
		base = 0
	}
	if base > 100 {
		base = 100
	}
	return base
}

func (t *TeachingSystem) teachingTags() []string {
	tags := []string{"teaching"}
	switch t.poble.Archetype {
	case entities.ArchetypeSage:
		tags = append(tags, "archetype:sage", "knowledge")
	case entities.ArchetypeCaretaker:
		tags = append(tags, "archetype:caretaker", "nurture")
	default:
		tags = append(tags, "voluntary")
	}
	return tags
}

// ApplyTeachingResult modifies Poble state based on a teaching event.
func ApplyTeachingResult(result *TeachingResult, teacher, student *entities.Poble) {
	if result == nil || teacher == nil || student == nil {
		return
	}

	// Teaching satisfies teacher's purpose need.
	teacher.Needs.Purpose = clampPercent(teacher.Needs.Purpose - result.Effectiveness*0.2)

	// Student gains esteem.
	student.Needs.Esteem = clampPercent(student.Needs.Esteem - result.Effectiveness*0.15)

	// Both gain belonging.
	teacher.Needs.Belonging = clampPercent(teacher.Needs.Belonging - result.Effectiveness*0.1)
	student.Needs.Belonging = clampPercent(student.Needs.Belonging - result.Effectiveness*0.1)
}
