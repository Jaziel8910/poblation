package events

import (
	"fmt"
	"math/rand"
	"strings"

	"github.com/user/poblation/internal/entities"
)

// EventType identifies event categories from the GDD (Personal, Social, World, Economy, Meta).
type EventType string

const (
	// Personal events.
	EventBirthday         EventType = "BIRTHDAY"
	EventIllnessOnset     EventType = "ILLNESS_ONSET"
	EventRecovery         EventType = "RECOVERY"
	EventMentalBreakdown  EventType = "MENTAL_BREAKDOWN"
	EventRevelation       EventType = "REVELATION"
	EventDecisionPoint    EventType = "DECISION_POINT"
	EventSexualEncounter  EventType = "SEXUAL_ENCOUNTER"
	EventPregnancy        EventType = "PREGNANCY"
	EventMiscarriage      EventType = "MISCARRIAGE"
	EventBirth            EventType = "BIRTH"
	EventDeathNatural     EventType = "DEATH_NATURAL"
	EventDeathAccident    EventType = "DEATH_ACCIDENT"
	EventDeathMurder      EventType = "DEATH_MURDER"
	EventSuicide          EventType = "SUICIDE"
	EventKinkDiscovery    EventType = "KINK_DISCOVERY"
	EventAffairStart      EventType = "AFFAIR_START"
	EventAffairEnd        EventType = "AFFAIR_END"
	EventObsessionPeak    EventType = "OBSESSION_PEAK"
	EventStalking         EventType = "STALKING"
	EventRestrainingOrder EventType = "RESTRAINING_ORDER"

	// Social events.
	EventFightVerbal       EventType = "FIGHT_VERBAL"
	EventFightPhysical     EventType = "FIGHT_PHYSICAL"
	EventWarDeclaration    EventType = "WAR_DECLARATION"
	EventPeaceTreaty       EventType = "PEACE_TREATY"
	EventMarriage          EventType = "MARRIAGE"
	EventDivorce           EventType = "DIVORCE"
	EventAdoption          EventType = "ADOPTION"
	EventBetrayalRevealed  EventType = "BETRAYAL_REVEALED"
	EventForgiveness       EventType = "FORGIVENESS"
	EventExile             EventType = "EXILE"
	EventRumourSpread      EventType = "RUMOUR_SPREAD"
	EventGossipChain       EventType = "GOSSIP_CHAIN"
	EventPublicHumiliation EventType = "PUBLIC_HUMILIATION"
	EventParty             EventType = "PARTY"
	EventFuneral           EventType = "FUNERAL"
	EventRitual            EventType = "RITUAL"
	EventReligionFounded   EventType = "RELIGION_FOUNDED"
	EventElection          EventType = "ELECTION"
	EventCoup              EventType = "COUP"
	EventRevolution        EventType = "REVOLUTION"

	// World events.
	EventEarthquake        EventType = "EARTHQUAKE"
	EventStorm             EventType = "STORM"
	EventDrought           EventType = "DROUGHT"
	EventPlague            EventType = "PLAGUE"
	EventIslandDiscovery   EventType = "ISLAND_DISCOVERY"
	EventResourceDepletion EventType = "RESOURCE_DEPLETION"
	EventTechDiscovered    EventType = "TECHNOLOGY_DISCOVERED"
	EventBuildingCollapsed EventType = "BUILDING_COLLAPSED"
	EventAnimalAttack      EventType = "ANIMAL_ATTACK"
	EventFire              EventType = "FIRE"
	EventFlood             EventType = "FLOOD"

	// Economy events.
	EventTradeEstablished EventType = "TRADE_ESTABLISHED"
	EventMonopolyFormed   EventType = "MONOPOLY_FORMED"
	EventTheft            EventType = "THEFT"
	EventDebt             EventType = "DEBT"
	EventInheritance      EventType = "INHERITANCE"
	EventGamblingWin      EventType = "GAMBLING_WIN"
	EventGamblingLoss     EventType = "GAMBLING_LOSS"
	EventGamblingResult   EventType = "GAMBLING_RESULT"

	// Meta events.
	EventGenerationEnd        EventType = "GENERATION_END"
	EventLastPersonAlive      EventType = "LAST_PERSON_ALIVE"
	EventCivilizationCollapse EventType = "CIVILIZATION_COLLAPSE"
	EventPopulationMilestone  EventType = "POPULATION_MILESTONE"
	EventEraChange            EventType = "ERA_CHANGE"

	// System-specific events.
	EventTeaching         EventType = "TEACHING"
	EventNicknameRevealed EventType = "NICKNAME_REVEALED"
)

// ConsequenceType identifies what aspect of the world a consequence modifies.
type ConsequenceType string

const (
	ConsequenceRelationshipChange ConsequenceType = "RELATIONSHIP_CHANGE"
	ConsequenceHealthChange       ConsequenceType = "HEALTH_CHANGE"
	ConsequenceMentalChange       ConsequenceType = "MENTAL_CHANGE"
	ConsequenceNeedChange         ConsequenceType = "NEED_CHANGE"
	ConsequenceMemoryCreated      ConsequenceType = "MEMORY_CREATED"
	ConsequenceSecretRevealed     ConsequenceType = "SECRET_REVEALED"
	ConsequenceRumourCreated      ConsequenceType = "RUMOUR_CREATED"
	ConsequenceDeathCaused        ConsequenceType = "DEATH_CAUSED"
	ConsequencePregnancyCaused    ConsequenceType = "PREGNANCY_CAUSED"
	ConsequenceItemGained         ConsequenceType = "ITEM_GAINED"
	ConsequenceItemLost           ConsequenceType = "ITEM_LOST"
	ConsequenceMoneyChange        ConsequenceType = "MONEY_CHANGE"
	ConsequenceMoodShift          ConsequenceType = "MOOD_SHIFT"
	ConsequenceConditionAdded     ConsequenceType = "CONDITION_ADDED"
	ConsequenceConditionRemoved   ConsequenceType = "CONDITION_REMOVED"
)

// Consequence stores one deferred or immediate effect of an event.
type Consequence struct {
	// TargetID is the Poble ID affected by this consequence.
	TargetID string `json:"target_id"`
	// Type identifies what changes.
	Type ConsequenceType `json:"type"`
	// Value is the magnitude of the change (interpretation depends on Type).
	Value float32 `json:"value"`
	// Delay is how many in-game hours before this consequence activates.
	// Zero means immediate.
	Delay int `json:"delay"`
}

// GameEvent is the canonical event representation for the event queue system.
type GameEvent struct {
	// ID uniquely identifies this event.
	ID string `json:"id"`
	// Type categorizes the event per GDD section 9.
	Type EventType `json:"type"`
	// Timestamp records when the event occurred or will occur.
	Timestamp entities.GameTime `json:"timestamp"`
	// Participants stores Poble IDs involved in this event.
	Participants []string `json:"participants"`
	// IsPublic marks whether all pobles can perceive this event.
	IsPublic bool `json:"is_public"`
	// Description is the narrator-generated text for this event.
	Description string `json:"description"`
	// Consequences lists effects this event produces.
	Consequences []Consequence `json:"consequences"`
	// IsResolved marks whether this event has been fully processed.
	IsResolved bool `json:"is_resolved"`
	// ChildEvents stores IDs of events spawned by this one.
	ChildEvents []string `json:"child_events"`
}

// World is the minimal interface the event system needs.
// Satisfied by internal/world without creating import cycles.
type World interface {
	GetAllPobles() []*entities.Poble
	GetWorldState() entities.WorldState
	GetPoble(id string) *entities.Poble
}

// RumourImpactProvider is optional. Rumour systems can implement it so social
// event scans can turn sensitive rumour arrivals into emotional events.
type RumourImpactProvider interface {
	GetRumourImpacts() []RumourImpact
}

// RumourImpact is a small adapter shape for rumour systems without coupling
// events directly to a specific rumour implementation.
type RumourImpact struct {
	ID             string
	RumourID       string
	ActorID        string
	TargetID       string
	SensitiveForID string
	Severity       float32
	IsPublic       bool
}

// TemplateRenderer generates narrator text for events.
// Satisfied by the template engine without importing it directly.
type TemplateRenderer interface {
	RenderEventDescription(event GameEvent) string
}

// TemplateContext carries renderer data for event descriptions.
type TemplateContext struct {
	Renderer   TemplateRenderer
	Speaker    *entities.Poble
	Target     *entities.Poble
	WorldState entities.WorldState
	ExtraVars  map[string]string
}

// lifeExpectancy returns approximate max age based on era.
func lifeExpectancy(era entities.Era) int {
	switch era {
	case entities.EraZero:
		return 55
	case entities.EraOne:
		return 62
	case entities.EraTwo:
		return 70
	case entities.EraThree:
		return 78
	case entities.EraFour:
		return 85
	default:
		return 60
	}
}

// CheckNaturalEvents scans for death by age, random illness, pregnancy,
// and weather events.
func CheckNaturalEvents(world World, rng *rand.Rand) []GameEvent {
	if world == nil || rng == nil {
		return nil
	}

	events := make([]GameEvent, 0, 4)
	ws := world.GetWorldState()

	for _, poble := range world.GetAllPobles() {
		if poble == nil || !poble.IsAlive {
			continue
		}
		events = append(events, checkDeathByAge(poble, ws, rng)...)
		events = append(events, checkIllness(poble, ws, rng)...)
		events = append(events, checkPregnancy(poble, world, rng)...)
	}

	events = append(events, checkWeatherEvents(ws, rng)...)
	return events
}

func checkDeathByAge(poble *entities.Poble, ws entities.WorldState, rng *rand.Rand) []GameEvent {
	maxAge := lifeExpectancy(ws.Era)
	if poble.Age <= maxAge {
		return nil
	}
	// Probability increases with years past life expectancy.
	yearsOver := poble.Age - maxAge
	chance := float32(yearsOver) * 0.08
	if rng.Float32() >= chance {
		return nil
	}
	return []GameEvent{{
		ID:           fmt.Sprintf("death_natural:%s:%d", poble.ID, ws.Day.Day),
		Type:         EventDeathNatural,
		Timestamp:    ws.Day,
		Participants: []string{poble.ID},
		IsPublic:     true,
		Consequences: []Consequence{
			{TargetID: poble.ID, Type: ConsequenceDeathCaused, Value: 1},
		},
	}}
}

func checkIllness(poble *entities.Poble, ws entities.WorldState, rng *rand.Rand) []GameEvent {
	baseChance := float32(0.002)
	if poble.Age > 50 {
		baseChance += float32(poble.Age-50) * 0.001
	}
	if poble.Health.HP < 40 {
		baseChance += 0.01
	}
	if rng.Float32() >= baseChance {
		return nil
	}
	return []GameEvent{{
		ID:           fmt.Sprintf("illness:%s:%d", poble.ID, ws.Day.Day),
		Type:         EventIllnessOnset,
		Timestamp:    ws.Day,
		Participants: []string{poble.ID},
		IsPublic:     false,
		Consequences: []Consequence{
			{TargetID: poble.ID, Type: ConsequenceHealthChange, Value: -15},
			{TargetID: poble.ID, Type: ConsequenceConditionAdded, Value: 1},
		},
	}}
}

func checkPregnancy(poble *entities.Poble, world World, rng *rand.Rand) []GameEvent {
	if poble.Sex != entities.Female && poble.Sex != entities.Intersex {
		return nil
	}
	if poble.Health.Fertility < 0.1 || poble.Age < 16 || poble.Age > 48 {
		return nil
	}
	if hasCondition(poble, entities.ConditionPregnant) {
		return nil
	}
	if poble.Needs.Sex < 60 {
		return nil
	}

	for targetID, rel := range poble.Relationships {
		if rel.Type != entities.RelationshipLover &&
			rel.Type != entities.RelationshipSpouse &&
			rel.Type != entities.RelationshipFriendsWithBenefits {
			continue
		}
		partner := world.GetPoble(targetID)
		if partner == nil || !partner.IsAlive {
			continue
		}
		if !canImpregnate(partner) {
			continue
		}
		chance := poble.Health.Fertility * 0.04
		if rng.Float32() < chance {
			ws := world.GetWorldState()
			return []GameEvent{{
				ID:           fmt.Sprintf("pregnancy:%s:%s:%d", poble.ID, targetID, ws.Day.Day),
				Type:         EventPregnancy,
				Timestamp:    ws.Day,
				Participants: []string{poble.ID, targetID},
				IsPublic:     false,
				Consequences: []Consequence{
					{TargetID: poble.ID, Type: ConsequencePregnancyCaused, Value: 1},
					{TargetID: poble.ID, Type: ConsequenceNeedChange, Value: -20},
				},
			}}
		}
	}
	return nil
}

func checkWeatherEvents(ws entities.WorldState, rng *rand.Rand) []GameEvent {
	// Storm: ~5% chance per day. Drought: ~1% after day 30.
	events := make([]GameEvent, 0, 1)
	if rng.Float32() < 0.05 {
		events = append(events, GameEvent{
			ID:        fmt.Sprintf("storm:%d", ws.Day.Day),
			Type:      EventStorm,
			Timestamp: ws.Day,
			IsPublic:  true,
			Consequences: []Consequence{
				{TargetID: "", Type: ConsequenceNeedChange, Value: 15},
			},
		})
	}
	if ws.Day.Day > 30 && rng.Float32() < 0.01 {
		events = append(events, GameEvent{
			ID:        fmt.Sprintf("drought:%d", ws.Day.Day),
			Type:      EventDrought,
			Timestamp: ws.Day,
			IsPublic:  true,
		})
	}
	return events
}

// CheckSocialEvents scans for fights, confessions, and rumour impacts.
func CheckSocialEvents(world World, rng *rand.Rand) []GameEvent {
	if world == nil || rng == nil {
		return nil
	}

	events := make([]GameEvent, 0, 4)
	ws := world.GetWorldState()

	for _, poble := range world.GetAllPobles() {
		if poble == nil || !poble.IsAlive {
			continue
		}
		events = append(events, checkResentmentFights(poble, ws, rng)...)
		events = append(events, checkAttractionConfession(poble, ws, rng)...)
		events = append(events, checkSecretConfession(poble, ws, rng)...)
	}
	events = append(events, checkRumourImpacts(world, ws)...)
	return events
}

func checkResentmentFights(poble *entities.Poble, ws entities.WorldState, rng *rand.Rand) []GameEvent {
	for targetID, rel := range poble.Relationships {
		if rel.Resentment <= 85 {
			continue
		}
		chance := (rel.Resentment - 85) * 0.02
		if rng.Float32() >= chance {
			continue
		}
		eventType := EventFightVerbal
		if rel.Resentment > 95 && rng.Float32() < 0.3 {
			eventType = EventFightPhysical
		}
		return []GameEvent{{
			ID:           fmt.Sprintf("fight:%s:%s:%d", poble.ID, targetID, ws.Day.Day),
			Type:         eventType,
			Timestamp:    ws.Day,
			Participants: []string{poble.ID, targetID},
			IsPublic:     true,
			Consequences: []Consequence{
				{TargetID: poble.ID, Type: ConsequenceMentalChange, Value: -10},
				{TargetID: targetID, Type: ConsequenceMentalChange, Value: -10},
				{TargetID: poble.ID, Type: ConsequenceRelationshipChange, Value: -15},
			},
		}}
	}
	return nil
}

func checkAttractionConfession(poble *entities.Poble, ws entities.WorldState, rng *rand.Rand) []GameEvent {
	for targetID, rel := range poble.Relationships {
		if rel.Attraction <= 90 {
			continue
		}
		chance := (rel.Attraction - 90) * 0.03
		if rng.Float32() >= chance {
			continue
		}
		return []GameEvent{{
			ID:           fmt.Sprintf("confession_love:%s:%s:%d", poble.ID, targetID, ws.Day.Day),
			Type:         EventRevelation,
			Timestamp:    ws.Day,
			Participants: []string{poble.ID, targetID},
			IsPublic:     false,
			Consequences: []Consequence{
				{TargetID: poble.ID, Type: ConsequenceMoodShift, Value: 20},
				{TargetID: targetID, Type: ConsequenceMemoryCreated, Value: 1},
			},
		}}
	}
	return nil
}

func checkSecretConfession(poble *entities.Poble, ws entities.WorldState, rng *rand.Rand) []GameEvent {
	if poble.Personality.Neuroticism <= 80 {
		return nil
	}
	for _, secret := range poble.Secrets {
		if secret.IsRevealed {
			continue
		}
		chance := float32(0.02) + (poble.Personality.Neuroticism-80)*0.003
		if rng.Float32() >= chance {
			continue
		}
		return []GameEvent{{
			ID:           fmt.Sprintf("secret_confess:%s:%s:%d", poble.ID, secret.ID, ws.Day.Day),
			Type:         EventRevelation,
			Timestamp:    ws.Day,
			Participants: []string{poble.ID},
			IsPublic:     false,
			Consequences: []Consequence{
				{TargetID: poble.ID, Type: ConsequenceSecretRevealed, Value: 1},
				{TargetID: poble.ID, Type: ConsequenceMentalChange, Value: 5},
			},
		}}
	}
	return nil
}

func checkRumourImpacts(world World, ws entities.WorldState) []GameEvent {
	provider, ok := world.(RumourImpactProvider)
	if !ok {
		return nil
	}

	impacts := provider.GetRumourImpacts()
	events := make([]GameEvent, 0, len(impacts))
	for _, impact := range impacts {
		targetID := impact.SensitiveForID
		if targetID == "" {
			targetID = impact.TargetID
		}
		if targetID == "" {
			continue
		}
		id := impact.ID
		if id == "" {
			id = fmt.Sprintf("rumour_impact:%s:%s:%d", impact.RumourID, targetID, ws.Day.Day)
		}
		participants := uniqueStrings([]string{impact.ActorID, targetID})
		events = append(events, GameEvent{
			ID:           id,
			Type:         EventRumourSpread,
			Timestamp:    ws.Day,
			Participants: participants,
			IsPublic:     impact.IsPublic,
			Consequences: []Consequence{
				{TargetID: targetID, Type: ConsequenceMentalChange, Value: -maxFloat32(10, impact.Severity*0.35)},
				{TargetID: targetID, Type: ConsequenceMemoryCreated, Value: 1},
			},
		})
	}
	return events
}

// CheckWorldEvents scans for population milestones, tech discoveries, era changes.
func CheckWorldEvents(world World, rng *rand.Rand) []GameEvent {
	if world == nil || rng == nil {
		return nil
	}

	events := make([]GameEvent, 0, 2)
	ws := world.GetWorldState()
	events = append(events, checkPopulationMilestone(ws)...)
	events = append(events, checkTechnologyDiscovery(world, ws, rng)...)
	events = append(events, checkEraChange(ws)...)
	return events
}

func checkPopulationMilestone(ws entities.WorldState) []GameEvent {
	milestones := []int{10, 50, 100, 200, 500, 1000}
	for _, m := range milestones {
		if ws.Population == m {
			return []GameEvent{{
				ID:        fmt.Sprintf("milestone:%d:%d", m, ws.Day.Day),
				Type:      EventPopulationMilestone,
				Timestamp: ws.Day,
				IsPublic:  true,
			}}
		}
	}
	return nil
}

func checkTechnologyDiscovery(world World, ws entities.WorldState, rng *rand.Rand) []GameEvent {
	if ws.TechTree.Unlocked["NAVIGATION"] {
		return nil
	}
	for _, poble := range world.GetAllPobles() {
		if poble == nil || !poble.IsAlive {
			continue
		}
		if poble.Personality.Openness < 82 || poble.Needs.Sleep > 70 || poble.Needs.Hunger > 70 {
			continue
		}
		chance := float32(0.01) + ((poble.Personality.Openness - 82) * 0.002)
		if rng.Float32() >= chance {
			continue
		}
		return []GameEvent{{
			ID:           fmt.Sprintf("tech:NAVIGATION:%s:%d", poble.ID, ws.Day.Day),
			Type:         EventTechDiscovered,
			Timestamp:    ws.Day,
			Participants: []string{poble.ID},
			IsPublic:     true,
		}}
	}
	return nil
}

func checkEraChange(ws entities.WorldState) []GameEvent {
	expectedEra := expectedEraForPopulation(ws.Population)
	if expectedEra == ws.Era {
		return nil
	}
	return []GameEvent{{
		ID:        fmt.Sprintf("era_change:%s:%d", expectedEra, ws.Day.Day),
		Type:      EventEraChange,
		Timestamp: ws.Day,
		IsPublic:  true,
	}}
}

func expectedEraForPopulation(pop int) entities.Era {
	switch {
	case pop >= 501:
		return entities.EraFour
	case pop >= 101:
		return entities.EraThree
	case pop >= 21:
		return entities.EraTwo
	case pop >= 5:
		return entities.EraOne
	default:
		return entities.EraZero
	}
}

// ApplyConsequences modifies the world based on an event's consequences.
// Consequences with Delay > 0 are returned as scheduled events.
func ApplyConsequences(event GameEvent, world World) []GameEvent {
	if world == nil {
		return nil
	}

	deferred := make([]GameEvent, 0)
	for _, c := range event.Consequences {
		if c.Delay > 0 {
			deferred = append(deferred, GameEvent{
				ID:           fmt.Sprintf("deferred:%s:%s:%d", event.ID, c.TargetID, c.Delay),
				Type:         EventType("DEFERRED_" + string(c.Type)),
				Timestamp:    event.Timestamp.Add(c.Delay),
				Participants: []string{c.TargetID},
				Consequences: []Consequence{{TargetID: c.TargetID, Type: c.Type, Value: c.Value, Delay: 0}},
			})
			continue
		}
		applyImmediateConsequence(c, world)
	}
	return deferred
}

func applyImmediateConsequence(c Consequence, world World) {
	if c.TargetID == "" {
		return
	}
	poble := world.GetPoble(c.TargetID)
	if poble == nil {
		return
	}

	switch c.Type {
	case ConsequenceHealthChange:
		poble.Health.HP = clampInt(poble.Health.HP+int(c.Value), 0, 100)
	case ConsequenceMentalChange:
		poble.Mental.Stability = clampInt(poble.Mental.Stability+int(c.Value), 0, 100)
	case ConsequenceNeedChange:
		// Distributes value across highest needs.
		applyNeedChange(poble, c.Value)
	case ConsequenceMoodShift:
		applyMoodShift(poble, c.Value)
	case ConsequenceDeathCaused:
		poble.IsAlive = false
		poble.Health.HP = 0
	case ConsequenceConditionAdded:
		if !hasCondition(poble, entities.ConditionSick) {
			poble.Health.Conditions = append(poble.Health.Conditions, entities.ConditionSick)
		}
	case ConsequenceConditionRemoved:
		poble.Health.Conditions = removeCondition(poble.Health.Conditions, entities.ConditionSick)
	case ConsequenceMoneyChange:
		poble.Money = maxInt(0, poble.Money+int(c.Value))
	case ConsequenceSecretRevealed:
		for i := range poble.Secrets {
			if !poble.Secrets[i].IsRevealed {
				poble.Secrets[i].IsRevealed = true
				break
			}
		}
	case ConsequenceMemoryCreated:
		// Memory creation is handled by the memory system; this is a signal only.
	case ConsequenceRelationshipChange:
		// Relationship changes are handled by the relationship system; signal only.
	case ConsequenceRumourCreated:
		// Rumour creation is handled by the rumour system; signal only.
	case ConsequencePregnancyCaused:
		if !hasCondition(poble, entities.ConditionPregnant) {
			poble.Health.Conditions = append(poble.Health.Conditions, entities.ConditionPregnant)
		}
	case ConsequenceItemGained, ConsequenceItemLost:
		// Item changes handled by inventory system; signal only.
	}
}

func applyNeedChange(poble *entities.Poble, value float32) {
	poble.Needs.Safety = clampFloat(poble.Needs.Safety+value, 0, 100)
}

func applyMoodShift(poble *entities.Poble, value float32) {
	newValence := poble.EmotionalState.Valence + (value / 100.0)
	if newValence < -1 {
		newValence = -1
	}
	if newValence > 1 {
		newValence = 1
	}
	poble.EmotionalState.Valence = newValence
}

// GenerateEventDescription uses narrator templates for external events only.
func GenerateEventDescription(event GameEvent, ctx TemplateContext) string {
	if event.Description != "" {
		return event.Description
	}
	if !usesNarratorTemplate(event.Type) {
		return ""
	}
	if ctx.Renderer != nil {
		desc := ctx.Renderer.RenderEventDescription(event)
		if desc != "" {
			return desc
		}
	}
	return ""
}

func usesNarratorTemplate(eventType EventType) bool {
	switch eventType {
	case EventDeathNatural, EventDeathAccident, EventDeathMurder, EventSuicide,
		EventStorm, EventDrought, EventEarthquake, EventPlague, EventFire, EventFlood,
		EventBirth, EventPopulationMilestone, EventCivilizationCollapse, EventLastPersonAlive,
		EventGenerationEnd, EventEraChange, EventTechDiscovered:
		return true
	default:
		return false
	}
}

// Helpers.

func hasCondition(poble *entities.Poble, condition entities.ConditionID) bool {
	for _, c := range poble.Health.Conditions {
		if c == condition {
			return true
		}
	}
	return false
}

func removeCondition(conditions []entities.ConditionID, target entities.ConditionID) []entities.ConditionID {
	result := conditions[:0]
	for _, c := range conditions {
		if c != target {
			result = append(result, c)
		}
	}
	return result
}

func canImpregnate(poble *entities.Poble) bool {
	return poble.Sex == entities.Male || poble.Sex == entities.Intersex
}

func clampInt(v, low, high int) int {
	if v < low {
		return low
	}
	if v > high {
		return high
	}
	return v
}

func clampFloat(v, low, high float32) float32 {
	if v < low {
		return low
	}
	if v > high {
		return high
	}
	return v
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func maxFloat32(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}

func uniqueStrings(values []string) []string {
	result := make([]string, 0, len(values))
	seen := map[string]struct{}{}
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
	return result
}
