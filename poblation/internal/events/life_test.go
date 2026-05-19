package events

import (
	"strings"
	"testing"

	"github.com/user/poblation/internal/entities"
	simworld "github.com/user/poblation/internal/world"
)

func TestHandleDeathCreatesInheritanceRumourAndGenerationEnd(t *testing.T) {
	founder := entities.NewPoble("founder", "Noah", 68, entities.Male)
	child := entities.NewPoble("child", "Kira", 18, entities.Female)

	founder.Money = 120
	child.Parents[0] = founder.ID
	child.Relationships[founder.ID] = entities.NewRelationship(founder.ID, entities.RelationshipParent)

	world := testWorld{
		pobles: map[string]*entities.Poble{
			founder.ID: &founder,
			child.ID:   &child,
		},
		state: entities.WorldState{
			Day:         entities.NewGameTime(4, 6, 0),
			TechTree:    entities.NewTechTree(),
			Settlements: []entities.Settlement{},
			Islands:     []entities.Island{},
		},
	}

	event := HandleDeath(&founder, DeathCauseMurder, world)
	if founder.IsAlive {
		t.Fatal("expected founder to be marked dead")
	}
	if child.Money != 120 {
		t.Fatalf("expected inheritance to reach child, got %d", child.Money)
	}
	if !containsChildEvent(event.ChildEvents, "generation_end") {
		t.Fatalf("expected generation_end child event, got %#v", event.ChildEvents)
	}
	if !containsChildEvent(event.ChildEvents, "rumour") {
		t.Fatalf("expected rumour child event, got %#v", event.ChildEvents)
	}
}

func TestNaturalDeathScanUsesFullLifecycleFallout(t *testing.T) {
	founder := entities.NewPoble("founder", "Noah", 91, entities.Male)
	child := entities.NewPoble("child", "Kira", 15, entities.Female)
	founder.Money = 77
	child.Parents[0] = founder.ID

	world := testWorld{
		pobles: map[string]*entities.Poble{
			founder.ID: &founder,
			child.ID:   &child,
		},
		state: entities.WorldState{
			Day:      entities.NewGameTime(10, 8, 0),
			Era:      entities.EraZero,
			TechTree: entities.NewTechTree(),
		},
	}

	generated := checkDeathByAge(&founder, world, world.state, lifecycleRNG(world.state.Day, "force"))
	if len(generated) != 1 {
		t.Fatalf("expected natural death event, got %#v", generated)
	}
	if founder.IsAlive {
		t.Fatal("expected natural death scan to mark founder dead")
	}
	if child.Money != 77 {
		t.Fatalf("expected inheritance from natural death scan, got %d", child.Money)
	}
	if !containsChildEvent(generated[0].ChildEvents, "orphaned_child") {
		t.Fatalf("expected orphan fallout, got %#v", generated[0].ChildEvents)
	}
}

func TestHandleBirthAddsBabyToConcreteWorld(t *testing.T) {
	world := simworld.NewWorld(1)
	mother := entities.NewPoble("mother", "Amina", 24, entities.Female)
	father := entities.NewPoble("father", "Mateo", 27, entities.Male)

	location := simworld.Location{IslandID: "island_0", X: 3, Y: 4}
	if !world.AddPoble(&mother, location) || !world.AddPoble(&father, location) {
		t.Fatal("expected parents to be added to world")
	}

	baby, event := HandleBirth(mother.ID, father.ID, world)
	if baby == nil {
		t.Fatal("expected baby to be created")
	}
	if got := world.GetPoble(baby.ID); got == nil {
		t.Fatal("expected baby to be registered in concrete world")
	}
	if event.Type != EventBirth {
		t.Fatalf("expected birth event, got %s", event.Type)
	}
	if len(mother.Children) == 0 || mother.Children[0] != baby.ID {
		t.Fatalf("expected mother to reference baby, got %#v", mother.Children)
	}
}

func TestHandlePregnancyGuaranteesDramaForThirdPartyCase(t *testing.T) {
	mother := entities.NewPoble("mother", "Lina", 26, entities.Female)
	official := entities.NewPoble("official", "Jonas", 28, entities.Male)
	actual := entities.NewPoble("actual", "Ilya", 29, entities.Male)

	actual.Orientation.Sexual = 0.9

	spouseRel := entities.NewRelationship(official.ID, entities.RelationshipSpouse)
	spouseRel.Trust = 90
	spouseRel.Affection = 80
	spouseRel.Familiarity = 90
	mother.Relationships[official.ID] = spouseRel

	loverRel := entities.NewRelationship(actual.ID, entities.RelationshipLover)
	loverRel.Attraction = 95
	loverRel.Dependency = 75
	loverRel.Affection = 82
	mother.Relationships[actual.ID] = loverRel

	world := testWorld{
		pobles: map[string]*entities.Poble{
			mother.ID:   &mother,
			official.ID: &official,
			actual.ID:   &actual,
		},
		state: entities.WorldState{
			Day:      entities.NewGameTime(2, 5, 0),
			TechTree: entities.NewTechTree(),
		},
	}

	arc := HandlePregnancy(mother.ID, world)
	if !arc.DramaGuaranteed {
		t.Fatalf("expected pregnancy drama guarantee, got %#v", arc)
	}
	if !hasCondition(&mother, entities.ConditionPregnant) {
		t.Fatal("expected mother to become pregnant")
	}
}

func TestHandleSuicideRequiresLowStabilityAndNoSupport(t *testing.T) {
	poble := entities.NewPoble("p1", "Rue", 22, entities.Female)
	poble.Mental.Stability = 5
	poble.Mental.TherapyLevel = 0

	world := testWorld{
		pobles: map[string]*entities.Poble{
			poble.ID: &poble,
		},
		state: entities.WorldState{
			Day:      entities.NewGameTime(7, 10, 0),
			TechTree: entities.NewTechTree(),
		},
	}

	event := HandleSuicide(&poble, "fight:1", world)
	if event.ID == "" {
		t.Fatal("expected suicide event to be created")
	}
	if !containsChildEvent(event.ChildEvents, "suicide_note") {
		t.Fatalf("expected suicide note child event, got %#v", event.ChildEvents)
	}
}

func containsChildEvent(values []string, fragment string) bool {
	for _, value := range values {
		if strings.Contains(value, fragment) {
			return true
		}
	}
	return false
}
