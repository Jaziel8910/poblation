package ai

import (
	"math/rand"
	"testing"

	"github.com/user/poblation/internal/entities"
)

type decisionTestWorld struct {
	pobles    []*Poble
	events    []GameEvent
	now       entities.GameTime
	proximity map[string]float32
}

func (w decisionTestWorld) GetAllPobles() []*Poble {
	return w.pobles
}

func (w decisionTestWorld) GetActiveEvents() []GameEvent {
	return w.events
}

func (w decisionTestWorld) GetCurrentTime() entities.GameTime {
	return w.now
}

func (w decisionTestWorld) GetProximityScore(fromID, targetID string) float32 {
	return w.proximity[fromID+"->"+targetID]
}

func TestDecisionSurvivalNeedsOverrideCurrentSleep(t *testing.T) {
	poble := newDecisionTestPoble("self", entities.ArchetypeGhost)
	poble.Needs.Hunger = 96
	poble.Needs.Thirst = 94
	world := decisionTestWorld{pobles: []*Poble{&poble}}
	engine := NewDecisionEngine(&poble, world, rand.New(rand.NewSource(1)))
	engine.currentAction = &Action{Type: ActionSleep, Priority: 70, Duration: 8}

	actions := engine.Decide(1)
	if len(actions) < 2 {
		t.Fatalf("expected hunger and thirst actions, got %+v", actions)
	}
	if actions[0].Priority != 100 {
		t.Fatalf("expected maximum priority survival action, got %+v", actions[0])
	}
	if actions[0].Type != ActionDrink && actions[0].Type != ActionEat {
		t.Fatalf("expected eat or drink first, got %+v", actions[0])
	}
	if engine.currentAction == nil || (engine.currentAction.Type != ActionDrink && engine.currentAction.Type != ActionEat) {
		t.Fatalf("expected survival action to interrupt sleep, got %+v", engine.currentAction)
	}
}

func TestDecisionArchetypesDivergeUnderSameStress(t *testing.T) {
	target := newDecisionTestPoble("target", entities.ArchetypeCustom)
	ruler := newDecisionTestPoble("ruler", entities.ArchetypeRuler)
	innocent := newDecisionTestPoble("innocent", entities.ArchetypeInnocent)
	ruler.Needs.Power = 65
	innocent.Needs.Power = 65
	innocent.Relationships[target.ID] = trustedRelationship(target.ID)

	rulerWorld := decisionTestWorld{pobles: []*Poble{&ruler, &target}}
	innocentWorld := decisionTestWorld{pobles: []*Poble{&innocent, &target}}
	rulerEngine := NewDecisionEngine(&ruler, rulerWorld, rand.New(rand.NewSource(2)))
	innocentEngine := NewDecisionEngine(&innocent, innocentWorld, rand.New(rand.NewSource(2)))

	rulerActions := rulerEngine.Decide(1)
	innocentActions := innocentEngine.Decide(1)
	if len(rulerActions) == 0 || len(innocentActions) == 0 {
		t.Fatalf("expected actions, got ruler=%+v innocent=%+v", rulerActions, innocentActions)
	}
	if rulerActions[0].Type != ActionGovern && rulerActions[0].Type != ActionFormAlliance {
		t.Fatalf("expected ruler to seek control under power pressure, got %+v", rulerActions[0])
	}
	if innocentActions[0].Type != ActionTalkTo {
		t.Fatalf("expected innocent to seek trust/social contact, got %+v", innocentActions[0])
	}
}

func TestEvaluateTargetPrioritizesRelationshipAndProximity(t *testing.T) {
	poble := newDecisionTestPoble("self", entities.ArchetypeLover)
	closeCrush := newDecisionTestPoble("close", entities.ArchetypeCustom)
	distantCrush := newDecisionTestPoble("distant", entities.ArchetypeCustom)

	closeRel := entities.NewRelationship(closeCrush.ID, entities.RelationshipCrush)
	closeRel.Attraction = 78
	closeRel.Affection = 50
	distantRel := entities.NewRelationship(distantCrush.ID, entities.RelationshipCrush)
	distantRel.Attraction = 78
	distantRel.Affection = 50
	poble.Relationships[closeCrush.ID] = closeRel
	poble.Relationships[distantCrush.ID] = distantRel

	world := decisionTestWorld{
		pobles: []*Poble{&poble, &closeCrush, &distantCrush},
		proximity: map[string]float32{
			"self->close":   15,
			"self->distant": -10,
		},
	}
	engine := NewDecisionEngine(&poble, world, rand.New(rand.NewSource(3)))

	targets := engine.EvaluateTarget(ActionFlirtWith)
	if len(targets) < 2 {
		t.Fatalf("expected both crushes as targets, got %+v", targets)
	}
	if targets[0] != closeCrush.ID {
		t.Fatalf("expected closer crush first, got %+v", targets)
	}
}

func TestEmotionalGuiltWithBurningSecretConfesses(t *testing.T) {
	poble := newDecisionTestPoble("self", entities.ArchetypeCustom)
	target := newDecisionTestPoble("target", entities.ArchetypeCustom)
	poble.Relationships[target.ID] = trustedRelationship(target.ID)
	poble.Secrets = []entities.Secret{
		{
			ID:            "secret_1",
			Type:          entities.SecretCriminalAct,
			RevealTrigger: "burning",
		},
	}
	poble.EmotionalState.ActiveEmotions = []entities.EmotionType{entities.EmotionGuilt}
	poble.EmotionalState.Valence = -1
	poble.EmotionalState.Arousal = 1
	poble.EmotionalState.Dominance = -1
	poble.CurrentMood = entities.MoodAnxious

	world := decisionTestWorld{pobles: []*Poble{&poble, &target}}
	engine := NewDecisionEngine(&poble, world, rand.New(rand.NewSource(4)))

	actions := engine.CheckEmotionalUrgency()
	if len(actions) == 0 {
		t.Fatal("expected urgent confession action")
	}
	if actions[0].Type != ActionConfessTo || actions[0].TargetID != target.ID {
		t.Fatalf("expected confession to trusted target, got %+v", actions[0])
	}
}

func TestRelationshipMemoryCreatesPastDrivenAction(t *testing.T) {
	poble := newDecisionTestPoble("self", entities.ArchetypeCustom)
	target := newDecisionTestPoble("target", entities.ArchetypeCustom)
	relationship := entities.NewRelationship(target.ID, entities.RelationshipBetrayer)
	relationship.Resentment = 82
	relationship.Familiarity = 80
	poble.Relationships[target.ID] = relationship
	memory := entities.NewMemory("betrayal_1", entities.NewGameTime(4, 8, 0), entities.MemoryBetrayal, "Target left Self alone during the storm")
	memory.Participants = []string{poble.ID, target.ID}
	memory.EmotionIntensity = 94
	memory.Tags = []string{"betrayal", "storm", "unresolved"}
	poble.Memories = []entities.Memory{memory}

	world := decisionTestWorld{pobles: []*Poble{&poble, &target}}
	engine := NewDecisionEngine(&poble, world, rand.New(rand.NewSource(8)))

	actions := engine.CheckRelationshipGoals()
	if !hasMemoryDrivenAction(actions, target.ID) {
		t.Fatalf("expected a memory-driven action toward target, got %+v", actions)
	}
}

func TestOldBetrayalMemoryChangesTargetRanking(t *testing.T) {
	poble := newDecisionTestPoble("self", entities.ArchetypeCustom)
	plain := newDecisionTestPoble("plain", entities.ArchetypeCustom)
	betrayer := newDecisionTestPoble("betrayer", entities.ArchetypeCustom)

	plainRel := entities.NewRelationship(plain.ID, entities.RelationshipAcquaintance)
	plainRel.Familiarity = 85
	plainRel.Resentment = 35
	betrayerRel := entities.NewRelationship(betrayer.ID, entities.RelationshipAcquaintance)
	betrayerRel.Familiarity = 85
	betrayerRel.Resentment = 35
	poble.Relationships[plain.ID] = plainRel
	poble.Relationships[betrayer.ID] = betrayerRel

	memory := entities.NewMemory("old_betrayal", entities.NewGameTime(1, 9, 0), entities.MemoryBetrayal, "Betrayer sold Self out and nobody repaired it")
	memory.Participants = []string{poble.ID, betrayer.ID}
	memory.EmotionIntensity = 90
	memory.Tags = []string{"betrayal", "unresolved"}
	poble.Memories = []entities.Memory{memory}

	world := decisionTestWorld{
		pobles: []*Poble{&poble, &plain, &betrayer},
		now:    entities.NewGameTime(80, 8, 0),
	}
	engine := NewDecisionEngine(&poble, world, rand.New(rand.NewSource(10)))

	targets := engine.EvaluateTarget(ActionArgueWith)
	if len(targets) == 0 || targets[0] != betrayer.ID {
		t.Fatalf("expected old betrayal to rank betrayer first for argument, got %+v", targets)
	}
}

func TestWarmOldMemoryCanDriveReconciliation(t *testing.T) {
	poble := newDecisionTestPoble("self", entities.ArchetypeCustom)
	target := newDecisionTestPoble("target", entities.ArchetypeCustom)
	relationship := entities.NewRelationship(target.ID, entities.RelationshipFriend)
	relationship.Affection = 68
	relationship.Trust = 62
	relationship.Resentment = 44
	relationship.Familiarity = 90
	poble.Relationships[target.ID] = relationship

	memory := entities.NewMemory("old_good", entities.NewGameTime(2, 7, 0), entities.MemoryPositive, "Target stayed when Self expected everyone to leave")
	memory.Participants = []string{poble.ID, target.ID}
	memory.EmotionIntensity = 86
	poble.Memories = []entities.Memory{memory}

	world := decisionTestWorld{pobles: []*Poble{&poble, &target}, now: entities.NewGameTime(40, 12, 0)}
	engine := NewDecisionEngine(&poble, world, rand.New(rand.NewSource(11)))

	actions := engine.CheckRelationshipGoals()
	for _, action := range actions {
		if action.TargetID == target.ID && action.Type == ActionTalkTo && hasActionTag(action, "reconciliation") {
			return
		}
	}
	t.Fatalf("expected warm memory to create reconciliation talk, got %+v", actions)
}

func TestShouldInterruptCurrentActionTreatsSleepAsSticky(t *testing.T) {
	poble := newDecisionTestPoble("self", entities.ArchetypeGhost)
	engine := NewDecisionEngine(&poble, nil, rand.New(rand.NewSource(5)))
	engine.currentAction = &Action{Type: ActionSleep, Priority: 72, Duration: 8}

	if engine.ShouldInterruptCurrentAction(Action{Type: ActionTalkTo, Priority: 94, TargetID: "friend"}) {
		t.Fatal("expected non-emergency social action not to interrupt sleep")
	}
	if !engine.ShouldInterruptCurrentAction(Action{Type: ActionDrink, Priority: 100, Tags: []string{"survival"}}) {
		t.Fatal("expected survival emergency to interrupt sleep")
	}
}

func hasMemoryDrivenAction(actions []Action, targetID string) bool {
	for _, action := range actions {
		if action.TargetID != targetID {
			continue
		}
		for _, tag := range action.Tags {
			if tag == "memory" {
				return true
			}
		}
	}
	return false
}

func hasActionTag(action Action, target string) bool {
	for _, tag := range action.Tags {
		if tag == target {
			return true
		}
	}
	return false
}

func newDecisionTestPoble(id string, archetype entities.ArchetypeID) Poble {
	poble := entities.NewPoble(id, id, 30, entities.Female)
	poble.Archetype = archetype
	return poble
}

func trustedRelationship(targetID string) entities.Relationship {
	relationship := entities.NewRelationship(targetID, entities.RelationshipBestFriend)
	relationship.Affection = 88
	relationship.Trust = 92
	relationship.Respect = 76
	relationship.Familiarity = 80
	return relationship
}
