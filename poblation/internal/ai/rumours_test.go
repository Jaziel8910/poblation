package ai

import (
	"testing"

	"github.com/user/poblation/internal/entities"
)

type rumourTestWorld struct {
	pobles []*Poble
}

func (w rumourTestWorld) GetAllPobles() []*Poble {
	return w.pobles
}

func TestMutateRumourEscalatesConflictDramatically(t *testing.T) {
	rumour := &Rumour{
		ID:               "r1",
		OriginalFactType: RumourFactConflict,
		OriginalContent:  "Noah y Kira tuvieron una pelea",
		CurrentContent:   "Noah y Kira tuvieron una pelea",
		TruthScore:       1,
		Spreadings:       4,
	}

	mutated := MutateRumour(rumour)
	if mutated.CurrentContent != "Noah golpeo a Kira" {
		t.Fatalf("expected dramatic conflict mutation, got %q", mutated.CurrentContent)
	}
	if mutated.TruthScore >= rumour.TruthScore {
		t.Fatalf("expected truth score to drop, got %.2f", mutated.TruthScore)
	}
}

func TestSpreadRumourUsesReceiverTrustToBelieve(t *testing.T) {
	from := entities.NewPoble("noah", "Noah", 29, entities.Male)
	to := entities.NewPoble("kira", "Kira", 28, entities.Female)
	relationship := entities.NewRelationship(from.ID, entities.RelationshipFriend)
	relationship.Trust = 96
	to.Relationships[from.ID] = relationship

	system := NewRumourSystem(nil)
	system.AddRumour(Rumour{
		ID:               "r2",
		OriginalFactType: RumourFactGeneric,
		OriginalContent:  "Algo raro paso cerca del refugio",
		CurrentContent:   "Algo raro paso cerca del refugio",
		TruthScore:       1,
		KnownBy:          []string{from.ID},
	})

	system.SpreadRumour("r2", from.ID, to.ID, rumourTestWorld{pobles: []*Poble{&from, &to}})
	rumour, ok := system.GetRumour("r2")
	if !ok {
		t.Fatal("expected rumour to remain in system")
	}
	if !containsString(rumour.KnownBy, to.ID) {
		t.Fatalf("expected receiver to know rumour, got %#v", rumour.KnownBy)
	}
	if !containsString(rumour.BelievedBy, to.ID) {
		t.Fatalf("expected trusted speaker to be believed, got %#v", rumour.BelievedBy)
	}
}

func TestDetectRumourImpactWhenSensitiveTargetLearns(t *testing.T) {
	noah := entities.NewPoble("noah", "Noah", 29, entities.Male)
	kira := entities.NewPoble("kira", "Kira", 28, entities.Female)
	relationship := entities.NewRelationship(kira.ID, entities.RelationshipLover)
	relationship.Attraction = 85
	relationship.Trust = 88
	noah.Relationships[kira.ID] = relationship

	rumour := &Rumour{
		ID:               "r3",
		OriginalFactType: RumourFactBetrayal,
		OriginalContent:  "Kira oculto algo importante",
		CurrentContent:   "Kira oculto algo importante",
		TruthScore:       0.9,
		KnownBy:          []string{noah.ID},
		IsSensitive:      true,
		SensitiveForID:   noah.ID,
		OriginatorID:     kira.ID,
		SubjectIDs:       []string{kira.ID},
	}

	impacts := DetectRumourImpact(rumour, rumourTestWorld{pobles: []*Poble{&noah, &kira}})
	if len(impacts) == 0 {
		t.Fatal("expected sensitive rumour impact")
	}
	if impacts[0].RumourID != rumour.ID {
		t.Fatalf("expected impact to reference rumour, got %#v", impacts[0])
	}
}
