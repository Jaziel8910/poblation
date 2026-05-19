package templates

import (
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestRepositoryTotalTemplateFloor(t *testing.T) {
	engine := NewTemplateEngine(rand.New(rand.NewSource(41)))
	if err := engine.LoadTemplates("../../templates"); err != nil {
		t.Fatalf("load all repository templates: %v", err)
	}

	stats := engine.GetStats()
	if stats.Total < 1800 {
		t.Fatalf("expected at least 1800 curated templates for beta floor, got %d", stats.Total)
	}
	requiredTopLevel := map[string]int{
		"thoughts":  600,
		"dialogues": 600,
		"dreams":    100,
		"diary":     100,
		"narrator":  160,
		"reactions": 90,
		"letters":   50,
	}
	for topLevel, minimum := range requiredTopLevel {
		total := 0
		for category, count := range stats.TotalByCategory {
			if category == topLevel || len(category) > len(topLevel) && category[:len(topLevel)+1] == topLevel+"/" {
				total += count
			}
		}
		if total < minimum {
			t.Fatalf("expected top-level category %s to have at least %d templates, got %d", topLevel, minimum, total)
		}
	}
}

func TestRepositoryTemplateIDsAreUniqueAndClean(t *testing.T) {
	header := regexp.MustCompile(`^\[TEMPLATE:([A-Za-z0-9_.:_-]+)\]$`)
	seen := map[string]string{}
	banned := []string{
		"felt a wave of",
		"couldn't help but",
		"deep down",
		"a part of me",
		"something inside",
		"eyes filled with",
		"heart racing",
		"took a deep breath",
		"couldn't shake the feeling",
	}

	err := filepath.WalkDir("../../templates", func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || strings.ToLower(filepath.Ext(path)) != ".txt" {
			return nil
		}
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		lower := strings.ToLower(string(content))
		for _, phrase := range banned {
			if strings.Contains(lower, phrase) {
				t.Fatalf("template file %s contains banned phrase %q", path, phrase)
			}
		}
		for _, line := range strings.Split(strings.ReplaceAll(string(content), "\r\n", "\n"), "\n") {
			match := header.FindStringSubmatch(strings.TrimSpace(line))
			if match == nil {
				continue
			}
			if previous, ok := seen[match[1]]; ok {
				t.Fatalf("duplicate template id %s in %s and %s", match[1], previous, path)
			}
			seen[match[1]] = path
		}
		return nil
	})
	if err != nil {
		t.Fatalf("scan templates: %v", err)
	}
	if len(seen) < 1800 {
		t.Fatalf("expected scan to find at least 1800 curated template ids, got %d", len(seen))
	}
}

func TestRepositoryRejectsGeneratedFillerPacks(t *testing.T) {
	for _, path := range []string{
		"../../templates/dialogues/generated",
		"../../templates/diary/generated",
		"../../templates/dreams/generated",
		"../../templates/letters/generated",
		"../../templates/narrator/generated",
		"../../templates/reactions/generated",
		"../../templates/thoughts/generated",
		"../../templates/rumours/generated",
		"../../templates/newspaper/generated",
	} {
		if _, err := os.Stat(path); err == nil {
			t.Fatalf("generated filler pack must not exist: %s", path)
		}
	}
}

func TestRepositoryThoughtTemplatesLoad(t *testing.T) {
	engine := NewTemplateEngine(rand.New(rand.NewSource(42)))
	if err := engine.LoadTemplates("../../templates/thoughts"); err != nil {
		t.Fatalf("load repository thought templates: %v", err)
	}

	stats := engine.GetStats()
	if stats.Total < 390 {
		t.Fatalf("expected at least 390 thought templates, got %d", stats.Total)
	}
	minimums := map[string]int{
		"about_other/grief":    25,
		"about_self/desire":    20,
		"by_archetype/ghost":   25,
		"by_archetype/prophet": 25,
		"about_other/jealousy": 30,
	}
	for category, count := range stats.TotalByCategory {
		minimum := 30
		if value, ok := minimums[category]; ok {
			minimum = value
		}
		if count < minimum {
			t.Fatalf("expected category %s to have at least %d templates, got %d", category, minimum, count)
		}
	}
}

func TestRepositoryDialogueTemplatesLoad(t *testing.T) {
	engine := NewTemplateEngine(rand.New(rand.NewSource(43)))
	if err := engine.LoadTemplates("../../templates/dialogues"); err != nil {
		t.Fatalf("load repository dialogue templates: %v", err)
	}

	requiredCategories := []string{
		"greeting/post_fight",
		"greeting/post_sex",
		"argument/petty",
		"argument/escalating",
		"flirt/subtle",
		"flirt/rejected",
		"confession/secret",
		"confession/love",
		"gossip/general",
		"reconciliation/after_betrayal",
	}
	stats := engine.GetStats()
	for _, category := range requiredCategories {
		if count := stats.TotalByCategory[category]; count < 25 {
			t.Fatalf("expected category %s to have at least 25 templates, got %d", category, count)
		}
	}
	specialCategories := map[string]int{
		"confession/betrayal": 20,
		"threat/general":      20,
		"sex/tension":         25,
		"sex/aftermath":       25,
		"political/general":   10,
		"religious/general":   10,
	}
	for category, minimum := range specialCategories {
		if count := stats.TotalByCategory[category]; count < minimum {
			t.Fatalf("expected dialogue category %s to have at least %d templates, got %d", category, minimum, count)
		}
	}

	encounterCategories := map[string]int{
		"encounter/tender/general":      20,
		"encounter/passionate/general":  20,
		"encounter/desperate/general":   20,
		"encounter/angry/general":       20,
		"encounter/curious/general":     20,
		"encounter/secret/general":      20,
		"encounter/complicated/general": 20,
		"encounter/last/general":        20,
		"encounter/first_ever/general":  20,
		"encounter/aftermath/general":   20,
	}
	for category, minimum := range encounterCategories {
		if count := stats.TotalByCategory[category]; count < minimum {
			t.Fatalf("expected encounter category %s to have at least %d templates, got %d", category, minimum, count)
		}
	}
}

func TestRepositoryDreamTemplatesLoad(t *testing.T) {
	engine := NewTemplateEngine(rand.New(rand.NewSource(44)))
	if err := engine.LoadTemplates("../../templates/dreams"); err != nil {
		t.Fatalf("load repository dream templates: %v", err)
	}

	requiredCategories := map[string]int{
		"wish_fulfillment/general": 25,
		"nightmare/general":        25,
		"nonsense/general":         30,
		"prophetic/general":        15,
		"erotic/general":           30,
	}
	stats := engine.GetStats()
	for category, minimum := range requiredCategories {
		if count := stats.TotalByCategory[category]; count < minimum {
			t.Fatalf("expected category %s to have at least %d templates, got %d", category, minimum, count)
		}
	}
}

func TestRepositoryDiaryTemplatesLoad(t *testing.T) {
	engine := NewTemplateEngine(rand.New(rand.NewSource(45)))
	if err := engine.LoadTemplates("../../templates/diary"); err != nil {
		t.Fatalf("load repository diary templates: %v", err)
	}

	requiredCategories := map[string]int{
		"daily/general":          30,
		"heartbreak/general":     25,
		"secret_keeping/general": 20,
		"planning/general":       20,
	}
	stats := engine.GetStats()
	for category, minimum := range requiredCategories {
		if count := stats.TotalByCategory[category]; count < minimum {
			t.Fatalf("expected category %s to have at least %d templates, got %d", category, minimum, count)
		}
	}
}

func TestRepositoryLetterTemplatesLoad(t *testing.T) {
	engine := NewTemplateEngine(rand.New(rand.NewSource(48)))
	if err := engine.LoadTemplates("../../templates/letters"); err != nil {
		t.Fatalf("load repository letter templates: %v", err)
	}

	requiredCategories := map[string]int{
		"love/general":    15,
		"hate/general":    10,
		"apology/general": 8,
	}
	stats := engine.GetStats()
	for category, minimum := range requiredCategories {
		if count := stats.TotalByCategory[category]; count < minimum {
			t.Fatalf("expected letter category %s to have at least %d templates, got %d", category, minimum, count)
		}
	}
}

func TestRepositoryNarratorTemplatesLoad(t *testing.T) {
	engine := NewTemplateEngine(rand.New(rand.NewSource(46)))
	if err := engine.LoadTemplates("../../templates/narrator"); err != nil {
		t.Fatalf("load repository narrator templates: %v", err)
	}

	requiredCategories := map[string]int{
		"death/natural":                         20,
		"death/murder":                          15,
		"death/illness":                         15,
		"death/suicide":                         10,
		"birth/general":                         20,
		"disaster/storm":                        10,
		"disaster/earthquake":                   10,
		"disaster/plague":                       10,
		"end_of_game/love":                      5,
		"end_of_game/dynasty":                   4,
		"end_of_game/war":                       4,
		"end_of_game/utopia":                    4,
		"end_of_game/cult":                      4,
		"end_of_game/alone":                     4,
		"end_of_game/reset":                     4,
		"end_of_game/myth":                      4,
		"end_of_game/plague/chapter1/general":   1,
		"end_of_game/monopoly/chapter1/general": 1,
	}
	stats := engine.GetStats()
	for category, minimum := range requiredCategories {
		if count := stats.TotalByCategory[category]; count < minimum {
			t.Fatalf("expected narrator category %s to have at least %d templates, got %d", category, minimum, count)
		}
	}
}

func TestRepositoryReactionTemplatesLoad(t *testing.T) {
	engine := NewTemplateEngine(rand.New(rand.NewSource(47)))
	if err := engine.LoadTemplates("../../templates/reactions"); err != nil {
		t.Fatalf("load repository reaction templates: %v", err)
	}

	requiredCategories := map[string]int{
		"death_of_loved_one":      8,
		"betrayal":                8,
		"discovery":               8,
		"birth":                   8,
		"violence/fight_physical": 5,
		"war/general":             8,
		"illness/general":         8,
		"weather/general":         8,
	}
	stats := engine.GetStats()
	for category, minimum := range requiredCategories {
		if count := stats.TotalByCategory[category]; count < minimum {
			t.Fatalf("expected reaction category %s to have at least %d templates, got %d", category, minimum, count)
		}
	}
}
