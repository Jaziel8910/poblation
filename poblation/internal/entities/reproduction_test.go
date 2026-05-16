package entities

import (
	"math/rand"
	"testing"
)

func TestCanReproduceMaleFemaleNatural(t *testing.T) {
	male := fertileAdult("m", Male)
	female := fertileAdult("f", Female)
	system := NewReproductionSystem(&World{State: NewWorldState()}, rand.New(rand.NewSource(1)))

	analysis := system.CanReproduce(male, female)
	if !analysis.IsBiologicallyPossible {
		t.Fatalf("expected male/female pair to be biologically possible: %+v", analysis)
	}
	if analysis.RequiresThirdParty {
		t.Fatalf("natural pair should not require third party: %+v", analysis)
	}
	if analysis.FertilityChance <= 0.5 || analysis.FertilityChance > 1 {
		t.Fatalf("unexpected fertility chance: %.2f", analysis.FertilityChance)
	}
}

func TestCanReproduceSameSexRequiresEraPaths(t *testing.T) {
	first := fertileAdult("a", Female)
	second := fertileAdult("b", Female)
	world := &World{State: NewWorldState(), Pobles: map[string]*Poble{first.ID: first, second.ID: second}}
	system := NewReproductionSystem(world, rand.New(rand.NewSource(2)))

	analysis := system.CanReproduce(first, second)
	if analysis.IsBiologicallyPossible || !analysis.RequiresThirdParty {
		t.Fatalf("same sex pair should need a third party: %+v", analysis)
	}
	if analysis.ThirdPartyType != ThirdPartyDonor {
		t.Fatalf("female/female pair should need donor, got %s", analysis.ThirdPartyType)
	}
	if !hasPath(analysis.AlternativePaths, ReproductionPathAdoption, true) {
		t.Fatalf("expected adoption event path in era zero: %+v", analysis.AlternativePaths)
	}
	if hasPath(analysis.AlternativePaths, ReproductionPathDonorNeeded, true) {
		t.Fatalf("donor should not be broadly available in era zero: %+v", analysis.AlternativePaths)
	}

	world.State.Era = EraThree
	analysis = system.CanReproduce(first, second)
	if !hasPath(analysis.AlternativePaths, ReproductionPathDonorNeeded, true) {
		t.Fatalf("donor should be available by era three: %+v", analysis.AlternativePaths)
	}
	if !hasPath(analysis.AlternativePaths, ReproductionPathTechRequired, true) {
		t.Fatalf("tech should be available by era three: %+v", analysis.AlternativePaths)
	}
}

func TestCanReproduceConsanguinityRisk(t *testing.T) {
	first := fertileAdult("a", Male)
	second := fertileAdult("b", Female)
	first.Parents = [2]string{"p1", "p2"}
	second.Parents = [2]string{"p1", "p3"}

	analysis := CanReproduce(first, second)
	if analysis.ConsanguinityLevel != 1 {
		t.Fatalf("siblings should be consanguinity level 1, got %d", analysis.ConsanguinityLevel)
	}
	if analysis.ConsanguinityRisk != 0.25 {
		t.Fatalf("siblings should have 25%% genetic risk, got %.2f", analysis.ConsanguinityRisk)
	}
}

func TestInheritTraitsAddsRiskAboveThreshold(t *testing.T) {
	parent := fertileAdult("same", Intersex)
	genetics := InheritTraits(parent, parent)
	if genetics.InbreedingCoefficient <= 0.25 {
		t.Fatalf("expected coefficient above threshold, got %.2f", genetics.InbreedingCoefficient)
	}
	if len(genetics.RecessiveRisks) == 0 {
		t.Fatal("expected descriptive recessive risk above threshold")
	}
}

func TestAdoptionEventAddsBabyAndSecret(t *testing.T) {
	world := &World{State: NewWorldState(), Pobles: map[string]*Poble{}}
	baby, event := AdoptionEvent(world)
	if baby == nil {
		t.Fatal("expected adopted baby")
	}
	if baby.Age != 0 || !baby.IsAlive {
		t.Fatalf("expected living newborn, got %+v", baby)
	}
	if len(baby.Secrets) == 0 {
		t.Fatal("expected future secret on adopted baby")
	}
	if world.Pobles[baby.ID] == nil || event.Type != "ADOPTION" {
		t.Fatalf("expected baby and event registered, event=%+v", event)
	}
}

func TestDonorProcessCreatesRelationshipPressure(t *testing.T) {
	donor := fertileAdult("donor", Male)
	recipient := fertileAdult("recipient", Female)
	world := &World{
		State:  NewWorldState(),
		Pobles: map[string]*Poble{donor.ID: donor, recipient.ID: recipient},
	}

	arc, err := DonorProcess(donor.ID, recipient.ID, world)
	if err != nil {
		t.Fatalf("DonorProcess failed: %v", err)
	}
	if arc.DonorID != donor.ID || !arc.ChildMayDiscoverDonor {
		t.Fatalf("unexpected donor arc: %+v", arc)
	}
	if len(world.Events) == 0 {
		t.Fatal("expected donor event to be registered")
	}
}

func fertileAdult(id string, sex Sex) *Poble {
	poble := NewPoble(id, id, 28, sex)
	poble.Health.Fertility = 1
	poble.Health.HP = 100
	poble.Mental.Stability = 90
	poble.IsAlive = true
	return &poble
}

func hasPath(paths []ReproductionPath, pathType ReproductionPathType, available bool) bool {
	for _, path := range paths {
		if path.Type == pathType && path.Availability == available {
			return true
		}
	}
	return false
}
