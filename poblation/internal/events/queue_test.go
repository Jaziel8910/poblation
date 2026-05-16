package events

import (
	"math/rand"
	"testing"

	"github.com/user/poblation/internal/entities"
)

type testWorld struct {
	pobles map[string]*entities.Poble
	state  entities.WorldState
}

func (w testWorld) GetAllPobles() []*entities.Poble {
	result := make([]*entities.Poble, 0, len(w.pobles))
	for _, poble := range w.pobles {
		result = append(result, poble)
	}
	return result
}

func (w testWorld) GetWorldState() entities.WorldState {
	return w.state
}

func (w testWorld) GetPoble(id string) *entities.Poble {
	return w.pobles[id]
}

func TestEventQueueProcessesFourLanes(t *testing.T) {
	queue := NewEventQueue(rand.New(rand.NewSource(1)))
	now := entities.NewGameTime(0, 0, 0)

	queue.Push(GameEvent{ID: "immediate", Timestamp: now}, Immediate())
	queue.Push(GameEvent{ID: "scheduled", Timestamp: now}, InHours(2))
	queue.Push(GameEvent{ID: "dormant", Timestamp: now}, TriggeredBy(func(entities.GameTime, World) bool {
		return true
	}))
	queue.PushGossip(GameEvent{ID: "gossip", Timestamp: now})

	firstTick := queue.Process(now.Add(1), testWorld{})
	if len(firstTick) != 3 {
		t.Fatalf("expected immediate, dormant, and gossip events; got %d", len(firstTick))
	}
	if queue.Len() != 1 {
		t.Fatalf("expected one scheduled event left, got %d", queue.Len())
	}

	secondTick := queue.Process(now.Add(2), testWorld{})
	if len(secondTick) != 1 || secondTick[0].ID != "scheduled" {
		t.Fatalf("expected scheduled event on second tick, got %#v", secondTick)
	}
}

func TestApplyConsequencesImmediateAndDeferred(t *testing.T) {
	poble := entities.NewPoble("p1", "Kira", 23, entities.Female)
	world := testWorld{
		pobles: map[string]*entities.Poble{"p1": &poble},
		state:  entities.NewWorldState(),
	}

	deferred := ApplyConsequences(GameEvent{
		ID:        "pregnancy",
		Timestamp: entities.NewGameTime(3, 4, 0),
		Consequences: []Consequence{
			{TargetID: "p1", Type: ConsequencePregnancyCaused, Value: 1},
			{TargetID: "p1", Type: ConsequenceHealthChange, Value: -10, Delay: 5},
		},
	}, world)

	if !hasCondition(&poble, entities.ConditionPregnant) {
		t.Fatal("expected pregnancy consequence to add pregnant condition")
	}
	if len(deferred) != 1 {
		t.Fatalf("expected one deferred consequence event, got %d", len(deferred))
	}
	if deferred[0].Timestamp != entities.NewGameTime(3, 9, 0) {
		t.Fatalf("expected deferred event five hours later, got %s", deferred[0].Timestamp)
	}
}
