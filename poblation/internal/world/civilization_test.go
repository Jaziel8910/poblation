package world

import (
	"testing"

	"github.com/user/poblation/internal/ai"
	"github.com/user/poblation/internal/entities"
)

func TestCheckEraTransitionFromZeroToOne(t *testing.T) {
	world := NewWorld(10)
	addTestPobles(world, 5, entities.ArchetypeInnocent)

	manager := CivilizationManager{}
	transition := manager.CheckEraTransition(world)
	if transition == nil {
		t.Fatal("expected era transition")
	}
	if transition.FromEra != entities.EraZero || transition.ToEra != entities.EraOne {
		t.Fatalf("unexpected transition: %+v", transition)
	}
	if world.Era != entities.EraOne {
		t.Fatalf("expected world era one, got %s", world.Era)
	}
}

func TestCheckEraTransitionFromOneToTwoCanEmergeBasicBuildings(t *testing.T) {
	world := NewWorld(11)
	world.Era = entities.EraOne
	addTestPobles(world, 21, entities.ArchetypeCaretaker)

	transition := (CivilizationManager{}).CheckEraTransition(world)
	if transition == nil || transition.ToEra != entities.EraTwo {
		t.Fatalf("expected era two transition, got %+v", transition)
	}
	if !hasBasicBuildingSet(world) {
		t.Fatal("expected basic building set to emerge before transition")
	}
}

func TestDetermineGovernmentTypePrefersTheocracyForProphets(t *testing.T) {
	world := NewWorld(12)
	addTestPobles(world, 6, entities.ArchetypeProphet)
	addTestPobles(world, 2, entities.ArchetypeInnocent)

	if got := DetermineGovernmentType(world); got != GovernmentTheocracy {
		t.Fatalf("expected theocracy, got %s", got)
	}
}

func TestEmergentLawsReactToMurderHistory(t *testing.T) {
	world := NewWorld(13)
	world.EventHistory = append(world.EventHistory, GameEvent{
		ID:   "murder",
		Type: ai.GameEventDeath,
		Tags: []string{"murder", "violence"},
	})

	laws := emergentLaws(world)
	found := false
	for _, law := range laws {
		if law.ID == "law_no_murder" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected anti-murder law, got %+v", laws)
	}
}

func TestMaintainGovernmentStoresLegitimacyAndStability(t *testing.T) {
	world := NewWorld(130)
	addTestPobles(world, 21, entities.ArchetypeCaretaker)

	event := MaintainGovernment(world)
	if event == nil {
		t.Fatal("expected government formation event")
	}
	if world.Government == nil {
		t.Fatal("expected government")
	}
	if world.Government.Legitimacy <= 0 || world.Government.Stability <= 0 {
		t.Fatalf("expected visible government metrics, got legitimacy=%d stability=%d", world.Government.Legitimacy, world.Government.Stability)
	}
	if world.Government.LastElection.ToMinutes() != world.Calendar.ToMinutes() {
		t.Fatalf("expected first election clock to start at formation, got %+v", world.Government.LastElection)
	}
}

func TestRunElectionCanReplaceWeakLeader(t *testing.T) {
	world := NewWorld(131)
	addTestPobles(world, 21, entities.ArchetypeCaretaker)
	pobles := world.GetAllPobles()
	weak := pobles[0]
	strong := pobles[1]
	weak.Personality.Ambition = 1
	weak.Personality.Loyalty = 1
	weak.Personality.Conscientiousness = 1
	weak.Personality.Agreeableness = 1
	strong.Personality.Ambition = 100
	strong.Personality.Loyalty = 100
	strong.Personality.Conscientiousness = 100
	strong.Personality.Agreeableness = 100
	oldLeader := weak.ID
	world.Government = &GovernmentSystem{
		Type:         GovernmentDemocracy,
		Leader:       &oldLeader,
		Laws:         emergentLaws(world),
		Legitimacy:   45,
		Stability:    55,
		LastElection: entities.NewGameTime(0, 0, 0),
	}
	world.Calendar = entities.NewGameTime(31, 0, 0)

	event := runElection(world)
	if event == nil {
		t.Fatal("expected election event")
	}
	if world.Government.Leader == nil || *world.Government.Leader != strong.ID {
		t.Fatalf("expected strongest candidate to win, got %v", world.Government.Leader)
	}
	if !hasAnyTag(event.Tags, "election") {
		t.Fatalf("expected election tag, got %+v", event.Tags)
	}
}

func TestRunGovernmentCrisisChangesBrokenGovernment(t *testing.T) {
	world := NewWorld(132)
	addTestPobles(world, 21, entities.ArchetypeWarrior)
	leader := world.GetAllPobles()[0].ID
	world.Government = &GovernmentSystem{
		Type:       GovernmentTribal,
		Leader:     &leader,
		Laws:       emergentLaws(world),
		Legitimacy: 10,
		Stability:  12,
		LastCrisis: entities.NewGameTime(0, 0, 0),
	}
	world.Calendar = entities.NewGameTime(20, 0, 0)
	world.EventHistory = append(world.EventHistory, GameEvent{
		ID:           "fight",
		Type:         ai.GameEventConflict,
		Time:         entities.NewGameTime(19, 0, 0),
		PrimaryActor: leader,
		Tags:         []string{"violence"},
		Description:  "the settlement was cracking in public",
	})

	event := runGovernmentCrisis(world)
	if event == nil {
		t.Fatal("expected crisis event")
	}
	if !hasAnyTag(event.Tags, "coup", "revolution") {
		t.Fatalf("expected coup or revolution tag, got %+v", event.Tags)
	}
	if world.Government.Type == GovernmentTribal {
		t.Fatal("expected government type to change after crisis")
	}
}

func TestRunLawEnforcementTurnsLawsIntoEvents(t *testing.T) {
	world := NewWorld(133)
	addTestPobles(world, 21, entities.ArchetypeCaretaker)
	leader := world.GetAllPobles()[0].ID
	world.Government = &GovernmentSystem{
		Type:       GovernmentDemocracy,
		Leader:     &leader,
		Laws:       []Law{{ID: "law_property", Description: "property is protected", Penalty: "restitution", IsEnforced: true}},
		Legitimacy: 50,
		Stability:  50,
	}
	world.Calendar = entities.NewGameTime(7, 0, 0)
	world.EventHistory = append(world.EventHistory, GameEvent{
		ID:           "theft",
		Type:         ai.GameEventSocialNegative,
		Time:         entities.NewGameTime(6, 23, 0),
		PrimaryActor: leader,
		Tags:         []string{"theft"},
		Description:  "stored food vanished",
	})

	event := runLawEnforcement(world)
	if event == nil {
		t.Fatal("expected law enforcement event")
	}
	if !hasAnyTag(event.Tags, "law", "enforcement", "law_property") {
		t.Fatalf("expected law enforcement tags, got %+v", event.Tags)
	}
}

func TestAttemptDiscoveryUnlocksTechAndFeatures(t *testing.T) {
	world := NewWorld(14)
	addTestPobles(world, 4, entities.ArchetypeSage)
	for _, poble := range world.GetAllPobles() {
		poble.Personality.Openness = 95
		poble.Needs.Hunger = 0
		poble.Needs.Thirst = 0
		poble.Needs.Sleep = 0
	}
	origin := world.GetIsland("island_0")
	origin.Resources[ResourceWood] = 400
	origin.Resources[ResourceStone] = 400

	var discovered *TechNode
	for i := 0; i < 20; i++ {
		if node := AttemptDiscovery(world); node != nil {
			discovered = node
			break
		}
		world.Calendar = world.Calendar.Add(24)
	}
	if discovered == nil {
		t.Fatal("expected at least one discovery")
	}
	if !world.TechTree.Unlocked[discovered.ID] {
		t.Fatalf("expected tech unlocked: %+v", discovered)
	}
	if len(world.TechTree.GetUnlockedFeatures()) == 0 {
		t.Fatal("expected unlocked features")
	}
}

func TestTechnologyLevelReachesThree(t *testing.T) {
	world := NewWorld(15)
	for _, id := range []TechID{
		TechFireControl, TechBasicShelter, TechStoneTools, TechAgriculture,
		TechWriting, TechBasicMedicine, TechCurrency, TechMetalworking,
		TechContraception, TechSurgery, TechNavigation, TechElectricity,
	} {
		world.TechTree.Unlocked[id] = true
		world.TechTree.Discovered[id] = world.Calendar
	}

	if got := technologyLevel(world); got < 3 {
		t.Fatalf("expected tech level 3+, got %d", got)
	}
}

func addTestPobles(world *World, count int, archetype entities.ArchetypeID) {
	start := world.GetPopulation()
	for i := 0; i < count; i++ {
		id := start + i
		poble := entities.NewPoble(
			string(rune('a'+(id%26)))+string(rune('a'+((id/26)%26))),
			"Test",
			24,
			entities.Female,
		)
		poble.Archetype = archetype
		poble.IsAlive = true
		poble.Personality.Openness = 80
		poble.Personality.Ambition = 70
		poble.Personality.Conscientiousness = 70
		poble.Needs = entities.NewNeeds()
		world.AddPoble(&poble, Location{IslandID: "island_0", BuildingID: "island_0_home_0"})
	}
}
