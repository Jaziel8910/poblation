package main

import (
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/user/poblation/internal/templates"
)

var (
	headerPattern   = regexp.MustCompile(`^\[TEMPLATE:([A-Za-z0-9_.:-]+)\]$`)
	variablePattern = regexp.MustCompile(`\{([^{}]+)\}`)
	validVariables  = map[string]struct{}{
		"self_name": {}, "dreamer_name": {}, "target_name": {}, "target_pronoun": {},
		"memory_recent": {}, "memory_old": {}, "location": {}, "time_of_day": {},
		"weather": {}, "relationship_status": {}, "population": {}, "settlement_name": {},
		"secret_hint": {}, "child_name": {}, "age": {}, "surviving_relationships": {},
		"legacy_hint": {}, "victim_name": {}, "killer_name": {}, "method_hint": {},
		"days_ill": {}, "last_words_hint": {}, "trigger_hint": {}, "mother_name": {},
		"father_name": {}, "circumstances": {}, "event_description": {}, "specific_grievance": {},
		"the_thing_wanted": {}, "future_event_hint": {}, "survivor_name": {}, "deceased_name": {},
		"years_together": {}, "shared_history_hint": {}, "last_thing_they_said": {},
		"location_atmosphere": {}, "location_memory": {}, "location_name": {}, "power_dynamic": {},
		"encounter_mood": {}, "encounter_type": {}, "self_vocab_filler": {}, "settlement_slang": {},
		"self_nickname": {}, "target_nickname": {}, "profanity_mild": {}, "profanity_strong": {},
		"self_expletive": {}, "self_love_lang": {}, "self_quirk": {}, "self_defense": {},
		"self_superstition": {}, "ending_title": {}, "ending_type": {}, "ending_day": {},
		"ending_population": {}, "ending_total_pobles": {}, "ending_total_deaths": {},
		"ending_total_births": {}, "ending_total_wars": {}, "ending_total_affairs": {},
		"ending_total_secrets": {}, "ending_total_rumours": {}, "ending_subject": {},
		"ending_second_subject": {}, "ending_survivor": {}, "ending_deceased": {},
		"ending_first_couple": {}, "ending_last_event": {}, "ending_dramatic_event": {},
		"ending_longest_lived": {}, "ending_most_loved": {}, "ending_most_hated": {},
		"ending_richest": {}, "ending_mutated_rumour": {}, "ending_settlement": {},
		"ending_era": {}, "ending_tech": {}, "ending_belief": {}, "ending_leader": {},
		"ending_prophet": {}, "ending_family": {},
	}
)

type templateEntry struct {
	ID        string
	Category  string
	File      string
	Line      int
	Variables []string
}

func main() {
	root := flag.String("root", "templates", "template root directory")
	threshold := flag.Int("threshold", 20, "coverage threshold")
	flag.Parse()

	entries, err := collectTemplates(*root)
	if err != nil {
		fatal(err)
	}

	if err := validateUniqueIDs(entries); err != nil {
		fatal(err)
	}

	if err := validateVariables(entries); err != nil {
		fatal(err)
	}

	engine := templates.NewTemplateEngine(nil)
	if err := engine.LoadTemplates(*root); err != nil {
		fatal(err)
	}

	printCounts(entries)
	printCoverage(entries, *threshold)
}

func collectTemplates(root string) ([]templateEntry, error) {
	entries := []templateEntry{}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() || strings.ToLower(filepath.Ext(path)) != ".txt" {
			return nil
		}
		fileEntries, fileErr := parseTemplateFile(root, path)
		if fileErr != nil {
			return fileErr
		}
		entries = append(entries, fileEntries...)
		return nil
	})
	return entries, err
}

func parseTemplateFile(root, path string) ([]templateEntry, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	category, err := categoryFromPath(root, path)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(strings.ReplaceAll(string(content), "\r\n", "\n"), "\n")
	entries := []templateEntry{}
	for i := 0; i < len(lines); {
		line := strings.TrimSpace(strings.TrimPrefix(lines[i], "\ufeff"))
		if line == "" || strings.HasPrefix(line, "#") {
			i++
			continue
		}
		match := headerPattern.FindStringSubmatch(line)
		if match == nil {
			i++
			continue
		}
		bodyStart := -1
		bodyEnd := -1
		vars := []string{}
		j := i + 1
		for ; j < len(lines); j++ {
			trimmed := strings.TrimSpace(lines[j])
			if trimmed == "---" {
				bodyStart = j + 1
				break
			}
			if trimmed == "" || strings.HasPrefix(trimmed, "#") {
				continue
			}
			if !strings.Contains(trimmed, ":") {
				return nil, fmt.Errorf("%s:%d invalid metadata line %q", path, j+1, trimmed)
			}
		}
		if bodyStart == -1 {
			return nil, fmt.Errorf("%s: missing body separator", path)
		}
		for k := bodyStart; k < len(lines); k++ {
			if strings.TrimSpace(lines[k]) == "---" {
				bodyEnd = k
				break
			}
		}
		if bodyEnd == -1 {
			return nil, fmt.Errorf("%s: missing closing separator", path)
		}
		body := strings.Join(lines[bodyStart:bodyEnd], "\n")
		for _, v := range variablePattern.FindAllStringSubmatch(body, -1) {
			name := strings.TrimSpace(v[1])
			if name == "" || strings.HasPrefix(name, "if:") || name == "endif" || strings.Contains(name, "|") {
				continue
			}
			if _, ok := validVariables[name]; ok {
				vars = append(vars, name)
				continue
			}
			if strings.HasPrefix(name, "days_ago:") {
				continue
			}
			return nil, fmt.Errorf("%s: invalid variable %q", path, name)
		}
		entries = append(entries, templateEntry{
			ID:        match[1],
			Category:  category,
			File:      path,
			Line:      i + 1,
			Variables: uniqueStrings(vars),
		})
		i = bodyEnd + 1
	}
	return entries, nil
}

func validateUniqueIDs(entries []templateEntry) error {
	seen := map[string]templateEntry{}
	for _, entry := range entries {
		if previous, ok := seen[entry.ID]; ok {
			return fmt.Errorf("duplicate template id %q in %s and %s", entry.ID, previous.File, entry.File)
		}
		seen[entry.ID] = entry
	}
	return nil
}

func validateVariables(entries []templateEntry) error {
	missing := []string{}
	for _, entry := range entries {
		for _, variable := range entry.Variables {
			if _, ok := validVariables[variable]; !ok && !strings.HasPrefix(variable, "days_ago:") {
				missing = append(missing, fmt.Sprintf("%s:%d uses %q", entry.File, entry.Line, variable))
			}
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("invalid variables:\n%s", strings.Join(missing, "\n"))
	}
	return nil
}

func printCounts(entries []templateEntry) {
	counts := map[string]int{}
	for _, entry := range entries {
		counts[entry.Category]++
	}
	cats := make([]string, 0, len(counts))
	for category := range counts {
		cats = append(cats, category)
	}
	sort.Strings(cats)

	fmt.Println("Template counts by category")
	for _, category := range cats {
		fmt.Printf("  %-32s %d\n", category, counts[category])
	}
	fmt.Printf("\nTotal templates: %d\n", len(entries))
}

func printCoverage(entries []templateEntry, threshold int) {
	counts := map[string]int{}
	byTopLevel := map[string]int{}
	for _, entry := range entries {
		counts[entry.Category]++
		top := strings.Split(entry.Category, "/")[0]
		byTopLevel[top]++
	}

	type sparse struct {
		category string
		count    int
	}
	rows := []sparse{}
	for category, count := range counts {
		rows = append(rows, sparse{category: category, count: count})
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].count == rows[j].count {
			return rows[i].category < rows[j].category
		}
		return rows[i].count < rows[j].count
	})

	fmt.Printf("\nCoverage gaps (< %d)\n", threshold)
	found := false
	for _, row := range rows {
		if row.count >= threshold {
			continue
		}
		found = true
		fmt.Printf("  %-32s %d\n", row.category, row.count)
	}
	if !found {
		fmt.Println("  none")
	}

	topKeys := make([]string, 0, len(byTopLevel))
	for key := range byTopLevel {
		topKeys = append(topKeys, key)
	}
	sort.Strings(topKeys)

	fmt.Println("\nCoverage by top-level group")
	for _, key := range topKeys {
		fmt.Printf("  %-16s %d\n", key, byTopLevel[key])
	}
}

func categoryFromPath(root, path string) (string, error) {
	rel, err := filepath.Rel(root, filepath.Dir(path))
	if err != nil {
		return "", err
	}
	rel = filepath.ToSlash(rel)
	if rel == "." {
		return "", nil
	}
	return rel, nil
}

func uniqueStrings(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
