package engine

import (
	"math/rand"
	"path/filepath"
	"strings"
	"testing"

	"github.com/user/poblation/internal/ai"
	"github.com/user/poblation/internal/entities"
	"github.com/user/poblation/internal/events"
	"github.com/user/poblation/internal/templates"
	"github.com/user/poblation/internal/world"
)

func TestOnTickProcessesSocialEvents(t *testing.T) {
	gameWorld := world.NewWorld(101)
	first := entities.NewPoble("first", "First", 31, entities.Female)
	second := entities.NewPoble("second", "Second", 33, entities.Male)
	firstRel := entities.NewRelationship(second.ID, entities.RelationshipEnemy)
	firstRel.Resentment = 100
	firstRel.Familiarity = 90
	secondRel := entities.NewRelationship(first.ID, entities.RelationshipEnemy)
	secondRel.Resentment = 100
	secondRel.Familiarity = 90
	first.Relationships[second.ID] = firstRel
	second.Relationships[first.ID] = secondRel

	if !gameWorld.AddPoble(&first, world.Location{IslandID: "island_0", X: 4, Y: 4}) {
		t.Fatal("add first")
	}
	if !gameWorld.AddPoble(&second, world.Location{IslandID: "island_0", X: 5, Y: 4}) {
		t.Fatal("add second")
	}

	orchestrator := &Orchestrator{
		world:      gameWorld,
		eventQueue: events.NewEventQueue(rand.New(rand.NewSource(3))),
		rng:        rand.New(rand.NewSource(3)),
	}

	current := entities.NewGameTime(0, 0, 0)
	generated := []events.GameEvent{}
	for i := 0; i < 24; i++ {
		current = current.Add(1)
		generated = append(generated, orchestrator.OnTick(GameTick{CurrentTime: current, DeltaHours: 1})...)
	}

	for _, event := range generated {
		if strings.HasPrefix(event.ID, "fight:") &&
			(event.Type == events.EventFightVerbal || event.Type == events.EventFightPhysical) {
			return
		}
	}
	t.Fatalf("expected social fight event from orchestrator tick, got %+v", generated)
}

func TestNarrativeArtifactsCreateInnerLife(t *testing.T) {
	gameWorld := world.NewWorld(202)
	writer := entities.NewPoble("writer", "Ami", 24, entities.Female)
	target := entities.NewPoble("target", "Sol", 26, entities.Male)
	writer.IsAlive = true
	target.IsAlive = true
	writer.CurrentMood = entities.MoodSad
	writer.Secrets = []entities.Secret{entities.NewSecret("secret_1", entities.SecretDarkDesire, "Ami wants something dangerous.")}
	writer.Memories = []entities.Memory{
		{
			ID:               "memory_1",
			Timestamp:        entities.NewGameTime(1, 8, 0),
			Type:             entities.MemoryRomantic,
			Participants:     []string{"writer", "target"},
			EmotionIntensity: 82,
			Summary:          "Ami and Sol almost confessed the truth.",
		},
	}
	relationship := entities.NewRelationship(target.ID, entities.RelationshipComplicated)
	relationship.Attraction = 80
	relationship.Familiarity = 90
	relationship.Affection = 70
	writer.Relationships[target.ID] = relationship

	if !gameWorld.AddPoble(&writer, world.Location{IslandID: "island_0", X: 2, Y: 2}) {
		t.Fatal("add writer")
	}
	if !gameWorld.AddPoble(&target, world.Location{IslandID: "island_0", X: 3, Y: 2}) {
		t.Fatal("add target")
	}

	rng := rand.New(rand.NewSource(8))
	templateEngine := templates.NewTemplateEngine(rng)
	if err := templateEngine.LoadTemplates(filepath.Join("..", "..", "templates")); err != nil {
		t.Fatalf("load templates: %v", err)
	}
	orchestrator := &Orchestrator{
		world:          gameWorld,
		templateEngine: templateEngine,
		rng:            rng,
		eventQueue:     events.NewEventQueue(rng),
	}

	poble := gameWorld.GetPoble("writer")
	now := entities.NewGameTime(2, 18, 0)
	gameWorld.Calendar = now
	orchestrator.applyActionNarrativeArtifacts(poble, ai.Action{Type: ai.ActionWriteDiary}, now)
	event, ok := orchestrator.eventFromAction(poble, ai.Action{Type: ai.ActionWriteDiary}, now)
	if !ok || strings.TrimSpace(event.Description) == "" {
		t.Fatalf("expected diary action event to carry artifact description, got %+v", event)
	}
	orchestrator.applyActionNarrativeArtifacts(poble, ai.Action{Type: ai.ActionSendLetter, TargetID: "target"}, now)
	orchestrator.applyActionNarrativeArtifacts(poble, ai.Action{Type: ai.ActionSleep}, now)
	orchestrator.generatePassiveThought(poble)

	if len(poble.DiaryEntries) != 1 || strings.TrimSpace(poble.DiaryEntries[0].Text) == "" {
		t.Fatalf("expected diary artifact, got %+v", poble.DiaryEntries)
	}
	if len(poble.Letters) != 1 || !poble.Letters[0].IsSent || strings.TrimSpace(poble.Letters[0].Text) == "" {
		t.Fatalf("expected sent letter artifact, got %+v", poble.Letters)
	}
	if len(poble.Dreams) != 1 || strings.TrimSpace(poble.Dreams[0].Text) == "" {
		t.Fatalf("expected dream artifact, got %+v", poble.Dreams)
	}
	if len(poble.Thoughts) != 1 || strings.TrimSpace(poble.Thoughts[0].Text) == "" {
		t.Fatalf("expected passive thought artifact, got %+v", poble.Thoughts)
	}
}

func TestPromisedActionTypesReachEventFeed(t *testing.T) {
	cases := []struct {
		action ai.ActionType
		event  events.EventType
		public bool
	}{
		{action: ai.ActionWriteDiary, event: events.EventDecisionPoint, public: false},
		{action: ai.ActionSendLetter, event: events.EventDecisionPoint, public: false},
		{action: ai.ActionBetray, event: events.EventBetrayalRevealed, public: false},
		{action: ai.ActionGovern, event: events.EventElection, public: true},
		{action: ai.ActionTrade, event: events.EventTradeEstablished, public: true},
		{action: ai.ActionPray, event: events.EventRitual, public: true},
		{action: ai.ActionParty, event: events.EventParty, public: true},
		{action: ai.ActionHaveBreakdown, event: events.EventMentalBreakdown, public: false},
	}

	for _, tc := range cases {
		eventType, public, ok := eventTypeForAction(tc.action)
		if !ok {
			t.Fatalf("expected action %s to be mapped to event feed", tc.action)
		}
		if eventType != tc.event || public != tc.public {
			t.Fatalf("action %s mapped to %s/public=%v, want %s/public=%v", tc.action, eventType, public, tc.event, tc.public)
		}
	}
}

func TestOnTickRunsDailyCivilizationSystems(t *testing.T) {
	gameWorld := world.NewWorld(303)
	for i := 0; i < 21; i++ {
		poble := entities.NewPoble(
			"daily_"+string(rune('a'+i)),
			"Daily",
			24,
			entities.Female,
		)
		poble.IsAlive = true
		poble.Archetype = entities.ArchetypeCaretaker
		poble.Personality.Ambition = 70
		poble.Personality.Conscientiousness = 75
		if !gameWorld.AddPoble(&poble, world.Location{IslandID: "island_0", BuildingID: "island_0_home_0"}) {
			t.Fatalf("add daily poble %d", i)
		}
	}
	origin := gameWorld.GetIsland("island_0")
	origin.Resources[world.ResourceFood] = 300
	origin.Resources[world.ResourceWater] = 300
	origin.Resources[world.ResourceWood] = 300
	origin.Resources[world.ResourceStone] = 300

	orchestrator := &Orchestrator{
		world:               gameWorld,
		eventQueue:          events.NewEventQueue(rand.New(rand.NewSource(9))),
		rng:                 rand.New(rand.NewSource(9)),
		civilizationManager: &world.CivilizationManager{},
	}

	generated := orchestrator.OnTick(GameTick{
		CurrentTime: entities.NewGameTime(3, 0, 0),
		DeltaHours:  1,
		IsNewDay:    true,
	})
	if gameWorld.Government == nil {
		t.Fatal("expected daily civilization tick to establish government")
	}
	for _, event := range generated {
		if event.Type == events.EventElection || event.Type == events.EventTradeEstablished || event.Type == events.EventTechDiscovered {
			return
		}
	}
	t.Fatalf("expected civilization event in feed, got %+v", generated)
}
