package minigames

import (
	"fmt"
	"hash/fnv"
	"sort"
	"strings"

	"github.com/user/poblation/internal/ai"
	"github.com/user/poblation/internal/config"
	"github.com/user/poblation/internal/entities"
	"github.com/user/poblation/internal/events"
	"github.com/user/poblation/internal/world"
)

// EncounterType identifies the emotional shape of one encounter.
type EncounterType string

const (
	EncounterTender        EncounterType = "TENDER"
	EncounterPassionate    EncounterType = "PASSIONATE"
	EncounterDesperate     EncounterType = "DESPERATE"
	EncounterAngry         EncounterType = "ANGRY"
	EncounterCurious       EncounterType = "CURIOUS"
	EncounterSecret        EncounterType = "SECRET"
	EncounterTransactional EncounterType = "TRANSACTIONAL"
	EncounterComplicated   EncounterType = "COMPLICATED"
	EncounterLast          EncounterType = "LAST"
	EncounterFirstEver     EncounterType = "FIRST_EVER"
)

func (e EncounterType) String() string { return string(e) }

// EncounterContext carries the facts needed to classify and resolve one encounter.
type EncounterContext struct {
	A            *entities.Poble
	B            *entities.Poble
	Location     *world.Location
	World        *world.World
	StartedAt    entities.GameTime
	TriggerEvent string
	Relationship entities.Relationship
	Power        PowerDynamic
	IsFirstTime  bool
	Witnesses    []string
}

// PowerDynamic stores asymmetries shaping the scene.
type PowerDynamic struct {
	LeadID          string `json:"lead_id"`
	FollowerID      string `json:"follower_id"`
	TrustGap        int    `json:"trust_gap"`
	SocialGap       int    `json:"social_gap"`
	ExperienceGap   int    `json:"experience_gap"`
	SecrecyPressure int    `json:"secrecy_pressure"`
	Description     string `json:"description"`
}

// AftermathReaction stores how one Poble leaves the encounter.
type AftermathReaction struct {
	PobleID           string               `json:"poble_id"`
	DominantEmotion   entities.EmotionType `json:"dominant_emotion"`
	Mood              entities.MoodType    `json:"mood"`
	RelationshipDelta float32              `json:"relationship_delta"`
	TrustDelta        float32              `json:"trust_delta"`
	AttractionDelta   float32              `json:"attraction_delta"`
	ResentmentDelta   float32              `json:"resentment_delta"`
	Summary           string               `json:"summary"`
	Discovered        []string             `json:"discovered"`
	WantsDistance     bool                 `json:"wants_distance"`
	WantsMore         bool                 `json:"wants_more"`
}

// EncounterPreferenceMatch stores hidden-preference compatibility.
type EncounterPreferenceMatch struct {
	SharedNames        []string `json:"shared_names"`
	FrictionPoints     []string `json:"friction_points"`
	NewlyDiscovered    []string `json:"newly_discovered"`
	CompatibilityScore int      `json:"compatibility_score"`
	CuriosityBoost     int      `json:"curiosity_boost"`
}

// EncounterAftermath stores the combined outcome.
type EncounterAftermath struct {
	Type               EncounterType            `json:"type"`
	Power              PowerDynamic             `json:"power"`
	PreferenceMatch    EncounterPreferenceMatch `json:"preference_match"`
	Reactions          [2]AftermathReaction     `json:"reactions"`
	Witnesses          []string                 `json:"witnesses"`
	CreatedSecrets     []entities.Secret        `json:"created_secrets"`
	CreatedMemories    [2]entities.Memory       `json:"created_memories"`
	HealthFallout      EncounterHealthFallout   `json:"health_fallout"`
	RelationshipShift  int                      `json:"relationship_shift"`
	IsSecret           bool                     `json:"is_secret"`
	PregnancyTriggered bool                     `json:"pregnancy_triggered"`
	STITransmitted     entities.STIType         `json:"sti_transmitted"`
	VisibleSummary     string                   `json:"visible_summary"`
	InternalSummary    string                   `json:"internal_summary"`
}

// EncounterHealthFallout stores concrete physical consequences from the encounter.
type EncounterHealthFallout struct {
	UsedProtection bool                      `json:"used_protection"`
	PregnancyRisk  int                       `json:"pregnancy_risk"`
	STIRisk        int                       `json:"sti_risk"`
	HPDelta        [2]int                    `json:"hp_delta"`
	Conditions     [2][]entities.ConditionID `json:"conditions"`
	DeathCause     events.DeathCause         `json:"death_cause,omitempty"`
	Summary        string                    `json:"summary"`
}

// EncounterRecord stores one saved summary of the encounter.
type EncounterRecord struct {
	Type         EncounterType            `json:"type"`
	StartedAt    entities.GameTime        `json:"started_at"`
	LocationID   string                   `json:"location_id"`
	LocationName string                   `json:"location_name"`
	Participants [2]string                `json:"participants"`
	Power        PowerDynamic             `json:"power"`
	Preference   EncounterPreferenceMatch `json:"preference"`
	Aftermath    EncounterAftermath       `json:"aftermath"`
	IsRestricted bool                     `json:"is_restricted"`
}

// ArchetypeEncounterProfile shapes before/during/after tendencies per archetype.
type ArchetypeEncounterProfile struct {
	BeforeBehavior string `json:"before_behavior"`
	DuringBehavior string `json:"during_behavior"`
	AfterBehavior  string `json:"after_behavior"`
	RiskTolerance  int    `json:"risk_tolerance"`
	NeedForControl int    `json:"need_for_control"`
	SecrecyBias    int    `json:"secrecy_bias"`
	AttachmentBias int    `json:"attachment_bias"`
	TendernessBias int    `json:"tenderness_bias"`
	NoveltyBias    int    `json:"novelty_bias"`
}

// Back-compat types still used by the current Bubble Tea encounter view.
type EncounterPhase string

const (
	EncounterPhaseTension    EncounterPhase = "TENSION"
	EncounterPhaseInitiation EncounterPhase = "INITIATION"
	EncounterPhaseEncounter  EncounterPhase = "ENCOUNTER"
	EncounterPhaseAftermath  EncounterPhase = "AFTERMATH"
)

type EncounterMood string

const (
	EncounterMoodTender        EncounterMood = "tender"
	EncounterMoodPassionate    EncounterMood = "passionate"
	EncounterMoodDesperate     EncounterMood = "desperate"
	EncounterMoodAngry         EncounterMood = "angry"
	EncounterMoodAwkward       EncounterMood = "awkward"
	EncounterMoodCurious       EncounterMood = "curious"
	EncounterMoodSecret        EncounterMood = "secret"
	EncounterMoodTransactional EncounterMood = "transactional"
	EncounterMoodComplicated   EncounterMood = "complicated"
	EncounterMoodLast          EncounterMood = "last"
	EncounterMoodFirstEver     EncounterMood = "first_ever"
)

type ChoiceOption struct {
	Text              string
	RelationshipDelta int
	TrustDelta        int
	AttractionDelta   int
	UsesProtection    bool
	MoodShift         entities.EmotionType
}

type ChoiceConsequence struct {
	RelationshipDelta int
	TrustDelta        int
	AttractionDelta   int
	FutureHint        string
	MoodShift         entities.EmotionType
}

type EncounterChoice struct {
	Phase               EncounterPhase
	Text                string
	OptionsForPoble1    []ChoiceOption
	AutoChoiceForPoble1 ChoiceOption
	AutoChoiceForPoble2 ChoiceOption
	Consequence         ChoiceConsequence
	Delay               int
}

type EncounterState struct {
	Participants            [2]*entities.Poble
	Phase                   EncounterPhase
	Choices                 []EncounterChoice
	Mood                    EncounterMood
	Type                    EncounterType
	WillLeadToPregnancy     bool
	STITransmissionOccurred bool
	RelationshipDelta       map[string]int
	WasConsensual           bool
	StartedAt               entities.GameTime
	UsedProtection          bool
	FutureHints             []string
	CurrentStep             int
	ResultApplied           bool
	TransmittedSTI          entities.STIType
	Context                 EncounterContext
	PreferenceMatch         EncounterPreferenceMatch
	Aftermath               EncounterAftermath
	Record                  EncounterRecord
}

// NewEncounterState builds a compatibility encounter state for the UI flow.
func NewEncounterState(participants [2]*entities.Poble, startedAt entities.GameTime) EncounterState {
	level := config.GetContentLevel()
	ctx := newEncounterContext(nil, participants[0], participants[1], startedAt, "")
	encounterType := DetermineEncounterType(ctx)
	usedProtection := shouldUseProtection(ctx, encounterType)
	stiType, transmitted := deterministicSTIResult(ctx, usedProtection)

	return EncounterState{
		Participants:            participants,
		Phase:                   EncounterPhaseTension,
		Choices:                 []EncounterChoice{},
		Mood:                    encounterMoodFromType(encounterType),
		Type:                    encounterType,
		WillLeadToPregnancy:     shouldTriggerPregnancy(ctx, usedProtection),
		STITransmissionOccurred: transmitted,
		RelationshipDelta:       map[string]int{},
		WasConsensual:           true,
		StartedAt:               startedAt,
		UsedProtection:          usedProtection,
		FutureHints:             futureHintsFor(ctx, encounterType, level),
		CurrentStep:             0,
		TransmittedSTI:          stiType,
		Context:                 ctx,
	}
}

// ResolveChoice keeps the older encounter UI working while feeding the new result model.
func ResolveChoice(state *EncounterState, choice EncounterChoice, playerChoice *ChoiceOption) (ChoiceOption, ChoiceOption) {
	if state == nil {
		return ChoiceOption{}, ChoiceOption{}
	}
	first := choice.AutoChoiceForPoble1
	if playerChoice != nil {
		first = *playerChoice
	}
	second := choice.AutoChoiceForPoble2

	state.Phase = choice.Phase
	state.CurrentStep++
	state.Choices = append(state.Choices, choice)
	state.RelationshipDelta[relationshipKey(state.Participants[0], state.Participants[1])] += choice.Consequence.RelationshipDelta + first.RelationshipDelta + second.RelationshipDelta
	state.RelationshipDelta[trustKey(state.Participants[0], state.Participants[1])] += choice.Consequence.TrustDelta + first.TrustDelta + second.TrustDelta
	state.RelationshipDelta[attractionKey(state.Participants[0], state.Participants[1])] += choice.Consequence.AttractionDelta + first.AttractionDelta + second.AttractionDelta
	if choice.Consequence.FutureHint != "" {
		state.FutureHints = append(state.FutureHints, choice.Consequence.FutureHint)
	}
	if first.UsesProtection || second.UsesProtection {
		state.UsedProtection = true
		state.WillLeadToPregnancy = false
		state.STITransmissionOccurred = false
		state.TransmittedSTI = entities.STINone
	}
	return first, second
}

// DetermineEncounterType classifies the emotional shape from context, not randomness.
func DetermineEncounterType(ctx EncounterContext) EncounterType {
	level := config.GetContentLevel()
	_ = level

	scores := map[EncounterType]int{
		EncounterTender:        0,
		EncounterPassionate:    0,
		EncounterDesperate:     0,
		EncounterAngry:         0,
		EncounterCurious:       0,
		EncounterSecret:        0,
		EncounterTransactional: 0,
		EncounterComplicated:   0,
		EncounterLast:          0,
		EncounterFirstEver:     0,
	}

	addRelationshipScores(scores, ctx)
	addEmotionScores(scores, ctx)
	addLocationScores(scores, ctx)
	addHistoryScores(scores, ctx)
	addTriggerScores(scores, ctx)
	addPowerScores(scores, ctx.Power)

	if ctx.IsFirstTime {
		scores[EncounterFirstEver] += 32
	}
	if witnessCount(ctx) == 0 && ctx.Location != nil && ctx.Location.PrivacyLevel >= 75 {
		scores[EncounterSecret] += 10
	}
	if ctx.Location != nil && ctx.Location.Type == world.LocationCemetery {
		scores[EncounterLast] += 8
	}

	return highestEncounterScore(scores)
}

// DetermineArchetypeRole returns an archetype-shaped encounter profile.
func DetermineArchetypeRole(poble *entities.Poble, ctx EncounterContext) ArchetypeEncounterProfile {
	level := config.GetContentLevel()
	_ = level
	if poble == nil {
		return ArchetypeEncounterProfile{
			BeforeBehavior: "hesitates",
			DuringBehavior: "follows the room",
			AfterBehavior:  "stays unreadable",
			RiskTolerance:  45,
			NeedForControl: 40,
			SecrecyBias:    40,
			AttachmentBias: 45,
			TendernessBias: 45,
			NoveltyBias:    45,
		}
	}

	switch poble.Archetype {
	case entities.ArchetypeRuler:
		return ArchetypeEncounterProfile{"frames it like a decision", "likes command and structure", "measures the shift in leverage", 68, 84, 58, 40, 35, 28}
	case entities.ArchetypeLover:
		return ArchetypeEncounterProfile{"leans in through longing", "turns touch into confession", "replays every signal as meaning", 60, 34, 40, 88, 82, 52}
	case entities.ArchetypeJester:
		return ArchetypeEncounterProfile{"hides nerves behind a grin", "cuts tension with jokes", "acts lighter than it felt", 57, 22, 48, 38, 46, 70}
	case entities.ArchetypeSage:
		return ArchetypeEncounterProfile{"observes before committing", "stays attentive and exact", "dissects what changed", 42, 44, 52, 50, 58, 38}
	case entities.ArchetypeRebel:
		return ArchetypeEncounterProfile{"wants whatever feels forbidden", "pushes against boundaries to test them", "bristles if it starts feeling owned", 76, 49, 45, 28, 24, 75}
	case entities.ArchetypeCaretaker:
		return ArchetypeEncounterProfile{"checks the other's face first", "prioritizes comfort and steadiness", "worries about fallout more than pleasure", 38, 20, 55, 78, 86, 22}
	case entities.ArchetypeVillain:
		return ArchetypeEncounterProfile{"smells power before softness", "likes edges, leverage and reaction", "counts what the other will hide now", 85, 88, 70, 16, 10, 58}
	case entities.ArchetypeGhost:
		return ArchetypeEncounterProfile{"arrives half elsewhere", "keeps one part absent", "retreats before the room cools", 36, 18, 82, 22, 18, 44}
	case entities.ArchetypeAddict:
		return ArchetypeEncounterProfile{"chases relief harder than wisdom", "mistakes intensity for necessity", "crashes into need again after it ends", 80, 27, 42, 48, 24, 66}
	case entities.ArchetypeProphet:
		return ArchetypeEncounterProfile{"reads omen into attraction", "treats the moment like destiny talking", "frames aftermath as meaning, not accident", 52, 41, 64, 59, 44, 31}
	case entities.ArchetypeSchemer:
		return ArchetypeEncounterProfile{"tracks openings and future cost", "keeps the upper hand if possible", "files the scene as usable information", 71, 79, 88, 32, 20, 54}
	case entities.ArchetypeInnocent:
		return ArchetypeEncounterProfile{"arrives earnest and open", "treats closeness like trust", "takes the aftermath personally", 33, 14, 26, 74, 79, 35}
	case entities.ArchetypeWarrior:
		return ArchetypeEncounterProfile{"goes direct once decided", "brings urgency and physical certainty", "turns shame into bluntness if hurt", 77, 64, 29, 36, 30, 42}
	case entities.ArchetypeDrifter:
		return ArchetypeEncounterProfile{"lets the night steer a little", "stays present without promising much", "slips away if permanence appears", 63, 16, 52, 24, 20, 81}
	case entities.ArchetypeMirror:
		return ArchetypeEncounterProfile{"matches the energy offered", "reflects the other's pace and need", "absorbs the other person's reading of it", 49, 37, 61, 56, 53, 57}
	default:
		return ArchetypeEncounterProfile{"moves by whatever the moment allows", "follows instinct more than script", "makes sense of it afterward", 50, 40, 40, 45, 45, 45}
	}
}

// ResolvePreferenceMatch compares structured hidden preferences.
func ResolvePreferenceMatch(a, b *entities.Poble) EncounterPreferenceMatch {
	level := config.GetContentLevel()
	match := EncounterPreferenceMatch{
		SharedNames:        []string{},
		FrictionPoints:     []string{},
		NewlyDiscovered:    []string{},
		CompatibilityScore: 45,
		CuriosityBoost:     0,
	}
	if a == nil || b == nil {
		return match
	}
	if level == config.ContentRestricted {
		match.CompatibilityScore = 50
		return match
	}

	for _, first := range a.HiddenPrefs.Preferences {
		for _, second := range b.HiddenPrefs.Preferences {
			if samePreference(first, second) {
				label := preferenceLabel(first)
				match.SharedNames = appendUniqueString(match.SharedNames, label)
				match.CompatibilityScore += 8
				if !first.IsDiscovered || !second.IsDiscovered {
					match.NewlyDiscovered = appendUniqueString(match.NewlyDiscovered, label)
					match.CuriosityBoost += 6
				}
				continue
			}
			if frictionBetween(first, second) {
				match.FrictionPoints = appendUniqueString(match.FrictionPoints, frictionLabel(first, second))
				match.CompatibilityScore -= 6
			}
		}
	}

	openGap := absInt(int(a.HiddenPrefs.Openness) - int(b.HiddenPrefs.Openness))
	if openGap > 35 {
		match.FrictionPoints = appendUniqueString(match.FrictionPoints, "ritmo distinto para explorar")
		match.CompatibilityScore -= 10
	}
	if a.HiddenPrefs.DominantCategory == b.HiddenPrefs.DominantCategory {
		match.CompatibilityScore += 10
	}
	match.CompatibilityScore = clampInt(match.CompatibilityScore, 0, 100)
	return match
}

// GenerateAftermath produces archetype-specific fallout from the same encounter.
func GenerateAftermath(ctx EncounterContext, prefMatch EncounterPreferenceMatch) EncounterAftermath {
	level := config.GetContentLevel()
	encounterType := DetermineEncounterType(ctx)
	profileA := DetermineArchetypeRole(ctx.A, ctx)
	profileB := DetermineArchetypeRole(ctx.B, ctx)

	reactionA := buildReaction(ctx.A, encounterType, profileA, prefMatch, ctx.Relationship, true)
	reactionB := buildReaction(ctx.B, encounterType, profileB, prefMatch, reverseRelationship(ctx), false)

	usedProtection := shouldUseProtection(ctx, encounterType)
	aftermath := EncounterAftermath{
		Type:               encounterType,
		Power:              ctx.Power,
		PreferenceMatch:    prefMatch,
		Reactions:          [2]AftermathReaction{reactionA, reactionB},
		Witnesses:          append([]string{}, ctx.Witnesses...),
		CreatedSecrets:     buildEncounterSecrets(ctx, encounterType),
		HealthFallout:      buildEncounterHealthFallout(ctx, encounterType, usedProtection),
		RelationshipShift:  int(reactionA.RelationshipDelta + reactionB.RelationshipDelta),
		IsSecret:           encounterType == EncounterSecret || encounterType == EncounterTransactional || encounterType == EncounterLast || placeFeelsSecret(ctx.Location),
		PregnancyTriggered: shouldTriggerPregnancy(ctx, usedProtection),
		STITransmitted:     pickSTITransmission(ctx, usedProtection),
		VisibleSummary:     restrictedSummary(ctx),
		InternalSummary:    internalSummary(ctx, encounterType),
	}
	if level == config.ContentFull {
		aftermath.VisibleSummary = aftermath.InternalSummary
	}
	aftermath.CreatedMemories = buildEncounterMemories(ctx, aftermath)
	return aftermath
}

// ApplyEncounterResult writes an encounter state into the world for the current UI flow.
func ApplyEncounterResult(state EncounterState, gameWorld *world.World) {
	if gameWorld == nil {
		return
	}
	ctx := state.Context
	if ctx.A == nil {
		ctx.A = state.Participants[0]
	}
	if ctx.B == nil {
		ctx.B = state.Participants[1]
	}
	if ctx.StartedAt.ToMinutes() == 0 {
		ctx.StartedAt = state.StartedAt
	}
	if ctx.World == nil {
		ctx.World = gameWorld
	}
	if ctx.Location == nil && ctx.A != nil {
		if loc, ok := gameWorld.GetLocation(ctx.A.ID); ok {
			ctx.Location = &loc
		}
	}
	if len(ctx.Witnesses) == 0 && ctx.Location != nil {
		ctx.Witnesses = witnessesAtLocation(gameWorld, *ctx.Location, idOf(ctx.A), idOf(ctx.B))
	}
	if ctx.Relationship.TargetID == "" && ctx.A != nil && ctx.B != nil {
		ctx.Relationship = relationBetween(ctx.A, ctx.B)
	}
	if ctx.Power.LeadID == "" {
		ctx.Power = derivePowerDynamic(ctx.A, ctx.B, ctx.Relationship)
	}
	prefMatch := state.PreferenceMatch
	if prefMatch.CompatibilityScore == 0 && ctx.A != nil && ctx.B != nil {
		prefMatch = ResolvePreferenceMatch(ctx.A, ctx.B)
	}
	aftermath := state.Aftermath
	if aftermath.Type == "" {
		aftermath = GenerateAftermath(ctx, prefMatch)
	}
	ApplyEncounterAftermath(aftermath, gameWorld)
}

// ApplyEncounterAftermath writes aftermath changes into the world.
func ApplyEncounterAftermath(aftermath EncounterAftermath, gameWorld *world.World) {
	level := config.GetContentLevel()
	_ = level
	if gameWorld == nil {
		return
	}

	a := gameWorld.GetPoble(aftermath.Reactions[0].PobleID)
	b := gameWorld.GetPoble(aftermath.Reactions[1].PobleID)
	if a == nil || b == nil {
		return
	}

	applyReactionToRelationship(a, b, aftermath.Reactions[0])
	applyReactionToRelationship(b, a, aftermath.Reactions[1])
	applyReactionToBodyAndMood(a, aftermath.Reactions[0])
	applyReactionToBodyAndMood(b, aftermath.Reactions[1])

	a.Memories = append(a.Memories, aftermath.CreatedMemories[0])
	b.Memories = append(b.Memories, aftermath.CreatedMemories[1])

	for _, secret := range aftermath.CreatedSecrets {
		a.Secrets = append(a.Secrets, secret)
		b.Secrets = append(b.Secrets, secret)
	}

	if aftermath.PregnancyTriggered {
		if carrier := pregnancyCarrier(a, b); carrier != nil {
			_ = events.HandlePregnancy(carrier.ID, gameWorld)
			appendEncounterWorldEvent(gameWorld, ai.GameEventIntimacy, aftermath, "pregnancy", []string{carrier.ID, idOf(a), idOf(b)}, fmt.Sprintf("%s left the encounter carrying a possible pregnancy arc.", nameOf(carrier)), 0.74, -0.05)
		}
	}
	if aftermath.STITransmitted != entities.STINone {
		applySTITransmission(a, b, aftermath.STITransmitted)
		source, _, target := stiCarrier(a, b)
		if source != nil && target != nil {
			appendEncounterWorldEvent(gameWorld, ai.GameEventSocialNegative, aftermath, "sti", []string{source.ID, target.ID}, fmt.Sprintf("%s may have transmitted %s to %s.", nameOf(source), aftermath.STITransmitted, nameOf(target)), 0.72, -0.42)
		}
	}
	applyEncounterHealthFallout(a, b, aftermath, gameWorld)
	appendEncounterWorldEvent(gameWorld, ai.GameEventIntimacy, aftermath, "encounter", []string{idOf(a), idOf(b)}, aftermath.VisibleSummary, 0.44, 0.08)
	registerWitnesses(gameWorld, a, b, aftermath)
}

func newEncounterContext(gameWorld *world.World, a, b *entities.Poble, startedAt entities.GameTime, trigger string) EncounterContext {
	ctx := EncounterContext{
		A:            a,
		B:            b,
		World:        gameWorld,
		StartedAt:    startedAt,
		TriggerEvent: trigger,
		IsFirstTime:  isFirstEroticMemory(a, b),
	}
	ctx.Relationship = relationBetween(a, b)
	ctx.Power = derivePowerDynamic(a, b, ctx.Relationship)
	if gameWorld != nil && a != nil {
		if loc, ok := gameWorld.GetLocation(a.ID); ok {
			ctx.Location = &loc
			ctx.Witnesses = witnessesAtLocation(gameWorld, loc, a.ID, idOf(b))
		}
	}
	return ctx
}

func derivePowerDynamic(a, b *entities.Poble, relationship entities.Relationship) PowerDynamic {
	power := PowerDynamic{
		LeadID:          idOf(a),
		FollowerID:      idOf(b),
		TrustGap:        int(relationship.Trust - relationship.Resentment),
		SocialGap:       absInt(a.Money - b.Money),
		ExperienceGap:   absInt(len(a.Memories) - len(b.Memories)),
		SecrecyPressure: secrecyPressure(a, b, relationship),
		Description:     "balanced enough to stay private",
	}
	if a == nil || b == nil {
		return power
	}
	if b.EmotionalState.Dominance > a.EmotionalState.Dominance {
		power.LeadID, power.FollowerID = b.ID, a.ID
	}
	switch {
	case power.SecrecyPressure >= 60:
		power.Description = "both know this gets heavier if anyone sees it"
	case power.SocialGap >= 30:
		power.Description = "one of them can afford consequences better than the other"
	case power.TrustGap <= -20:
		power.Description = "the room is carrying more leverage than comfort"
	default:
		power.Description = "they are not equal, but neither is fully steering"
	}
	return power
}

func addRelationshipScores(scores map[EncounterType]int, ctx EncounterContext) {
	relationship := ctx.Relationship
	switch relationship.Type {
	case entities.RelationshipLover, entities.RelationshipSpouse:
		scores[EncounterTender] += 18
		scores[EncounterPassionate] += 14
	case entities.RelationshipFriendsWithBenefits:
		scores[EncounterTransactional] += 12
		scores[EncounterSecret] += 8
	case entities.RelationshipEnemy, entities.RelationshipNemesis:
		scores[EncounterAngry] += 24
		scores[EncounterComplicated] += 12
	case entities.RelationshipCrush:
		scores[EncounterCurious] += 16
		scores[EncounterFirstEver] += 10
	case entities.RelationshipSecretObsession:
		scores[EncounterSecret] += 20
		scores[EncounterDesperate] += 14
	case entities.RelationshipComplicated, entities.RelationshipToxicAttraction:
		scores[EncounterComplicated] += 18
		scores[EncounterSecret] += 7
		scores[EncounterAngry] += 6
	}
	if relationship.Attraction >= 75 {
		scores[EncounterPassionate] += 16
	}
	if relationship.Trust >= 72 {
		scores[EncounterTender] += 14
	}
	if relationship.Resentment >= 65 {
		scores[EncounterAngry] += 18
		scores[EncounterComplicated] += 10
	}
	if relationship.Dependency >= 70 {
		scores[EncounterDesperate] += 12
	}
	if relationship.IsSecret {
		scores[EncounterSecret] += 14
	}
}

func addEmotionScores(scores map[EncounterType]int, ctx EncounterContext) {
	for _, poble := range []*entities.Poble{ctx.A, ctx.B} {
		if poble == nil {
			continue
		}
		addMoodScore(scores, poble.CurrentMood)
		for _, emotion := range poble.EmotionalState.ActiveEmotions {
			switch emotion {
			case entities.EmotionLust:
				scores[EncounterPassionate] += 8
			case entities.EmotionHope:
				scores[EncounterTender] += 6
			case entities.EmotionCuriosity:
				scores[EncounterCurious] += 10
			case entities.EmotionAnger, entities.EmotionResentment:
				scores[EncounterAngry] += 12
			case entities.EmotionGrief, entities.EmotionLoneliness:
				scores[EncounterDesperate] += 10
				scores[EncounterLast] += 6
			case entities.EmotionShame, entities.EmotionFear:
				scores[EncounterSecret] += 8
			}
		}
	}
}

func addMoodScore(scores map[EncounterType]int, mood entities.MoodType) {
	switch mood {
	case entities.MoodContent, entities.MoodHappy:
		scores[EncounterTender] += 6
	case entities.MoodEuphoric:
		scores[EncounterPassionate] += 8
	case entities.MoodAnxious:
		scores[EncounterSecret] += 5
		scores[EncounterCurious] += 4
	case entities.MoodAngry:
		scores[EncounterAngry] += 14
	case entities.MoodSad, entities.MoodDepressed:
		scores[EncounterDesperate] += 10
		scores[EncounterLast] += 8
	case entities.MoodObsessive:
		scores[EncounterComplicated] += 10
		scores[EncounterSecret] += 6
	}
}

func addLocationScores(scores map[EncounterType]int, ctx EncounterContext) {
	if ctx.Location == nil {
		return
	}
	switch ctx.Location.Type {
	case world.LocationTavern, world.LocationDiveBar:
		scores[EncounterPassionate] += 8
		scores[EncounterDesperate] += 6
	case world.LocationCemetery:
		scores[EncounterLast] += 14
		scores[EncounterTender] += 4
	case world.LocationTemple:
		scores[EncounterSecret] += 10
		scores[EncounterComplicated] += 4
	case world.LocationHotel:
		scores[EncounterSecret] += 12
		scores[EncounterTransactional] += 10
	case world.LocationMentalHealthClinic, world.LocationClinic, world.LocationHospital:
		scores[EncounterTender] += 8
		scores[EncounterLast] += 6
	case world.LocationPort:
		scores[EncounterLast] += 8
		scores[EncounterSecret] += 6
	case world.LocationMarket, world.LocationNewspaperOffice:
		scores[EncounterTransactional] += 8
		scores[EncounterSecret] += 5
	}
	if ctx.Location.PrivacyLevel >= 75 {
		scores[EncounterSecret] += 14
	}
	if len(ctx.Witnesses) > 0 {
		scores[EncounterSecret] += 8
		scores[EncounterComplicated] += 6
	}
}

func addHistoryScores(scores map[EncounterType]int, ctx EncounterContext) {
	if ctx.A == nil || ctx.B == nil {
		return
	}
	if ctx.IsFirstTime {
		scores[EncounterFirstEver] += 20
	}
	if hasEroticMemoryWith(ctx.A, ctx.B.ID) || hasEroticMemoryWith(ctx.B, ctx.A.ID) {
		scores[EncounterPassionate] += 6
	}
	if hasMemoryTagBetween(ctx.A, ctx.B.ID, "betrayal") || hasMemoryTagBetween(ctx.B, ctx.A.ID, "betrayal") {
		scores[EncounterComplicated] += 12
		scores[EncounterAngry] += 8
	}
	if hasMemoryTagBetween(ctx.A, ctx.B.ID, "grief") || hasMemoryTagBetween(ctx.B, ctx.A.ID, "grief") {
		scores[EncounterLast] += 8
	}
}

func addTriggerScores(scores map[EncounterType]int, ctx EncounterContext) {
	trigger := strings.ToLower(strings.TrimSpace(ctx.TriggerEvent))
	switch {
	case strings.Contains(trigger, "fight"), strings.Contains(trigger, "betrayal"):
		scores[EncounterAngry] += 16
		scores[EncounterComplicated] += 10
	case strings.Contains(trigger, "confession"), strings.Contains(trigger, "secret"):
		scores[EncounterSecret] += 14
		scores[EncounterTender] += 6
	case strings.Contains(trigger, "funeral"), strings.Contains(trigger, "goodbye"):
		scores[EncounterLast] += 18
	case strings.Contains(trigger, "lonely"), strings.Contains(trigger, "panic"):
		scores[EncounterDesperate] += 14
	case strings.Contains(trigger, "barter"), strings.Contains(trigger, "money"):
		scores[EncounterTransactional] += 18
	}
}

func addPowerScores(scores map[EncounterType]int, power PowerDynamic) {
	if power.SecrecyPressure >= 60 {
		scores[EncounterSecret] += 12
	}
	if power.SocialGap >= 30 {
		scores[EncounterTransactional] += 14
	}
	if power.TrustGap <= -20 {
		scores[EncounterComplicated] += 12
		scores[EncounterAngry] += 6
	}
	if power.ExperienceGap >= 20 {
		scores[EncounterCurious] += 6
	}
}

func highestEncounterScore(scores map[EncounterType]int) EncounterType {
	order := []EncounterType{
		EncounterFirstEver,
		EncounterLast,
		EncounterSecret,
		EncounterAngry,
		EncounterTransactional,
		EncounterComplicated,
		EncounterDesperate,
		EncounterPassionate,
		EncounterTender,
		EncounterCurious,
	}
	best := EncounterTender
	bestScore := -1
	for _, candidate := range order {
		if scores[candidate] > bestScore {
			best = candidate
			bestScore = scores[candidate]
		}
	}
	return best
}

func buildReaction(poble *entities.Poble, encounterType EncounterType, profile ArchetypeEncounterProfile, prefMatch EncounterPreferenceMatch, relationship entities.Relationship, primary bool) AftermathReaction {
	if poble == nil {
		return AftermathReaction{}
	}
	base := reactionBase(encounterType)
	matchLift := float32(prefMatch.CompatibilityScore-50) / 10
	controlDrag := float32(profile.NeedForControl-50) / 8
	secrecyDrag := float32(profile.SecrecyBias-50) / 10

	reaction := AftermathReaction{
		PobleID:           poble.ID,
		DominantEmotion:   dominantEmotionFor(encounterType, profile, primary),
		Mood:              aftermathMoodFor(encounterType, profile),
		RelationshipDelta: clampReaction(base.relationship + matchLift + float32(profile.AttachmentBias-50)/8),
		TrustDelta:        clampReaction(base.trust + matchLift - secrecyDrag),
		AttractionDelta:   clampReaction(base.attraction + float32(profile.NoveltyBias-50)/10),
		ResentmentDelta:   clampReaction(base.resentment + controlDrag - matchLift),
		Summary:           reactionSummary(poble, encounterType, profile, prefMatch),
		Discovered:        append([]string{}, prefMatch.NewlyDiscovered...),
		WantsDistance:     encounterType == EncounterAngry || encounterType == EncounterTransactional || (encounterType == EncounterComplicated && profile.AttachmentBias < 45),
		WantsMore:         encounterType == EncounterTender || encounterType == EncounterPassionate || (encounterType == EncounterCurious && profile.NoveltyBias >= 55),
	}

	if relationship.Type == entities.RelationshipEnemy || relationship.Type == entities.RelationshipNemesis {
		reaction.ResentmentDelta += 8
		reaction.TrustDelta -= 4
	}
	if profile.AfterBehavior == "retreats before the room cools" {
		reaction.WantsDistance = true
	}
	return reaction
}

type reactionWeights struct {
	relationship float32
	trust        float32
	attraction   float32
	resentment   float32
}

func reactionBase(encounterType EncounterType) reactionWeights {
	switch encounterType {
	case EncounterTender:
		return reactionWeights{8, 10, 6, -3}
	case EncounterPassionate:
		return reactionWeights{5, 3, 10, -1}
	case EncounterDesperate:
		return reactionWeights{2, -1, 7, 3}
	case EncounterAngry:
		return reactionWeights{-3, -6, 4, 10}
	case EncounterCurious:
		return reactionWeights{3, 2, 5, 0}
	case EncounterSecret:
		return reactionWeights{4, -1, 6, 2}
	case EncounterTransactional:
		return reactionWeights{-1, -4, 3, 5}
	case EncounterComplicated:
		return reactionWeights{1, -3, 7, 6}
	case EncounterLast:
		return reactionWeights{5, 1, 4, 4}
	case EncounterFirstEver:
		return reactionWeights{6, 6, 6, 1}
	default:
		return reactionWeights{}
	}
}

func dominantEmotionFor(encounterType EncounterType, profile ArchetypeEncounterProfile, primary bool) entities.EmotionType {
	switch encounterType {
	case EncounterTender:
		return entities.EmotionRelief
	case EncounterPassionate:
		return entities.EmotionLust
	case EncounterDesperate:
		if profile.AttachmentBias >= 60 {
			return entities.EmotionHope
		}
		return entities.EmotionLoneliness
	case EncounterAngry:
		if primary {
			return entities.EmotionAnger
		}
		return entities.EmotionResentment
	case EncounterCurious:
		return entities.EmotionCuriosity
	case EncounterSecret:
		return entities.EmotionShame
	case EncounterTransactional:
		return entities.EmotionContempt
	case EncounterComplicated:
		return entities.EmotionConfusion
	case EncounterLast:
		return entities.EmotionGrief
	case EncounterFirstEver:
		return entities.EmotionSurprise
	default:
		return entities.EmotionHope
	}
}

func aftermathMoodFor(encounterType EncounterType, profile ArchetypeEncounterProfile) entities.MoodType {
	switch encounterType {
	case EncounterTender:
		return entities.MoodContent
	case EncounterPassionate:
		return entities.MoodEuphoric
	case EncounterDesperate:
		if profile.AttachmentBias >= 60 {
			return entities.MoodObsessive
		}
		return entities.MoodSad
	case EncounterAngry:
		return entities.MoodAngry
	case EncounterCurious, EncounterFirstEver:
		return entities.MoodAnxious
	case EncounterSecret, EncounterComplicated:
		return entities.MoodObsessive
	case EncounterTransactional:
		return entities.MoodNumb
	case EncounterLast:
		return entities.MoodSad
	default:
		return entities.MoodNeutral
	}
}

func reactionSummary(poble *entities.Poble, encounterType EncounterType, profile ArchetypeEncounterProfile, prefMatch EncounterPreferenceMatch) string {
	if poble == nil {
		return ""
	}
	switch encounterType {
	case EncounterTender:
		return fmt.Sprintf("%s sale con el cuerpo menos armado y la cabeza queriendo creer en eso.", poble.Name)
	case EncounterPassionate:
		return fmt.Sprintf("%s se queda acelerado, como si la escena todavia siguiera decidiendo cosas.", poble.Name)
	case EncounterDesperate:
		return fmt.Sprintf("%s siente alivio, pero del tipo que no dura mucho.", poble.Name)
	case EncounterAngry:
		return fmt.Sprintf("%s no sabe si aquello fue cierre o gasolina.", poble.Name)
	case EncounterCurious:
		return fmt.Sprintf("%s se va pensando menos en culpa y mas en preguntas nuevas.", poble.Name)
	case EncounterSecret:
		return fmt.Sprintf("%s ya esta calculando cuanto de esto puede quedarse debajo del piso.", poble.Name)
	case EncounterTransactional:
		return fmt.Sprintf("%s trata de llamarlo intercambio para no llamarlo hambre.", poble.Name)
	case EncounterComplicated:
		return fmt.Sprintf("%s confirma que querer algo no lo vuelve simple.", poble.Name)
	case EncounterLast:
		return fmt.Sprintf("%s siente que el momento peso mas por lo que termina que por lo que dio.", poble.Name)
	case EncounterFirstEver:
		if len(prefMatch.NewlyDiscovered) > 0 {
			return fmt.Sprintf("%s sale cambiado y un poco asustado de haber aprendido algo propio.", poble.Name)
		}
		return fmt.Sprintf("%s sale tocando el borde de una version nueva de si mismo.", poble.Name)
	default:
		return fmt.Sprintf("%s no logra dejarlo en una sola palabra.", poble.Name)
	}
}

func buildEncounterSecrets(ctx EncounterContext, encounterType EncounterType) []entities.Secret {
	if !placeFeelsSecret(ctx.Location) && encounterType != EncounterSecret && len(ctx.Witnesses) == 0 {
		return nil
	}
	content := "guardan un encuentro que seria mas facil mentir que explicar"
	if encounterType == EncounterLast {
		content = "guardan un ultimo encuentro que parece despedida aunque nadie lo haya dicho"
	}
	secret := entities.NewSecret(
		fmt.Sprintf("encounter_secret_%s_%s_%d", idOf(ctx.A), idOf(ctx.B), ctx.StartedAt.ToMinutes()),
		entities.SecretPastRelationship,
		content,
	)
	secret.KnownBy = append(secret.KnownBy, idOf(ctx.A), idOf(ctx.B))
	return []entities.Secret{secret}
}

func buildEncounterMemories(ctx EncounterContext, aftermath EncounterAftermath) [2]entities.Memory {
	level := config.GetContentLevel()
	summary := aftermath.InternalSummary
	if level == config.ContentRestricted {
		summary = restrictedSummary(ctx)
	}
	first := privateEroticMemory(ctx.A, ctx.B, ctx.StartedAt, aftermath.Type, summary)
	second := privateEroticMemory(ctx.B, ctx.A, ctx.StartedAt, aftermath.Type, summary)
	return [2]entities.Memory{first, second}
}

func privateEroticMemory(owner, other *entities.Poble, at entities.GameTime, encounterType EncounterType, summary string) entities.Memory {
	memory := entities.NewMemory(
		fmt.Sprintf("encounter_%s_%s_%d", idOf(owner), idOf(other), at.ToMinutes()),
		at,
		entities.MemoryErotic,
		summary,
	)
	memory.Participants = []string{idOf(owner), idOf(other)}
	memory.EmotionIntensity = 74
	memory.Tags = []string{"private", "erotic", strings.ToLower(encounterType.String())}
	return memory
}

func internalSummary(ctx EncounterContext, encounterType EncounterType) string {
	return fmt.Sprintf("%s y %s tuvieron un encuentro %s en %s.", nameOf(ctx.A), nameOf(ctx.B), strings.ToLower(encounterType.String()), locationName(ctx.Location))
}

func restrictedSummary(ctx EncounterContext) string {
	return fmt.Sprintf("[%s y %s pasaron tiempo juntos]", nameOf(ctx.A), nameOf(ctx.B))
}

func applyReactionToRelationship(owner, other *entities.Poble, reaction AftermathReaction) {
	if owner == nil || other == nil {
		return
	}
	relationship := ensureRelationship(owner, other.ID)
	relationship.Affection = clampPercent(relationship.Affection + reaction.RelationshipDelta)
	relationship.Trust = clampPercent(relationship.Trust + reaction.TrustDelta)
	relationship.Attraction = clampPercent(relationship.Attraction + reaction.AttractionDelta)
	relationship.Resentment = clampPercent(relationship.Resentment + reaction.ResentmentDelta)
	relationship.LastInteraction = other.DayOfBirth
	owner.Relationships[other.ID] = relationship
}

func applyReactionToBodyAndMood(poble *entities.Poble, reaction AftermathReaction) {
	if poble == nil {
		return
	}
	poble.Needs.Sex = clampPercent(poble.Needs.Sex - 28)
	poble.Needs.Belonging = clampPercent(poble.Needs.Belonging - 6)
	poble.CurrentMood = reaction.Mood
	poble.EmotionalState.CurrentMood = reaction.Mood
	poble.EmotionalState.ActiveEmotions = appendUniqueEmotion(poble.EmotionalState.ActiveEmotions, reaction.DominantEmotion)
}

func buildEncounterHealthFallout(ctx EncounterContext, encounterType EncounterType, usedProtection bool) EncounterHealthFallout {
	fallout := EncounterHealthFallout{
		UsedProtection: usedProtection,
		PregnancyRisk:  encounterRiskPercent(ctx, "pregnancy_risk", usedProtection),
		STIRisk:        encounterRiskPercent(ctx, "sti_risk", usedProtection),
		HPDelta:        [2]int{-2, -2},
		Summary:        "the encounter changed bodies lightly",
	}
	switch encounterType {
	case EncounterTender, EncounterCurious, EncounterFirstEver:
		fallout.HPDelta = [2]int{-1, -1}
	case EncounterPassionate, EncounterComplicated:
		fallout.HPDelta = [2]int{-4, -4}
	case EncounterDesperate, EncounterAngry:
		fallout.HPDelta = [2]int{-9, -7}
		fallout.Conditions[0] = append(fallout.Conditions[0], entities.ConditionExhausted)
		fallout.Conditions[1] = append(fallout.Conditions[1], entities.ConditionExhausted)
		fallout.Summary = "the encounter left exhaustion and visible fallout"
	case EncounterLast:
		fallout.HPDelta = [2]int{-12, -10}
		fallout.Conditions[0] = append(fallout.Conditions[0], entities.ConditionExhausted)
		fallout.Summary = "the encounter felt like a body running out of road"
	}
	if ctx.A != nil && ctx.A.Health.HP+fallout.HPDelta[0] <= 0 {
		fallout.DeathCause = events.DeathCauseAccident
	}
	if ctx.B != nil && ctx.B.Health.HP+fallout.HPDelta[1] <= 0 {
		fallout.DeathCause = events.DeathCauseAccident
	}
	return fallout
}

func encounterRiskPercent(ctx EncounterContext, risk string, usedProtection bool) int {
	if usedProtection {
		return 0
	}
	base := 8
	if risk == "sti_risk" {
		base = 12
		if source, _, target := stiCarrier(ctx.A, ctx.B); source != nil && target != nil {
			base = 42
		}
	}
	if risk == "pregnancy_risk" && pregnancyCarrier(ctx.A, ctx.B) != nil {
		base = 26
	}
	if DetermineEncounterType(ctx) == EncounterDesperate || DetermineEncounterType(ctx) == EncounterPassionate {
		base += 8
	}
	return clampInt(base, 0, 95)
}

func applyEncounterHealthFallout(a, b *entities.Poble, aftermath EncounterAftermath, gameWorld *world.World) {
	participants := [2]*entities.Poble{a, b}
	for index, poble := range participants {
		if poble == nil || !poble.IsAlive {
			continue
		}
		delta := aftermath.HealthFallout.HPDelta[index]
		if delta != 0 {
			poble.Health.HP = clampInt(poble.Health.HP+delta, 0, 100)
		}
		for _, condition := range aftermath.HealthFallout.Conditions[index] {
			if condition.IsValid() && !hasCondition(poble, condition) {
				poble.Health.Conditions = append(poble.Health.Conditions, condition)
			}
		}
		if delta < -5 || len(aftermath.HealthFallout.Conditions[index]) > 0 {
			appendEncounterWorldEvent(gameWorld, ai.GameEventSocialNegative, aftermath, "health", []string{poble.ID}, fmt.Sprintf("%s left the encounter with health fallout: %s.", nameOf(poble), aftermath.HealthFallout.Summary), 0.48, -0.24)
		}
		if aftermath.HealthFallout.DeathCause != "" && poble.Health.HP <= 0 {
			death := events.HandleDeath(poble, aftermath.HealthFallout.DeathCause, gameWorld)
			death.Description = fmt.Sprintf("%s died after an encounter turned physically unsafe.", nameOf(poble))
			appendEncounterWorldEvent(gameWorld, ai.GameEventDeath, aftermath, "death", []string{poble.ID}, death.Description, 0.98, -0.95)
		}
	}
}

func appendEncounterWorldEvent(gameWorld *world.World, eventType ai.GameEventType, aftermath EncounterAftermath, suffix string, participants []string, description string, severity, valence float32) {
	if gameWorld == nil {
		return
	}
	if strings.TrimSpace(description) == "" {
		description = aftermath.VisibleSummary
	}
	tags := []string{"encounter", "intimacy", strings.ToLower(aftermath.Type.String()), suffix}
	if aftermath.IsSecret {
		tags = append(tags, "secret", "affair")
	}
	if aftermath.PregnancyTriggered {
		tags = append(tags, "pregnancy")
	}
	if aftermath.STITransmitted != entities.STINone {
		tags = append(tags, "sti", strings.ToLower(aftermath.STITransmitted.String()))
	}
	event := world.GameEvent{
		ID:           fmt.Sprintf("encounter_%s_%d_%s", suffix, gameWorld.Calendar.ToMinutes(), strings.Join(uniqueStrings(participants), "_")),
		Type:         eventType,
		Time:         gameWorld.Calendar,
		PrimaryActor: firstNonEmpty(participants),
		Participants: uniqueStrings(participants),
		Severity:     severity,
		Valence:      valence,
		Description:  description,
		Tags:         uniqueStrings(tags),
	}
	gameWorld.EventHistory = append(gameWorld.EventHistory, event)
}

func registerWitnesses(gameWorld *world.World, a, b *entities.Poble, aftermath EncounterAftermath) {
	if gameWorld == nil {
		return
	}
	for _, witnessID := range aftermath.Witnesses {
		witness := gameWorld.GetPoble(witnessID)
		if witness == nil {
			continue
		}
		memory := entities.NewMemory(
			fmt.Sprintf("encounter_witness_%s_%d", witnessID, gameWorld.Calendar.ToMinutes()),
			gameWorld.Calendar,
			entities.MemoryNegative,
			fmt.Sprintf("%s vio a %s y %s demasiado cerca para fingir que no habia historia ahi.", witness.Name, nameOf(a), nameOf(b)),
		)
		memory.Participants = []string{witnessID, idOf(a), idOf(b)}
		memory.EmotionIntensity = 58
		memory.Tags = []string{"witness", "private", "encounter"}
		witness.Memories = append(witness.Memories, memory)
	}
}

func witnessesAtLocation(gameWorld *world.World, location world.Location, ignored ...string) []string {
	if gameWorld == nil {
		return nil
	}
	ignore := map[string]struct{}{}
	for _, id := range ignored {
		if id != "" {
			ignore[id] = struct{}{}
		}
	}
	result := []string{}
	for _, candidate := range gameWorld.GetAllPobles() {
		if candidate == nil {
			continue
		}
		if _, blocked := ignore[candidate.ID]; blocked {
			continue
		}
		loc, ok := gameWorld.GetLocation(candidate.ID)
		if ok && loc.ID == location.ID {
			result = append(result, candidate.ID)
		}
	}
	return result
}

func shouldUseProtection(ctx EncounterContext, encounterType EncounterType) bool {
	base := ctx.Relationship.Trust >= 65
	switch encounterType {
	case EncounterTender, EncounterCurious, EncounterFirstEver:
		return true
	case EncounterPassionate:
		return base
	case EncounterAngry, EncounterDesperate:
		return false
	default:
		return ctx.Power.SecrecyPressure < 55 && base
	}
}

func shouldTriggerPregnancy(ctx EncounterContext, usedProtection bool) bool {
	if usedProtection || ctx.A == nil || ctx.B == nil {
		return false
	}
	carrier := pregnancyCarrier(ctx.A, ctx.B)
	source := pregnancySource(ctx.A, ctx.B)
	if carrier == nil || source == nil {
		return false
	}
	chance := carrier.Health.Fertility * 0.22
	if ctx.Relationship.Attraction >= 70 {
		chance += 0.08
	}
	if DetermineEncounterType(ctx) == EncounterPassionate || DetermineEncounterType(ctx) == EncounterDesperate {
		chance += 0.05
	}
	return encounterContextRoll(ctx, "pregnancy") < chance
}

func deterministicSTIResult(ctx EncounterContext, usedProtection bool) (entities.STIType, bool) {
	if usedProtection || ctx.A == nil || ctx.B == nil {
		return entities.STINone, false
	}
	return pickSTITransmission(ctx, usedProtection), pickSTITransmission(ctx, usedProtection) != entities.STINone
}

func pickSTITransmission(ctx EncounterContext, usedProtection bool) entities.STIType {
	if usedProtection {
		return entities.STINone
	}
	source, stiType, target := stiCarrier(ctx.A, ctx.B)
	if source == nil || target == nil || stiType == entities.STINone {
		return entities.STINone
	}
	chance := float32(0.30)
	if DetermineEncounterType(ctx) == EncounterPassionate || DetermineEncounterType(ctx) == EncounterDesperate {
		chance += 0.08
	}
	if encounterContextRoll(ctx, "sti:"+stiType.String()) < chance {
		return stiType
	}
	return entities.STINone
}

func applySTITransmission(a, b *entities.Poble, stiType entities.STIType) {
	source, _, target := stiCarrier(a, b)
	if source == nil || target == nil || stiType == entities.STINone || hasSTI(target, stiType) {
		return
	}
	target.Health.STIs = append(target.Health.STIs, stiType)
}

func futureHintsFor(ctx EncounterContext, encounterType EncounterType, level config.ContentLevel) []string {
	hints := []string{strings.ToLower(encounterType.String())}
	if ctx.IsFirstTime {
		hints = append(hints, "first_time")
	}
	if level == config.ContentRestricted {
		hints = append(hints, "restricted")
	}
	if placeFeelsSecret(ctx.Location) {
		hints = append(hints, "secret")
	}
	return uniqueStrings(hints)
}

func encounterMoodFromType(encounterType EncounterType) EncounterMood {
	switch encounterType {
	case EncounterTender:
		return EncounterMoodTender
	case EncounterPassionate:
		return EncounterMoodPassionate
	case EncounterDesperate:
		return EncounterMoodDesperate
	case EncounterAngry:
		return EncounterMoodAngry
	case EncounterCurious:
		return EncounterMoodCurious
	case EncounterSecret:
		return EncounterMoodSecret
	case EncounterTransactional:
		return EncounterMoodTransactional
	case EncounterComplicated:
		return EncounterMoodComplicated
	case EncounterLast:
		return EncounterMoodLast
	case EncounterFirstEver:
		return EncounterMoodFirstEver
	default:
		return EncounterMoodComplicated
	}
}

func relationBetween(first, second *entities.Poble) entities.Relationship {
	if first == nil || second == nil {
		return entities.NewRelationship("", entities.RelationshipStranger)
	}
	if first.Relationships == nil {
		return entities.NewRelationship(second.ID, entities.RelationshipStranger)
	}
	if relationship, ok := first.Relationships[second.ID]; ok {
		return relationship
	}
	return entities.NewRelationship(second.ID, entities.RelationshipStranger)
}

func reverseRelationship(ctx EncounterContext) entities.Relationship {
	if ctx.B == nil || ctx.A == nil {
		return entities.NewRelationship("", entities.RelationshipStranger)
	}
	return relationBetween(ctx.B, ctx.A)
}

func ensureRelationship(owner *entities.Poble, targetID string) entities.Relationship {
	if owner == nil {
		return entities.NewRelationship(targetID, entities.RelationshipStranger)
	}
	if owner.Relationships == nil {
		owner.Relationships = map[string]entities.Relationship{}
	}
	if relationship, ok := owner.Relationships[targetID]; ok {
		return relationship
	}
	return entities.NewRelationship(targetID, entities.RelationshipStranger)
}

func samePreference(a, b entities.HiddenPreference) bool {
	return strings.EqualFold(a.Name, b.Name) || (a.Category == b.Category && absInt(int(a.Intensity)-int(b.Intensity)) <= 18)
}

func frictionBetween(a, b entities.HiddenPreference) bool {
	if a.Category != b.Category {
		return false
	}
	return absInt(int(a.Intensity)-int(b.Intensity)) >= 45
}

func preferenceLabel(pref entities.HiddenPreference) string {
	if strings.TrimSpace(pref.Name) != "" {
		return strings.TrimSpace(pref.Name)
	}
	return strings.ToLower(pref.Category.String())
}

func frictionLabel(a, b entities.HiddenPreference) string {
	return fmt.Sprintf("%s no coincide en ritmo con %s", preferenceLabel(a), preferenceLabel(b))
}

func secrecyPressure(a, b *entities.Poble, relationship entities.Relationship) int {
	pressure := 0
	if relationship.IsSecret {
		pressure += 35
	}
	if a != nil {
		pressure += int(a.Personality.Jealousy / 5)
	}
	if b != nil {
		pressure += int(b.Personality.Jealousy / 5)
	}
	return clampInt(pressure, 0, 100)
}

func witnessCount(ctx EncounterContext) int {
	return len(ctx.Witnesses)
}

func placeFeelsSecret(location *world.Location) bool {
	return location != nil && (location.PrivacyLevel >= 70 || location.Type == world.LocationHotel || location.Type == world.LocationApartment)
}

func locationName(location *world.Location) string {
	if location == nil || strings.TrimSpace(location.Name) == "" {
		return "un lugar sin nombre"
	}
	return location.Name
}

func hasEroticMemoryWith(poble *entities.Poble, targetID string) bool {
	if poble == nil {
		return false
	}
	for _, memory := range poble.Memories {
		if memory.Type == entities.MemoryErotic && memoryInvolves(memory, targetID) {
			return true
		}
	}
	return false
}

func hasMemoryTagBetween(poble *entities.Poble, targetID string, tag string) bool {
	if poble == nil {
		return false
	}
	for _, memory := range poble.Memories {
		if !memoryInvolves(memory, targetID) {
			continue
		}
		for _, current := range memory.Tags {
			if strings.EqualFold(current, tag) {
				return true
			}
		}
	}
	return false
}

func isFirstEroticMemory(first, second *entities.Poble) bool {
	return !hasEroticMemoryWith(first, idOf(second)) && !hasEroticMemoryWith(second, idOf(first))
}

func memoryInvolves(memory entities.Memory, targetID string) bool {
	for _, participant := range memory.Participants {
		if participant == targetID {
			return true
		}
	}
	return false
}

func pregnancyCarrier(first, second *entities.Poble) *entities.Poble {
	if canCarryPregnancy(first) && canImpregnate(second) {
		return first
	}
	if canCarryPregnancy(second) && canImpregnate(first) {
		return second
	}
	return nil
}

func pregnancySource(first, second *entities.Poble) *entities.Poble {
	if canImpregnate(first) && canCarryPregnancy(second) {
		return first
	}
	if canImpregnate(second) && canCarryPregnancy(first) {
		return second
	}
	return nil
}

func canCarryPregnancy(poble *entities.Poble) bool {
	if poble == nil {
		return false
	}
	return (poble.Sex == entities.Female || poble.Sex == entities.Intersex) && poble.Age >= 16 && poble.Age <= 48
}

func canImpregnate(poble *entities.Poble) bool {
	if poble == nil {
		return false
	}
	return (poble.Sex == entities.Male || poble.Sex == entities.Intersex) && poble.Health.Fertility >= 0.1
}

func stiCarrier(first, second *entities.Poble) (*entities.Poble, entities.STIType, *entities.Poble) {
	if first == nil || second == nil {
		return nil, entities.STINone, nil
	}
	if stiType, ok := firstSTI(first); ok && !hasSTI(second, stiType) {
		return first, stiType, second
	}
	if stiType, ok := firstSTI(second); ok && !hasSTI(first, stiType) {
		return second, stiType, first
	}
	return nil, entities.STINone, nil
}

func firstSTI(poble *entities.Poble) (entities.STIType, bool) {
	if poble == nil {
		return entities.STINone, false
	}
	for _, current := range poble.Health.STIs {
		if current.IsValid() && current != entities.STINone {
			return current, true
		}
	}
	return entities.STINone, false
}

func hasSTI(poble *entities.Poble, stiType entities.STIType) bool {
	if poble == nil {
		return false
	}
	for _, current := range poble.Health.STIs {
		if current == stiType {
			return true
		}
	}
	return false
}

func encounterContextRoll(ctx EncounterContext, salt string) float32 {
	hasher := fnv.New32a()
	parts := []string{
		salt,
		fmt.Sprintf("%d", ctx.StartedAt.Day),
		fmt.Sprintf("%d", ctx.StartedAt.Hour),
		idOf(ctx.A),
		idOf(ctx.B),
	}
	if ctx.Location != nil {
		parts = append(parts, ctx.Location.ID)
	}
	_, _ = hasher.Write([]byte(strings.Join(parts, "|")))
	return float32(hasher.Sum32()%1000) / 1000
}

func encounterRoll(a, b *entities.Poble, salt string, at entities.GameTime) float32 {
	return encounterContextRoll(EncounterContext{A: a, B: b, StartedAt: at}, salt)
}

func relationshipKey(a, b any) string { return pairKey(a, b, "relationship") }
func trustKey(a, b any) string        { return pairKey(a, b, "trust") }
func attractionKey(a, b any) string   { return pairKey(a, b, "attraction") }

func pairKey(a, b any, suffix string) string {
	return pairID(a) + ":" + pairID(b) + ":" + suffix
}

func pairID(value any) string {
	switch typed := value.(type) {
	case *entities.Poble:
		return idOf(typed)
	case entities.Poble:
		return idOf(&typed)
	case string:
		return typed
	default:
		return ""
	}
}

func worldParticipant(gameWorld *world.World, fallback *entities.Poble) *entities.Poble {
	if gameWorld == nil || fallback == nil {
		return fallback
	}
	if current := gameWorld.GetPoble(fallback.ID); current != nil {
		return current
	}
	return fallback
}

func hasCondition(poble *entities.Poble, condition entities.ConditionID) bool {
	if poble == nil {
		return false
	}
	for _, current := range poble.Health.Conditions {
		if current == condition {
			return true
		}
	}
	return false
}

func appendUniqueEmotion(values []entities.EmotionType, target entities.EmotionType) []entities.EmotionType {
	for _, current := range values {
		if current == target {
			return values
		}
	}
	return append(values, target)
}

func appendUniqueString(values []string, target string) []string {
	for _, current := range values {
		if current == target {
			return values
		}
	}
	return append(values, target)
}

func uniqueStrings(values []string) []string {
	if len(values) == 0 {
		return values
	}
	seen := map[string]struct{}{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func firstNonEmpty(values []string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func clampPercent(value float32) float32 {
	if value < 0 {
		return 0
	}
	if value > 100 {
		return 100
	}
	return value
}

func clampReaction(value float32) float32 {
	if value < -25 {
		return -25
	}
	if value > 25 {
		return 25
	}
	return value
}

func clampInt(value, low, high int) int {
	if value < low {
		return low
	}
	if value > high {
		return high
	}
	return value
}

func absInt(value int) int {
	if value < 0 {
		return -value
	}
	return value
}

func idOf(poble *entities.Poble) string {
	if poble == nil {
		return ""
	}
	return poble.ID
}

func nameOf(poble *entities.Poble) string {
	if poble == nil || strings.TrimSpace(poble.Name) == "" {
		return "alguien"
	}
	return poble.Name
}
