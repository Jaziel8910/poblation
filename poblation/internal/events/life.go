package events

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/user/poblation/internal/entities"
	simworld "github.com/user/poblation/internal/world"
)

// DeathCause identifies why a Poble died.
type DeathCause string

const (
	DeathCauseNaturalAge DeathCause = "NATURAL_AGE"
	DeathCauseIllness    DeathCause = "ILLNESS"
	DeathCauseAccident   DeathCause = "ACCIDENT"
	DeathCauseMurder     DeathCause = "MURDER"
	DeathCauseSuicide    DeathCause = "SUICIDE"
	DeathCauseWar        DeathCause = "WAR"
	DeathCauseExecution  DeathCause = "EXECUTION"
	DeathCauseChildbirth DeathCause = "CHILDBIRTH"
	DeathCauseStarvation DeathCause = "STARVATION"
)

// PregnancyArc tracks an active pregnancy storyline.
type PregnancyArc struct {
	MotherID           string            `json:"mother_id"`
	StartTime          entities.GameTime `json:"start_time"`
	DurationHours      int               `json:"duration_hours"`
	DueTime            entities.GameTime `json:"due_time"`
	RiskComplication   float32           `json:"risk_complication"`
	MiscarriageRisk    float32           `json:"miscarriage_risk"`
	DifficultBirthRisk float32           `json:"difficult_birth_risk"`
	FatherKnown        bool              `json:"father_known"`
	FatherOfficial     string            `json:"father_official"`
	FatherActual       string            `json:"father_actual"`
	DramaGuaranteed    bool              `json:"drama_guaranteed"`
	ComplicationFlags  []string          `json:"complication_flags"`
}

const (
	pregnancyArcPrefix = "pregnancy_arc|"
	stiSecretPrefix    = "sti_secret|"
	suicideNoteTag     = "suicide_note"
)

// HandleDeath resolves the death of one Poble and its immediate fallout.
func HandleDeath(poble *entities.Poble, cause DeathCause, world World) GameEvent {
	if poble == nil || world == nil {
		return GameEvent{}
	}

	now := currentWorldTime(world)
	rng := lifecycleRNG(now, "death", poble.ID, string(cause))

	poble.IsAlive = false
	poble.Health.HP = 0
	poble.CurrentMood = entities.MoodNumb
	poble.EmotionalState.CurrentMood = entities.MoodNumb
	appendMemory(poble, entities.Memory{
		ID:               lifecycleID("last_moment", now, poble.ID),
		Timestamp:        now,
		Type:             deathMemoryType(cause),
		Participants:     []string{poble.ID},
		EmotionIntensity: 100,
		Tags:             []string{"death", strings.ToLower(string(cause))},
		Summary:          deathMemorySummary(cause),
	})

	consequences := []Consequence{
		{TargetID: poble.ID, Type: ConsequenceDeathCaused, Value: 1},
	}
	childEvents := []string{}

	for _, observer := range world.GetAllPobles() {
		if observer == nil || observer.ID == poble.ID {
			continue
		}
		consequences = append(consequences, applyDeathReaction(observer, poble, cause, now)...)
	}

	childEvents = append(childEvents, resolveDependentsAndInheritance(poble, world, now, rng, &consequences)...)
	if isUnexpectedDeath(cause) {
		childEvents = append(childEvents, createDeathRumour(poble, cause, world, now))
	}
	if isLastOfGeneration(poble, world) {
		childEvents = append(childEvents, lifecycleID("generation_end", now, poble.ID))
	}

	return GameEvent{
		ID:           lifecycleID("death", now, poble.ID, string(cause)),
		Type:         deathEventType(cause),
		Timestamp:    now,
		Participants: []string{poble.ID},
		IsPublic:     true,
		Consequences: consequences,
		ChildEvents:  uniqueStrings(childEvents),
	}
}

// HandleBirth creates a baby, links family relationships, and returns the birth event.
func HandleBirth(motherID string, fatherID string, world World) (*entities.Poble, GameEvent) {
	if world == nil || motherID == "" {
		return nil, GameEvent{}
	}

	now := currentWorldTime(world)
	mother := world.GetPoble(motherID)
	if mother == nil {
		return nil, GameEvent{}
	}

	secretArc := decodePregnancyArcSecret(mother)
	officialFatherID := fatherID
	actualFatherID := fatherID
	if secretArc != nil {
		if secretArc["official_father"] != "" {
			officialFatherID = secretArc["official_father"]
		}
		if secretArc["actual_father"] != "" {
			actualFatherID = secretArc["actual_father"]
		}
	}

	actualFather := world.GetPoble(actualFatherID)
	officialFather := world.GetPoble(officialFatherID)
	rng := lifecycleRNG(now, "birth", motherID, actualFatherID, officialFatherID)
	baby := createBaby(mother, actualFather, now, rng)

	if officialFather != nil {
		baby.Parents[1] = officialFather.ID
	}
	if actualFather != nil && officialFather == nil {
		baby.Parents[1] = actualFather.ID
	}

	linkParentAndChild(mother, baby, now)
	if actualFather != nil {
		linkParentAndChild(actualFather, baby, now)
	}
	if officialFather != nil && (actualFather == nil || officialFather.ID != actualFather.ID) {
		linkSocialParent(officialFather, baby, now)
	}

	consequences := []Consequence{
		{TargetID: mother.ID, Type: ConsequenceNeedChange, Value: -25},
		{TargetID: mother.ID, Type: ConsequenceMoodShift, Value: 22},
	}
	childEvents := []string{}

	mother.Health.Conditions = removeCondition(mother.Health.Conditions, entities.ConditionPregnant)
	mother.Needs.Belonging = clampFloat(mother.Needs.Belonging-18, 0, 100)
	mother.CurrentMood = entities.MoodContent
	mother.EmotionalState.CurrentMood = entities.MoodContent
	appendMemory(mother, newLifeMemory("birth", now, baby.ID, "birth"))

	if actualFather != nil {
		consequences = append(consequences, Consequence{TargetID: actualFather.ID, Type: ConsequenceMoodShift, Value: 18})
		appendMemory(actualFather, newLifeMemory("birth", now, baby.ID, "birth"))
	}

	for _, observer := range world.GetAllPobles() {
		if observer == nil || observer.ID == mother.ID {
			continue
		}
		if observer.ID == baby.ID {
			continue
		}
		applyBirthReaction(observer, baby, mother, now, &consequences)
	}

	if parentsAreTooClose(mother, actualFather) && rng.Float32() < 0.15 {
		if !hasCondition(baby, entities.ConditionSick) {
			baby.Health.Conditions = append(baby.Health.Conditions, entities.ConditionSick)
		}
		baby.Health.HP = clampInt(baby.Health.HP-18, 0, 100)
		childEvents = append(childEvents, lifecycleID("genetic_condition", now, baby.ID))
	}

	if actualFather != nil && officialFather != nil && actualFather.ID != officialFather.ID {
		childEvents = append(childEvents, lifecycleID("secret_father_drama", now, mother.ID, baby.ID))
	}

	registerBabyInWorld(world, mother.ID, baby)

	return baby, GameEvent{
		ID:           lifecycleID("birth", now, mother.ID, baby.ID),
		Type:         EventBirth,
		Timestamp:    now,
		Participants: uniqueStrings([]string{mother.ID, baby.ID, actualFatherID, officialFatherID}),
		IsPublic:     true,
		Consequences: consequences,
		ChildEvents:  uniqueStrings(childEvents),
	}
}

// HandleIllness progresses one illness and may return contagion, recovery, or death events.
func HandleIllness(poble *entities.Poble, illnessType entities.ConditionID, world World) []GameEvent {
	if poble == nil || world == nil {
		return nil
	}

	now := currentWorldTime(world)
	rng := lifecycleRNG(now, "illness", poble.ID, illnessType.String())
	events := make([]GameEvent, 0, 4)

	if !hasCondition(poble, illnessType) {
		poble.Health.Conditions = append(poble.Health.Conditions, illnessType)
	}

	damage := illnessDamageForStage(poble)
	poble.Health.HP = clampInt(poble.Health.HP-damage, 0, 100)
	poble.Mental.Stability = clampInt(poble.Mental.Stability-(damage/2), 0, 100)
	appendMemory(poble, entities.Memory{
		ID:               lifecycleID("illness", now, poble.ID, illnessType.String()),
		Timestamp:        now,
		Type:             entities.MemoryNegative,
		Participants:     []string{poble.ID},
		EmotionIntensity: clampFloat(float32(35+damage), 0, 100),
		Tags:             []string{"illness", strings.ToLower(illnessType.String())},
		Summary:          illnessSummary(illnessType, poble.Health.HP),
	})

	events = append(events, GameEvent{
		ID:           lifecycleID("illness_event", now, poble.ID, illnessType.String()),
		Type:         EventIllnessOnset,
		Timestamp:    now,
		Participants: []string{poble.ID},
		IsPublic:     false,
		Consequences: []Consequence{{TargetID: poble.ID, Type: ConsequenceHealthChange, Value: -float32(damage)}},
	})

	for _, contact := range closeContactsForIllness(poble, world) {
		if contact == nil || hasCondition(contact, illnessType) {
			continue
		}
		if rng.Float32() >= illnessSpreadChance(poble, contact) {
			continue
		}
		contact.Health.Conditions = append(contact.Health.Conditions, illnessType)
		events = append(events, GameEvent{
			ID:           lifecycleID("illness_spread", now, poble.ID, contact.ID),
			Type:         EventPlague,
			Timestamp:    now,
			Participants: []string{poble.ID, contact.ID},
			IsPublic:     false,
			Consequences: []Consequence{{TargetID: contact.ID, Type: ConsequenceConditionAdded, Value: 1}},
		})
	}

	recoveryChance := recoveryChanceForIllness(world, poble)
	if rng.Float32() < recoveryChance {
		poble.Health.Conditions = removeCondition(poble.Health.Conditions, illnessType)
		poble.Health.HP = clampInt(poble.Health.HP+12, 0, 100)
		events = append(events, GameEvent{
			ID:           lifecycleID("recovery", now, poble.ID),
			Type:         EventRecovery,
			Timestamp:    now,
			Participants: []string{poble.ID},
			IsPublic:     false,
			Consequences: []Consequence{{TargetID: poble.ID, Type: ConsequenceHealthChange, Value: 12}},
		})
		return events
	}

	if poble.Health.HP <= 15 {
		events = append(events, HandleDeath(poble, DeathCauseIllness, world))
		return events
	}

	if poble.Health.HP <= 35 {
		events = append(events, GameEvent{
			ID:           lifecycleID("terminal_arc", now, poble.ID),
			Type:         EventDecisionPoint,
			Timestamp:    now,
			Participants: []string{poble.ID},
			IsPublic:     false,
			Consequences: []Consequence{{TargetID: poble.ID, Type: ConsequenceMentalChange, Value: -8}},
		})
	}

	return events
}

// HandlePregnancy creates a pregnancy arc and marks the mother as pregnant.
func HandlePregnancy(motherID string, world World) PregnancyArc {
	if world == nil || motherID == "" {
		return PregnancyArc{}
	}

	now := currentWorldTime(world)
	mother := world.GetPoble(motherID)
	if mother == nil {
		return PregnancyArc{}
	}

	rng := lifecycleRNG(now, "pregnancy", motherID)
	actualFather, officialFather := choosePregnancyFathers(mother, world, rng)
	arc := PregnancyArc{
		MotherID:           mother.ID,
		StartTime:          now,
		DurationHours:      9 * 24,
		DueTime:            now.Add(9 * 24),
		MiscarriageRisk:    10,
		DifficultBirthRisk: 15,
		FatherKnown:        officialFather != "",
		FatherOfficial:     officialFather,
		FatherActual:       actualFather,
		ComplicationFlags:  []string{},
	}

	arc.RiskComplication = clampFloat(float32(arc.MiscarriageRisk+arc.DifficultBirthRisk), 0, 100)
	if mother.Age < 18 || mother.Age > 38 || mother.Health.HP < 55 {
		arc.RiskComplication = clampFloat(arc.RiskComplication+12, 0, 100)
		arc.MiscarriageRisk = clampFloat(arc.MiscarriageRisk+5, 0, 100)
		arc.ComplicationFlags = append(arc.ComplicationFlags, "high_risk")
	}
	if parentsAreTooClose(mother, world.GetPoble(actualFather)) {
		arc.RiskComplication = clampFloat(arc.RiskComplication+10, 0, 100)
		arc.ComplicationFlags = append(arc.ComplicationFlags, "consanguinity")
	}

	if actualFather != "" && officialFather != "" && actualFather != officialFather {
		father := world.GetPoble(actualFather)
		if father != nil && father.Orientation.Sexual >= 0.75 {
			arc.DramaGuaranteed = true
			arc.ComplicationFlags = append(arc.ComplicationFlags, "third_party_drama")
		}
		storePregnancyArcSecret(mother, actualFather, officialFather, now)
	}

	if !hasCondition(mother, entities.ConditionPregnant) {
		mother.Health.Conditions = append(mother.Health.Conditions, entities.ConditionPregnant)
	}
	mother.Needs.Sex = clampFloat(mother.Needs.Sex-20, 0, 100)
	mother.Needs.Safety = clampFloat(mother.Needs.Safety+10, 0, 100)

	return arc
}

// HandleSTI models transmission, incubation, and later discovery drama.
func HandleSTI(fromID string, toID string, stiType entities.STIType, world World) []GameEvent {
	if world == nil || fromID == "" || toID == "" || !stiType.IsValid() || stiType == entities.STINone {
		return nil
	}

	from := world.GetPoble(fromID)
	to := world.GetPoble(toID)
	if from == nil || to == nil {
		return nil
	}

	now := currentWorldTime(world)
	rng := lifecycleRNG(now, "sti", fromID, toID, stiType.String())
	if rng.Float32() >= stiTransmissionChance(stiType) {
		return nil
	}

	if !hasSTI(to, stiType) {
		to.Health.STIs = append(to.Health.STIs, stiType)
	}
	storeSTISecret(to, stiType, incubationDaysForSTI(stiType), now)

	events := []GameEvent{{
		ID:           lifecycleID("sti_transmission", now, fromID, toID, stiType.String()),
		Type:         EventSexualEncounter,
		Timestamp:    now,
		Participants: []string{fromID, toID},
		IsPublic:     false,
		Consequences: []Consequence{{TargetID: toID, Type: ConsequenceConditionAdded, Value: 1}},
	}}

	revealTime := now.Add(incubationDaysForSTI(stiType) * 24)
	revealConsequences := []Consequence{
		{TargetID: toID, Type: ConsequenceMentalChange, Value: -12},
		{TargetID: toID, Type: ConsequenceSecretRevealed, Value: 1},
	}
	if hasSTI(from, stiType) {
		from.Secrets = append(from.Secrets, entities.NewSecret(
			lifecycleID("sti_knowledge", now, fromID, stiType.String()),
			entities.SecretCriminalAct,
			fmt.Sprintf("%s knowingly carried %s", fromID, stiType.String()),
		))
	}
	events = append(events, GameEvent{
		ID:           lifecycleID("sti_reveal", revealTime, fromID, toID, stiType.String()),
		Type:         EventRevelation,
		Timestamp:    revealTime,
		Participants: []string{fromID, toID},
		IsPublic:     false,
		Consequences: revealConsequences,
	})

	return events
}

// HandleSuicide enforces the support gate before resolving the death event.
func HandleSuicide(poble *entities.Poble, triggerEventID string, world World) GameEvent {
	if poble == nil || world == nil || poble.Mental.Stability >= 10 || supportAvailableFor(poble, world) {
		return GameEvent{}
	}

	event := HandleDeath(poble, DeathCauseSuicide, world)
	now := event.Timestamp
	poble.Secrets = append(poble.Secrets, entities.NewSecret(
		lifecycleID("suicide_note", now, poble.ID),
		entities.SecretTraumaEvent,
		fmt.Sprintf("%s|trigger=%s", suicideNoteTag, triggerEventID),
	))
	event.ChildEvents = append(event.ChildEvents, lifecycleID("suicide_note", now, poble.ID))

	for _, observer := range world.GetAllPobles() {
		if observer == nil || observer.ID == poble.ID {
			continue
		}
		observer.Mental.Stability = clampInt(observer.Mental.Stability-18, 0, 100)
		observer.CurrentMood = entities.MoodDepressed
		observer.EmotionalState.CurrentMood = entities.MoodDepressed
		appendMemory(observer, entities.Memory{
			ID:               lifecycleID("suicide_impact", now, observer.ID, poble.ID),
			Timestamp:        now,
			Type:             entities.MemoryTraumatic,
			Participants:     []string{observer.ID, poble.ID},
			EmotionIntensity: 95,
			Tags:             []string{"death", "suicide", "trigger:" + triggerEventID},
			Summary:          "A death arrived after the mind had already been losing ground.",
		})
	}

	return event
}

func deathEventType(cause DeathCause) EventType {
	switch cause {
	case DeathCauseNaturalAge:
		return EventDeathNatural
	case DeathCauseMurder:
		return EventDeathMurder
	case DeathCauseSuicide:
		return EventSuicide
	default:
		return EventDeathAccident
	}
}

func deathMemoryType(cause DeathCause) entities.MemoryType {
	switch cause {
	case DeathCauseSuicide, DeathCauseMurder, DeathCauseWar, DeathCauseExecution:
		return entities.MemoryTraumatic
	default:
		return entities.MemoryNegative
	}
}

func deathMemorySummary(cause DeathCause) string {
	switch cause {
	case DeathCauseNaturalAge:
		return "The body gave out after a long life."
	case DeathCauseIllness:
		return "Illness slowly closed every remaining door."
	case DeathCauseMurder:
		return "Violence arrived with human intent behind it."
	case DeathCauseSuicide:
		return "Pain won a private fight that others only saw too late."
	case DeathCauseChildbirth:
		return "Life and death crossed during birth."
	case DeathCauseStarvation:
		return "Hunger took what the world failed to protect."
	default:
		return "Death came suddenly and changed the settlement."
	}
}

func applyDeathReaction(observer *entities.Poble, deceased *entities.Poble, cause DeathCause, now entities.GameTime) []Consequence {
	relationship, hasRelationship := observer.Relationships[deceased.ID]
	importance := relationshipImportance(relationship, hasRelationship)
	enemy := relationshipIsEnemy(relationship, hasRelationship)

	mentalDelta := float32(-10 - importance*0.22)
	moodDelta := float32(-18 - importance*0.20)
	memoryType := entities.MemoryNegative
	summary := "Someone died and the loss landed heavily."

	if enemy {
		mentalDelta = -4
		moodDelta = 8
		memoryType = entities.MemoryNegative
		summary = "A death brought relief tangled up with guilt."
	}

	if importance >= 75 || cause == DeathCauseMurder || cause == DeathCauseSuicide {
		mentalDelta -= 12
		memoryType = entities.MemoryTraumatic
	}

	observer.Mental.Stability = clampInt(observer.Mental.Stability+int(mentalDelta), 0, 100)
	observer.EmotionalState.Valence = clampSignedUnitLocal(observer.EmotionalState.Valence + (moodDelta / 100.0))
	if enemy && observer.EmotionalState.CurrentMood != entities.MoodAngry {
		observer.EmotionalState.CurrentMood = entities.MoodContent
		observer.CurrentMood = entities.MoodContent
	} else {
		observer.EmotionalState.CurrentMood = entities.MoodSad
		observer.CurrentMood = entities.MoodSad
	}
	appendMemory(observer, entities.Memory{
		ID:               lifecycleID("death_reaction", now, observer.ID, deceased.ID),
		Timestamp:        now,
		Type:             memoryType,
		Participants:     []string{observer.ID, deceased.ID},
		EmotionIntensity: clampFloat(45+importance*0.45, 0, 100),
		Tags:             []string{"death", strings.ToLower(string(cause))},
		Summary:          summary,
	})

	consequences := []Consequence{
		{TargetID: observer.ID, Type: ConsequenceMentalChange, Value: mentalDelta},
		{TargetID: observer.ID, Type: ConsequenceMoodShift, Value: moodDelta},
	}

	if importance >= 75 {
		observer.Mental.Traumas = uniqueStrings(append(observer.Mental.Traumas, lifecycleID("death_reaction", now, observer.ID, deceased.ID)))
		if !containsMentalConditionLocal(observer.Mental.Conditions, entities.MentalPTSD) && (cause == DeathCauseMurder || cause == DeathCauseSuicide) {
			observer.Mental.Conditions = append(observer.Mental.Conditions, entities.MentalPTSD)
		}
		if observer.Mental.Stability < 25 {
			consequences = append(consequences, Consequence{TargetID: observer.ID, Type: ConsequenceMentalChange, Value: -8})
		}
	}

	return consequences
}

func resolveDependentsAndInheritance(deceased *entities.Poble, world World, now entities.GameTime, rng *rand.Rand, consequences *[]Consequence) []string {
	childEvents := []string{}
	heirs := make([]*entities.Poble, 0, 4)

	for _, candidate := range world.GetAllPobles() {
		if candidate == nil || candidate.ID == deceased.ID {
			continue
		}

		if isChildOf(candidate, deceased.ID) {
			candidate.Needs.Safety = clampFloat(candidate.Needs.Safety+25, 0, 100)
			candidate.Needs.Belonging = clampFloat(candidate.Needs.Belonging+20, 0, 100)
			candidate.Mental.Stability = clampInt(candidate.Mental.Stability-12, 0, 100)
			*consequences = append(*consequences,
				Consequence{TargetID: candidate.ID, Type: ConsequenceNeedChange, Value: 25},
				Consequence{TargetID: candidate.ID, Type: ConsequenceMentalChange, Value: -12},
			)
			if otherParentMissing(candidate, deceased.ID, world) {
				childEvents = append(childEvents, lifecycleID("orphaned_child", now, candidate.ID))
			}
			heirs = append(heirs, candidate)
			continue
		}

		if isLikelyHeir(candidate, deceased.ID) {
			heirs = append(heirs, candidate)
		}
	}

	if deceased.HomeID != "" {
		childEvents = append(childEvents, lifecycleID("empty_home", now, deceased.HomeID))
	}

	if deceased.Money > 0 && len(heirs) > 0 {
		share := maxInt(1, deceased.Money/len(heirs))
		for _, heir := range heirs {
			if heir == nil {
				continue
			}
			heir.Money += share
			*consequences = append(*consequences, Consequence{TargetID: heir.ID, Type: ConsequenceMoneyChange, Value: float32(share)})
			childEvents = append(childEvents, lifecycleID("inheritance", now, deceased.ID, heir.ID))
		}
		deceased.Money = 0
	}

	if rng.Float32() < 0.15 && len(heirs) == 0 {
		childEvents = append(childEvents, lifecycleID("estate_lost", now, deceased.ID))
	}

	return childEvents
}

func relationshipImportance(relationship entities.Relationship, ok bool) float32 {
	if !ok {
		return 8
	}
	score := relationship.Affection*0.28 +
		relationship.Trust*0.22 +
		relationship.Attraction*0.18 +
		relationship.Dependency*0.20 +
		relationship.Familiarity*0.12
	if relationship.Type == entities.RelationshipParent || relationship.Type == entities.RelationshipChild ||
		relationship.Type == entities.RelationshipSibling || relationship.Type == entities.RelationshipSpouse ||
		relationship.Type == entities.RelationshipLover || relationship.Type == entities.RelationshipBestFriend {
		score += 22
	}
	return clampFloat(score, 0, 100)
}

func relationshipIsEnemy(relationship entities.Relationship, ok bool) bool {
	if !ok {
		return false
	}
	if relationship.Type == entities.RelationshipEnemy || relationship.Type == entities.RelationshipNemesis || relationship.Type == entities.RelationshipRival {
		return true
	}
	return relationship.Resentment >= 75 && relationship.Affection <= 20
}

func isUnexpectedDeath(cause DeathCause) bool {
	switch cause {
	case DeathCauseNaturalAge:
		return false
	default:
		return true
	}
}

func createDeathRumour(deceased *entities.Poble, cause DeathCause, world World, now entities.GameTime) string {
	if deceased == nil {
		return ""
	}
	if concrete, ok := world.(*simworld.World); ok {
		concrete.RumourPool = append(concrete.RumourPool, simworld.Rumour{
			ID:          lifecycleID("rumour", now, deceased.ID),
			SourceID:    deceased.ID,
			AboutID:     deceased.ID,
			Content:     "A violent death is being talked about in pieces.",
			Truthiness:  1.0,
			SpreadLevel: 0.15,
			Tags:        []string{"death", strings.ToLower(string(cause))},
		})
	}
	return lifecycleID("rumour", now, deceased.ID)
}

func isLastOfGeneration(poble *entities.Poble, world World) bool {
	if poble == nil || world == nil {
		return false
	}
	targetLevel := generationLevel(poble)
	for _, other := range world.GetAllPobles() {
		if other == nil || other.ID == poble.ID {
			continue
		}
		if generationLevel(other) == targetLevel {
			return false
		}
	}
	return true
}

func generationLevel(poble *entities.Poble) int {
	if poble == nil {
		return 0
	}
	level := 0
	if poble.Parents[0] != "" || poble.Parents[1] != "" {
		level++
	}
	if strings.TrimSpace(poble.Parents[0]) != "" && strings.TrimSpace(poble.Parents[1]) != "" {
		level++
	}
	return level
}

func createBaby(mother *entities.Poble, father *entities.Poble, now entities.GameTime, rng *rand.Rand) *entities.Poble {
	baby, err := entities.GeneratePople(entities.PoblConfig{}, rng)
	if err != nil || baby == nil {
		fallback := entities.NewPoble(lifecycleID("baby", now, mother.ID), "Newborn", 0, entities.Female)
		baby = &fallback
	}

	baby.ID = lifecycleID("baby", now, mother.ID)
	baby.Age = 0
	baby.DayOfBirth = now
	baby.Health = entities.NewHealthState(0)
	baby.Mental = entities.NewMentalState()
	baby.Needs = entities.NewNeeds()
	baby.Inventory = []entities.Item{}
	baby.Memories = []entities.Memory{}
	baby.Children = []string{}
	baby.Secrets = []entities.Secret{}
	baby.IsAlive = true
	baby.CurrentMood = entities.MoodNeutral
	baby.EmotionalState = entities.NewEmotionalState()
	baby.EmotionalState.CurrentMood = entities.MoodNeutral
	baby.Relationships = map[string]entities.Relationship{}
	baby.Health.Fertility = 0
	baby.Parents[0] = mother.ID

	if father != nil {
		inheritFromParents(baby, mother, father, rng)
	} else {
		baby.Personality.Agreeableness = clampFloat((mother.Personality.Agreeableness+55)/2, 0, 100)
		baby.Personality.Neuroticism = clampFloat((mother.Personality.Neuroticism+45)/2, 0, 100)
	}

	return baby
}

func inheritFromParents(baby *entities.Poble, mother *entities.Poble, father *entities.Poble, rng *rand.Rand) {
	mix := func(a, b float32, swing float64) float32 {
		return clampFloat((a+b)/2+float32(rng.NormFloat64()*swing), 0, 100)
	}
	baby.Personality.Openness = mix(mother.Personality.Openness, father.Personality.Openness, 6)
	baby.Personality.Conscientiousness = mix(mother.Personality.Conscientiousness, father.Personality.Conscientiousness, 6)
	baby.Personality.Extraversion = mix(mother.Personality.Extraversion, father.Personality.Extraversion, 6)
	baby.Personality.Agreeableness = mix(mother.Personality.Agreeableness, father.Personality.Agreeableness, 6)
	baby.Personality.Neuroticism = mix(mother.Personality.Neuroticism, father.Personality.Neuroticism, 6)
	baby.Personality.Cruelty = mix(mother.Personality.Cruelty, father.Personality.Cruelty, 5)
	baby.Personality.Horniness = mix(mother.Personality.Horniness, father.Personality.Horniness, 5)
	baby.Personality.Ambition = mix(mother.Personality.Ambition, father.Personality.Ambition, 6)
	baby.Personality.Jealousy = mix(mother.Personality.Jealousy, father.Personality.Jealousy, 5)
	baby.Personality.Loyalty = mix(mother.Personality.Loyalty, father.Personality.Loyalty, 5)
	baby.Orientation.Romantic = clampFloat((mother.Orientation.Romantic+father.Orientation.Romantic)/2, 0, 1)
	baby.Orientation.Sexual = clampFloat((mother.Orientation.Sexual+father.Orientation.Sexual)/2, 0, 1)
	baby.Orientation.Intensity = clampFloat((mother.Orientation.Intensity+father.Orientation.Intensity)/2, 0, 1)
	baby.Orientation.Fluidity = clampFloat((mother.Orientation.Fluidity+father.Orientation.Fluidity)/2, 0, 1)
}

func linkParentAndChild(parent *entities.Poble, child *entities.Poble, now entities.GameTime) {
	if parent == nil || child == nil {
		return
	}
	parent.Children = uniqueStrings(append(parent.Children, child.ID))
	parentRel := entities.NewRelationship(child.ID, entities.RelationshipChild)
	parentRel.Affection = 75
	parentRel.Trust = 70
	parentRel.Familiarity = 100
	parentRel.LastInteraction = now
	parent.Relationships[child.ID] = parentRel

	childRel := entities.NewRelationship(parent.ID, entities.RelationshipParent)
	childRel.Affection = 80
	childRel.Trust = 80
	childRel.Dependency = 100
	childRel.Familiarity = 100
	childRel.LastInteraction = now
	child.Relationships[parent.ID] = childRel
}

func linkSocialParent(parent *entities.Poble, child *entities.Poble, now entities.GameTime) {
	if parent == nil || child == nil {
		return
	}
	rel := entities.NewRelationship(child.ID, entities.RelationshipComplicated)
	rel.Affection = 55
	rel.Trust = 38
	rel.Resentment = 18
	rel.LastInteraction = now
	parent.Relationships[child.ID] = rel
}

func applyBirthReaction(observer *entities.Poble, baby *entities.Poble, mother *entities.Poble, now entities.GameTime, consequences *[]Consequence) {
	if observer == nil || baby == nil || mother == nil {
		return
	}
	value := float32(6)
	if relationship, ok := observer.Relationships[mother.ID]; ok {
		value += relationshipImportance(relationship, true) * 0.08
	}
	observer.EmotionalState.Valence = clampSignedUnitLocal(observer.EmotionalState.Valence + value/120.0)
	if observer.EmotionalState.CurrentMood != entities.MoodAngry {
		observer.EmotionalState.CurrentMood = entities.MoodContent
		observer.CurrentMood = entities.MoodContent
	}
	appendMemory(observer, newLifeMemory("birth", now, baby.ID, "settlement_birth"))
	*consequences = append(*consequences, Consequence{TargetID: observer.ID, Type: ConsequenceMoodShift, Value: value})
}

func parentsAreTooClose(mother *entities.Poble, father *entities.Poble) bool {
	if mother == nil || father == nil {
		return false
	}
	if mother.ID == father.ID {
		return true
	}
	for _, parentID := range mother.Parents {
		if parentID != "" && parentID == father.ID {
			return true
		}
	}
	for _, parentID := range father.Parents {
		if parentID != "" && parentID == mother.ID {
			return true
		}
	}
	for _, motherParent := range mother.Parents {
		if motherParent == "" {
			continue
		}
		for _, fatherParent := range father.Parents {
			if fatherParent != "" && fatherParent == motherParent {
				return true
			}
		}
	}
	if relationship, ok := mother.Relationships[father.ID]; ok {
		return relationship.Type == entities.RelationshipSibling || relationship.Type == entities.RelationshipFamily
	}
	return false
}

func registerBabyInWorld(world World, motherID string, baby *entities.Poble) {
	if baby == nil {
		return
	}
	concrete, ok := world.(*simworld.World)
	if !ok {
		return
	}
	location, found := concrete.GetLocation(motherID)
	if !found {
		state := concrete.GetWorldState()
		if len(state.Islands) == 0 {
			return
		}
		location = simworld.Location{IslandID: state.Islands[0].ID}
	}
	concrete.AddPoble(baby, location)
}

func illnessDamageForStage(poble *entities.Poble) int {
	if poble == nil {
		return 0
	}
	switch {
	case poble.Health.HP > 70:
		return 8
	case poble.Health.HP > 40:
		return 14
	default:
		return 22
	}
}

func illnessSummary(illnessType entities.ConditionID, hp int) string {
	switch {
	case hp <= 20:
		return fmt.Sprintf("%s turned critical and every hour felt narrower.", illnessType)
	case hp <= 45:
		return fmt.Sprintf("%s settled in deeper and started changing decisions.", illnessType)
	default:
		return fmt.Sprintf("%s began to shape the day around weakness.", illnessType)
	}
}

func closeContactsForIllness(poble *entities.Poble, world World) []*entities.Poble {
	contacts := make([]*entities.Poble, 0, 4)
	for targetID, relationship := range poble.Relationships {
		if relationship.Familiarity < 55 && relationship.Affection < 50 && relationship.Dependency < 45 {
			continue
		}
		target := world.GetPoble(targetID)
		if target != nil {
			contacts = append(contacts, target)
		}
	}
	return contacts
}

func illnessSpreadChance(source *entities.Poble, target *entities.Poble) float32 {
	if source == nil || target == nil {
		return 0
	}
	chance := float32(0.12)
	if relationship, ok := source.Relationships[target.ID]; ok {
		chance += relationship.Familiarity / 500.0
		chance += relationship.Dependency / 700.0
	}
	return clampFloat(chance, 0, 0.85)
}

func recoveryChanceForIllness(world World, poble *entities.Poble) float32 {
	ws := world.GetWorldState()
	chance := float32(0.18)
	if ws.TechTree.Unlocked["MEDICINE"] || ws.TechTree.Unlocked["BASIC_MEDICINE"] || ws.TechTree.Unlocked["SURGERY"] {
		chance += 0.28
	}
	if poble != nil {
		chance += poble.Health.Fertility * 0.06
		if poble.Health.HP < 30 {
			chance -= 0.10
		}
	}
	return clampFloat(chance, 0.05, 0.80)
}

func choosePregnancyFathers(mother *entities.Poble, world World, rng *rand.Rand) (string, string) {
	if mother == nil {
		return "", ""
	}
	official := ""
	actual := ""
	bestOfficial := float32(-1)
	bestActual := float32(-1)

	for targetID, relationship := range mother.Relationships {
		target := world.GetPoble(targetID)
		if target == nil || !canImpregnate(target) {
			continue
		}

		officialScore := relationship.Trust + relationship.Familiarity + relationship.Affection
		if relationship.Type == entities.RelationshipSpouse || relationship.Type == entities.RelationshipLover {
			officialScore += 30
		}
		if officialScore > bestOfficial {
			bestOfficial = officialScore
			official = targetID
		}

		actualScore := relationship.Attraction + relationship.Affection + relationship.Dependency
		if relationship.Type == entities.RelationshipFriendsWithBenefits {
			actualScore += 18
		}
		actualScore += rng.Float32() * 8
		if actualScore > bestActual {
			bestActual = actualScore
			actual = targetID
		}
	}

	if actual == "" {
		actual = official
	}
	return actual, official
}

func storePregnancyArcSecret(mother *entities.Poble, actualFather string, officialFather string, now entities.GameTime) {
	if mother == nil || actualFather == "" || officialFather == "" || actualFather == officialFather {
		return
	}
	content := pregnancyArcPrefix + "actual_father=" + actualFather + "|official_father=" + officialFather
	mother.Secrets = append(mother.Secrets, entities.NewSecret(
		lifecycleID("pregnancy_secret", now, mother.ID, actualFather, officialFather),
		entities.SecretPlannedBetrayal,
		content,
	))
}

func decodePregnancyArcSecret(mother *entities.Poble) map[string]string {
	if mother == nil {
		return nil
	}
	for _, secret := range mother.Secrets {
		if strings.HasPrefix(secret.Content, pregnancyArcPrefix) {
			return decodeTaggedFields(strings.TrimPrefix(secret.Content, pregnancyArcPrefix))
		}
	}
	return nil
}

func stiTransmissionChance(stiType entities.STIType) float32 {
	switch stiType {
	case entities.STIHIV:
		return 0.22
	case entities.STISyphilis:
		return 0.34
	default:
		return 0.42
	}
}

func incubationDaysForSTI(stiType entities.STIType) int {
	switch stiType {
	case entities.STIHIV:
		return 6
	case entities.STISyphilis:
		return 4
	default:
		return 2
	}
}

func storeSTISecret(poble *entities.Poble, stiType entities.STIType, incubationDays int, now entities.GameTime) {
	if poble == nil {
		return
	}
	content := fmt.Sprintf("%stype=%s|reveal_at=%d", stiSecretPrefix, stiType, now.Add(incubationDays*24).ToMinutes())
	poble.Secrets = append(poble.Secrets, entities.NewSecret(
		lifecycleID("sti_secret", now, poble.ID, stiType.String()),
		entities.SecretChild,
		content,
	))
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

func supportAvailableFor(poble *entities.Poble, world World) bool {
	if poble == nil || world == nil {
		return false
	}
	if poble.Mental.TherapyLevel >= 35 {
		return true
	}
	ws := world.GetWorldState()
	if ws.TechTree.Unlocked["MEDICINE"] || ws.TechTree.Unlocked["BASIC_MEDICINE"] {
		return true
	}
	for targetID, relationship := range poble.Relationships {
		if relationship.Trust < 65 || relationship.Affection < 55 {
			continue
		}
		target := world.GetPoble(targetID)
		if target != nil && target.Mental.TherapyLevel >= 20 {
			return true
		}
	}
	return false
}

func isChildOf(candidate *entities.Poble, parentID string) bool {
	if candidate == nil || parentID == "" {
		return false
	}
	return candidate.Parents[0] == parentID || candidate.Parents[1] == parentID
}

func otherParentMissing(candidate *entities.Poble, deceasedID string, world World) bool {
	if candidate == nil || world == nil {
		return false
	}
	for _, parentID := range candidate.Parents {
		if parentID == "" || parentID == deceasedID {
			continue
		}
		return world.GetPoble(parentID) == nil
	}
	return true
}

func isLikelyHeir(candidate *entities.Poble, deceasedID string) bool {
	if candidate == nil {
		return false
	}
	relationship, ok := candidate.Relationships[deceasedID]
	if !ok {
		return false
	}
	return relationship.Type == entities.RelationshipSpouse ||
		relationship.Type == entities.RelationshipChild ||
		relationship.Type == entities.RelationshipParent ||
		relationship.Affection >= 70
}

func newLifeMemory(prefix string, now entities.GameTime, targetID string, tag string) entities.Memory {
	return entities.Memory{
		ID:               lifecycleID(prefix+"_memory", now, targetID),
		Timestamp:        now,
		Type:             entities.MemoryPositive,
		Participants:     []string{targetID},
		EmotionIntensity: 64,
		Tags:             []string{tag},
		Summary:          "Life changed shape in a way the settlement will keep feeling.",
	}
}

func appendMemory(poble *entities.Poble, memory entities.Memory) {
	if poble == nil || memory.ID == "" {
		return
	}
	poble.Memories = append(poble.Memories, memory)
}

func lifecycleID(prefix string, now entities.GameTime, parts ...string) string {
	items := []string{prefix, fmt.Sprintf("%d", now.ToMinutes())}
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			items = append(items, part)
		}
	}
	return strings.Join(items, ":")
}

func lifecycleRNG(now entities.GameTime, parts ...string) *rand.Rand {
	seed := int64(now.ToMinutes() + 1)
	for _, part := range parts {
		for _, r := range part {
			seed = (seed * 33) + int64(r)
		}
	}
	if seed == 0 {
		seed = time.Now().UnixNano()
	}
	return rand.New(rand.NewSource(seed))
}

func currentWorldTime(world World) entities.GameTime {
	if world == nil {
		return entities.NewGameTime(0, 0, 0)
	}
	return world.GetWorldState().Day
}

func decodeTaggedFields(raw string) map[string]string {
	fields := map[string]string{}
	for _, part := range strings.Split(raw, "|") {
		key, value, ok := strings.Cut(part, "=")
		if ok && strings.TrimSpace(key) != "" {
			fields[strings.TrimSpace(key)] = strings.TrimSpace(value)
		}
	}
	return fields
}

func containsMentalConditionLocal(values []entities.MentalCondition, target entities.MentalCondition) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func clampSignedUnitLocal(value float32) float32 {
	if value < -1 {
		return -1
	}
	if value > 1 {
		return 1
	}
	return value
}
