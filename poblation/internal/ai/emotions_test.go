package ai

import (
	"testing"

	"github.com/user/poblation/internal/entities"
)

func TestDeathEventHitsFriendAndEnemyDifferently(t *testing.T) {
	friend := entities.NewPoble("self", "Self", 29, entities.Female)
	friendRel := entities.NewRelationship("alex", entities.RelationshipFriend)
	friendRel.Affection = 92
	friendRel.Trust = 88
	friendRel.Respect = 70
	friend.Relationships["alex"] = friendRel

	friendSystem := NewEmotionSystem(&friend)
	friendChanges := friendSystem.ProcessEvent(GameEvent{
		ID:       "death-friend",
		Type:     GameEventDeath,
		TargetID: "alex",
		Severity: 90,
	})
	if !hasEmotion(friendChanges, entities.EmotionGrief) {
		t.Fatalf("expected grief for friend death, got %+v", friendChanges)
	}

	enemy := entities.NewPoble("self2", "Self2", 29, entities.Male)
	enemyRel := entities.NewRelationship("alex", entities.RelationshipEnemy)
	enemyRel.Resentment = 95
	enemyRel.Fear = 40
	enemy.Relationships["alex"] = enemyRel

	enemySystem := NewEmotionSystem(&enemy)
	enemyChanges := enemySystem.ProcessEvent(GameEvent{
		ID:       "death-enemy",
		Type:     GameEventDeath,
		TargetID: "alex",
		Severity: 90,
	})
	if !hasEmotion(enemyChanges, entities.EmotionJoy) && !hasEmotion(enemyChanges, entities.EmotionRelief) {
		t.Fatalf("expected joy or relief for enemy death, got %+v", enemyChanges)
	}
}

func TestConflictMarksPobleAsEmotionallyInteresting(t *testing.T) {
	poble := entities.NewPoble("dramatic", "Dramatic", 34, entities.Female)
	system := NewEmotionSystem(&poble)

	system.applyChange(EmotionChange{Emotion: entities.EmotionLove, Intensity: 70, DurationHours: 24})
	system.applyChange(EmotionChange{Emotion: entities.EmotionResentment, Intensity: 65, DurationHours: 24})

	conflicts := system.DetectEmotionConflict()
	if len(conflicts) == 0 {
		t.Fatal("expected emotional conflict")
	}
	if !system.EmotionallyInteresting {
		t.Fatal("expected poble to be marked emotionally interesting")
	}
	if len(system.InternalConflictThoughts) == 0 {
		t.Fatal("expected internal conflict thoughts")
	}
}

func TestUpdateDecaysJoyFasterThanGrief(t *testing.T) {
	poble := entities.NewPoble("mood", "Mood", 31, entities.Male)
	system := NewEmotionSystem(&poble)

	system.applyChange(EmotionChange{Emotion: entities.EmotionJoy, Intensity: 80, DurationHours: 12})
	system.applyChange(EmotionChange{Emotion: entities.EmotionGrief, Intensity: 80, DurationHours: 120})
	system.Update(6)

	values := system.intensityByEmotion()
	if values[entities.EmotionJoy] >= values[entities.EmotionGrief] {
		t.Fatalf("expected joy to decay faster than grief, got joy=%.2f grief=%.2f", values[entities.EmotionJoy], values[entities.EmotionGrief])
	}
}

func hasEmotion(changes []EmotionChange, target entities.EmotionType) bool {
	for _, change := range changes {
		if change.Emotion == target {
			return true
		}
	}
	return false
}
