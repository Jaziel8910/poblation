package templates

import (
	"fmt"
	"io/fs"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/user/poblation/internal/ai"
	"github.com/user/poblation/internal/config"
	"github.com/user/poblation/internal/entities"
	eventpkg "github.com/user/poblation/internal/events"
)

// Core narrative types come from entities.go, the source of truth.
type ArchetypeID = entities.ArchetypeID
type GameEvent = ai.GameEvent
type GameTime = entities.GameTime
type Memory = entities.Memory
type MoodType = entities.MoodType
type Poble = entities.Poble
type Relationship = entities.Relationship
type WorldState = entities.WorldState

// VariableResolver turns one variable name into renderable text.
type VariableResolver func(TemplateContext) string

// TemplateEngine loads, selects, and renders procedural text templates.
type TemplateEngine struct {
	mu                sync.RWMutex
	templates         map[string][]*Template
	variableResolvers map[string]VariableResolver
	usageHistory      map[string][]int
	usageCounts       map[string]int
	rng               *rand.Rand
}

// Template stores one parsed text template.
type Template struct {
	ID              string
	RawText         string
	Tags            []string
	Weight          int
	ArchetypeFilter []ArchetypeID
	MoodFilter      []MoodType
	Variables       []string
	RequiredContext []string
	MinRelationship *float32
	MaxRelationship *float32
	Category        string
	SourcePath      string
}

// TemplateContext carries all facts available during template render.
type TemplateContext struct {
	Speaker                *Poble
	Target                 *Poble
	Location               string
	GameTime               GameTime
	RecentMemory           *Memory
	OldMemory              *Memory
	CurrentEvent           *GameEvent
	RelationshipWithTarget *Relationship
	WorldState             WorldState
	ExtraVars              map[string]string
}

// TemplateStats exposes load and repetition health.
type TemplateStats struct {
	TotalByCategory  map[string]int
	TopUsed          []TemplateUsage
	SparseCategories []string
	Total            int
}

// TemplateUsage stores one high-usage template entry.
type TemplateUsage struct {
	Category string
	ID       string
	Count    int
}

type scoredTemplate struct {
	template *Template
	index    int
	weight   int
}

var (
	templateHeaderPattern = regexp.MustCompile(`^\[TEMPLATE:([A-Za-z0-9_.:_-]+)\]$`)
	variablePattern       = regexp.MustCompile(`\{([^{}]+)\}`)
	conditionPattern      = regexp.MustCompile(`(?s)\{if:([^{}]+)\}(.*?)\{endif\}`)
	variantPattern        = regexp.MustCompile(`\{([^{}\n]*\|[^{}\n]*)\}`)
)

// NewTemplateEngine creates a template engine with built-in variable resolvers.
func NewTemplateEngine(rng *rand.Rand) *TemplateEngine {
	if rng == nil {
		rng = rand.New(rand.NewSource(time.Now().UnixNano()))
	}
	engine := &TemplateEngine{
		templates:         map[string][]*Template{},
		variableResolvers: map[string]VariableResolver{},
		usageHistory:      map[string][]int{},
		usageCounts:       map[string]int{},
		rng:               rng,
	}
	engine.registerDefaultResolvers()
	return engine
}

// RegisterVariableResolver adds or replaces a variable resolver before loading.
func (e *TemplateEngine) RegisterVariableResolver(name string, resolver VariableResolver) {
	if e == nil || name == "" || resolver == nil {
		return
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	e.variableResolvers[name] = resolver
}

// LoadTemplates recursively parses every .txt file under basePath.
func (e *TemplateEngine) LoadTemplates(basePath string) error {
	if e == nil {
		return fmt.Errorf("templates.LoadTemplates: nil engine")
	}
	if basePath == "" {
		return fmt.Errorf("templates.LoadTemplates: empty base path")
	}

	loaded := map[string][]*Template{}
	errors := []string{}
	walkErr := filepath.WalkDir(basePath, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", path, err))
			return nil
		}
		if entry.IsDir() || strings.ToLower(filepath.Ext(path)) != ".txt" {
			return nil
		}
		parsed, parseErr := e.parseTemplateFile(basePath, path)
		if parseErr != nil {
			errors = append(errors, parseErr.Error())
			return nil
		}
		for _, template := range parsed {
			loaded[template.Category] = append(loaded[template.Category], template)
		}
		return nil
	})
	if walkErr != nil {
		return fmt.Errorf("templates.LoadTemplates: walk %s: %w", basePath, walkErr)
	}
	if len(errors) > 0 {
		return fmt.Errorf("templates.LoadTemplates: %s", strings.Join(errors, "; "))
	}

	e.mu.Lock()
	for category, values := range loaded {
		e.templates[category] = append(e.templates[category], values...)
	}
	e.mu.Unlock()
	e.logLoadReport(loaded)
	return nil
}

// Select chooses one template using filters, relevance, and anti-repetition.
func (e *TemplateEngine) Select(category string, ctx TemplateContext) (*Template, error) {
	if e == nil {
		return nil, fmt.Errorf("templates.Select: nil engine")
	}
	if categoryRequiresAdultContent(category) && config.GetContentLevel() != config.ContentFull {
		return nil, fmt.Errorf("templates.Select: restricted content level blocks %q", category)
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	candidates := e.filteredCandidatesLocked(category, ctx)
	if len(candidates) == 0 {
		return nil, fmt.Errorf("templates.Select: no templates matched category %q", category)
	}
	scored := e.scoreCandidatesLocked(category, candidates, ctx)
	if len(scored) == 0 {
		return nil, fmt.Errorf("templates.Select: all templates penalized for category %q", category)
	}

	selected := e.weightedPick(scored)
	e.recordUsageLocked(category, selected.index, selected.template)
	return selected.template, nil
}

// Render resolves variables, conditionals, variants, and whitespace.
func (e *TemplateEngine) Render(template *Template, ctx TemplateContext) (string, error) {
	if e == nil {
		return "", fmt.Errorf("templates.Render: nil engine")
	}
	if template == nil {
		return "", fmt.Errorf("templates.Render: nil template")
	}

	rendered := e.processConditionals(template.RawText, ctx)
	rendered = e.processVariants(rendered)
	rendered = variablePattern.ReplaceAllStringFunc(rendered, func(match string) string {
		name := strings.TrimSuffix(strings.TrimPrefix(match, "{"), "}")
		if isControlToken(name) || strings.Contains(name, "|") {
			return match
		}
		return e.ResolveVariable(name, ctx)
	})

	return cleanTemplateWhitespace(rendered), nil
}

// RenderEventDescription renders narrator-only text for external events.
func (e *TemplateEngine) RenderEventDescription(event eventpkg.GameEvent) string {
	category := narratorCategoryForEvent(event.Type)
	if category == "" {
		return ""
	}
	ctx := TemplateContext{
		GameTime:     event.Timestamp,
		CurrentEvent: nil,
		ExtraVars:    eventExtraVars(event),
	}
	for _, candidate := range []string{category, strings.TrimPrefix(category, "narrator/")} {
		template, err := e.Select(candidate, ctx)
		if err != nil {
			continue
		}
		rendered, err := e.Render(template, ctx)
		if err == nil {
			return rendered
		}
	}
	return ""
}

// ResolveVariable turns one variable token into text, logging unknown variables.
func (e *TemplateEngine) ResolveVariable(name string, ctx TemplateContext) string {
	if e == nil {
		return ""
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	if resolver, ok := e.variableResolvers[name]; ok {
		return resolver(ctx)
	}
	if strings.HasPrefix(name, "days_ago:") {
		return daysAgoValue(strings.TrimPrefix(name, "days_ago:"), ctx)
	}
	if value, ok := ctx.ExtraVars[name]; ok {
		return value
	}

	log.Printf("templates: unknown variable %q", name)
	return ""
}

// GetStats returns current load counts and repetition hotspots.
func (e *TemplateEngine) GetStats() TemplateStats {
	stats := TemplateStats{
		TotalByCategory:  map[string]int{},
		TopUsed:          []TemplateUsage{},
		SparseCategories: []string{},
	}
	if e == nil {
		return stats
	}

	e.mu.RLock()
	defer e.mu.RUnlock()

	for category, values := range e.templates {
		stats.TotalByCategory[category] = len(values)
		stats.Total += len(values)
		if len(values) < 20 {
			stats.SparseCategories = append(stats.SparseCategories, category)
		}
	}
	stats.TopUsed = e.topUsedLocked()
	sort.Strings(stats.SparseCategories)
	return stats
}

func (e *TemplateEngine) parseTemplateFile(basePath, path string) ([]*Template, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("templates.parseTemplateFile: read %s: %w", path, err)
	}

	category, err := categoryFromPath(basePath, path)
	if err != nil {
		return nil, err
	}
	templates, err := e.parseTemplates(string(content), category, path)
	if err != nil {
		return nil, fmt.Errorf("templates.parseTemplateFile: %s: %w", path, err)
	}
	return templates, nil
}

func (e *TemplateEngine) parseTemplates(content, category, sourcePath string) ([]*Template, error) {
	lines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
	result := []*Template{}
	errors := []string{}

	for index := 0; index < len(lines); {
		line := strings.TrimPrefix(strings.TrimSpace(lines[index]), "\ufeff")
		if line == "" || strings.HasPrefix(line, "#") {
			index++
			continue
		}

		match := templateHeaderPattern.FindStringSubmatch(line)
		if match == nil {
			errors = append(errors, fmt.Sprintf("expected [TEMPLATE:ID] near line %d", index+1))
			index++
			continue
		}

		metadata, body, next, err := readTemplateSections(lines, index+1)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s near line %d", err, index+1))
			index = next
			continue
		}
		template, err := e.buildTemplate(match[1], metadata, body, category, sourcePath)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", match[1], err))
		} else {
			result = append(result, template)
		}
		index = next
	}

	if len(errors) > 0 {
		return result, fmt.Errorf("%s", strings.Join(errors, "; "))
	}
	return result, nil
}

func readTemplateSections(lines []string, start int) (map[string]string, string, int, error) {
	metadata := map[string]string{}
	index := start
	for index < len(lines) && strings.TrimSpace(lines[index]) != "---" {
		line := strings.TrimSpace(lines[index])
		if line != "" && !strings.HasPrefix(line, "#") {
			key, value, ok := strings.Cut(line, ":")
			if !ok {
				return metadata, "", index + 1, fmt.Errorf("invalid metadata %q", line)
			}
			metadata[strings.ToLower(strings.TrimSpace(key))] = strings.TrimSpace(value)
		}
		index++
	}
	if index >= len(lines) {
		return metadata, "", index, fmt.Errorf("missing body separator")
	}

	index++
	bodyLines := []string{}
	for index < len(lines) && strings.TrimSpace(lines[index]) != "---" {
		bodyLines = append(bodyLines, lines[index])
		index++
	}
	if index >= len(lines) {
		return metadata, "", index, fmt.Errorf("missing closing separator")
	}
	return metadata, strings.Join(bodyLines, "\n"), index + 1, nil
}

func (e *TemplateEngine) buildTemplate(id string, metadata map[string]string, body, category, sourcePath string) (*Template, error) {
	template := &Template{
		ID:              id,
		RawText:         strings.TrimSpace(body),
		Tags:            splitComma(metadata["tags"]),
		Weight:          parsePositiveInt(metadata["weight"], 1),
		RequiredContext: splitComma(metadata["required_context"]),
		Category:        category,
		SourcePath:      sourcePath,
	}

	var err error
	template.ArchetypeFilter, err = parseArchetypes(metadata["archetypes"])
	if err != nil {
		return nil, err
	}
	template.MoodFilter, err = parseMoods(metadata["moods"])
	if err != nil {
		return nil, err
	}
	template.MinRelationship, err = parseOptionalFloat(metadata["min_relationship"])
	if err != nil {
		return nil, fmt.Errorf("invalid min_relationship: %w", err)
	}
	template.MaxRelationship, err = parseOptionalFloat(metadata["max_relationship"])
	if err != nil {
		return nil, fmt.Errorf("invalid max_relationship: %w", err)
	}

	template.Variables = extractVariables(template.RawText)
	template.RequiredContext = uniqueSortedStrings(append(template.RequiredContext, requiredContextForVariables(template.Variables)...))
	if err := e.validateTemplateVariables(template); err != nil {
		return nil, err
	}
	return template, nil
}

func (e *TemplateEngine) validateTemplateVariables(template *Template) error {
	unknown := []string{}
	for _, variable := range template.Variables {
		if _, ok := e.variableResolvers[variable]; ok {
			continue
		}
		if strings.HasPrefix(variable, "days_ago:") {
			continue
		}
		unknown = append(unknown, variable)
	}
	if len(unknown) > 0 {
		return fmt.Errorf("unknown variables: %s", strings.Join(unknown, ", "))
	}
	return nil
}

func (e *TemplateEngine) filteredCandidatesLocked(category string, ctx TemplateContext) []scoredTemplate {
	candidates := []scoredTemplate{}
	for storedCategory, values := range e.templates {
		if !categoryMatches(storedCategory, category) {
			continue
		}
		for index, template := range values {
			if !templateMatchesContext(template, ctx) {
				continue
			}
			candidates = append(candidates, scoredTemplate{template: template, index: index})
		}
	}
	return candidates
}

func (e *TemplateEngine) scoreCandidatesLocked(category string, candidates []scoredTemplate, ctx TemplateContext) []scoredTemplate {
	scored := []scoredTemplate{}
	contextTags := contextTagSet(ctx)
	for _, candidate := range candidates {
		weight := adjustedWeight(candidate.template, contextTags)
		weight = e.applyHistoryPenalty(category, candidate.index, weight, len(candidates))
		if weight <= 0 {
			continue
		}
		candidate.weight = weight
		scored = append(scored, candidate)
	}
	return scored
}

func (e *TemplateEngine) applyHistoryPenalty(category string, index, weight, candidateCount int) int {
	history := e.usageHistory[category]
	for position := len(history) - 1; position >= 0; position-- {
		usedIndex := history[position]
		if usedIndex != index {
			continue
		}
		if position == len(history)-1 && candidateCount > 1 {
			return 0
		}
		distanceFromNewest := len(history) - position
		penalty := distanceFromNewest + 1
		if penalty > weight {
			return 1
		}
		return weight / penalty
	}
	return weight
}

func (e *TemplateEngine) weightedPick(candidates []scoredTemplate) scoredTemplate {
	total := 0
	for _, candidate := range candidates {
		total += candidate.weight
	}
	if total <= 0 {
		return candidates[0]
	}

	roll := e.rng.Intn(total)
	for _, candidate := range candidates {
		if roll < candidate.weight {
			return candidate
		}
		roll -= candidate.weight
	}
	return candidates[len(candidates)-1]
}

func (e *TemplateEngine) recordUsageLocked(category string, index int, template *Template) {
	history := append(e.usageHistory[category], index)
	if len(history) > 8 {
		history = history[len(history)-8:]
	}
	e.usageHistory[category] = history
	e.usageCounts[usageCountKey(category, template.ID)]++
}

func (e *TemplateEngine) processConditionals(text string, ctx TemplateContext) string {
	return conditionPattern.ReplaceAllStringFunc(text, func(match string) string {
		parts := conditionPattern.FindStringSubmatch(match)
		if len(parts) != 3 {
			return ""
		}
		if conditionIsTrue(parts[1], ctx) {
			return parts[2]
		}
		return ""
	})
}

func (e *TemplateEngine) processVariants(text string) string {
	return variantPattern.ReplaceAllStringFunc(text, func(match string) string {
		inner := strings.TrimSuffix(strings.TrimPrefix(match, "{"), "}")
		options := splitVariantOptions(inner)
		if len(options) == 0 {
			return ""
		}
		return options[e.rng.Intn(len(options))]
	})
}

func (e *TemplateEngine) logLoadReport(loaded map[string][]*Template) {
	categories := make([]string, 0, len(loaded))
	for category := range loaded {
		categories = append(categories, category)
	}
	sort.Strings(categories)
	for _, category := range categories {
		log.Printf("templates: loaded %d templates for %s", len(loaded[category]), category)
	}
}

func (e *TemplateEngine) registerDefaultResolvers() {
	e.variableResolvers["self_name"] = func(ctx TemplateContext) string {
		return nameOrExtra(ctx.Speaker, ctx.ExtraVars, "self_name", "alguien")
	}
	e.variableResolvers["dreamer_name"] = func(ctx TemplateContext) string {
		return nameOrExtra(ctx.Speaker, ctx.ExtraVars, "dreamer_name", "alguien")
	}
	e.variableResolvers["target_name"] = func(ctx TemplateContext) string {
		return nameOrExtra(ctx.Target, ctx.ExtraVars, "target_name", "alguien")
	}
	e.variableResolvers["target_pronoun"] = func(ctx TemplateContext) string { return pronounFor(ctx.Target) }
	e.variableResolvers["memory_recent"] = func(ctx TemplateContext) string { return memorySummary(ctx.RecentMemory) }
	e.variableResolvers["memory_old"] = func(ctx TemplateContext) string { return memorySummary(ctx.OldMemory) }
	e.variableResolvers["location"] = func(ctx TemplateContext) string { return fallbackString(ctx.Location, "un lugar sin nombre") }
	e.variableResolvers["time_of_day"] = func(ctx TemplateContext) string { return timeOfDay(ctx.GameTime) }
	e.variableResolvers["weather"] = func(ctx TemplateContext) string { return weatherValue(ctx) }
	e.variableResolvers["relationship_status"] = func(ctx TemplateContext) string { return relationshipStatus(ctx.RelationshipWithTarget) }
	e.variableResolvers["population"] = func(ctx TemplateContext) string { return strconv.Itoa(ctx.WorldState.Population) }
	e.variableResolvers["settlement_name"] = func(ctx TemplateContext) string { return settlementName(ctx.WorldState) }
	e.variableResolvers["secret_hint"] = func(ctx TemplateContext) string { return secretHint(ctx.Speaker) }
	e.variableResolvers["child_name"] = func(ctx TemplateContext) string { return childName(ctx.Speaker, ctx.ExtraVars) }
	e.variableResolvers["age"] = func(ctx TemplateContext) string { return fallbackString(ctx.ExtraVars["age"], "muchos") }
	e.variableResolvers["surviving_relationships"] = func(ctx TemplateContext) string {
		return fallbackString(ctx.ExtraVars["surviving_relationships"], "quienes todavia sabian su nombre")
	}
	e.variableResolvers["legacy_hint"] = func(ctx TemplateContext) string {
		return fallbackString(ctx.ExtraVars["legacy_hint"], "lo que dejo sigue trabajando sin hacer ruido")
	}
	e.variableResolvers["victim_name"] = func(ctx TemplateContext) string { return fallbackString(ctx.ExtraVars["victim_name"], "alguien") }
	e.variableResolvers["killer_name"] = func(ctx TemplateContext) string { return fallbackString(ctx.ExtraVars["killer_name"], "alguien") }
	e.variableResolvers["method_hint"] = func(ctx TemplateContext) string {
		return fallbackString(ctx.ExtraVars["method_hint"], "un metodo demasiado humano")
	}
	e.variableResolvers["days_ill"] = func(ctx TemplateContext) string { return fallbackString(ctx.ExtraVars["days_ill"], "varios dias") }
	e.variableResolvers["last_words_hint"] = func(ctx TemplateContext) string {
		return fallbackString(ctx.ExtraVars["last_words_hint"], "lo ultimo quedo pequeno y dificil de repetir")
	}
	e.variableResolvers["trigger_hint"] = func(ctx TemplateContext) string {
		return fallbackString(ctx.ExtraVars["trigger_hint"], "algo que venia pesando desde antes")
	}
	e.variableResolvers["mother_name"] = func(ctx TemplateContext) string { return fallbackString(ctx.ExtraVars["mother_name"], "alguien") }
	e.variableResolvers["father_name"] = func(ctx TemplateContext) string { return fallbackString(ctx.ExtraVars["father_name"], "alguien") }
	e.variableResolvers["circumstances"] = func(ctx TemplateContext) string {
		return fallbackString(ctx.ExtraVars["circumstances"], "nadie sabe contar esto sin dejar algo fuera")
	}
	e.variableResolvers["event_description"] = func(ctx TemplateContext) string {
		return fallbackString(ctx.ExtraVars["event_description"], "paso algo y todos lo sintieron")
	}
	e.variableResolvers["specific_grievance"] = func(ctx TemplateContext) string {
		return fallbackString(ctx.ExtraVars["specific_grievance"], "lo que hiciste y todavia quieres llamar accidente")
	}
	e.variableResolvers["the_thing_wanted"] = func(ctx TemplateContext) string {
		return fallbackString(ctx.ExtraVars["the_thing_wanted"], "eso que no deberia querer")
	}
	e.variableResolvers["future_event_hint"] = func(ctx TemplateContext) string { return futureEventHint(ctx.ExtraVars) }
	e.variableResolvers["survivor_name"] = func(ctx TemplateContext) string { return fallbackString(ctx.ExtraVars["survivor_name"], "quien quedo") }
	e.variableResolvers["deceased_name"] = func(ctx TemplateContext) string {
		return fallbackString(ctx.ExtraVars["deceased_name"], "quien se fue")
	}
	e.variableResolvers["years_together"] = func(ctx TemplateContext) string {
		return fallbackString(ctx.ExtraVars["years_together"], "demasiado tiempo")
	}
	e.variableResolvers["shared_history_hint"] = func(ctx TemplateContext) string {
		return fallbackString(ctx.ExtraVars["shared_history_hint"], "lo que vivieron juntos quedo sin testigos")
	}
	e.variableResolvers["last_thing_they_said"] = func(ctx TemplateContext) string {
		return fallbackString(ctx.ExtraVars["last_thing_they_said"], "no hubo frase suficiente")
	}
	e.variableResolvers["location_atmosphere"] = func(ctx TemplateContext) string {
		return fallbackString(ctx.ExtraVars["location_atmosphere"], "el aire no decide nada todavia")
	}
	e.variableResolvers["location_memory"] = func(ctx TemplateContext) string {
		return fallbackString(ctx.ExtraVars["location_memory"], "el lugar no ha dicho que recuerda")
	}
	e.variableResolvers["location_name"] = func(ctx TemplateContext) string {
		if value := strings.TrimSpace(ctx.ExtraVars["location_name"]); value != "" {
			return value
		}
		return fallbackString(ctx.Location, "un lugar sin nombre")
	}
	e.variableResolvers["power_dynamic"] = func(ctx TemplateContext) string {
		return fallbackString(ctx.ExtraVars["power_dynamic"], "nadie logra mandar del todo")
	}
	e.variableResolvers["encounter_mood"] = func(ctx TemplateContext) string { return fallbackString(ctx.ExtraVars["encounter_mood"], "complicado") }
	e.variableResolvers["encounter_type"] = func(ctx TemplateContext) string {
		return fallbackString(ctx.ExtraVars["encounter_type"], "complicated")
	}

	// New system resolvers.
	e.variableResolvers["self_vocab_filler"] = func(ctx TemplateContext) string { return vocabFiller(ctx.Speaker) }
	e.variableResolvers["settlement_slang"] = func(ctx TemplateContext) string { return settlementSlang(ctx.WorldState) }
	e.variableResolvers["self_nickname"] = func(ctx TemplateContext) string { return selfNickname(ctx.Speaker) }
	e.variableResolvers["target_nickname"] = func(ctx TemplateContext) string { return selfNickname(ctx.Target) }
	e.variableResolvers["profanity_mild"] = func(ctx TemplateContext) string { return profanityMild(ctx.Speaker) }
	e.variableResolvers["profanity_strong"] = func(ctx TemplateContext) string { return profanityStrong(ctx.Speaker) }
	e.variableResolvers["self_expletive"] = func(ctx TemplateContext) string { return selfExpletive(ctx.Speaker) }
	e.variableResolvers["self_love_lang"] = func(ctx TemplateContext) string { return loveLanguageLabel(ctx.Speaker) }
	e.variableResolvers["self_quirk"] = func(ctx TemplateContext) string { return selfQuirk(ctx.Speaker) }
	e.variableResolvers["self_defense"] = func(ctx TemplateContext) string { return selfDefense(ctx.Speaker) }
	e.variableResolvers["self_superstition"] = func(ctx TemplateContext) string { return selfSuperstition(ctx.Speaker) }
	e.registerEndingResolvers()
}

func (e *TemplateEngine) registerEndingResolvers() {
	names := []string{
		"ending_title",
		"ending_type",
		"ending_day",
		"ending_population",
		"ending_total_pobles",
		"ending_total_deaths",
		"ending_total_births",
		"ending_total_wars",
		"ending_total_affairs",
		"ending_total_secrets",
		"ending_total_rumours",
		"ending_subject",
		"ending_second_subject",
		"ending_survivor",
		"ending_deceased",
		"ending_first_couple",
		"ending_last_event",
		"ending_dramatic_event",
		"ending_longest_lived",
		"ending_most_loved",
		"ending_most_hated",
		"ending_richest",
		"ending_mutated_rumour",
		"ending_settlement",
		"ending_era",
		"ending_tech",
		"ending_belief",
		"ending_leader",
		"ending_prophet",
		"ending_family",
	}
	for _, name := range names {
		key := name
		e.variableResolvers[key] = func(ctx TemplateContext) string {
			return fallbackString(ctx.ExtraVars[key], "")
		}
	}
}

func templateMatchesContext(template *Template, ctx TemplateContext) bool {
	return archetypeAllowed(template.ArchetypeFilter, ctx.Speaker) &&
		moodAllowed(template.MoodFilter, ctx.Speaker) &&
		relationshipAllowed(template, relationshipScore(ctx.RelationshipWithTarget))
}

func archetypeAllowed(filter []ArchetypeID, speaker *Poble) bool {
	if len(filter) == 0 {
		return true
	}
	if speaker == nil {
		return false
	}
	for _, archetype := range filter {
		if archetype == speaker.Archetype {
			return true
		}
	}
	return false
}

func moodAllowed(filter []MoodType, speaker *Poble) bool {
	if len(filter) == 0 {
		return true
	}
	if speaker == nil {
		return false
	}
	for _, mood := range filter {
		if mood == speaker.CurrentMood || mood == speaker.EmotionalState.CurrentMood {
			return true
		}
	}
	return false
}

func relationshipAllowed(template *Template, score float32) bool {
	if template.MinRelationship != nil && score < *template.MinRelationship {
		return false
	}
	if template.MaxRelationship != nil && score > *template.MaxRelationship {
		return false
	}
	return true
}

func adjustedWeight(template *Template, contextTags map[string]struct{}) int {
	weight := template.Weight
	if weight <= 0 {
		weight = 1
	}
	for _, tag := range template.Tags {
		normalized := normalizeTag(tag)
		if _, ok := contextTags[normalized]; ok {
			weight += maxInt(1, template.Weight/2)
		}
	}
	return weight
}

func contextTagSet(ctx TemplateContext) map[string]struct{} {
	tags := map[string]struct{}{}
	addTag(tags, "any")
	if ctx.Speaker != nil {
		addTag(tags, "archetype:"+strings.ToLower(ctx.Speaker.Archetype.String()))
		addTag(tags, strings.ToLower(ctx.Speaker.CurrentMood.String()))
		addEmotionTags(tags, ctx.Speaker.EmotionalState)
		addNeedTags(tags, ctx.Speaker.Needs)
	}
	if ctx.RelationshipWithTarget != nil {
		addRelationshipTags(tags, *ctx.RelationshipWithTarget)
	}
	if ctx.CurrentEvent != nil {
		addTag(tags, "event:"+strings.ToLower(string(ctx.CurrentEvent.Type)))
		for _, tag := range ctx.CurrentEvent.Tags {
			addTag(tags, tag)
		}
	}
	if modifiers := strings.TrimSpace(ctx.ExtraVars["template_modifiers"]); modifiers != "" {
		for _, tag := range strings.Split(modifiers, ",") {
			addTag(tags, strings.TrimSpace(tag))
		}
	}
	addMemoryTags(tags, ctx.RecentMemory)
	addMemoryTags(tags, ctx.OldMemory)
	return tags
}

func addEmotionTags(tags map[string]struct{}, state entities.EmotionalState) {
	if state.Valence < -0.25 {
		addTag(tags, "negative_valence")
	}
	if state.Valence > 0.25 {
		addTag(tags, "positive_valence")
	}
	if state.Arousal > 0.35 {
		addTag(tags, "high_arousal")
	}
	if state.Arousal < -0.25 {
		addTag(tags, "low_arousal")
	}
	for _, emotion := range state.ActiveEmotions {
		addTag(tags, emotion.String())
	}
}

func addNeedTags(tags map[string]struct{}, needs entities.Needs) {
	if needs.Sex >= 75 {
		addTag(tags, "high_horniness")
	}
	if needs.Belonging >= 75 {
		addTag(tags, "lonely")
		addTag(tags, "high_belonging")
	}
	if needs.Power >= 75 {
		addTag(tags, "power")
	}
}

func addRelationshipTags(tags map[string]struct{}, relationship Relationship) {
	addTag(tags, "relationship:"+strings.ToLower(relationship.Type.String()))
	if relationship.Attraction >= 70 {
		addTag(tags, "attraction")
	}
	if relationship.Resentment >= 60 {
		addTag(tags, "resentment")
	}
	if relationship.Trust >= 70 {
		addTag(tags, "trust")
	}
}

func addMemoryTags(tags map[string]struct{}, memory *Memory) {
	if memory == nil {
		return
	}
	addTag(tags, "memory:"+strings.ToLower(memory.Type.String()))
	for _, tag := range memory.Tags {
		addTag(tags, tag)
	}
}

func addTag(tags map[string]struct{}, tag string) {
	tag = normalizeTag(tag)
	if tag != "" {
		tags[tag] = struct{}{}
	}
}

func categoryMatches(storedCategory, requested string) bool {
	requested = filepath.ToSlash(strings.Trim(strings.TrimSpace(requested), "/"))
	return storedCategory == requested || strings.HasPrefix(storedCategory, requested+"/")
}

func extractVariables(text string) []string {
	matches := variablePattern.FindAllStringSubmatch(text, -1)
	variables := []string{}
	for _, match := range matches {
		if len(match) != 2 {
			continue
		}
		name := strings.TrimSpace(match[1])
		if isControlToken(name) || strings.Contains(name, "|") {
			continue
		}
		variables = append(variables, name)
	}
	return uniqueSortedStrings(variables)
}

func isControlToken(name string) bool {
	return strings.HasPrefix(name, "if:") || name == "endif"
}

func requiredContextForVariables(variables []string) []string {
	required := []string{}
	for _, variable := range variables {
		switch variable {
		case "target_name", "target_pronoun":
			required = append(required, "target")
		case "memory_recent", "days_ago:memory", "days_ago:recent":
			required = append(required, "recent_memory")
		case "memory_old", "days_ago:old":
			required = append(required, "old_memory")
		case "relationship_status":
			required = append(required, "relationship")
		case "secret_hint":
			required = append(required, "secret")
		}
	}
	return required
}

func parseArchetypes(value string) ([]ArchetypeID, error) {
	items := splitComma(value)
	result := make([]ArchetypeID, 0, len(items))
	for _, item := range items {
		archetype := entities.ArchetypeID(strings.ToUpper(item))
		if !archetype.IsValid() {
			return nil, fmt.Errorf("invalid archetype %q", item)
		}
		result = append(result, archetype)
	}
	return result, nil
}

func parseMoods(value string) ([]MoodType, error) {
	items := splitComma(value)
	result := make([]MoodType, 0, len(items))
	for _, item := range items {
		mood := entities.MoodType(strings.ToUpper(item))
		if !mood.IsValid() {
			return nil, fmt.Errorf("invalid mood %q", item)
		}
		result = append(result, mood)
	}
	return result, nil
}

func parseOptionalFloat(value string) (*float32, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}
	parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 32)
	if err != nil {
		return nil, err
	}
	value32 := float32(parsed)
	return &value32, nil
}

func parsePositiveInt(value string, fallback int) int {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func splitComma(value string) []string {
	if strings.TrimSpace(value) == "" {
		return []string{}
	}
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}

func categoryRequiresAdultContent(category string) bool {
	category = filepath.ToSlash(strings.Trim(strings.TrimSpace(category), "/"))
	return strings.HasPrefix(category, "dialogues/encounter/") || strings.HasPrefix(category, "dreams/erotic")
}

func narratorCategoryForEvent(eventType eventpkg.EventType) string {
	switch eventType {
	case eventpkg.EventDeathNatural:
		return "narrator/death/natural"
	case eventpkg.EventDeathMurder:
		return "narrator/death/murder"
	case eventpkg.EventIllnessOnset:
		return "narrator/death/illness"
	case eventpkg.EventSuicide:
		return "narrator/death/suicide"
	case eventpkg.EventBirth:
		return "narrator/birth/general"
	case eventpkg.EventStorm:
		return "narrator/disaster/storm"
	case eventpkg.EventEarthquake:
		return "narrator/disaster/earthquake"
	case eventpkg.EventPlague:
		return "narrator/disaster/plague"
	case eventpkg.EventCivilizationCollapse:
		return "narrator/end_of_game/reset"
	case eventpkg.EventLastPersonAlive:
		return "narrator/end_of_game/alone"
	case eventpkg.EventGenerationEnd:
		return "narrator/end_of_game/love"
	default:
		return ""
	}
}

func eventExtraVars(event eventpkg.GameEvent) map[string]string {
	vars := map[string]string{
		"event_description": fallbackString(event.Description, strings.ToLower(string(event.Type))),
		"location":          "donde todos pudieron mirar tarde",
	}
	if len(event.Participants) > 0 {
		vars["self_name"] = event.Participants[0]
		vars["victim_name"] = event.Participants[0]
		vars["child_name"] = event.Participants[0]
	}
	if len(event.Participants) > 1 {
		vars["target_name"] = event.Participants[1]
		vars["killer_name"] = event.Participants[1]
		vars["mother_name"] = event.Participants[1]
	}
	if len(event.Participants) > 2 {
		vars["father_name"] = event.Participants[2]
	}
	return vars
}

func splitVariantOptions(value string) []string {
	options := strings.Split(value, "|")
	result := make([]string, 0, len(options))
	for _, option := range options {
		option = strings.TrimSpace(option)
		if option != "" {
			result = append(result, option)
		}
	}
	return result
}

func categoryFromPath(basePath, path string) (string, error) {
	relative, err := filepath.Rel(basePath, path)
	if err != nil {
		return "", fmt.Errorf("templates.categoryFromPath: %w", err)
	}
	relative = strings.TrimSuffix(relative, filepath.Ext(relative))
	return filepath.ToSlash(relative), nil
}

func conditionIsTrue(name string, ctx TemplateContext) bool {
	name = strings.ToLower(strings.TrimSpace(name))
	if value, ok := ctx.ExtraVars[name]; ok {
		return truthy(value)
	}
	switch name {
	case "high_horniness":
		return ctx.Speaker != nil && (ctx.Speaker.Personality.Horniness >= 70 || ctx.Speaker.Needs.Sex >= 70)
	case "high_belonging":
		return ctx.Speaker != nil && ctx.Speaker.Needs.Belonging >= 75
	case "has_target":
		return ctx.Target != nil
	case "has_recent_memory":
		return ctx.RecentMemory != nil
	case "has_old_memory":
		return ctx.OldMemory != nil
	case "has_secret":
		return ctx.Speaker != nil && len(ctx.Speaker.Secrets) > 0
	case "low_trust":
		return ctx.RelationshipWithTarget != nil && ctx.RelationshipWithTarget.Trust < 35
	case "high_trust":
		return ctx.RelationshipWithTarget != nil && ctx.RelationshipWithTarget.Trust >= 70
	case "high_attraction":
		return ctx.RelationshipWithTarget != nil && ctx.RelationshipWithTarget.Attraction >= 70
	case "high_resentment":
		return ctx.RelationshipWithTarget != nil && ctx.RelationshipWithTarget.Resentment >= 60
	case "morning":
		return ctx.GameTime.Hour >= 5 && ctx.GameTime.Hour < 12
	case "night":
		return ctx.GameTime.Hour >= 20 || ctx.GameTime.Hour < 5
	default:
		return ctx.Speaker != nil && strings.EqualFold(ctx.Speaker.CurrentMood.String(), name)
	}
}

func truthy(value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	return value == "1" || value == "true" || value == "yes" || value == "si"
}

func nameOrFallback(poble *Poble, fallback string) string {
	if poble == nil || strings.TrimSpace(poble.Name) == "" {
		return fallback
	}
	return poble.Name
}

func nameOrExtra(poble *Poble, extraVars map[string]string, key, fallback string) string {
	if poble != nil && strings.TrimSpace(poble.Name) != "" {
		return poble.Name
	}
	return fallbackString(extraVars[key], fallback)
}

func pronounFor(poble *Poble) string {
	if poble == nil {
		return "elle"
	}
	switch poble.Sex {
	case entities.Male:
		return "él"
	case entities.Female:
		return "ella"
	default:
		return "elle"
	}
}

func memorySummary(memory *Memory) string {
	if memory == nil || strings.TrimSpace(memory.Summary) == "" {
		return "algo que no termina de nombrar"
	}
	return memory.Summary
}

func timeOfDay(gameTime GameTime) string {
	switch {
	case gameTime.Hour >= 5 && gameTime.Hour < 12:
		return "mañana"
	case gameTime.Hour >= 12 && gameTime.Hour < 18:
		return "tarde"
	default:
		return "noche"
	}
}

func weatherValue(ctx TemplateContext) string {
	if value, ok := ctx.ExtraVars["weather"]; ok && strings.TrimSpace(value) != "" {
		return value
	}
	return "clima incierto"
}

func relationshipStatus(relationship *Relationship) string {
	if relationship == nil {
		return "distancia sin historia clara"
	}
	switch {
	case relationship.Resentment >= 70 && relationship.Attraction >= 40:
		return "deseo mezclado con rencor"
	case relationship.Resentment >= 75:
		return "rencor abierto"
	case relationship.Trust >= 75 && relationship.Attraction >= 70:
		return "intimidad dificil de negar"
	case relationship.Trust >= 70:
		return "confianza fragil pero real"
	case relationship.Attraction >= 70:
		return "atraccion mal disimulada"
	default:
		return strings.ToLower(relationship.Type.String())
	}
}

func settlementName(worldState WorldState) string {
	for _, settlement := range worldState.Settlements {
		if strings.TrimSpace(settlement.Name) != "" {
			return settlement.Name
		}
	}
	return "el asentamiento"
}

func secretHint(poble *Poble) string {
	if poble == nil {
		return "algo guardado"
	}
	for _, secret := range poble.Secrets {
		if secret.IsRevealed {
			continue
		}
		switch secret.Type {
		case entities.SecretPastRelationship:
			return "un nombre viejo que evita decir"
		case entities.SecretDarkDesire:
			return "un deseo que no quiere mirar de frente"
		case entities.SecretPlannedBetrayal:
			return "un plan que todavia no tiene testigos"
		case entities.SecretObsession:
			return "una fijacion demasiado quieta"
		default:
			return "algo que mantiene fuera de la luz"
		}
	}
	return "algo que nunca dice completo"
}

func childName(poble *Poble, extraVars map[string]string) string {
	if value, ok := extraVars["child_name"]; ok && strings.TrimSpace(value) != "" {
		return value
	}
	if poble != nil && len(poble.Children) > 0 {
		return poble.Children[0]
	}
	return "el hijo de alguien"
}

func futureEventHint(extraVars map[string]string) string {
	if value, ok := extraVars["future_event_hint"]; ok && strings.TrimSpace(value) != "" {
		return value
	}
	return "algo viene, y no trae permiso"
}

func daysAgoValue(kind string, ctx TemplateContext) string {
	var memory *Memory
	switch kind {
	case "memory", "recent":
		memory = ctx.RecentMemory
	case "old":
		memory = ctx.OldMemory
	default:
		return ""
	}
	if memory == nil {
		return ""
	}
	hours := ctx.GameTime.Diff(memory.Timestamp)
	if hours < 0 {
		hours = 0
	}
	days := hours / 24
	if days <= 0 {
		return "hoy"
	}
	if days == 1 {
		return "hace 1 día"
	}
	return fmt.Sprintf("hace %d días", days)
}

func relationshipScore(relationship *Relationship) float32 {
	if relationship == nil {
		return 0
	}
	score := ((relationship.Trust - 50) * 1.2) +
		(relationship.Attraction * 0.35) +
		(relationship.Respect * 0.25) -
		(relationship.Resentment * 0.75)
	return clampFloat(score, -100, 100)
}

func cleanTemplateWhitespace(text string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	lines := strings.Split(text, "\n")
	cleaned := make([]string, 0, len(lines))
	blankCount := 0
	for _, line := range lines {
		line = strings.TrimRight(line, " \t")
		if strings.TrimSpace(line) == "" {
			blankCount++
			if blankCount <= 1 {
				cleaned = append(cleaned, "")
			}
			continue
		}
		blankCount = 0
		cleaned = append(cleaned, line)
	}
	return strings.TrimSpace(strings.Join(cleaned, "\n"))
}

func fallbackString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func normalizeTag(tag string) string {
	tag = strings.TrimSpace(strings.ToLower(tag))
	tag = strings.ReplaceAll(tag, " ", "_")
	return tag
}

func uniqueSortedStrings(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	seen := map[string]struct{}{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func usageCountKey(category, id string) string {
	return category + "\x00" + id
}

func splitUsageCountKey(key string) (string, string) {
	parts := strings.SplitN(key, "\x00", 2)
	if len(parts) != 2 {
		return "", key
	}
	return parts[0], parts[1]
}

func (e *TemplateEngine) topUsedLocked() []TemplateUsage {
	usages := make([]TemplateUsage, 0, len(e.usageCounts))
	for key, count := range e.usageCounts {
		category, id := splitUsageCountKey(key)
		usages = append(usages, TemplateUsage{Category: category, ID: id, Count: count})
	}
	sort.SliceStable(usages, func(i, j int) bool {
		if usages[i].Count == usages[j].Count {
			return usages[i].ID < usages[j].ID
		}
		return usages[i].Count > usages[j].Count
	})
	if len(usages) > 10 {
		return usages[:10]
	}
	return usages
}

func clampFloat(value, minValue, maxValue float32) float32 {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func vocabFiller(poble *Poble) string {
	if poble == nil || len(poble.Vocabulary.FillerWords) == 0 {
		return ""
	}
	return poble.Vocabulary.FillerWords[0]
}

func settlementSlang(ws WorldState) string {
	if len(ws.Settlements) == 0 {
		return ""
	}
	for _, s := range ws.Settlements {
		if len(s.Language.SlangWords) > 0 {
			return s.Language.SlangWords[0]
		}
	}
	return ""
}

func selfNickname(poble *Poble) string {
	if poble == nil || len(poble.Nicknames) == 0 {
		return ""
	}
	for _, nick := range poble.Nicknames {
		if nick.IsAccepted {
			return nick.Nickname
		}
	}
	return poble.Nicknames[0].Nickname
}

func profanityMild(poble *Poble) string {
	if poble == nil || len(poble.Profanity.MildExpletives) == 0 {
		return ""
	}
	return poble.Profanity.MildExpletives[0]
}

func profanityStrong(poble *Poble) string {
	if poble == nil || len(poble.Profanity.StrongExpletives) == 0 {
		return ""
	}
	return poble.Profanity.StrongExpletives[0]
}

func selfExpletive(poble *Poble) string {
	if poble == nil || poble.Profanity.SignatureExpletive == "" {
		return ""
	}
	return poble.Profanity.SignatureExpletive
}

func loveLanguageLabel(poble *Poble) string {
	if poble == nil {
		return ""
	}
	switch poble.LoveLang.Primary {
	case entities.LoveLangWordsOfAffirmation:
		return "palabras"
	case entities.LoveLangActsOfService:
		return "acciones"
	case entities.LoveLangGifts:
		return "regalos"
	case entities.LoveLangQualityTime:
		return "tiempo"
	case entities.LoveLangPhysicalTouch:
		return "contacto"
	default:
		return ""
	}
}

func selfQuirk(poble *Poble) string {
	if poble == nil || len(poble.Quirks) == 0 {
		return ""
	}
	return poble.Quirks[0].Description
}

func selfDefense(poble *Poble) string {
	if poble == nil {
		return ""
	}
	switch poble.Defense.Primary {
	case entities.DefenseDenial:
		return "niega lo que duele"
	case entities.DefenseProjection:
		return "culpa a otros"
	case entities.DefenseRationalization:
		return "lo razona todo"
	case entities.DefenseDisplacement:
		return "descarga en otro lado"
	case entities.DefenseRepression:
		return "entierra lo que siente"
	case entities.DefenseHumor:
		return "se rie para no romperse"
	case entities.DefenseSublimation:
		return "lo convierte en trabajo"
	case entities.DefenseIntellectualize:
		return "lo analiza hasta que no duele"
	case entities.DefenseRegression:
		return "se vuelve mas pequeño"
	case entities.DefenseDissociation:
		return "se desconecta"
	default:
		return ""
	}
}

func selfSuperstition(poble *Poble) string {
	if poble == nil || len(poble.Superstitions) == 0 {
		return ""
	}
	return poble.Superstitions[0].Belief
}
