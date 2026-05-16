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

const maxEventFeed = 80

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
	ID        string
	Name      string
	Age       int
	Archetype entities.ArchetypeID
	Mood      entities.MoodType
	IsAlive   bool
	Needs     entities.Needs
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
	processed = append(processed, o.checkWorldEvents(tick)...)
	processed = append(processed, o.checkEraTransition(tick)...)
	o.checkEndingConditions()

	if tick.IsNewDay {
		o.handleNewDay()
	}
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
		snapshot.Pobles = snapshotPobles(o.world.GetAllKnownPobles())
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
}

func (o *Orchestrator) enqueueActionEvents(poble *entities.Poble, actions []ai.Action, now GameTime) {
	if len(actions) == 0 {
		return
	}
	action := actions[0]
	o.applyActionNeedResult(poble, action)
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

func (o *Orchestrator) handleNewDay() {
	save.ExportNewspaper(o.world)
	if o.saveSystem != nil {
		o.lastError = o.saveSystem.AutoSave(o.world)
	}
}

func (o *Orchestrator) applyEvents(input []events.GameEvent) []events.GameEvent {
	if len(input) == 0 {
		return nil
	}
	processed := make([]events.GameEvent, 0, len(input))
	for _, event := range input {
		if strings.TrimSpace(event.Description) == "" {
			event.Description = events.GenerateEventDescription(event, events.TemplateContext{
				WorldState: o.world.GetWorldState(),
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
		Consequences: actionConsequences(poble.ID, action),
	}, true
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

func snapshotPobles(pobles []*entities.Poble) []PobleSnapshot {
	snapshots := make([]PobleSnapshot, 0, len(pobles))
	for _, poble := range pobles {
		if poble == nil {
			continue
		}
		snapshots = append(snapshots, PobleSnapshot{
			ID:        poble.ID,
			Name:      poble.Name,
			Age:       poble.Age,
			Archetype: poble.Archetype,
			Mood:      poble.CurrentMood,
			IsAlive:   poble.IsAlive,
			Needs:     poble.Needs,
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

func (o *Orchestrator) startSpeed() float64 {
	if o.options.Speed <= 0 {
		return 1.0
	}
	return o.options.Speed
}

func templateRoot() string {
	if _, err := os.Stat("templates"); err == nil {
		return "templates"
	}
	wd, err := os.Getwd()
	if err != nil {
		return "templates"
	}
	return filepath.Join(wd, "templates")
}
