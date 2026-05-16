package tests

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/user/poblation/internal/ai"
	"github.com/user/poblation/internal/engine"
	"github.com/user/poblation/internal/entities"
	"github.com/user/poblation/internal/events"
	"github.com/user/poblation/internal/templates"
	"github.com/user/poblation/internal/world"
)

func TestFullGameLoop(t *testing.T) {
	enterRepoRoot(t)
	isolateUserHome(t)

	orchestrator := engine.NewOrchestrator(engine.OrchestratorOptions{Debug: true, Speed: 1})
	if err := orchestrator.Init(424242); err != nil {
		t.Fatalf("init orchestrator: %v", err)
	}
	defer orchestrator.Stop()

	initial := orchestrator.GetWorldSnapshot()
	if len(initial.Pobles) != 2 {
		t.Fatalf("expected 2 starting pobles, got %d", len(initial.Pobles))
	}

	initialNeeds := map[string]entities.Needs{}
	for _, snapshot := range initial.Pobles {
		initialNeeds[snapshot.ID] = snapshot.Needs
	}
	primeFoundersForInteraction(t, orchestrator.World())

	generated := runTicks(orchestrator, 100)
	final := orchestrator.GetWorldSnapshot()
	if len(final.Pobles) == 0 {
		t.Fatal("expected at least one living poble after 100 hours")
	}

	for _, poble := range orchestrator.World().GetAllKnownPobles() {
		if poble.IsAlive {
			if !needsChanged(initialNeeds[poble.ID], poble.Needs) {
				t.Fatalf("expected needs to evolve for %s: before=%+v after=%+v", poble.ID, initialNeeds[poble.ID], poble.Needs)
			}
			continue
		}
		if !hasDeathEventFor(orchestrator.World(), generated, poble.ID) {
			t.Fatalf("poble %s died without an appropriate death event", poble.ID)
		}
	}

	for _, poble := range orchestrator.World().GetAllKnownPobles() {
		thought := renderThought(t, orchestrator.TemplateEngine(), poble, orchestrator.World())
		if strings.TrimSpace(thought) == "" {
			t.Fatalf("expected a generated thought for %s", poble.ID)
		}
	}

	if !hasInteraction(generated, orchestrator.EventFeed()) {
		t.Fatalf("expected at least one interaction event, generated=%+v feed=%+v", generated, orchestrator.EventFeed())
	}
}

func TestReproductionPaths(t *testing.T) {
	cases := []struct {
		name   string
		a      *entities.Poble
		b      *entities.Poble
		era    entities.Era
		assert func(*testing.T, entities.ReproductionAnalysis)
	}{
		{
			name: "male female normal fertility allows pregnancy",
			a:    fertilePoble("m", entities.Male),
			b:    fertilePoble("f", entities.Female),
			era:  entities.EraZero,
			assert: func(t *testing.T, analysis entities.ReproductionAnalysis) {
				t.Helper()
				if !analysis.IsBiologicallyPossible || analysis.FertilityChance <= 0 {
					t.Fatalf("expected possible natural pregnancy, got %+v", analysis)
				}
				if !hasReproductionPath(analysis.AlternativePaths, entities.ReproductionPathNatural, true) {
					t.Fatalf("expected natural path, got %+v", analysis.AlternativePaths)
				}
			},
		},
		{
			name: "male male era zero only event or adoption available",
			a:    fertilePoble("m1", entities.Male),
			b:    fertilePoble("m2", entities.Male),
			era:  entities.EraZero,
			assert: func(t *testing.T, analysis entities.ReproductionAnalysis) {
				t.Helper()
				if analysis.IsBiologicallyPossible || !analysis.RequiresThirdParty {
					t.Fatalf("expected third-party route for male/male, got %+v", analysis)
				}
				if !hasReproductionPath(analysis.AlternativePaths, entities.ReproductionPathAdoption, true) {
					t.Fatalf("expected adoption/event path in era zero, got %+v", analysis.AlternativePaths)
				}
				if hasReproductionPath(analysis.AlternativePaths, entities.ReproductionPathTechRequired, true) {
					t.Fatalf("tech reproduction should not be available in era zero: %+v", analysis.AlternativePaths)
				}
			},
		},
		{
			name: "male male era three allows tech route",
			a:    fertilePoble("m3", entities.Male),
			b:    fertilePoble("m4", entities.Male),
			era:  entities.EraThree,
			assert: func(t *testing.T, analysis entities.ReproductionAnalysis) {
				t.Helper()
				if !hasReproductionPath(analysis.AlternativePaths, entities.ReproductionPathTechRequired, true) {
					t.Fatalf("expected tech route by era three, got %+v", analysis.AlternativePaths)
				}
			},
		},
		{
			name: "female female close kin activates risk flag",
			a:    fertileSibling("f1", entities.Female),
			b:    fertileSibling("f2", entities.Female),
			era:  entities.EraZero,
			assert: func(t *testing.T, analysis entities.ReproductionAnalysis) {
				t.Helper()
				if analysis.ConsanguinityLevel != 1 || analysis.ConsanguinityRisk <= 0 {
					t.Fatalf("expected level 1 consanguinity risk, got %+v", analysis)
				}
				if !hasReproductionPath(analysis.AlternativePaths, entities.ReproductionPathNatural, true) {
					t.Fatalf("expected explicit consanguinity risk path, got %+v", analysis.AlternativePaths)
				}
			},
		},
		{
			name: "level one consanguinity calculates genetic risk",
			a:    fertileSibling("s1", entities.Male),
			b:    fertileSibling("s2", entities.Female),
			era:  entities.EraZero,
			assert: func(t *testing.T, analysis entities.ReproductionAnalysis) {
				t.Helper()
				if analysis.ConsanguinityLevel != 1 {
					t.Fatalf("expected consanguinity level 1, got %+v", analysis)
				}
				if analysis.ConsanguinityRisk != 0.25 {
					t.Fatalf("expected 0.25 genetic risk, got %.2f", analysis.ConsanguinityRisk)
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			entityWorld := &entities.World{
				State: entities.NewWorldState(),
				Pobles: map[string]*entities.Poble{
					tc.a.ID:    tc.a,
					tc.b.ID:    tc.b,
					"observer": fertilePoble("observer", entities.Intersex),
				},
			}
			entityWorld.State.Era = tc.era
			system := entities.NewReproductionSystem(entityWorld, rand.New(rand.NewSource(99)))
			tc.assert(t, system.CanReproduce(tc.a, tc.b))
		})
	}
}

func TestRumourMutation(t *testing.T) {
	rumour := &ai.Rumour{
		ID:               "rumour_test",
		OriginalFactType: ai.RumourFactConflict,
		OriginalContent:  "Noah shouted at Kira near the well",
		CurrentContent:   "Noah shouted at Kira near the well",
		TruthScore:       1.0,
		KnownBy:          []string{"noah"},
		SubjectIDs:       []string{"noah", "kira"},
	}

	for i := 0; i < 10; i++ {
		rumour = ai.MutateRumour(rumour)
	}

	if rumour.TruthScore >= 0.5 {
		t.Fatalf("expected truth score below 0.5 after 10 spreads, got %.2f", rumour.TruthScore)
	}
	if rumour.CurrentContent == rumour.OriginalContent {
		t.Fatalf("expected mutated content, got %q", rumour.CurrentContent)
	}
}

func TestEndingConditions(t *testing.T) {
	enterRepoRoot(t)

	checker := engine.NewEndingChecker()
	for _, tc := range []struct {
		name        string
		endingType  engine.EndingType
		buildWorld  func(t *testing.T) *world.World
		useDetector bool
	}{
		{name: "love", endingType: engine.END_LOVE, buildWorld: loveEndingWorld},
		{name: "dynasty", endingType: engine.END_DYNASTY, buildWorld: dynastyEndingWorld},
		{name: "war", endingType: engine.END_WAR, buildWorld: warEndingWorld},
		{name: "utopia", endingType: engine.END_UTOPIA, buildWorld: utopiaEndingWorld},
		{name: "cult", endingType: engine.END_CULT, buildWorld: cultEndingWorld},
		{name: "alone", endingType: engine.END_ALONE, buildWorld: aloneEndingWorld},
		{name: "reset", endingType: engine.END_RESET, buildWorld: resetEndingWorld},
		{name: "myth", endingType: engine.END_MYTH, buildWorld: mythEndingWorld},
	} {
		t.Run(tc.name, func(t *testing.T) {
			gameWorld := tc.buildWorld(t)
			ending := checker.CheckEndingConditions(gameWorld)
			if ending == nil {
				t.Fatalf("expected %s ending to be detected", tc.endingType)
			}
			if ending.Type != tc.endingType {
				t.Fatalf("expected %s, got %s", tc.endingType, ending.Type)
			}

			generated := engine.GenerateEnding(tc.endingType, gameWorld)
			if generated.Type != tc.endingType || strings.TrimSpace(generated.Title) == "" {
				t.Fatalf("generated ending missing identity: %+v", generated)
			}
			if len(generated.NarrativeChapters) == 0 || strings.TrimSpace(generated.LastWords) == "" {
				t.Fatalf("generated ending missing narrative content: %+v", generated)
			}
		})
	}
}

func TestTemplateSystem(t *testing.T) {
	basePath := t.TempDir()
	writeTemplateFile(t, basePath, "thoughts/random/e2e.txt", `
[TEMPLATE:E2E_A]
tags: neutral, any
weight: 10
---
{self_name} counts the cups again while {target_name} pretends not to notice.
---
[TEMPLATE:E2E_B]
tags: anxious, memory:positive
weight: 9
---
{memory_recent}. {self_name} keeps that sentence under the tongue.
---
[TEMPLATE:E2E_C]
tags: relationship:lover, attraction
weight: 8
---
{target_name} says nothing. That is somehow worse than a speech.
---
[TEMPLATE:E2E_D]
tags: night, negative_valence
weight: 7
---
The house is quiet, except for the part of {self_name} still arguing.
---
[TEMPLATE:E2E_E]
tags: settlement
weight: 6
---
In {settlement_name}, even breakfast has witnesses.
---
`)

	engine := templates.NewTemplateEngine(rand.New(rand.NewSource(123)))
	if err := engine.LoadTemplates(basePath); err != nil {
		t.Fatalf("load test templates: %v", err)
	}

	speaker := testPoble("speaker", "Noah", entities.Male)
	target := testPoble("target", "Kira", entities.Female)
	relationship := entities.NewRelationship(target.ID, entities.RelationshipLover)
	relationship.Attraction = 90
	relationship.Trust = 70
	memory := entities.NewMemory("m1", entities.NewGameTime(1, 8, 0), entities.MemoryPositive, "Kira fixed the roof without bragging")
	worldState := entities.NewWorldState()
	worldState.Population = 2
	worldState.Settlements = []entities.Settlement{entities.NewSettlement("s0", "El Origen", "island_0")}
	ctx := templates.TemplateContext{
		Speaker:                speaker,
		Target:                 target,
		GameTime:               entities.NewGameTime(5, 22, 0),
		RecentMemory:           &memory,
		RelationshipWithTarget: &relationship,
		WorldState:             worldState,
	}

	counts := map[string]int{}
	previous := ""
	immediateRepeats := 0
	for i := 0; i < 1000; i++ {
		template, err := engine.Select("thoughts/random", ctx)
		if err != nil {
			t.Fatalf("select template %d: %v", i, err)
		}
		rendered, err := engine.Render(template, ctx)
		if err != nil {
			t.Fatalf("render template %s: %v", template.ID, err)
		}
		if strings.Contains(rendered, "{") || strings.Contains(rendered, "}") {
			t.Fatalf("unresolved variable in rendered text: %q", rendered)
		}
		if template.ID == previous {
			immediateRepeats++
		}
		previous = template.ID
		counts[template.ID]++
	}

	if immediateRepeats != 0 {
		t.Fatalf("anti-repetition failed with %d immediate repeats", immediateRepeats)
	}
	if len(counts) < 4 {
		t.Fatalf("expected broad distribution across templates, got %+v", counts)
	}
	for id, count := range counts {
		if count > 450 {
			t.Fatalf("template %s dominated selection despite anti-repetition: %+v", id, counts)
		}
	}
}

func enterRepoRoot(t *testing.T) {
	t.Helper()
	root := findRepoRoot(t)
	previous, err := os.Getwd()
	if err != nil {
		t.Fatalf("get cwd: %v", err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir repo root: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(previous)
	})
}

func isolateUserHome(t *testing.T) {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("USERPROFILE", tmp)
}

func findRepoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get cwd: %v", err)
	}
	for i := 0; i < 8; i++ {
		if _, err := os.Stat(filepath.Join(wd, "go.mod")); err == nil {
			return wd
		}
		next := filepath.Dir(wd)
		if next == wd {
			break
		}
		wd = next
	}
	t.Fatal("could not find repo root")
	return ""
}

func runTicks(orchestrator *engine.Orchestrator, hours int) []events.GameEvent {
	current := entities.NewGameTime(0, 0, 0)
	generated := []events.GameEvent{}
	for i := 0; i < hours; i++ {
		previous := current
		current = current.Add(1)
		tick := engine.GameTick{
			CurrentTime: current,
			DeltaHours:  1,
			IsNewDay:    current.Day != previous.Day,
			IsNewWeek:   current.Day != previous.Day && current.Day > 0 && current.Day%7 == 0,
			IsNewMonth:  current.Day != previous.Day && current.Day > 0 && current.Day%30 == 0,
			IsNewYear:   current.Day != previous.Day && current.Day > 0 && current.Day%365 == 0,
		}
		generated = append(generated, orchestrator.OnTick(tick)...)
	}
	return generated
}

func primeFoundersForInteraction(t *testing.T, gameWorld *world.World) {
	t.Helper()
	pobles := gameWorld.GetAllPobles()
	if len(pobles) != 2 {
		t.Fatalf("expected 2 founders, got %d", len(pobles))
	}
	first, second := pobles[0], pobles[1]
	first.Needs.Sex = 95
	second.Needs.Sex = 95
	first.Needs.Belonging = 88
	second.Needs.Belonging = 88
	first.Personality.Horniness = 90
	second.Personality.Horniness = 90

	firstRel := entities.NewRelationship(second.ID, entities.RelationshipLover)
	firstRel.Attraction = 95
	firstRel.Affection = 80
	firstRel.Trust = 75
	firstRel.Familiarity = 90
	secondRel := entities.NewRelationship(first.ID, entities.RelationshipLover)
	secondRel.Attraction = 94
	secondRel.Affection = 82
	secondRel.Trust = 76
	secondRel.Familiarity = 90
	first.Relationships[second.ID] = firstRel
	second.Relationships[first.ID] = secondRel
}

func renderThought(t *testing.T, templateEngine *templates.TemplateEngine, speaker *entities.Poble, gameWorld *world.World) string {
	t.Helper()
	if templateEngine == nil || speaker == nil {
		t.Fatal("template engine and speaker are required")
	}
	ctx := templates.TemplateContext{
		Speaker:    speaker,
		GameTime:   gameWorld.Calendar,
		WorldState: gameWorld.GetWorldState(),
	}
	template, err := templateEngine.Select("thoughts/random", ctx)
	if err != nil {
		t.Fatalf("select thought for %s: %v", speaker.ID, err)
	}
	rendered, err := templateEngine.Render(template, ctx)
	if err != nil {
		t.Fatalf("render thought for %s: %v", speaker.ID, err)
	}
	return rendered
}

func needsChanged(before, after entities.Needs) bool {
	return before.Hunger != after.Hunger ||
		before.Thirst != after.Thirst ||
		before.Sleep != after.Sleep ||
		before.Safety != after.Safety ||
		before.Belonging != after.Belonging ||
		before.Esteem != after.Esteem ||
		before.Sex != after.Sex ||
		before.Power != after.Power ||
		before.Purpose != after.Purpose
}

func hasDeathEventFor(gameWorld *world.World, generated []events.GameEvent, pobleID string) bool {
	for _, event := range generated {
		if eventHasParticipant(event.Participants, pobleID) && isDeathEventType(event.Type) {
			return true
		}
	}
	for _, event := range gameWorld.EventHistory {
		if eventHasParticipant(event.Participants, pobleID) && event.Type == ai.GameEventDeath {
			return true
		}
	}
	return false
}

func isDeathEventType(eventType events.EventType) bool {
	switch eventType {
	case events.EventDeathNatural, events.EventDeathAccident, events.EventDeathMurder, events.EventSuicide:
		return true
	default:
		return false
	}
}

func hasInteraction(generated, feed []events.GameEvent) bool {
	all := append([]events.GameEvent{}, generated...)
	all = append(all, feed...)
	for _, event := range all {
		if len(event.Participants) < 2 {
			continue
		}
		switch event.Type {
		case events.EventFightVerbal, events.EventFightPhysical, events.EventRevelation,
			events.EventSexualEncounter, events.EventRumourSpread, events.EventDecisionPoint,
			events.EventAffairStart, events.EventBetrayalRevealed:
			return true
		}
	}
	return false
}

func eventHasParticipant(participants []string, id string) bool {
	for _, participant := range participants {
		if participant == id {
			return true
		}
	}
	return false
}

func fertilePoble(id string, sex entities.Sex) *entities.Poble {
	poble := entities.NewPoble(id, id, 28, sex)
	poble.Health.Fertility = 1
	poble.Health.HP = 100
	poble.Mental.Stability = 95
	return &poble
}

func fertileSibling(id string, sex entities.Sex) *entities.Poble {
	poble := fertilePoble(id, sex)
	poble.Parents = [2]string{"parent_a", "parent_b"}
	return poble
}

func hasReproductionPath(paths []entities.ReproductionPath, pathType entities.ReproductionPathType, available bool) bool {
	for _, path := range paths {
		if path.Type == pathType && path.Availability == available {
			return true
		}
	}
	return false
}

func testPoble(id, name string, sex entities.Sex) *entities.Poble {
	poble := entities.NewPoble(id, name, 30, sex)
	poble.Archetype = entities.ArchetypeLover
	poble.CurrentMood = entities.MoodAnxious
	poble.EmotionalState.CurrentMood = entities.MoodAnxious
	poble.EmotionalState.Valence = -0.4
	poble.EmotionalState.ActiveEmotions = []entities.EmotionType{entities.EmotionAnxiety}
	return &poble
}

func loveEndingWorld(t *testing.T) *world.World {
	gameWorld := worldWithPobles(t, 1,
		endingPoble("love_survivor", "Alba", entities.Female, true),
		endingPoble("love_dead", "Noe", entities.Male, false),
	)
	gameWorld.EventHistory = append(gameWorld.EventHistory,
		ai.GameEvent{ID: "death_noe", Type: ai.GameEventDeath, Time: entities.NewGameTime(7, 0, 0), Description: "Noe died.", Tags: []string{"death"}, Severity: 0.9},
		ai.GameEvent{ID: "no_reproduction", Type: ai.GameEventGeneric, Time: entities.NewGameTime(8, 0, 0), Description: "Alba chose no reproduction.", Tags: []string{"no_reproduction"}, Severity: 0.8},
	)
	return gameWorld
}

func dynastyEndingWorld(t *testing.T) *world.World {
	pobles := make([]*entities.Poble, 0, 10)
	for i := 1; i <= 10; i++ {
		poble := endingPoble(fmt.Sprintf("dynasty_%02d", i), fmt.Sprintf("Dynasty %02d", i), entities.Female, true)
		if i > 1 {
			poble.Parents = [2]string{fmt.Sprintf("dynasty_%02d", i-1), ""}
		}
		pobles = append(pobles, poble)
	}
	gameWorld := worldWithPobles(t, 2, pobles...)
	leader := pobles[len(pobles)-1].ID
	gameWorld.Government = &world.GovernmentSystem{Type: world.GovernmentMonarchy, Leader: &leader}
	return gameWorld
}

func warEndingWorld(t *testing.T) *world.World {
	gameWorld := worldWithPobles(t, 3,
		endingPoble("war_a", "Iris", entities.Female, false),
		endingPoble("war_b", "Luca", entities.Male, false),
	)
	gameWorld.EventHistory = append(gameWorld.EventHistory, ai.GameEvent{
		ID: "final_war", Type: ai.GameEventConflict, Time: entities.NewGameTime(30, 0, 0),
		Description: "The final war killed everyone.", Tags: []string{"war", "combat"}, Severity: 1,
	})
	return gameWorld
}

func utopiaEndingWorld(t *testing.T) *world.World {
	pobles := make([]*entities.Poble, 0, 500)
	for i := 0; i < 500; i++ {
		poble := endingPoble(fmt.Sprintf("utopia_%03d", i), fmt.Sprintf("Utopia %03d", i), entities.Female, true)
		poble.Mental.Stability = 92
		pobles = append(pobles, poble)
	}
	gameWorld := worldWithPobles(t, 4, pobles...)
	gameWorld.Calendar = entities.NewGameTime(50*365+5, 0, 0)
	gameWorld.Era = entities.EraThree
	gameWorld.State.Era = entities.EraThree
	return gameWorld
}

func cultEndingWorld(t *testing.T) *world.World {
	pobles := make([]*entities.Poble, 0, 10)
	for i := 0; i < 10; i++ {
		poble := endingPoble(fmt.Sprintf("cult_%02d", i), fmt.Sprintf("Cult %02d", i), entities.Female, true)
		poble.Superstitions = []entities.Superstition{entities.NewSuperstition("sun", "La Luz Manda", "dawn", "kneel", 90)}
		if i == 0 {
			poble.Archetype = entities.ArchetypeProphet
		}
		pobles = append(pobles, poble)
	}
	gameWorld := worldWithPobles(t, 5, pobles...)
	leader := pobles[0].ID
	gameWorld.Government = &world.GovernmentSystem{
		Type:   world.GovernmentTheocracy,
		Leader: &leader,
		Laws:   []world.Law{{ID: "law_faith", Description: "religious ritual is law", IsEnforced: true}},
	}
	return gameWorld
}

func aloneEndingWorld(t *testing.T) *world.World {
	return worldWithPobles(t, 6, endingPoble("alone", "Mara", entities.Female, true))
}

func resetEndingWorld(t *testing.T) *world.World {
	gameWorld := worldWithPobles(t, 7,
		endingPoble("reset_a", "Kira", entities.Female, true),
		endingPoble("reset_b", "Noah", entities.Male, true),
		endingPoble("reset_dead", "Mara", entities.Female, false),
	)
	gameWorld.EventHistory = append(gameWorld.EventHistory, ai.GameEvent{
		ID: "civilization_collapse", Type: ai.GameEventThreat, Time: entities.NewGameTime(70, 0, 0),
		Description: "Civilization collapsed and left two survivors.", Tags: []string{"collapse"}, Severity: 0.95,
	})
	return gameWorld
}

func mythEndingWorld(t *testing.T) *world.World {
	pobles := []*entities.Poble{
		endingPoble("myth_a", "Lia", entities.Female, true),
		endingPoble("myth_b", "Ren", entities.Male, true),
	}
	for _, poble := range pobles {
		poble.Parents = [2]string{"forgotten_parent_a", "forgotten_parent_b"}
		poble.DayOfBirth = entities.NewGameTime(20, 0, 0)
	}
	gameWorld := worldWithPobles(t, 8, pobles...)
	gameWorld.Era = entities.EraFour
	gameWorld.State.Era = entities.EraFour
	gameWorld.TechTree.Unlocked[world.TechAIResearch] = true
	return gameWorld
}

func worldWithPobles(t *testing.T, seed int64, pobles ...*entities.Poble) *world.World {
	t.Helper()
	gameWorld := world.NewWorld(seed)
	for index, poble := range pobles {
		if !gameWorld.AddPoble(poble, world.Location{IslandID: "island_0", X: index, Y: index}) {
			t.Fatalf("add poble %s", poble.ID)
		}
	}
	return gameWorld
}

func endingPoble(id, name string, sex entities.Sex, alive bool) *entities.Poble {
	poble := entities.NewPoble(id, name, 35, sex)
	poble.IsAlive = alive
	poble.Mental.Stability = 90
	return &poble
}

func writeTemplateFile(t *testing.T, basePath, relativePath, content string) {
	t.Helper()
	path := filepath.Join(basePath, relativePath)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir template dir: %v", err)
	}
	if err := os.WriteFile(path, []byte(strings.TrimSpace(content)+"\n"), 0o644); err != nil {
		t.Fatalf("write template file: %v", err)
	}
}
