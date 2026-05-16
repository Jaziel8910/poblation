package tests

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/user/poblation/internal/ai"
	"github.com/user/poblation/internal/engine"
	"github.com/user/poblation/internal/entities"
	"github.com/user/poblation/internal/templates"
)

func TestLongSimulation(t *testing.T) {
	enterRepoRoot(t)
	isolateUserHome(t)

	orchestrator := engine.NewOrchestrator(engine.OrchestratorOptions{Debug: false, Speed: 1})
	if err := orchestrator.Init(515151); err != nil {
		t.Fatalf("init orchestrator: %v", err)
	}
	defer orchestrator.Stop()

	start := time.Now()
	events := runTicks(orchestrator, 30*24)
	elapsed := time.Since(start)
	msPerTick := float64(elapsed.Milliseconds()) / float64(30*24)

	if orchestrator.World().GetPopulation() == 0 {
		t.Fatalf("population reached 0 after long simulation; events=%+v", events)
	}
	if msPerTick >= 50 {
		t.Fatalf("expected < 50ms per tick, got %.2fms over %s", msPerTick, elapsed)
	}
	t.Logf("long simulation generated %d events in %s (%.2fms/tick)", len(events), elapsed, msPerTick)
}

func BenchmarkDecisionEngine(b *testing.B) {
	pobles := benchmarkPobles(50)
	world := benchmarkDecisionWorld{
		pobles: pobles,
		now:    entities.NewGameTime(10, 12, 0),
	}
	engines := make([]*ai.DecisionEngine, 0, len(pobles))
	for i, poble := range pobles {
		engines = append(engines, ai.NewDecisionEngine(poble, world, rand.New(rand.NewSource(int64(i+1)))))
	}

	start := time.Now()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, decisionEngine := range engines {
			actions := decisionEngine.Decide(1)
			if len(actions) == 0 {
				b.Fatal("expected decision engine to produce actions")
			}
		}
	}
	b.StopTimer()

	elapsed := time.Since(start)
	nsPerPoble := float64(elapsed.Nanoseconds()) / float64(maxInt(1, b.N*len(engines)))
	b.ReportMetric(nsPerPoble/1000, "us_per_poble")
	if nsPerPoble >= float64(time.Millisecond) {
		b.Fatalf("decision engine too slow: %.2fus per poble", nsPerPoble/1000)
	}
}

func BenchmarkTemplateSelect(b *testing.B) {
	basePath := b.TempDir()
	writeBenchmarkTemplateFile(b, basePath, "thoughts/random/bench.txt", `
[TEMPLATE:BENCH_A]
tags: neutral, any
weight: 10
---
{self_name} checks the door.
---
[TEMPLATE:BENCH_B]
tags: neutral, any
weight: 10
---
{self_name} counts the water.
---
[TEMPLATE:BENCH_C]
tags: neutral, any
weight: 10
---
{self_name} hears {target_name} outside.
---
[TEMPLATE:BENCH_D]
tags: anxious, any
weight: 9
---
The room keeps one old argument alive.
---
[TEMPLATE:BENCH_E]
tags: settlement, any
weight: 8
---
{settlement_name} is awake too early.
---
`)

	templateEngine := templates.NewTemplateEngine(rand.New(rand.NewSource(404)))
	if err := templateEngine.LoadTemplates(basePath); err != nil {
		b.Fatalf("load templates: %v", err)
	}

	speaker := testPoble("bench_speaker", "Noah", entities.Male)
	target := testPoble("bench_target", "Kira", entities.Female)
	worldState := entities.NewWorldState()
	worldState.Population = 2
	worldState.Settlements = []entities.Settlement{entities.NewSettlement("s0", "El Origen", "island_0")}
	ctx := templates.TemplateContext{
		Speaker:    speaker,
		Target:     target,
		WorldState: worldState,
	}

	iterations := b.N
	if iterations < 10000 {
		iterations = 10000
	}
	latencies := make([]int64, iterations)

	b.ResetTimer()
	for i := 0; i < iterations; i++ {
		start := time.Now()
		template, err := templateEngine.Select("thoughts/random", ctx)
		if err != nil {
			b.Fatalf("select template: %v", err)
		}
		if _, err := templateEngine.Render(template, ctx); err != nil {
			b.Fatalf("render template: %v", err)
		}
		latencies[i] = time.Since(start).Nanoseconds()
	}
	b.StopTimer()

	sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })
	p50 := percentileLatency(latencies, 0.50)
	p95 := percentileLatency(latencies, 0.95)
	p99 := percentileLatency(latencies, 0.99)
	b.ReportMetric(float64(p50)/1000, "p50_us")
	b.ReportMetric(float64(p95)/1000, "p95_us")
	b.ReportMetric(float64(p99)/1000, "p99_us")
	if p95 >= int64(500*time.Microsecond) {
		b.Fatalf("template select/render p95 too slow: %.3fms", float64(p95)/float64(time.Millisecond))
	}
}

type benchmarkDecisionWorld struct {
	pobles []*entities.Poble
	now    entities.GameTime
}

func (w benchmarkDecisionWorld) GetAllPobles() []*entities.Poble {
	return w.pobles
}

func (w benchmarkDecisionWorld) GetActiveEvents() []ai.GameEvent {
	return nil
}

func (w benchmarkDecisionWorld) GetCurrentTime() entities.GameTime {
	return w.now
}

func (w benchmarkDecisionWorld) GetProximityScore(fromID, targetID string) float32 {
	return 4
}

func benchmarkPobles(count int) []*entities.Poble {
	archetypes := []entities.ArchetypeID{
		entities.ArchetypeRuler,
		entities.ArchetypeLover,
		entities.ArchetypeJester,
		entities.ArchetypeSage,
		entities.ArchetypeRebel,
		entities.ArchetypeCaretaker,
		entities.ArchetypeVillain,
		entities.ArchetypeGhost,
		entities.ArchetypeAddict,
		entities.ArchetypeProphet,
		entities.ArchetypeSchemer,
		entities.ArchetypeInnocent,
		entities.ArchetypeWarrior,
		entities.ArchetypeDrifter,
		entities.ArchetypeMirror,
	}

	pobles := make([]*entities.Poble, 0, count)
	for i := 0; i < count; i++ {
		sex := entities.Female
		if i%2 == 0 {
			sex = entities.Male
		}
		poble := entities.NewPoble(fmt.Sprintf("bench_%02d", i), fmt.Sprintf("Bench %02d", i), 25+(i%20), sex)
		poble.Archetype = archetypes[i%len(archetypes)]
		poble.Personality.Openness = float32(30 + (i*7)%70)
		poble.Personality.Extraversion = float32(25 + (i*11)%70)
		poble.Personality.Agreeableness = float32(25 + (i*13)%70)
		poble.Personality.Neuroticism = float32(20 + (i*17)%75)
		poble.Personality.Ambition = float32(30 + (i*19)%70)
		poble.Personality.Horniness = float32(20 + (i*23)%75)
		poble.Needs.Belonging = float32(35 + (i*5)%60)
		poble.Needs.Power = float32(30 + (i*3)%70)
		poble.Needs.Purpose = float32(40 + (i*2)%55)
		pobles = append(pobles, &poble)
	}
	for _, source := range pobles {
		for _, target := range pobles {
			if source.ID == target.ID {
				continue
			}
			relationship := entities.NewRelationship(target.ID, entities.RelationshipAcquaintance)
			relationship.Familiarity = 60
			relationship.Trust = 45
			relationship.Attraction = 35
			relationship.Resentment = 10
			source.Relationships[target.ID] = relationship
		}
	}
	return pobles
}

func percentileLatency(sorted []int64, percentile float64) int64 {
	if len(sorted) == 0 {
		return 0
	}
	index := int(float64(len(sorted)-1) * percentile)
	return sorted[index]
}

func writeBenchmarkTemplateFile(b *testing.B, basePath, relativePath, content string) {
	b.Helper()
	path := filepath.Join(basePath, relativePath)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		b.Fatalf("mkdir template dir: %v", err)
	}
	if err := os.WriteFile(path, []byte(strings.TrimSpace(content)+"\n"), 0o644); err != nil {
		b.Fatalf("write template file: %v", err)
	}
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
