package ai

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
	"strings"

	"github.com/user/poblation/internal/entities"
)

// Memory reuses the core memory type while keeping AI signatures local.
type Memory = entities.Memory

// MemoryType reuses the core memory category type.
type MemoryType = entities.MemoryType

const (
	defaultMaxMemories  = 150
	repressedStressGate = 80.0
)

// MemoryQuery filters episodic recall.
type MemoryQuery struct {
	AboutPersonID string
	Types         []MemoryType
	MinIntensity  float32
	MaxResults    int
	PreferRecent  bool
}

type scoredMemory struct {
	memory Memory
	score  float32
}

// MemorySystem stores and recalls episodic memories for one Poble.
type MemorySystem struct {
	poblID      string
	memories    []*Memory
	maxMemories int
	rng         *rand.Rand

	poble             *entities.Poble
	currentTime       entities.GameTime
	emotionalStress   float32
	hasStressOverride bool
}

// NewMemorySystem builds an episodic memory system for one Poble.
func NewMemorySystem(poble *entities.Poble, rng *rand.Rand) *MemorySystem {
	system := &MemorySystem{
		maxMemories: defaultMaxMemories,
		rng:         rng,
		poble:       poble,
	}

	if poble == nil {
		return system
	}

	system.poblID = poble.ID
	system.currentTime = entities.NewGameTime(0, 0, 0)
	for i := range poble.Memories {
		copied := poble.Memories[i]
		system.memories = append(system.memories, &copied)
		if copied.Timestamp.ToMinutes() > system.currentTime.ToMinutes() {
			system.currentTime = copied.Timestamp
		}
	}

	return system
}

// SetEmotionalStress lets other systems expose the current stress load directly.
func (s *MemorySystem) SetEmotionalStress(stress float32) {
	if s == nil {
		return
	}
	s.emotionalStress = clampPercent(stress)
	s.hasStressOverride = true
}

// AddMemory stores a new episodic memory, merging or replacing when needed.
func (s *MemorySystem) AddMemory(m Memory) {
	if s == nil || !m.IsValid() {
		return
	}

	m.Participants = uniqueSortedStrings(m.Participants)
	m.Tags = uniqueSortedStrings(m.Tags)
	if m.Timestamp.ToMinutes() > s.currentTime.ToMinutes() {
		s.currentTime = m.Timestamp
	}

	if similarIndex := s.findSimilarMemory(m); similarIndex >= 0 {
		s.mergeMemory(s.memories[similarIndex], m)
		s.syncToPoble()
		return
	}

	if s.maxMemories <= 0 {
		s.maxMemories = defaultMaxMemories
	}

	if len(s.memories) < s.maxMemories {
		copied := m
		s.memories = append(s.memories, &copied)
		s.syncToPoble()
		return
	}

	replacementIndex := s.lowestReplaceableMemoryIndex()
	if replacementIndex < 0 {
		return
	}

	copied := m
	s.memories[replacementIndex] = &copied
	s.syncToPoble()
}

// Recall returns memories most likely to surface for the current context.
func (s *MemorySystem) Recall(query MemoryQuery) []Memory {
	if s == nil || len(s.memories) == 0 {
		return nil
	}

	stress := s.currentStress()
	typeFilter := make(map[MemoryType]struct{}, len(query.Types))
	for _, memoryType := range query.Types {
		typeFilter[memoryType] = struct{}{}
	}

	matches := make([]scoredMemory, 0, len(s.memories))
	for _, stored := range s.memories {
		if stored == nil {
			continue
		}

		memory := *stored
		if query.AboutPersonID != "" && !containsString(memory.Participants, query.AboutPersonID) {
			continue
		}
		if len(typeFilter) > 0 {
			if _, ok := typeFilter[memory.Type]; !ok {
				continue
			}
		}
		if memory.IsRepressed && stress <= repressedStressGate {
			continue
		}

		ageFactor := s.ageFactor(memory)
		score := s.recallScore(memory, ageFactor, stress, query.PreferRecent)
		recalled := s.applyRecallDecay(memory, ageFactor)
		if recalled.EmotionIntensity < clampPercent(query.MinIntensity) || score <= 0 {
			continue
		}

		matches = append(matches, scoredMemory{
			memory: recalled,
			score:  score,
		})
	}

	sort.SliceStable(matches, func(i, j int) bool {
		if matches[i].score == matches[j].score {
			return matches[i].memory.Timestamp.ToMinutes() > matches[j].memory.Timestamp.ToMinutes()
		}
		return matches[i].score > matches[j].score
	})

	limit := query.MaxResults
	if limit <= 0 || limit > len(matches) {
		limit = len(matches)
	}

	result := make([]Memory, 0, limit)
	for i := 0; i < limit; i++ {
		result = append(result, matches[i].memory)
	}

	return result
}

// GetStrongestMemoryAbout returns the dominant memory about one target.
func (s *MemorySystem) GetStrongestMemoryAbout(targetID string) *Memory {
	if s == nil || targetID == "" {
		return nil
	}

	stress := s.currentStress()
	var best *Memory
	bestScore := float32(-1)
	for _, stored := range s.memories {
		if stored == nil || !containsString(stored.Participants, targetID) {
			continue
		}
		if stored.IsRepressed && stress <= repressedStressGate {
			continue
		}

		score := stored.EmotionIntensity * (0.70 + (s.ageFactor(*stored) * 0.30))
		if score > bestScore {
			copied := *stored
			best = &copied
			bestScore = score
		}
	}

	return best
}

// HasTrauma reports whether this Poble carries any traumatic memory burden.
func (s *MemorySystem) HasTrauma() bool {
	if s == nil {
		return false
	}

	for _, stored := range s.memories {
		if stored != nil && s.isTraumaticMemory(*stored) {
			return true
		}
	}

	return s.poble != nil && len(s.poble.Mental.Traumas) > 0
}

// GetActiveTraumas returns traumatic memories that are currently in play.
func (s *MemorySystem) GetActiveTraumas() []Memory {
	if s == nil {
		return nil
	}

	stress := s.currentStress()
	active := make([]Memory, 0, len(s.memories))
	for _, stored := range s.memories {
		if stored == nil || !s.isTraumaticMemory(*stored) {
			continue
		}
		if stored.IsRepressed && stress <= repressedStressGate {
			continue
		}

		active = append(active, s.applyRecallDecay(*stored, s.ageFactor(*stored)))
	}

	sort.SliceStable(active, func(i, j int) bool {
		return active[i].EmotionIntensity > active[j].EmotionIntensity
	})

	return active
}

// ShouldBringUpPastWith decides whether old baggage spills into a current exchange.
func (s *MemorySystem) ShouldBringUpPastWith(targetID string) bool {
	if s == nil || targetID == "" {
		return false
	}

	strongest := s.GetStrongestMemoryAbout(targetID)
	if strongest == nil {
		return false
	}

	neuroticism := float32(35)
	resentment := float32(0)
	if s.poble != nil {
		neuroticism = s.poble.Personality.Neuroticism
		if relationship, ok := s.poble.Relationships[targetID]; ok {
			resentment = relationship.Resentment
		}
	}

	unresolved := s.unresolvedIntensity(targetID)
	stress := s.currentStress()
	chance := 0.08 +
		(neuroticism / 360.0) +
		(resentment / 240.0) +
		(unresolved / 220.0) +
		(strongest.EmotionIntensity / 320.0)

	if isNegativeMemoryType(strongest.Type) || s.isTraumaticMemory(*strongest) {
		chance += 0.14
	}
	if stress > repressedStressGate {
		chance += 0.10
	}
	if isPositiveMemoryType(strongest.Type) && resentment < 25 {
		chance -= 0.18
	}

	chance = clampRange(chance, 0.02, 0.96)
	if s.rng == nil {
		return chance >= 0.50
	}

	return s.rng.Float32() < chance
}

// ProcessNewEvent creates a subjective memory from one world event.
func (s *MemorySystem) ProcessNewEvent(event GameEvent, thisPobleID string) {
	if s == nil {
		return
	}

	if thisPobleID == "" {
		thisPobleID = s.poblID
	}
	if thisPobleID == "" {
		return
	}

	if event.Time.IsValid() {
		s.currentTime = event.Time
	}

	participants := uniqueSortedStrings(eventParticipants(event))
	targetID := event.TargetID
	if targetID == "" {
		targetID = event.PrimaryActor
	}
	if targetID == thisPobleID {
		targetID = s.otherParticipant(participants, thisPobleID)
	}

	memoryType := s.subjectiveMemoryType(event, thisPobleID)
	intensity := s.subjectiveIntensity(event, thisPobleID, targetID)
	summary := s.subjectiveSummary(event, thisPobleID)
	if summary == "" {
		summary = "Algo paso y dejo una marca."
	}

	memoryID := subjectiveMemoryID(event, thisPobleID)
	memory := entities.NewMemory(memoryID, event.Time, memoryType, summary)
	memory.Participants = participants
	memory.EmotionIntensity = intensity
	memory.Tags = uniqueSortedStrings(append([]string{
		"event:" + fallbackEventID(event),
		"event_type:" + string(event.Type),
		"perspective:" + roleInEvent(event, thisPobleID),
	}, event.Tags...))
	memory.IsRepressed = s.shouldRepress(memory)

	s.AddMemory(memory)
	s.rememberTrauma(memory)
}

// GetRelationshipNarrative returns a short history summary for template use.
func (s *MemorySystem) GetRelationshipNarrative(targetID string) string {
	if s == nil || targetID == "" {
		return ""
	}

	memories := s.Recall(MemoryQuery{
		AboutPersonID: targetID,
		MaxResults:    6,
	})
	if len(memories) == 0 && s.poble == nil {
		return ""
	}

	positive := strongestMemoryByKind(memories, isPositiveMemoryType)
	negative := strongestMemoryByKind(memories, isNegativeMemoryType)

	parts := []string{}
	switch {
	case positive != nil && (positive.Type == entities.MemoryRomantic || positive.Type == entities.MemoryErotic):
		parts = append(parts, "Hubo deseo y cercania entre ellos.")
	case positive != nil:
		parts = append(parts, "Antes hubo confianza entre ellos.")
	case s.hasWarmRelationship(targetID):
		parts = append(parts, "Su historia comenzo con cierta cercania.")
	}

	if negative != nil {
		parts = append(parts, fmt.Sprintf("Luego paso %s.", summarizeMemoryFragment(negative.Summary)))
	}

	nowClause := s.currentRelationshipClause(targetID, positive, negative)
	if nowClause != "" {
		parts = append(parts, nowClause)
	}

	return strings.Join(parts, " ")
}

// EmotionalDecay softens memory charge over time and may surface locked trauma.
func (s *MemorySystem) EmotionalDecay(deltaHours int) {
	if s == nil || deltaHours <= 0 {
		return
	}

	s.currentTime = s.currentTime.Add(deltaHours)
	stress := s.currentStress()
	hours := float32(deltaHours)

	for _, stored := range s.memories {
		if stored == nil {
			continue
		}

		stored.EmotionIntensity = clampPercent(stored.EmotionIntensity - (s.decayRate(*stored) * hours))
		if stored.IsRepressed && s.isTraumaticMemory(*stored) && stress > 85 {
			chance := clampRange(0.12+((stress-85)/100.0)+(stored.EmotionIntensity/400.0), 0.10, 0.90)
			if s.rng == nil && chance >= 0.35 {
				stored.IsRepressed = false
			}
			if s.rng != nil && s.rng.Float32() < chance {
				stored.IsRepressed = false
			}
		}
	}

	s.syncToPoble()
}

func (s *MemorySystem) findSimilarMemory(memory Memory) int {
	for index, stored := range s.memories {
		if stored == nil {
			continue
		}

		if stored.ID == memory.ID {
			return index
		}
		if sameEventTag(*stored, memory) && sameParticipants(stored.Participants, memory.Participants) {
			return index
		}
		if stored.Type == memory.Type &&
			sameParticipants(stored.Participants, memory.Participants) &&
			normalizeSummary(stored.Summary) == normalizeSummary(memory.Summary) {
			return index
		}
	}

	return -1
}

func (s *MemorySystem) mergeMemory(target *Memory, incoming Memory) {
	if target == nil {
		return
	}

	if incoming.Timestamp.ToMinutes() > target.Timestamp.ToMinutes() {
		target.Timestamp = incoming.Timestamp
	}
	target.EmotionIntensity = clampPercent(
		maxFloat32(target.EmotionIntensity, incoming.EmotionIntensity) +
			minFloat32(8, incoming.EmotionIntensity*0.12),
	)
	if incoming.Summary != "" && (len(incoming.Summary) > len(target.Summary) || incoming.EmotionIntensity >= target.EmotionIntensity) {
		target.Summary = incoming.Summary
	}
	if s.isTraumaticMemory(incoming) && !s.isTraumaticMemory(*target) {
		target.Type = incoming.Type
	}
	target.IsRepressed = target.IsRepressed && incoming.IsRepressed
	target.Participants = uniqueSortedStrings(append(target.Participants, incoming.Participants...))
	target.Tags = uniqueSortedStrings(append(target.Tags, incoming.Tags...))
}

func (s *MemorySystem) lowestReplaceableMemoryIndex() int {
	lowestIndex := -1
	lowestIntensity := float32(101)
	for index, stored := range s.memories {
		if stored == nil || s.isTraumaticMemory(*stored) {
			continue
		}
		if stored.EmotionIntensity < lowestIntensity {
			lowestIndex = index
			lowestIntensity = stored.EmotionIntensity
		}
	}

	return lowestIndex
}

func (s *MemorySystem) recallScore(memory Memory, ageFactor, stress float32, preferRecent bool) float32 {
	score := (memory.EmotionIntensity / 100.0) * 0.58
	score += ageFactor * 0.22
	if preferRecent {
		score += ageFactor * 0.20
	} else {
		score += (memory.EmotionIntensity / 100.0) * 0.10
	}

	if memory.IsRepressed && stress > repressedStressGate {
		score += 0.16 + ((stress - repressedStressGate) / 200.0)
	}
	if s.isTraumaticMemory(memory) {
		score += 0.08
	}
	if s.rng != nil {
		score += s.rng.Float32() * 0.03
	}

	return score
}

func (s *MemorySystem) applyRecallDecay(memory Memory, ageFactor float32) Memory {
	recalled := memory
	recalled.EmotionIntensity = clampPercent(memory.EmotionIntensity * (0.45 + (ageFactor * 0.55)))
	recalled.Summary = summarizeByAge(memory.Summary, ageFactor)
	return recalled
}

func (s *MemorySystem) ageFactor(memory Memory) float32 {
	ageHours := s.ageHours(memory.Timestamp)
	agePenalty := float32(ageHours) / (24.0 * 120.0)
	factor := 1.0 / (1.0 + agePenalty)

	switch memory.Type {
	case entities.MemoryPositive, entities.MemoryFunny, entities.MemoryRomantic, entities.MemoryAchievement:
		factor -= 0.08
	case entities.MemoryNegative, entities.MemoryEmbarrassing, entities.MemoryViolent, entities.MemoryBetrayal:
		factor += 0.06
	case entities.MemoryTraumatic:
		factor += 0.25
	}
	if memory.IsRepressed {
		factor -= 0.05
	}

	return clampRange(factor, 0.12, 1.0)
}

func (s *MemorySystem) ageHours(timestamp entities.GameTime) int {
	if !timestamp.IsValid() {
		return 0
	}

	now := s.currentTime
	if !now.IsValid() || now.ToMinutes() < timestamp.ToMinutes() {
		now = timestamp
	}

	age := now.Diff(timestamp)
	if age < 0 {
		return 0
	}
	return age
}

func (s *MemorySystem) currentStress() float32 {
	if s == nil {
		return 0
	}
	if s.hasStressOverride {
		return s.emotionalStress
	}
	if s.poble == nil {
		return 0
	}

	stress := float32(0)
	stress += clampPercent(float32(math.Abs(float64(s.poble.EmotionalState.Arousal))) * 42.0)
	stress += clampPercent(maxFloat32(0, -s.poble.EmotionalState.Valence) * 35.0)
	stress += clampPercent(float32(100-s.poble.Mental.Stability) * 0.22)
	stress += clampPercent(s.poble.Needs.Safety * 0.18)
	stress += clampPercent(s.poble.Needs.Belonging * 0.06)

	for _, emotion := range s.poble.EmotionalState.ActiveEmotions {
		switch emotion {
		case entities.EmotionAnxiety, entities.EmotionFear, entities.EmotionGrief, entities.EmotionResentment, entities.EmotionAnger:
			stress += 7
		}
	}

	return clampPercent(stress)
}

func (s *MemorySystem) subjectiveMemoryType(event GameEvent, thisPobleID string) MemoryType {
	if event.IsTraumatic {
		return entities.MemoryTraumatic
	}

	switch event.Type {
	case GameEventBetrayal:
		return entities.MemoryBetrayal
	case GameEventThreat:
		if event.Severity >= 60 {
			return entities.MemoryViolent
		}
		return entities.MemoryNegative
	case GameEventConflict, GameEventSocialNegative:
		if event.Severity >= 70 {
			return entities.MemoryEmbarrassing
		}
		return entities.MemoryNegative
	case GameEventSocialPositive:
		return entities.MemoryPositive
	case GameEventGoalComplete:
		return entities.MemoryAchievement
	case GameEventIntimacy:
		if event.Valence >= 0 {
			return entities.MemoryRomantic
		}
		return entities.MemoryErotic
	case GameEventDeath:
		return entities.MemoryTraumatic
	default:
		if containsString(event.Tags, "funny") {
			return entities.MemoryFunny
		}
		if event.Valence >= 0 {
			return entities.MemoryPositive
		}
		return entities.MemoryNegative
	}
}

func (s *MemorySystem) subjectiveIntensity(event GameEvent, thisPobleID, targetID string) float32 {
	intensity := maxFloat32(12, clampPercent(event.Severity))
	intensity += clampPercent(float32(math.Abs(float64(clampSignedUnit(event.Valence)))) * 18.0)

	switch roleInEvent(event, thisPobleID) {
	case "target":
		intensity += 18
	case "actor":
		intensity += 8
	case "witness":
		intensity -= 8
	default:
		intensity += 4
	}

	if s.poble != nil {
		intensity += (s.poble.Personality.Neuroticism - 50.0) * 0.18
		if relationship, ok := s.poble.Relationships[targetID]; ok {
			intensity += (relationship.Resentment * 0.18)
			intensity += (relationship.Affection * 0.10)
			intensity += (relationship.Trust * 0.08)
		}
	}
	if event.IsTraumatic {
		intensity = maxFloat32(intensity, 72)
	}

	return clampPercent(intensity)
}

func (s *MemorySystem) subjectiveSummary(event GameEvent, thisPobleID string) string {
	otherID := event.TargetID
	if otherID == "" || otherID == thisPobleID {
		otherID = event.PrimaryActor
	}
	if otherID == "" || otherID == thisPobleID {
		otherID = s.otherParticipant(eventParticipants(event), thisPobleID)
	}
	if otherID == "" {
		otherID = "alguien"
	}

	role := roleInEvent(event, thisPobleID)
	switch event.Type {
	case GameEventBetrayal:
		if role == "target" {
			return fmt.Sprintf("%s me fallo cuando mas importaba.", otherID)
		}
		if role == "actor" {
			return fmt.Sprintf("Cruce una linea con %s y eso dejo marca.", otherID)
		}
		return fmt.Sprintf("Vi la traicion entre %s y %s.", event.PrimaryActor, event.TargetID)
	case GameEventConflict:
		if role == "target" {
			return fmt.Sprintf("%s volvio la situacion personal contra mi.", otherID)
		}
		if role == "actor" {
			return fmt.Sprintf("La pelea con %s se me quedo pegada.", otherID)
		}
		return fmt.Sprintf("Vi el choque entre %s y %s.", event.PrimaryActor, otherID)
	case GameEventSocialPositive:
		if role == "target" || role == "actor" {
			return fmt.Sprintf("Con %s hubo un momento bueno y raro de olvidar.", otherID)
		}
		return fmt.Sprintf("Vi a %s acercarse a %s.", event.PrimaryActor, otherID)
	case GameEventSocialNegative:
		if role == "target" {
			return fmt.Sprintf("%s me hizo sentir pequeño.", otherID)
		}
		return fmt.Sprintf("Con %s quedo una punzada fea.", otherID)
	case GameEventIntimacy:
		if role == "target" || role == "actor" {
			return fmt.Sprintf("Con %s hubo una cercania que no se fue del todo.", otherID)
		}
		return fmt.Sprintf("Vi una cercania cargada entre %s y %s.", event.PrimaryActor, otherID)
	case GameEventGoalComplete:
		return "Logre algo importante y todavia me sostiene."
	case GameEventThreat:
		if role == "target" {
			return fmt.Sprintf("%s me hizo sentir en peligro real.", otherID)
		}
		return fmt.Sprintf("El peligro alrededor de %s me dejo en alerta.", otherID)
	case GameEventDeath:
		if role == "target" {
			return "La muerte casi me llevo y no se me borra."
		}
		return fmt.Sprintf("La muerte alrededor de %s me cambio algo por dentro.", otherID)
	default:
		if event.Description != "" {
			return event.Description
		}
		return "Algo del pasado sigue tirando de mi."
	}
}

func (s *MemorySystem) shouldRepress(memory Memory) bool {
	if !s.isTraumaticMemory(memory) {
		return false
	}
	if s.poble == nil {
		return memory.EmotionIntensity >= 92
	}

	trigger := memory.EmotionIntensity >= 85 &&
		(s.poble.Personality.Neuroticism >= 72 || s.poble.Mental.Stability <= 48)
	if !trigger {
		return false
	}
	if s.rng == nil {
		return true
	}

	return s.rng.Float32() < 0.65
}

func (s *MemorySystem) rememberTrauma(memory Memory) {
	if s.poble == nil || !s.isTraumaticMemory(memory) {
		return
	}

	if !containsString(s.poble.Mental.Traumas, memory.ID) {
		s.poble.Mental.Traumas = append(s.poble.Mental.Traumas, memory.ID)
	}
	if memory.EmotionIntensity >= 88 && !containsMentalCondition(s.poble.Mental.Conditions, entities.MentalPTSD) {
		s.poble.Mental.Conditions = append(s.poble.Mental.Conditions, entities.MentalPTSD)
	}
}

func (s *MemorySystem) isTraumaticMemory(memory Memory) bool {
	if memory.Type == entities.MemoryTraumatic {
		return true
	}
	if containsString(memory.Tags, "trauma") || containsString(memory.Tags, "event_type:DEATH") {
		return true
	}
	if s.poble != nil && containsString(s.poble.Mental.Traumas, memory.ID) {
		return true
	}
	return false
}

func (s *MemorySystem) unresolvedIntensity(targetID string) float32 {
	total := float32(0)
	for _, stored := range s.memories {
		if stored == nil || !containsString(stored.Participants, targetID) {
			continue
		}
		if !isNegativeMemoryType(stored.Type) && !s.isTraumaticMemory(*stored) {
			continue
		}
		if containsString(stored.Tags, "resolved") {
			continue
		}
		total += maxFloat32(12, stored.EmotionIntensity) / 3.5
	}
	return clampPercent(total)
}

func (s *MemorySystem) hasWarmRelationship(targetID string) bool {
	if s.poble == nil {
		return false
	}

	relationship, ok := s.poble.Relationships[targetID]
	if !ok {
		return false
	}

	return relationship.Affection >= 55 || relationship.Trust >= 60 ||
		relationship.Type == entities.RelationshipFriend ||
		relationship.Type == entities.RelationshipBestFriend ||
		relationship.Type == entities.RelationshipLover ||
		relationship.Type == entities.RelationshipSpouse
}

func (s *MemorySystem) currentRelationshipClause(targetID string, positive, negative *Memory) string {
	if s.poble == nil {
		if negative != nil {
			return "Ahora quedan cosas sin cerrar."
		}
		if positive != nil {
			return "Ahora esa historia todavia pesa."
		}
		return ""
	}

	relationship, ok := s.poble.Relationships[targetID]
	if !ok {
		if negative != nil {
			return "Ahora hay distancia y una herida abierta."
		}
		if positive != nil {
			return "Ahora queda una cercania fragil."
		}
		return ""
	}

	switch relationship.Type {
	case entities.RelationshipEnemy, entities.RelationshipBetrayer, entities.RelationshipRival:
		return "Ahora el resentimiento manda."
	case entities.RelationshipLover, entities.RelationshipSpouse, entities.RelationshipFriendsWithBenefits, entities.RelationshipCrush, entities.RelationshipObsession:
		if relationship.Resentment >= 45 && negative != nil {
			return "Ahora deseo y rencor se mezclan."
		}
		return "Ahora la tension sigue siendo intima."
	case entities.RelationshipFriend, entities.RelationshipBestFriend, entities.RelationshipAlly:
		if negative != nil && relationship.Resentment >= 35 {
			return "Ahora la confianza esta rota pero no del todo muerta."
		}
		return "Ahora todavia queda un lazo."
	case entities.RelationshipFamily, entities.RelationshipParent, entities.RelationshipChild, entities.RelationshipSibling:
		if relationship.Resentment >= 40 {
			return "Ahora la familia no alcanza para tapar el dolor."
		}
		return "Ahora los sigue atando algo profundo."
	}

	if relationship.Resentment >= 55 {
		return "Ahora quedan cuentas abiertas."
	}
	if relationship.Trust >= 65 || relationship.Affection >= 60 {
		return "Ahora hay una calma fragil entre ellos."
	}
	if negative != nil {
		return "Ahora el pasado se mete en medio."
	}
	if positive != nil {
		return "Ahora aun queda eco de lo bueno."
	}
	return ""
}

func (s *MemorySystem) decayRate(memory Memory) float32 {
	switch memory.Type {
	case entities.MemoryPositive, entities.MemoryFunny, entities.MemoryRomantic, entities.MemoryAchievement:
		return 0.10
	case entities.MemoryNegative, entities.MemoryEmbarrassing:
		return 0.06
	case entities.MemoryBetrayal, entities.MemoryViolent:
		return 0.04
	case entities.MemoryTraumatic:
		return 0.012
	default:
		return 0.07
	}
}

func (s *MemorySystem) otherParticipant(participants []string, selfID string) string {
	for _, participant := range participants {
		if participant != "" && participant != selfID {
			return participant
		}
	}
	return ""
}

func (s *MemorySystem) syncToPoble() {
	if s == nil || s.poble == nil {
		return
	}

	s.poble.Memories = make([]entities.Memory, 0, len(s.memories))
	for _, stored := range s.memories {
		if stored != nil {
			s.poble.Memories = append(s.poble.Memories, *stored)
		}
	}
}

func eventParticipants(event GameEvent) []string {
	participants := make([]string, 0, len(event.Participants)+2)
	if event.PrimaryActor != "" {
		participants = append(participants, event.PrimaryActor)
	}
	if event.TargetID != "" {
		participants = append(participants, event.TargetID)
	}
	participants = append(participants, event.Participants...)
	return participants
}

func fallbackEventID(event GameEvent) string {
	if event.ID != "" {
		return event.ID
	}
	return fmt.Sprintf("%s-%d", event.Type, event.Time.ToMinutes())
}

func subjectiveMemoryID(event GameEvent, thisPobleID string) string {
	return fmt.Sprintf("%s:%s", thisPobleID, fallbackEventID(event))
}

func sameParticipants(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	if len(left) == 0 {
		return true
	}

	leftCopy := uniqueSortedStrings(left)
	rightCopy := uniqueSortedStrings(right)
	for i := range leftCopy {
		if leftCopy[i] != rightCopy[i] {
			return false
		}
	}
	return true
}

func sameEventTag(left, right Memory) bool {
	leftTag := extractEventTag(left.Tags)
	rightTag := extractEventTag(right.Tags)
	return leftTag != "" && leftTag == rightTag
}

func extractEventTag(tags []string) string {
	for _, tag := range tags {
		if strings.HasPrefix(tag, "event:") {
			return tag
		}
	}
	return ""
}

func normalizeSummary(summary string) string {
	summary = strings.ToLower(strings.TrimSpace(summary))
	replacer := strings.NewReplacer(",", "", ".", "", ";", "", ":", "", "!", "", "?", "")
	return replacer.Replace(summary)
}

func uniqueSortedStrings(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}

	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
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

func roleInEvent(event GameEvent, thisPobleID string) string {
	switch {
	case event.TargetID == thisPobleID:
		return "target"
	case event.PrimaryActor == thisPobleID:
		return "actor"
	case containsString(event.Participants, thisPobleID):
		return "participant"
	default:
		return "witness"
	}
}

func strongestMemoryByKind(memories []Memory, matcher func(MemoryType) bool) *Memory {
	var best *Memory
	bestIntensity := float32(-1)
	for i := range memories {
		if !matcher(memories[i].Type) {
			continue
		}
		if memories[i].EmotionIntensity > bestIntensity {
			copied := memories[i]
			best = &copied
			bestIntensity = memories[i].EmotionIntensity
		}
	}
	return best
}

func isPositiveMemoryType(memoryType MemoryType) bool {
	switch memoryType {
	case entities.MemoryPositive, entities.MemoryFunny, entities.MemoryRomantic, entities.MemoryErotic, entities.MemoryAchievement:
		return true
	default:
		return false
	}
}

func isNegativeMemoryType(memoryType MemoryType) bool {
	switch memoryType {
	case entities.MemoryNegative, entities.MemoryTraumatic, entities.MemoryEmbarrassing, entities.MemoryViolent, entities.MemoryBetrayal:
		return true
	default:
		return false
	}
}

func summarizeByAge(summary string, ageFactor float32) string {
	if summary == "" || ageFactor >= 0.75 {
		return summary
	}

	words := strings.Fields(summary)
	if len(words) <= 6 {
		return summary
	}

	keep := 8
	if ageFactor < 0.45 {
		keep = 5
	}
	if keep > len(words) {
		keep = len(words)
	}

	return strings.Join(words[:keep], " ") + "..."
}

func summarizeMemoryFragment(summary string) string {
	summary = strings.TrimSpace(summary)
	summary = strings.TrimSuffix(summary, ".")
	return strings.ToLower(summary)
}

func containsMentalCondition(values []entities.MentalCondition, target entities.MentalCondition) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
