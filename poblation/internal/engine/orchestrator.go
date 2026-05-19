package engine

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/user/poblation/internal/ai"
	"github.com/user/poblation/internal/entities"
	"github.com/user/poblation/internal/events"
	"github.com/user/poblation/internal/save"
	"github.com/user/poblation/internal/templates"
	"github.com/user/poblation/internal/world"
)

const (
	maxEventFeed        = 80
	maxPobleThoughts    = 32
	maxPobleDreams      = 16
	maxPobleDiaryPages  = 24
	maxPobleLetters     = 24
	thoughtEveryMinutes = 6 * 60
)

// OrchestratorOptions carries startup flags that are not part of Init(seed).
type OrchestratorOptions struct {
	Debug bool
	Speed float64
	Slot  int
}

// UISnapshot is a safe render-facing copy of simulation state.
type UISnapshot struct {
	WorldState   entities.WorldState
	Pobles       []PobleSnapshot
	EventFeed    []events.GameEvent
	ActiveEnding *Ending
	CurrentTime  GameTime
	Speed        float64
	IsPaused     bool
	Debug        bool
}

// PobleSnapshot keeps the UI away from direct mutable Poble pointers.
type PobleSnapshot struct {
	ID            string
	Name          string
	Age           int
	Archetype     entities.ArchetypeID
	Mood          entities.MoodType
	IsAlive       bool
	Needs         entities.Needs
	CurrentIntent string
	IntentReason  string
}

// Orchestrator connects the simulation systems. No subsystem should do this.
type Orchestrator struct {
	world               *world.World
	timeEngine          *TimeEngine
	templateEngine      *templates.TemplateEngine
	eventQueue          *events.EventQueue
	allPobles           map[string]*entities.Poble
	decisionEngines     map[string]*ai.DecisionEngine
	reproductionSystem  *entities.ReproductionSystem
	civilizationManager *world.CivilizationManager
	endingChecker       *EndingChecker
	saveSystem          *save.SaveSystem
	rng                 *rand.Rand

	options      OrchestratorOptions
	eventFeed    []events.GameEvent
	activeEnding *Ending
	lastError    error
	mu           sync.RWMutex
}

// NewOrchestrator creates the central runtime coordinator.
func NewOrchestrator(options OrchestratorOptions) *Orchestrator {
	return &Orchestrator{options: options}
}

// Init initializes templates, world, AI systems, persistence, and time.
func (o *Orchestrator) Init(seed int64) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if seed == 0 {
		seed = time.Now().UnixNano()
	}
	o.rng = rand.New(rand.NewSource(seed))
	o.timeEngine = NewTimeEngine()
	o.templateEngine = templates.NewTemplateEngine(o.rng)
	o.eventQueue = events.NewEventQueue(o.rng)
	o.saveSystem = save.NewSaveSystem()
	o.civilizationManager = &world.CivilizationManager{}
	checker := NewEndingChecker()
	o.endingChecker = &checker
	o.eventFeed = []events.GameEvent{}
	o.activeEnding = nil
	o.lastError = nil

	if err := o.templateEngine.LoadTemplates(templateRoot()); err != nil {
		return fmt.Errorf("orchestrator.Init load templates: %w", err)
	}
	if err := o.initializeWorld(seed); err != nil {
		return err
	}
	o.timeEngine.SetSpeed(o.startSpeed())
	o.rebuildPobleSystems()
	return nil
}

// OnTick is the main game loop. One tick equals one in-game hour.
func (o *Orchestrator) OnTick(tick GameTick) []events.GameEvent {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.world == nil {
		return nil
	}
	o.world.Calendar = tick.CurrentTime
	o.rebuildPobleSystems()
	o.updateLivingPobles(tick)

	processed := make([]events.GameEvent, 0, 8)
	processed = append(processed, o.processQueuedEvents(tick)...)
	processed = append(processed, o.checkNaturalEvents(tick)...)
	processed = append(processed, o.checkSocialEvents(tick)...)
	processed = append(processed, o.checkWorldEvents(tick)...)
	if tick.IsNewDay {
		processed = append(processed, o.handleNewDay()...)
	}
	processed = append(processed, o.checkEraTransition(tick)...)
	o.checkEndingConditions()
	o.publishEvents(processed)
	return append([]events.GameEvent(nil), processed...)
}

// GetWorldSnapshot returns immutable-ish data for rendering.
func (o *Orchestrator) GetWorldSnapshot() UISnapshot {
	o.mu.RLock()
	defer o.mu.RUnlock()

	snapshot := UISnapshot{
		EventFeed:    append([]events.GameEvent(nil), o.eventFeed...),
		ActiveEnding: o.activeEnding,
		Debug:        o.options.Debug,
	}
	if o.world != nil {
		snapshot.WorldState = o.world.GetWorldState()
		snapshot.CurrentTime = o.world.Calendar
		snapshot.Pobles = snapshotPobles(o.world.GetAllKnownPobles(), o.decisionEngines)
	}
	if o.timeEngine != nil {
		snapshot.Speed = o.timeEngine.Speed
		snapshot.IsPaused = o.timeEngine.IsPaused
	}
	return snapshot
}

// World returns the runtime world for legacy views that still need it.
func (o *Orchestrator) World() *world.World {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.world
}

// TimeEngine returns the central clock.
func (o *Orchestrator) TimeEngine() *TimeEngine {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.timeEngine
}

// TemplateEngine returns the loaded template engine.
func (o *Orchestrator) TemplateEngine() *templates.TemplateEngine {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.templateEngine
}

// EventFeed returns the newest events first.
func (o *Orchestrator) EventFeed() []events.GameEvent {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return append([]events.GameEvent(nil), o.eventFeed...)
}

// ActiveEnding returns the ending currently detected by the simulation.
func (o *Orchestrator) ActiveEnding() *Ending {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.activeEnding
}

// Stop shuts down the central clock cleanly.
func (o *Orchestrator) Stop() {
	o.mu.RLock()
	clock := o.timeEngine
	o.mu.RUnlock()
	if clock != nil {
		clock.Stop()
	}
}

func (o *Orchestrator) initializeWorld(seed int64) error {
	if o.options.Slot > 0 {
		return o.loadSlot(o.options.Slot)
	}
	o.world = world.NewWorld(seed)
	return o.createFounders()
}

func (o *Orchestrator) loadSlot(slot int) error {
	data, err := o.saveSystem.Load(slot)
	if err != nil {
		return fmt.Errorf("orchestrator.Init load slot %d: %w", slot, err)
	}
	if data == nil || data.WorldState == nil {
		return fmt.Errorf("orchestrator.Init load slot %d: save is empty", slot)
	}
	o.world = data.WorldState
	o.world.Calendar = data.CurrentTime
	o.eventFeed = append([]events.GameEvent(nil), data.EventHistory...)
	if o.timeEngine != nil {
		o.timeEngine.SetTime(data.CurrentTime)
	}
	return nil
}

func (o *Orchestrator) createFounders() error {
	first, second, err := o.generateFounders()
	if err != nil {
		return err
	}
	linkFounders(first, second)
	_ = o.world.AddPoble(first, world.Location{IslandID: "island_0", X: 12, Y: 10})
	_ = o.world.AddPoble(second, world.Location{IslandID: "island_0", X: 18, Y: 10})
	return nil
}

func (o *Orchestrator) generateFounders() (*entities.Poble, *entities.Poble, error) {
	male := entities.Male
	female := entities.Female
	first, err := entities.GeneratePople(entities.PoblConfig{Sex: &male, AgeRange: [2]int{22, 36}}, o.rng)
	if err != nil {
		return nil, nil, fmt.Errorf("orchestrator.generateFounders first: %w", err)
	}
	second, err := entities.GeneratePople(entities.PoblConfig{Sex: &female, AgeRange: [2]int{20, 34}}, o.rng)
	if err != nil {
		return nil, nil, fmt.Errorf("orchestrator.generateFounders second: %w", err)
	}
	return first, second, nil
}

func (o *Orchestrator) rebuildPobleSystems() {
	o.allPobles = map[string]*entities.Poble{}
	o.decisionEngines = map[string]*ai.DecisionEngine{}
	if o.world == nil {
		return
	}
	for _, poble := range o.world.GetAllKnownPobles() {
		if poble == nil {
			continue
		}
		o.allPobles[poble.ID] = poble
		if poble.IsAlive {
			o.decisionEngines[poble.ID] = ai.NewDecisionEngine(poble, o.world, o.rng)
		}
	}
	entityWorld := &entities.World{
		State:  o.world.GetWorldState(),
		Pobles: o.allPobles,
		Events: []entities.GameEvent{},
	}
	o.reproductionSystem = entities.NewReproductionSystem(entityWorld, o.rng)
}

func (o *Orchestrator) updateLivingPobles(tick GameTick) {
	for _, poble := range o.world.GetAllPobles() {
		if poble == nil || !poble.IsAlive {
			continue
		}
		o.updatePobleState(poble, tick.DeltaHours)
		actions := o.decisionEngines[poble.ID].Decide(tick.DeltaHours)
		o.enqueueActionEvents(poble, actions, tick.CurrentTime)
	}
}

func (o *Orchestrator) updatePobleState(poble *entities.Poble, deltaHours int) {
	ai.NewNeedsSystem(poble).Update(deltaHours, o.worldContextFor(poble))
	ai.NewEmotionSystem(poble).Update(deltaHours)
	ai.NewMemorySystem(poble, o.rng).EmotionalDecay(deltaHours)
	o.generatePassiveThought(poble)
}

func (o *Orchestrator) enqueueActionEvents(poble *entities.Poble, actions []ai.Action, now GameTime) {
	if len(actions) == 0 {
		return
	}
	action := actions[0]
	o.applyActionNeedResult(poble, action)
	o.applyActionNarrativeArtifacts(poble, action, now)
	event, ok := o.eventFromAction(poble, action, now)
	if ok {
		o.eventQueue.Push(event, events.Immediate())
	}
}

func (o *Orchestrator) processQueuedEvents(tick GameTick) []events.GameEvent {
	fired := o.eventQueue.Process(tick.CurrentTime, o.world)
	return o.applyEvents(fired)
}

func (o *Orchestrator) checkNaturalEvents(tick GameTick) []events.GameEvent {
	generated := events.CheckNaturalEvents(o.world, o.rng)
	return o.applyEvents(o.eventsAt(generated, tick.CurrentTime))
}

func (o *Orchestrator) checkSocialEvents(tick GameTick) []events.GameEvent {
	generated := events.CheckSocialEvents(o.world, o.rng)
	return o.applyEvents(o.eventsAt(generated, tick.CurrentTime))
}

func (o *Orchestrator) checkWorldEvents(tick GameTick) []events.GameEvent {
	generated := events.CheckWorldEvents(o.world, o.rng)
	return o.applyEvents(o.eventsAt(generated, tick.CurrentTime))
}

func (o *Orchestrator) checkEraTransition(tick GameTick) []events.GameEvent {
	if o.civilizationManager == nil {
		return nil
	}
	transition := o.civilizationManager.CheckEraTransition(o.world)
	if transition == nil {
		return nil
	}
	event := events.GameEvent{
		ID:           transition.NarrativeEvent.ID,
		Type:         events.EventEraChange,
		Timestamp:    tick.CurrentTime,
		Participants: append([]string(nil), transition.NarrativeEvent.Participants...),
		IsPublic:     true,
	}
	return o.applyEvents([]events.GameEvent{event})
}

func (o *Orchestrator) checkEndingConditions() {
	if o.activeEnding != nil || o.endingChecker == nil {
		return
	}
	o.activeEnding = o.endingChecker.CheckEndingConditions(o.world)
}

func (o *Orchestrator) handleNewDay() []events.GameEvent {
	processed := o.checkDailyCivilization()
	save.ExportNewspaper(o.world)
	if o.saveSystem != nil {
		o.lastError = o.saveSystem.AutoSave(o.world)
	}
	return processed
}

func (o *Orchestrator) checkDailyCivilization() []events.GameEvent {
	if o.civilizationManager == nil || o.world == nil {
		return nil
	}
	generated := o.civilizationManager.DailyUpdate(o.world)
	if len(generated) == 0 {
		return nil
	}
	processed := make([]events.GameEvent, 0, len(generated))
	for _, event := range generated {
		adapted := eventFromWorldHistory(event)
		if adapted.ID == "" {
			continue
		}
		o.applyEventToPobles(adapted)
		processed = append(processed, adapted)
	}
	return processed
}

func (o *Orchestrator) applyEvents(input []events.GameEvent) []events.GameEvent {
	if len(input) == 0 {
		return nil
	}
	processed := make([]events.GameEvent, 0, len(input))
	for _, event := range input {
		if o.hasRecordedEvent(event.ID) {
			continue
		}
		if strings.TrimSpace(event.Description) == "" {
			event.Description = events.GenerateEventDescription(event, events.TemplateContext{
				WorldState: o.world.GetWorldState(),
				Renderer:   o.templateEngine,
			})
		}
		deferred := events.ApplyConsequences(event, o.world)
		for _, delayed := range deferred {
			o.eventQueue.Push(delayed, events.InHours(0))
		}
		o.applyEventToPobles(event)
		o.recordWorldEvent(event)
		processed = append(processed, event)
	}
	return processed
}

func (o *Orchestrator) hasRecordedEvent(id string) bool {
	if strings.TrimSpace(id) == "" || o.world == nil {
		return false
	}
	for _, event := range o.world.EventHistory {
		if event.ID == id {
			return true
		}
	}
	for _, event := range o.world.ActiveEvents {
		if event.ID == id {
			return true
		}
	}
	return false
}

func (o *Orchestrator) applyEventToPobles(event events.GameEvent) {
	adapted := adaptEventForAI(event)
	for _, poble := range o.world.GetAllPobles() {
		if poble == nil {
			continue
		}
		emotions := ai.NewEmotionSystem(poble)
		emotions.ProcessEvent(adapted)
		memory := ai.NewMemorySystem(poble, o.rng)
		memory.ProcessNewEvent(adapted, poble.ID)
	}
}

func (o *Orchestrator) publishEvents(processed []events.GameEvent) {
	if len(processed) == 0 {
		return
	}
	next := append([]events.GameEvent(nil), processed...)
	next = append(next, o.eventFeed...)
	if len(next) > maxEventFeed {
		next = next[:maxEventFeed]
	}
	o.eventFeed = next
}

func (o *Orchestrator) recordWorldEvent(event events.GameEvent) {
	adapted := adaptEventForAI(event)
	o.world.ActiveEvents = append(o.world.ActiveEvents, adapted)
	o.world.EventHistory = append(o.world.EventHistory, adapted)
	if len(o.world.ActiveEvents) > maxEventFeed {
		o.world.ActiveEvents = o.world.ActiveEvents[len(o.world.ActiveEvents)-maxEventFeed:]
	}
}

func (o *Orchestrator) eventsAt(input []events.GameEvent, now GameTime) []events.GameEvent {
	for index := range input {
		if input[index].Timestamp.ToMinutes() == 0 {
			input[index].Timestamp = now
		}
	}
	return input
}

func (o *Orchestrator) worldContextFor(poble *entities.Poble) ai.WorldContext {
	if poble == nil || o.world == nil {
		return ai.WorldContext{}
	}
	return ai.WorldContext{
		ConflictActive: o.hasRecentConflict(poble.ID),
		IsAlone:        o.world.GetPopulation() <= 1,
		HasControl:     poble.Personality.Ambition >= 70 || poble.Archetype == entities.ArchetypeRuler,
		ActiveGoals:    countHighNeeds(poble.Needs),
		HoursSinceSex:  hoursSinceTaggedMemory(poble, "event:intimacy", o.world.Calendar),
	}
}

func (o *Orchestrator) hasRecentConflict(pobleID string) bool {
	for i := len(o.world.ActiveEvents) - 1; i >= 0 && i >= len(o.world.ActiveEvents)-12; i-- {
		event := o.world.ActiveEvents[i]
		if !containsString(event.Participants, pobleID) {
			continue
		}
		if event.Type == ai.GameEventConflict || event.Type == ai.GameEventThreat || event.Type == ai.GameEventBetrayal {
			return true
		}
	}
	return false
}

func (o *Orchestrator) applyActionNeedResult(poble *entities.Poble, action ai.Action) {
	needs := ai.NewNeedsSystem(poble)
	switch action.Type {
	case ai.ActionEat:
		needs.SatisfyNeed(ai.NeedHunger, 35)
	case ai.ActionDrink:
		needs.SatisfyNeed(ai.NeedThirst, 40)
	case ai.ActionSleep:
		needs.SatisfyNeed(ai.NeedSleep, 55)
	case ai.ActionRest:
		needs.SatisfyNeed(ai.NeedSleep, 20)
	case ai.ActionTalkTo, ai.ActionParty, ai.ActionGossipWith:
		needs.SatisfyNeed(ai.NeedBelonging, 18)
	case ai.ActionWork, ai.ActionGovern, ai.ActionBuild:
		needs.SatisfyNeed(ai.NeedPurpose, 14)
	case ai.ActionHaveSex, ai.ActionFlirtWith:
		needs.SatisfyNeed(ai.NeedSex, 25)
	}
}

func (o *Orchestrator) applyActionNarrativeArtifacts(poble *entities.Poble, action ai.Action, now GameTime) {
	if poble == nil || o.templateEngine == nil {
		return
	}
	target := o.actionTarget(poble, action)
	switch action.Type {
	case ai.ActionWriteDiary:
		o.appendDiaryEntry(poble, target, now)
	case ai.ActionSendLetter:
		o.appendLetter(poble, target, now, true)
	case ai.ActionSleep:
		o.appendDream(poble, target, now)
	case ai.ActionPlanRevenge:
		o.appendThought(poble, target, now, []string{"thoughts/about_other/resentment", "thoughts/random/general"}, []string{"thought", "revenge", "private"})
	case ai.ActionHaveBreakdown:
		o.appendThought(poble, target, now, []string{"thoughts/about_self/insecurity", "thoughts/night/general"}, []string{"thought", "breakdown", "private"})
	}
}

func (o *Orchestrator) generatePassiveThought(poble *entities.Poble) {
	if poble == nil || o.world == nil || o.templateEngine == nil {
		return
	}
	now := o.world.Calendar
	if now.ToMinutes() == 0 || now.Minute != 0 || now.Hour%6 != 0 {
		return
	}
	if len(poble.Thoughts) > 0 {
		last := poble.Thoughts[len(poble.Thoughts)-1].Timestamp
		if now.ToMinutes()-last.ToMinutes() < thoughtEveryMinutes {
			return
		}
	}
	target := o.narrativeTarget(poble)
	o.appendThought(poble, target, now, thoughtCategoriesForPoble(poble, target, now), []string{"thought", "passive", "private"})
}

func (o *Orchestrator) appendThought(poble *entities.Poble, target *entities.Poble, now GameTime, categories []string, tags []string) {
	text, category, ok := o.renderPobleTemplate(poble, target, now, categories)
	if !ok {
		return
	}
	poble.Thoughts = append(poble.Thoughts, entities.Thought{
		ID:        narrativeArtifactID("thought", poble.ID, now, len(poble.Thoughts)+1),
		Timestamp: now,
		Text:      text,
		Tags:      appendCategoryTag(tags, category),
	})
	if len(poble.Thoughts) > maxPobleThoughts {
		poble.Thoughts = poble.Thoughts[len(poble.Thoughts)-maxPobleThoughts:]
	}
}

func (o *Orchestrator) appendDream(poble *entities.Poble, target *entities.Poble, now GameTime) {
	text, category, ok := o.renderPobleTemplate(poble, target, now, dreamCategoriesForPoble(poble, target))
	if !ok {
		return
	}
	poble.Dreams = append(poble.Dreams, entities.Dream{
		ID:        narrativeArtifactID("dream", poble.ID, now, len(poble.Dreams)+1),
		Timestamp: now,
		Text:      text,
		Category:  category,
		IsPrivate: true,
		Tags:      appendCategoryTag([]string{"dream", "private"}, category),
	})
	if len(poble.Dreams) > maxPobleDreams {
		poble.Dreams = poble.Dreams[len(poble.Dreams)-maxPobleDreams:]
	}
}

func (o *Orchestrator) appendDiaryEntry(poble *entities.Poble, target *entities.Poble, now GameTime) {
	text, category, ok := o.renderPobleTemplate(poble, target, now, diaryCategoriesForPoble(poble, target))
	if !ok {
		return
	}
	poble.DiaryEntries = append(poble.DiaryEntries, entities.DiaryEntry{
		ID:        narrativeArtifactID("diary", poble.ID, now, len(poble.DiaryEntries)+1),
		Timestamp: now,
		Text:      text,
		Mood:      poble.CurrentMood,
		Tags:      appendCategoryTag([]string{"diary", "private"}, category),
	})
	if len(poble.DiaryEntries) > maxPobleDiaryPages {
		poble.DiaryEntries = poble.DiaryEntries[len(poble.DiaryEntries)-maxPobleDiaryPages:]
	}
}

func (o *Orchestrator) appendLetter(poble *entities.Poble, target *entities.Poble, now GameTime, sent bool) {
	if target == nil {
		return
	}
	text, category, ok := o.renderPobleTemplate(poble, target, now, letterCategoriesForPoble(poble, target))
	if !ok {
		return
	}
	poble.Letters = append(poble.Letters, entities.Letter{
		ID:        narrativeArtifactID("letter", poble.ID, now, len(poble.Letters)+1),
		Timestamp: now,
		ToID:      target.ID,
		Text:      text,
		IsSent:    sent,
		Tags:      appendCategoryTag([]string{"letter", "private"}, category),
	})
	if len(poble.Letters) > maxPobleLetters {
		poble.Letters = poble.Letters[len(poble.Letters)-maxPobleLetters:]
	}
}

func (o *Orchestrator) renderPobleTemplate(poble *entities.Poble, target *entities.Poble, now GameTime, categories []string) (string, string, bool) {
	if o.templateEngine == nil || len(categories) == 0 {
		return "", "", false
	}
	ctx := o.templateContextFor(poble, target, now)
	for _, category := range dedupeStrings(categories) {
		template, err := o.templateEngine.Select(category, ctx)
		if err != nil {
			continue
		}
		rendered, err := o.templateEngine.Render(template, ctx)
		if err != nil {
			continue
		}
		rendered = strings.TrimSpace(rendered)
		if rendered != "" {
			return rendered, category, true
		}
	}
	return "", "", false
}

func (o *Orchestrator) templateContextFor(poble *entities.Poble, target *entities.Poble, now GameTime) templates.TemplateContext {
	recent, old := narrativeMemoriesFor(poble, target)
	relationship := relationshipWithTarget(poble, target)
	location := o.locationNameFor(poble)
	worldState := entities.WorldState{}
	if o.world != nil {
		worldState = o.world.GetWorldState()
	}
	extra := map[string]string{
		"age":                 fmt.Sprintf("%d", poble.Age),
		"event_description":   memoryText(recent),
		"specific_grievance":  memoryText(recent),
		"the_thing_wanted":    targetName(target),
		"future_event_hint":   futureHintFor(poble, target),
		"encounter_mood":      strings.ToLower(poble.CurrentMood.String()),
		"location_name":       location,
		"location_atmosphere": locationAtmosphereFor(poble),
		"location_memory":     memoryText(old),
	}
	return templates.TemplateContext{
		Speaker:                poble,
		Target:                 target,
		Location:               location,
		GameTime:               now,
		RecentMemory:           recent,
		OldMemory:              old,
		RelationshipWithTarget: relationship,
		WorldState:             worldState,
		ExtraVars:              extra,
	}
}

func (o *Orchestrator) actionTarget(poble *entities.Poble, action ai.Action) *entities.Poble {
	if o.world == nil {
		return nil
	}
	if strings.TrimSpace(action.TargetID) != "" {
		return o.world.GetPoble(action.TargetID)
	}
	return o.narrativeTarget(poble)
}

func (o *Orchestrator) narrativeTarget(poble *entities.Poble) *entities.Poble {
	if o.world == nil || poble == nil || len(poble.Relationships) == 0 {
		return nil
	}
	ids := make([]string, 0, len(poble.Relationships))
	for id := range poble.Relationships {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	bestID := ""
	bestScore := float32(-1)
	for _, id := range ids {
		target := o.world.GetPoble(id)
		if target == nil {
			continue
		}
		rel := poble.Relationships[id]
		score := rel.Familiarity + rel.Affection + rel.Attraction + rel.Resentment + rel.Fear + rel.Dependency
		if score > bestScore {
			bestScore = score
			bestID = id
		}
	}
	if bestID == "" {
		return nil
	}
	return o.world.GetPoble(bestID)
}

func (o *Orchestrator) locationNameFor(poble *entities.Poble) string {
	if o.world == nil || poble == nil {
		return "el asentamiento"
	}
	location, ok := o.world.GetLocation(poble.ID)
	if !ok || strings.TrimSpace(location.Name) == "" {
		return "el asentamiento"
	}
	return location.Name
}

func thoughtCategoriesForPoble(poble *entities.Poble, target *entities.Poble, now GameTime) []string {
	categories := []string{}
	if rel := relationshipWithTarget(poble, target); rel != nil {
		switch {
		case (rel.Trust >= 50 || rel.Affection >= 50) && rel.Resentment >= 30:
			categories = append(categories, "thoughts/about_other/reconciliation", "thoughts/about_other/resentment")
		case rel.Resentment >= 55 || rel.Fear >= 70:
			categories = append(categories, "thoughts/about_other/resentment", "thoughts/about_other/grief")
		case rel.Attraction >= 55:
			categories = append(categories, "thoughts/about_other/jealousy", "thoughts/about_other/attraction")
		case rel.Dependency >= 60:
			categories = append(categories, "thoughts/about_other/obsession")
		}
	}
	if poble != nil {
		switch {
		case poble.CurrentMood == entities.MoodSad || poble.CurrentMood == entities.MoodDepressed || poble.Mental.Stability <= 45:
			categories = append(categories, "thoughts/about_other/grief", "thoughts/about_self/insecurity")
		case poble.CurrentMood == entities.MoodEuphoric || poble.Needs.Esteem <= 25:
			categories = append(categories, "thoughts/about_self/pride")
		case poble.Needs.Sex >= 75:
			categories = append(categories, "thoughts/about_self/desire")
		}
		if len(poble.Children) > 0 || hasParentLink(poble) {
			categories = append(categories, "thoughts/about_self/existential")
		}
		if len(poble.Secrets) > 0 {
			categories = append(categories, "thoughts/about_self/insecurity")
		}
		if category := archetypeThoughtCategory(poble.Archetype); category != "" {
			categories = append(categories, category)
		}
	}
	if now.Hour >= 5 && now.Hour <= 10 {
		categories = append(categories, "thoughts/morning/general")
	}
	if now.Hour >= 21 || now.Hour <= 4 {
		categories = append(categories, "thoughts/night/general")
	}
	return append(categories, "thoughts/random/hyperspecific", "thoughts/random/general")
}

func dreamCategoriesForPoble(poble *entities.Poble, target *entities.Poble) []string {
	categories := []string{}
	if poble != nil {
		switch {
		case poble.CurrentMood == entities.MoodAnxious || poble.CurrentMood == entities.MoodDepressed || poble.Mental.Stability <= 45:
			categories = append(categories, "dreams/nightmare/general")
		case poble.Archetype == entities.ArchetypeProphet:
			categories = append(categories, "dreams/prophetic/general")
		case poble.Needs.Sex >= 65:
			categories = append(categories, "dreams/erotic/general")
		case target != nil:
			categories = append(categories, "dreams/wish_fulfillment/general")
		}
	}
	return append(categories, "dreams/nonsense/general", "dreams/wish_fulfillment/general")
}

func diaryCategoriesForPoble(poble *entities.Poble, target *entities.Poble) []string {
	categories := []string{}
	if poble != nil {
		if len(poble.Secrets) > 0 {
			categories = append(categories, "diary/secret_keeping/general")
		}
		if len(poble.Children) > 0 || hasParentLink(poble) {
			categories = append(categories, "diary/family/general")
		}
		if rel := relationshipWithTarget(poble, target); rel != nil && rel.Attraction >= 45 && rel.Affection <= 35 {
			categories = append(categories, "diary/heartbreak/general")
		}
		if poble.Archetype == entities.ArchetypeSchemer || poble.Archetype == entities.ArchetypeRuler || poble.Personality.Ambition >= 70 {
			categories = append(categories, "diary/planning/general")
		}
	}
	return append(categories, "diary/daily/general")
}

func letterCategoriesForPoble(poble *entities.Poble, target *entities.Poble) []string {
	if rel := relationshipWithTarget(poble, target); rel != nil {
		if rel.Resentment >= 50 || rel.Fear >= 70 || rel.Trust <= 25 {
			return []string{"letters/hate/general", "letters/apology/general", "letters/love/general"}
		}
	}
	return []string{"letters/love/general", "letters/apology/general", "letters/hate/general"}
}

func hasParentLink(poble *entities.Poble) bool {
	if poble == nil {
		return false
	}
	return strings.TrimSpace(poble.Parents[0]) != "" || strings.TrimSpace(poble.Parents[1]) != ""
}

func archetypeThoughtCategory(archetype entities.ArchetypeID) string {
	switch archetype {
	case entities.ArchetypeRuler:
		return "thoughts/by_archetype/ruler"
	case entities.ArchetypeLover:
		return "thoughts/by_archetype/lover"
	case entities.ArchetypeJester:
		return "thoughts/by_archetype/jester"
	case entities.ArchetypeVillain:
		return "thoughts/by_archetype/villain"
	case entities.ArchetypeGhost:
		return "thoughts/by_archetype/ghost"
	case entities.ArchetypeProphet:
		return "thoughts/by_archetype/prophet"
	case entities.ArchetypeInnocent:
		return "thoughts/by_archetype/innocent"
	default:
		return ""
	}
}

func narrativeMemoriesFor(poble *entities.Poble, target *entities.Poble) (*entities.Memory, *entities.Memory) {
	if poble == nil {
		return nil, nil
	}
	targetID := ""
	if target != nil {
		targetID = target.ID
	}
	var recent *entities.Memory
	var old *entities.Memory
	for index := range poble.Memories {
		memory := &poble.Memories[index]
		if memory.IsRepressed || (targetID != "" && !containsString(memory.Participants, targetID)) {
			continue
		}
		if recent == nil || memory.Timestamp.ToMinutes() > recent.Timestamp.ToMinutes() {
			recent = memory
		}
		if old == nil || memory.EmotionIntensity > old.EmotionIntensity {
			old = memory
		}
	}
	return recent, old
}

func relationshipWithTarget(poble *entities.Poble, target *entities.Poble) *entities.Relationship {
	if poble == nil || target == nil || len(poble.Relationships) == 0 {
		return nil
	}
	relationship, ok := poble.Relationships[target.ID]
	if !ok {
		return nil
	}
	return &relationship
}

func narrativeArtifactID(kind string, pobleID string, now GameTime, index int) string {
	return fmt.Sprintf("%s:%s:%d:%d", kind, pobleID, now.ToMinutes(), index)
}

func appendCategoryTag(tags []string, category string) []string {
	out := append([]string(nil), tags...)
	if strings.TrimSpace(category) != "" {
		out = append(out, "category:"+category)
	}
	return out
}

func dedupeStrings(values []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

func memoryText(memory *entities.Memory) string {
	if memory == nil || strings.TrimSpace(memory.Summary) == "" {
		return "algo que no termina de acomodarse"
	}
	return memory.Summary
}

func targetName(target *entities.Poble) string {
	if target == nil || strings.TrimSpace(target.Name) == "" {
		return "eso que no deberia importar tanto"
	}
	return target.Name
}

func futureHintFor(poble *entities.Poble, target *entities.Poble) string {
	if rel := relationshipWithTarget(poble, target); rel != nil {
		switch {
		case rel.Resentment >= 55:
			return "una pelea que todavia esta buscando fecha"
		case rel.Attraction >= 55:
			return "una conversacion que nadie se atreve a empezar"
		case rel.Trust <= 25:
			return "una verdad que se va a caer sola"
		}
	}
	return "algo pequeno que podria cambiarlo todo"
}

func locationAtmosphereFor(poble *entities.Poble) string {
	if poble == nil {
		return "normal"
	}
	switch {
	case poble.Mental.Stability <= 45:
		return "demasiado apretado"
	case poble.CurrentMood == entities.MoodHappy || poble.CurrentMood == entities.MoodContent:
		return "casi amable"
	case poble.CurrentMood == entities.MoodAngry:
		return "afilado"
	default:
		return "inquieto"
	}
}

func (o *Orchestrator) eventFromAction(poble *entities.Poble, action ai.Action, now GameTime) (events.GameEvent, bool) {
	eventType, public, ok := eventTypeForAction(action.Type)
	if !ok {
		return events.GameEvent{}, false
	}
	participants := []string{poble.ID}
	if action.TargetID != "" {
		participants = append(participants, action.TargetID)
	}
	return events.GameEvent{
		ID:           actionEventID(poble.ID, action, now),
		Type:         eventType,
		Timestamp:    now,
		Participants: participants,
		IsPublic:     public,
		Description:  actionDescriptionFor(poble, action),
		Consequences: actionConsequences(poble.ID, action),
	}, true
}

func actionDescriptionFor(poble *entities.Poble, action ai.Action) string {
	if poble == nil {
		return ""
	}
	switch action.Type {
	case ai.ActionWriteDiary:
		if len(poble.DiaryEntries) > 0 {
			return poble.DiaryEntries[len(poble.DiaryEntries)-1].Text
		}
	case ai.ActionSendLetter:
		if len(poble.Letters) > 0 {
			return poble.Letters[len(poble.Letters)-1].Text
		}
	case ai.ActionPlanRevenge, ai.ActionHaveBreakdown:
		if len(poble.Thoughts) > 0 {
			return poble.Thoughts[len(poble.Thoughts)-1].Text
		}
	}
	return ""
}

func eventTypeForAction(action ai.ActionType) (events.EventType, bool, bool) {
	switch action {
	case ai.ActionArgueWith:
		return events.EventFightVerbal, true, true
	case ai.ActionFight:
		return events.EventFightPhysical, true, true
	case ai.ActionMurder:
		return events.EventDeathMurder, true, true
	case ai.ActionConfessTo:
		return events.EventRevelation, false, true
	case ai.ActionGossipWith:
		return events.EventRumourSpread, true, true
	case ai.ActionThreaten:
		return events.EventDecisionPoint, false, true
	case ai.ActionHaveSex:
		return events.EventSexualEncounter, false, true
	case ai.ActionFlirtWith, ai.ActionPropose, ai.ActionBreakUp:
		return events.EventDecisionPoint, false, true
	case ai.ActionExplore, ai.ActionResearch, ai.ActionBuild:
		return events.EventDecisionPoint, true, true
	case ai.ActionWriteDiary, ai.ActionSendLetter, ai.ActionObserveSecretly, ai.ActionPlanRevenge:
		return events.EventDecisionPoint, false, true
	case ai.ActionBetray:
		return events.EventBetrayalRevealed, false, true
	case ai.ActionFormAlliance, ai.ActionGovern:
		return events.EventElection, true, true
	case ai.ActionTrade:
		return events.EventTradeEstablished, true, true
	case ai.ActionPray:
		return events.EventRitual, true, true
	case ai.ActionParty:
		return events.EventParty, true, true
	case ai.ActionHaveBreakdown:
		return events.EventMentalBreakdown, false, true
	default:
		return "", false, false
	}
}

func actionConsequences(actorID string, action ai.Action) []events.Consequence {
	switch action.Type {
	case ai.ActionFight:
		return []events.Consequence{{TargetID: action.TargetID, Type: events.ConsequenceHealthChange, Value: -8}}
	case ai.ActionMurder:
		return []events.Consequence{{TargetID: action.TargetID, Type: events.ConsequenceDeathCaused, Value: 1}}
	case ai.ActionArgueWith:
		return []events.Consequence{{TargetID: actorID, Type: events.ConsequenceMoodShift, Value: -8}}
	case ai.ActionHaveSex:
		return []events.Consequence{{TargetID: actorID, Type: events.ConsequenceNeedChange, Value: -15}}
	case ai.ActionBetray, ai.ActionBreakUp:
		return []events.Consequence{{TargetID: action.TargetID, Type: events.ConsequenceRelationshipChange, Value: -18}}
	case ai.ActionSendLetter, ai.ActionWriteDiary:
		return []events.Consequence{{TargetID: actorID, Type: events.ConsequenceMemoryCreated, Value: 1}}
	case ai.ActionParty:
		return []events.Consequence{{TargetID: actorID, Type: events.ConsequenceMoodShift, Value: 10}}
	case ai.ActionHaveBreakdown:
		return []events.Consequence{{TargetID: actorID, Type: events.ConsequenceMentalChange, Value: -15}}
	default:
		return nil
	}
}

func adaptEventForAI(event events.GameEvent) ai.GameEvent {
	adapted := ai.GameEvent{
		ID:           event.ID,
		Type:         aiTypeForEvent(event.Type),
		Time:         event.Timestamp,
		Participants: append([]string(nil), event.Participants...),
		Severity:     severityForEvent(event.Type),
		Valence:      valenceForEvent(event.Type),
		Description:  event.Description,
		Tags:         []string{"event:" + strings.ToLower(string(event.Type))},
	}
	if len(event.Participants) > 0 {
		adapted.PrimaryActor = event.Participants[0]
	}
	if len(event.Participants) > 1 {
		adapted.TargetID = event.Participants[1]
	}
	adapted.IsTraumatic = adapted.Type == ai.GameEventDeath || adapted.Severity >= 0.85
	return adapted
}

func eventFromWorldHistory(event world.GameEvent) events.GameEvent {
	return events.GameEvent{
		ID:           event.ID,
		Type:         eventTypeForWorldHistory(event),
		Timestamp:    event.Time,
		Participants: append([]string(nil), event.Participants...),
		IsPublic:     true,
		Description:  event.Description,
	}
}

func eventTypeForWorldHistory(event world.GameEvent) events.EventType {
	if hasAnyString(event.Tags, "coup") {
		return events.EventCoup
	}
	if hasAnyString(event.Tags, "revolution") {
		return events.EventRevolution
	}
	if hasAnyString(event.Tags, "election") {
		return events.EventElection
	}
	if hasAnyString(event.Tags, "technology") {
		return events.EventTechDiscovered
	}
	if hasAnyString(event.Tags, "government", "institution") {
		return events.EventDecisionPoint
	}
	if hasAnyString(event.Tags, "trade") {
		return events.EventTradeEstablished
	}
	if hasAnyString(event.Tags, "gambling") {
		return events.EventGamblingResult
	}
	if hasAnyString(event.Tags, "theft") {
		return events.EventTheft
	}
	if hasAnyString(event.Tags, "pregnancy") {
		return events.EventPregnancy
	}
	if hasAnyString(event.Tags, "health", "illness") {
		return events.EventIllnessOnset
	}
	if hasAnyString(event.Tags, "sti", "encounter", "intimacy") {
		return events.EventSexualEncounter
	}
	if hasAnyString(event.Tags, "scarcity", "economy") {
		return events.EventResourceDepletion
	}
	switch event.Type {
	case ai.GameEventConflict:
		return events.EventFightVerbal
	case ai.GameEventBetrayal:
		return events.EventBetrayalRevealed
	case ai.GameEventDeath:
		return events.EventDeathNatural
	case ai.GameEventGoalComplete:
		return events.EventDecisionPoint
	default:
		return events.EventDecisionPoint
	}
}

func aiTypeForEvent(eventType events.EventType) ai.GameEventType {
	switch eventType {
	case events.EventDeathNatural, events.EventDeathAccident, events.EventDeathMurder, events.EventSuicide:
		return ai.GameEventDeath
	case events.EventFightVerbal, events.EventFightPhysical, events.EventWarDeclaration, events.EventCoup:
		return ai.GameEventConflict
	case events.EventBetrayalRevealed, events.EventAffairStart:
		return ai.GameEventBetrayal
	case events.EventBirth, events.EventMarriage, events.EventRecovery, events.EventTechDiscovered, events.EventEraChange:
		return ai.GameEventGoalComplete
	case events.EventSexualEncounter, events.EventPregnancy:
		return ai.GameEventIntimacy
	case events.EventParty, events.EventForgiveness, events.EventTradeEstablished, events.EventRitual, events.EventReligionFounded,
		events.EventElection, events.EventPeaceTreaty:
		return ai.GameEventSocialPositive
	case events.EventDivorce, events.EventExile, events.EventPublicHumiliation, events.EventTheft, events.EventDebt,
		events.EventGamblingLoss, events.EventMentalBreakdown:
		return ai.GameEventSocialNegative
	default:
		return ai.GameEventGeneric
	}
}

func severityForEvent(eventType events.EventType) float32 {
	switch eventType {
	case events.EventDeathNatural, events.EventDeathAccident, events.EventDeathMurder, events.EventSuicide:
		return 0.95
	case events.EventFightPhysical, events.EventWarDeclaration, events.EventPlague, events.EventCoup:
		return 0.82
	case events.EventPregnancy, events.EventBirth, events.EventEraChange, events.EventTechDiscovered:
		return 0.72
	default:
		return 0.42
	}
}

func valenceForEvent(eventType events.EventType) float32 {
	switch eventType {
	case events.EventDeathNatural, events.EventDeathAccident, events.EventDeathMurder, events.EventSuicide,
		events.EventFightVerbal, events.EventFightPhysical, events.EventWarDeclaration, events.EventPlague:
		return -0.75
	case events.EventBirth, events.EventRecovery, events.EventMarriage, events.EventEraChange, events.EventTechDiscovered:
		return 0.65
	default:
		return 0
	}
}

func linkFounders(a, b *entities.Poble) {
	if a == nil || b == nil {
		return
	}
	ab := entities.NewRelationship(b.ID, entities.RelationshipComplicated)
	ab.Familiarity = 70
	ab.Affection = 42
	ab.Trust = 48
	ab.Attraction = 38
	ab.Tags = []string{"founder", "day_zero"}
	ba := entities.NewRelationship(a.ID, entities.RelationshipComplicated)
	ba.Familiarity = 70
	ba.Affection = 40
	ba.Trust = 46
	ba.Attraction = 36
	ba.Tags = []string{"founder", "day_zero"}
	a.Relationships[b.ID] = ab
	b.Relationships[a.ID] = ba
}

func snapshotPobles(pobles []*entities.Poble, decisions map[string]*ai.DecisionEngine) []PobleSnapshot {
	snapshots := make([]PobleSnapshot, 0, len(pobles))
	for _, poble := range pobles {
		if poble == nil {
			continue
		}
		intent := "intent:IDLE"
		reason := "reason:idle"
		if decision := decisions[poble.ID]; decision != nil {
			intent = decision.GetCurrentIntent()
			reason = decision.GetCurrentReason()
		}
		snapshots = append(snapshots, PobleSnapshot{
			ID:            poble.ID,
			Name:          poble.Name,
			Age:           poble.Age,
			Archetype:     poble.Archetype,
			Mood:          poble.CurrentMood,
			IsAlive:       poble.IsAlive,
			Needs:         poble.Needs,
			CurrentIntent: intent,
			IntentReason:  reason,
		})
	}
	sort.Slice(snapshots, func(i, j int) bool {
		return snapshots[i].ID < snapshots[j].ID
	})
	return snapshots
}

func actionEventID(pobleID string, action ai.Action, now GameTime) string {
	target := action.TargetID
	if target == "" {
		target = "world"
	}
	return fmt.Sprintf("action:%s:%s:%s:%d", strings.ToLower(string(action.Type)), pobleID, target, now.ToMinutes())
}

func countHighNeeds(needs entities.Needs) int {
	values := []float32{needs.Hunger, needs.Thirst, needs.Sleep, needs.Safety, needs.Belonging, needs.Esteem, needs.Sex, needs.Power, needs.Purpose}
	count := 0
	for _, value := range values {
		if value >= 70 {
			count++
		}
	}
	return count
}

func hoursSinceTaggedMemory(poble *entities.Poble, tag string, now GameTime) int {
	if poble == nil {
		return 999
	}
	last := -1
	for _, memory := range poble.Memories {
		if containsString(memory.Tags, tag) && memory.Timestamp.ToMinutes() > last {
			last = memory.Timestamp.ToMinutes()
		}
	}
	if last < 0 {
		return 999
	}
	return (now.ToMinutes() - last) / 60
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func hasAnyString(values []string, targets ...string) bool {
	for _, value := range values {
		for _, target := range targets {
			if value == target {
				return true
			}
		}
	}
	return false
}

func (o *Orchestrator) startSpeed() float64 {
	if o.options.Speed <= 0 {
		return 1.0
	}
	return o.options.Speed
}

func templateRoot() string {
	if root := strings.TrimSpace(os.Getenv("POBLATION_TEMPLATES_DIR")); root != "" {
		if _, err := os.Stat(root); err == nil {
			return root
		}
	}
	if _, err := os.Stat("templates"); err == nil {
		return "templates"
	}
	wd, err := os.Getwd()
	if err != nil {
		return "templates"
	}
	return filepath.Join(wd, "templates")
}
