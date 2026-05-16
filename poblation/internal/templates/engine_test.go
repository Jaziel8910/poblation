package templates

import (
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/user/poblation/internal/entities"
)

func TestSelectUsesWeightedDistribution(t *testing.T) {
	basePath := t.TempDir()
	writeTemplateFile(t, basePath, "thoughts/random/pool.txt", `
[TEMPLATE:LOW_A]
tags: neutral
weight: 1
---
low a {self_name}
---
[TEMPLATE:LOW_B]
tags: neutral
weight: 1
---
low b {self_name}
---
[TEMPLATE:LOW_C]
tags: neutral
weight: 1
---
low c {self_name}
---
[TEMPLATE:HIGH]
tags: neutral
weight: 20
---
high {self_name}
---
`)

	engine := NewTemplateEngine(rand.New(rand.NewSource(7)))
	if err := engine.LoadTemplates(basePath); err != nil {
		t.Fatalf("load templates: %v", err)
	}

	speaker := testTemplatePoble("noah", entities.ArchetypeCustom)
	ctx := TemplateContext{Speaker: &speaker}
	counts := map[string]int{}
	for i := 0; i < 1200; i++ {
		template, err := engine.Select("thoughts/random", ctx)
		if err != nil {
			t.Fatalf("select template: %v", err)
		}
		counts[template.ID]++
	}

	if counts["HIGH"] <= counts["LOW_A"] || counts["HIGH"] <= counts["LOW_B"] || counts["HIGH"] <= counts["LOW_C"] {
		t.Fatalf("expected high-weight template to beat each low template, got %#v", counts)
	}
}

func TestRenderResolvesVariablesWithoutPanics(t *testing.T) {
	engine := NewTemplateEngine(rand.New(rand.NewSource(3)))
	speaker := testTemplatePoble("noah", entities.ArchetypeLover)
	target := testTemplatePoble("kira", entities.ArchetypeGhost)
	target.Sex = entities.Female
	speaker.Children = []string{"Liu"}
	speaker.Secrets = []entities.Secret{entities.NewSecret("s1", entities.SecretObsession, "hidden")}
	relationship := entities.NewRelationship(target.ID, entities.RelationshipLover)
	relationship.Trust = 82
	relationship.Attraction = 91
	memory := entities.NewMemory("m1", entities.NewGameTime(2, 8, 0), entities.MemoryRomantic, "Kira smiled at the fire")
	worldState := entities.NewWorldState()
	worldState.Population = 17
	worldState.Settlements = []entities.Settlement{entities.NewSettlement("s", "El Origen", "i")}

	template := &Template{
		ID: "VARS",
		RawText: `{self_name} vio a {target_name}. {target_pronoun} estaba en {location}.
{memory_recent}, {memory_old}, {days_ago:memory}. {time_of_day}, {weather}.
{relationship_status}. {population}. {settlement_name}. {secret_hint}. {child_name}.
{if:high_horniness}extra{endif} {quiet|sharp}`,
	}
	rendered, err := engine.Render(template, TemplateContext{
		Speaker:                &speaker,
		Target:                 &target,
		Location:               "la costa",
		GameTime:               entities.NewGameTime(5, 21, 0),
		RecentMemory:           &memory,
		OldMemory:              &memory,
		RelationshipWithTarget: &relationship,
		WorldState:             worldState,
		ExtraVars:              map[string]string{"weather": "lluvia rara"},
	})
	if err != nil {
		t.Fatalf("render template: %v", err)
	}
	if strings.Contains(rendered, "{") || strings.Contains(rendered, "}") {
		t.Fatalf("expected all variables to resolve, got %q", rendered)
	}
	if !strings.Contains(rendered, "Noah") || !strings.Contains(rendered, "Kira") || !strings.Contains(rendered, "17") {
		t.Fatalf("expected rendered text to include resolved context, got %q", rendered)
	}
}

func TestArchetypeFilterOnlyAllowsMatchingSpeaker(t *testing.T) {
	basePath := t.TempDir()
	writeTemplateFile(t, basePath, "dialogues/greeting/archetypes.txt", `
[TEMPLATE:RULER_ONLY]
tags: control
weight: 10
archetypes: RULER
---
Ruler text
---
[TEMPLATE:LOVER_ONLY]
tags: longing
weight: 10
archetypes: LOVER
---
Lover text
---
`)

	engine := NewTemplateEngine(rand.New(rand.NewSource(9)))
	if err := engine.LoadTemplates(basePath); err != nil {
		t.Fatalf("load templates: %v", err)
	}

	speaker := testTemplatePoble("kira", entities.ArchetypeLover)
	ctx := TemplateContext{Speaker: &speaker}
	for i := 0; i < 10; i++ {
		template, err := engine.Select("dialogues/greeting", ctx)
		if err != nil {
			t.Fatalf("select template: %v", err)
		}
		if template.ID != "LOVER_ONLY" {
			t.Fatalf("expected lover-only template, got %s", template.ID)
		}
	}
}

func TestAntiRepetitionAvoidsImmediateRepeat(t *testing.T) {
	basePath := t.TempDir()
	writeTemplateFile(t, basePath, "thoughts/night/two.txt", `
[TEMPLATE:FIRST]
tags: night
weight: 10
---
First
---
[TEMPLATE:SECOND]
tags: night
weight: 10
---
Second
---
`)

	engine := NewTemplateEngine(rand.New(rand.NewSource(1)))
	if err := engine.LoadTemplates(basePath); err != nil {
		t.Fatalf("load templates: %v", err)
	}

	speaker := testTemplatePoble("noah", entities.ArchetypeGhost)
	ctx := TemplateContext{Speaker: &speaker}
	first, err := engine.Select("thoughts/night", ctx)
	if err != nil {
		t.Fatalf("first select: %v", err)
	}
	second, err := engine.Select("thoughts/night", ctx)
	if err != nil {
		t.Fatalf("second select: %v", err)
	}
	if first.ID == second.ID {
		t.Fatalf("expected anti-repetition to avoid immediate repeat, got %s twice", first.ID)
	}
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

func testTemplatePoble(id string, archetype entities.ArchetypeID) entities.Poble {
	poble := entities.NewPoble(id, strings.Title(id), 30, entities.Male)
	poble.Archetype = archetype
	poble.CurrentMood = entities.MoodAnxious
	poble.EmotionalState.CurrentMood = entities.MoodAnxious
	poble.EmotionalState.ActiveEmotions = []entities.EmotionType{entities.EmotionAnxiety}
	poble.Personality.Horniness = 80
	poble.Needs.Sex = 80
	return poble
}
