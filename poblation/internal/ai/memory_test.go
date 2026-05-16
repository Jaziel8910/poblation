package ai

import (
	"math/rand"
	"testing"

	"github.com/user/poblation/internal/entities"
)

func TestAddMemoryReplacesWeakestNonTrauma(t *testing.T) {
	poble := entities.NewPoble("self", "Self", 30, entities.Female)
	system := NewMemorySystem(&poble, rand.New(rand.NewSource(1)))
	system.maxMemories = 3

	system.AddMemory(testMemory("m1", entities.MemoryPositive, 20, entities.NewGameTime(1, 8, 0), []string{"alex"}))
	system.AddMemory(testMemory("m2", entities.MemoryNegative, 30, entities.NewGameTime(1, 9, 0), []string{"alex"}))
	system.AddMemory(testMemory("m3", entities.MemoryTraumatic, 95, entities.NewGameTime(1, 10, 0), []string{"alex"}))
	system.AddMemory(testMemory("m4", entities.MemoryBetrayal, 70, entities.NewGameTime(1, 11, 0), []string{"alex"}))

	if len(system.memories) != 3 {
		t.Fatalf("expected 3 memories, got %d", len(system.memories))
	}
	if findMemoryByID(system.memories, "m1") != nil {
		t.Fatal("expected weakest non-trauma memory to be replaced")
	}
	if findMemoryByID(system.memories, "m3") == nil {
		t.Fatal("expected traumatic memory to remain protected")
	}
	if len(poble.Memories) != 3 {
		t.Fatalf("expected sync to poble memories, got %d", len(poble.Memories))
	}
}

func TestAddMemoryMergesSimilarEvents(t *testing.T) {
	system := NewMemorySystem(nil, rand.New(rand.NewSource(2)))

	first := testMemory("self:event-1", entities.MemoryNegative, 42, entities.NewGameTime(2, 5, 0), []string{"self", "alex"})
	first.Summary = "Alex me humillo en publico."
	first.Tags = []string{"event:event-1", "social"}

	second := testMemory("self:event-2", entities.MemoryNegative, 64, entities.NewGameTime(2, 5, 0), []string{"alex", "self"})
	second.Summary = "Alex me humillo en publico."
	second.Tags = []string{"event:event-1", "crowd"}

	system.AddMemory(first)
	system.AddMemory(second)

	if len(system.memories) != 1 {
		t.Fatalf("expected merged memory, got %d memories", len(system.memories))
	}
	merged := system.memories[0]
	if merged.EmotionIntensity <= 64 {
		t.Fatalf("expected merged intensity to reinforce memory, got %.2f", merged.EmotionIntensity)
	}
	if !containsString(merged.Tags, "social") || !containsString(merged.Tags, "crowd") {
		t.Fatalf("expected merged tags, got %+v", merged.Tags)
	}
}

func TestRecallBlocksRepressedMemoriesUntilStressGetsHigh(t *testing.T) {
	poble := entities.NewPoble("self", "Self", 30, entities.Female)
	system := NewMemorySystem(&poble, rand.New(rand.NewSource(3)))

	trauma := testMemory("trauma", entities.MemoryTraumatic, 92, entities.NewGameTime(3, 2, 0), []string{"alex"})
	trauma.IsRepressed = true
	trauma.Tags = []string{"event:trauma"}
	system.AddMemory(trauma)

	system.SetEmotionalStress(45)
	lowStress := system.Recall(MemoryQuery{AboutPersonID: "alex", MaxResults: 5})
	if len(lowStress) != 0 {
		t.Fatalf("expected repressed trauma to stay hidden at low stress, got %+v", lowStress)
	}

	system.SetEmotionalStress(92)
	highStress := system.Recall(MemoryQuery{AboutPersonID: "alex", MaxResults: 5})
	if len(highStress) != 1 || highStress[0].ID != "trauma" {
		t.Fatalf("expected trauma recall under high stress, got %+v", highStress)
	}
}

func TestHasTraumaAndActiveTraumasRespectRepression(t *testing.T) {
	poble := entities.NewPoble("self", "Self", 30, entities.Female)
	system := NewMemorySystem(&poble, rand.New(rand.NewSource(4)))

	trauma := testMemory("trauma", entities.MemoryTraumatic, 88, entities.NewGameTime(4, 1, 0), []string{"alex"})
	trauma.IsRepressed = true
	system.AddMemory(trauma)

	if !system.HasTrauma() {
		t.Fatal("expected trauma to be recorded")
	}

	system.SetEmotionalStress(40)
	if active := system.GetActiveTraumas(); len(active) != 0 {
		t.Fatalf("expected no active traumas under low stress, got %+v", active)
	}

	system.SetEmotionalStress(95)
	active := system.GetActiveTraumas()
	if len(active) != 1 || active[0].ID != "trauma" {
		t.Fatalf("expected trauma to become active under high stress, got %+v", active)
	}
}

func TestShouldBringUpPastWithUsesNegativeHistory(t *testing.T) {
	poble := entities.NewPoble("self", "Self", 30, entities.Female)
	poble.Personality.Neuroticism = 84
	relationship := entities.NewRelationship("alex", entities.RelationshipBetrayer)
	relationship.Resentment = 91
	poble.Relationships["alex"] = relationship

	system := NewMemorySystem(&poble, nil)
	memory := testMemory("betrayal", entities.MemoryBetrayal, 94, entities.NewGameTime(5, 6, 0), []string{"alex"})
	memory.Tags = []string{"event:betrayal", "unresolved"}
	system.AddMemory(memory)
	system.SetEmotionalStress(88)

	if !system.ShouldBringUpPastWith("alex") {
		t.Fatal("expected strong resentment and unresolved betrayal to resurface in dialogue")
	}
}

func TestEmotionalDecayCanUnlockRepressedTrauma(t *testing.T) {
	poble := entities.NewPoble("self", "Self", 30, entities.Female)
	system := NewMemorySystem(&poble, nil)

	trauma := testMemory("trauma", entities.MemoryTraumatic, 96, entities.NewGameTime(6, 4, 0), []string{"alex"})
	trauma.IsRepressed = true
	system.AddMemory(trauma)
	system.SetEmotionalStress(95)

	system.EmotionalDecay(12)

	if system.memories[0].IsRepressed {
		t.Fatal("expected extreme stress during decay to unlock repressed trauma")
	}
	if system.memories[0].EmotionIntensity >= 96 {
		t.Fatalf("expected some decay in trauma intensity, got %.2f", system.memories[0].EmotionIntensity)
	}
}

func testMemory(id string, memoryType entities.MemoryType, intensity float32, timestamp entities.GameTime, participants []string) entities.Memory {
	memory := entities.NewMemory(id, timestamp, memoryType, "Recuerdo de prueba")
	memory.Participants = participants
	memory.EmotionIntensity = intensity
	return memory
}

func findMemoryByID(memories []*Memory, id string) *Memory {
	for _, memory := range memories {
		if memory != nil && memory.ID == id {
			return memory
		}
	}
	return nil
}
