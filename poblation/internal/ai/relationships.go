package ai

import (
	"sort"
	"strings"
	"sync"

	"github.com/user/poblation/internal/entities"
)

// RelationshipType reuses the entity-level relationship contract.
type RelationshipType = entities.RelationshipType

const (
	RelationshipNemesis          = entities.RelationshipNemesis
	RelationshipToxicAttraction  = entities.RelationshipToxicAttraction
	RelationshipCodependent      = entities.RelationshipCodependent
	RelationshipSecretObsession  = entities.RelationshipSecretObsession
	RelationshipComplicated      = entities.RelationshipComplicated
	relationshipPublicObserverID = "__public__"
)

// RelationshipEventType identifies why a relationship changed.
type RelationshipEventType string

const (
	RelationshipEventGeneric              RelationshipEventType = "GENERIC"
	RelationshipEventKindness             RelationshipEventType = "KINDNESS"
	RelationshipEventConflict             RelationshipEventType = "CONFLICT"
	RelationshipEventIntimacy             RelationshipEventType = "INTIMACY"
	RelationshipEventBetrayal             RelationshipEventType = "BETRAYAL"
	RelationshipEventNeglect              RelationshipEventType = "NEGLECT"
	RelationshipEventSaturation           RelationshipEventType = "SATURATION"
	RelationshipEventInfidelityDiscovered RelationshipEventType = "INFIDELITY_DISCOVERED"
)

// RelationshipEvent carries one relationship change request.
type RelationshipEvent struct {
	Type             RelationshipEventType `json:"type"`
	TargetID         string                `json:"target_id"`
	ActorID          string                `json:"actor_id"`
	ThirdPartyID     string                `json:"third_party_id"`
	TrustDelta       float32               `json:"trust_delta"`
	AttractionDelta  float32               `json:"attraction_delta"`
	RespectDelta     float32               `json:"respect_delta"`
	ResentmentDelta  float32               `json:"resentment_delta"`
	IsPublic         bool                  `json:"is_public"`
	Time             entities.GameTime     `json:"time"`
	Tags             []string              `json:"tags"`
	DiscoveredSecret string                `json:"discovered_secret"`
}

// RelationshipManager owns directed relationship state for one Poble.
type RelationshipManager struct {
	mu               sync.RWMutex
	poble            *entities.Poble
	publicPerception map[string]map[string]entities.RelationshipType
	contactLog       map[string][]entities.GameTime
}

// NewRelationshipManager builds a manager around one Poble's relationship map.
func NewRelationshipManager(poble *entities.Poble) *RelationshipManager {
	manager := &RelationshipManager{
		poble:            poble,
		publicPerception: map[string]map[string]entities.RelationshipType{},
		contactLog:       map[string][]entities.GameTime{},
	}
	if poble != nil && poble.Relationships == nil {
		poble.Relationships = map[string]entities.Relationship{}
	}
	return manager
}

// UpdateRelationship applies one event while preserving contradictory feelings.
func (m *RelationshipManager) UpdateRelationship(targetID string, event RelationshipEvent) {
	if m == nil || m.poble == nil || targetID == "" {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	event.TargetID = targetID
	relationship := m.relationshipLocked(targetID)
	trust, attraction, respect, resentment := m.relationshipDeltas(targetID, event)

	relationship.Trust = clampPercent(relationship.Trust + trust)
	relationship.Attraction = clampPercent(relationship.Attraction + attraction)
	relationship.Respect = clampPercent(relationship.Respect + respect)
	relationship.Resentment = clampPercent(relationship.Resentment + resentment)
	relationship.Affection = clampPercent(relationship.Affection + affectionDelta(trust, attraction, resentment))
	relationship.Familiarity = clampPercent(relationship.Familiarity + familiarityDelta(event))

	if event.Time.IsValid() {
		relationship.LastInteraction = event.Time
		m.recordContactLocked(targetID, event.Time)
	}

	relationship.Tags = updatedRelationshipTags(relationship.Tags, event)
	inferred := m.inferRelationshipTypeLocked(relationship)
	if shouldStoreRelationshipType(relationship.Type, inferred) {
		relationship.Type = inferred
	}

	m.poble.Relationships[targetID] = relationship
	if event.IsPublic {
		m.setPublicPerceptionLocked(targetID, relationshipPublicObserverID, relationship.Type)
	}
}

// GetRelationship returns a copy of the current relationship.
func (m *RelationshipManager) GetRelationship(targetID string) (entities.Relationship, bool) {
	if m == nil || m.poble == nil || targetID == "" {
		return entities.Relationship{}, false
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	relationship, ok := m.poble.Relationships[targetID]
	return relationship, ok
}

// GetRelationshipType infers the live type from current relationship values.
func (m *RelationshipManager) GetRelationshipType(targetID string) entities.RelationshipType {
	if m == nil || m.poble == nil || targetID == "" {
		return entities.RelationshipStranger
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	relationship, ok := m.poble.Relationships[targetID]
	if !ok {
		return entities.RelationshipStranger
	}
	return m.inferRelationshipTypeLocked(relationship)
}

// RecordContact tracks repeated contact without changing relationship values.
func (m *RelationshipManager) RecordContact(targetID string, at entities.GameTime) {
	if m == nil || targetID == "" || !at.IsValid() {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.recordContactLocked(targetID, at)
}

// DetectRelationshipEvent finds the next relationship pressure that should fire.
func (m *RelationshipManager) DetectRelationshipEvent(currentTime entities.GameTime) *RelationshipEvent {
	if m == nil || m.poble == nil || !currentTime.IsValid() {
		return nil
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	if event := m.detectInfidelityLocked(currentTime); event != nil {
		return event
	}
	if event := m.detectSaturationLocked(currentTime); event != nil {
		return event
	}
	return m.detectNeglectLocked(currentTime)
}

// SetPublicPerception records what one observer believes about a relationship.
func (m *RelationshipManager) SetPublicPerception(targetID, observerID string, relationshipType entities.RelationshipType) {
	if m == nil || targetID == "" || observerID == "" || !relationshipType.IsValid() {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.setPublicPerceptionLocked(targetID, observerID, relationshipType)
}

// GetPublicPerception keeps the prompt-level signature for the most visible bond.
func (m *RelationshipManager) GetPublicPerception(observerID string) entities.RelationshipType {
	if m == nil {
		return entities.RelationshipStranger
	}
	return m.GetPublicPerceptionFor(m.defaultPublicTargetID(), observerID)
}

// GetPublicPerceptionFor returns what an observer thinks a specific bond is.
func (m *RelationshipManager) GetPublicPerceptionFor(targetID, observerID string) entities.RelationshipType {
	if m == nil || targetID == "" {
		return entities.RelationshipStranger
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	if observerID != "" {
		if perceived, ok := m.perceptionLocked(targetID, observerID); ok {
			return perceived
		}
	}
	if perceived, ok := m.perceptionLocked(targetID, relationshipPublicObserverID); ok {
		return perceived
	}
	return m.derivePublicPerceptionLocked(targetID)
}

func (m *RelationshipManager) relationshipLocked(targetID string) entities.Relationship {
	relationship, ok := m.poble.Relationships[targetID]
	if ok {
		return relationship
	}
	return entities.NewRelationship(targetID, entities.RelationshipStranger)
}

func (m *RelationshipManager) relationshipDeltas(targetID string, event RelationshipEvent) (float32, float32, float32, float32) {
	amplifier := m.thirdPartyNegativeAmplifier(targetID, event)
	trust := scaleNegative(event.TrustDelta, amplifier)
	respect := scaleNegative(event.RespectDelta, amplifier)
	resentment := scalePositive(event.ResentmentDelta, amplifier)

	if trust < 0 {
		resentment += -trust * 0.42
	}
	if trust > 0 {
		resentment -= trust * 0.28
	}
	if resentment < 0 {
		trust += -resentment * 0.12
	}
	if respect < 0 {
		resentment += -respect * 0.18
	}

	return trust, event.AttractionDelta, respect, resentment
}

func (m *RelationshipManager) thirdPartyNegativeAmplifier(targetID string, event RelationshipEvent) float32 {
	if m.poble == nil || !relationshipEventHasThirdParty(m.poble.ID, targetID, event) {
		return 1
	}
	return 1 + ((m.poble.Personality.Jealousy / 100.0) * 0.75)
}

func (m *RelationshipManager) inferRelationshipTypeLocked(relationship entities.Relationship) entities.RelationshipType {
	if relationship.Resentment >= 90 {
		return entities.RelationshipNemesis
	}
	if relationship.Resentment >= 70 && relationship.Attraction >= 40 {
		return entities.RelationshipToxicAttraction
	}
	if isSecretObsession(relationship) {
		return entities.RelationshipSecretObsession
	}
	if m.isCodependentLocked(relationship) {
		return entities.RelationshipCodependent
	}
	if relationship.Trust >= 80 && relationship.Attraction >= 80 {
		return entities.RelationshipLover
	}
	if signedTrust(relationship) <= -50 || relationship.Resentment >= 78 {
		return entities.RelationshipEnemy
	}
	if relationship.Trust >= 82 && relationship.Respect >= 70 && relationship.Affection >= 65 {
		return entities.RelationshipBestFriend
	}
	if relationship.Trust >= 62 && relationship.Respect >= 45 && relationship.Resentment < 45 {
		return entities.RelationshipFriend
	}
	if relationship.Attraction >= 65 && relationship.Resentment < 45 {
		return entities.RelationshipCrush
	}
	if relationship.Respect >= 60 && relationship.Resentment >= 45 {
		return entities.RelationshipRival
	}
	if isContradictoryRelationship(relationship) {
		return entities.RelationshipComplicated
	}
	if relationship.Familiarity < 20 && relationship.Trust <= 55 && relationship.Attraction < 25 {
		return entities.RelationshipStranger
	}
	return entities.RelationshipAcquaintance
}

func (m *RelationshipManager) isCodependentLocked(relationship entities.Relationship) bool {
	if m.poble == nil {
		return false
	}
	return m.poble.Needs.Belonging >= 78 &&
		relationship.Dependency >= 70 &&
		(relationship.Trust >= 50 || relationship.Affection >= 50 || relationship.Attraction >= 55)
}

func (m *RelationshipManager) detectInfidelityLocked(currentTime entities.GameTime) *RelationshipEvent {
	for _, targetID := range m.sortedRelationshipIDsLocked() {
		relationship := m.poble.Relationships[targetID]
		if !hasTag(relationship.Tags, "discovered_infidelity") || hasTag(relationship.Tags, "handled_infidelity") {
			continue
		}
		return &RelationshipEvent{
			Type:            RelationshipEventInfidelityDiscovered,
			TargetID:        targetID,
			TrustDelta:      -70,
			AttractionDelta: -15,
			RespectDelta:    -35,
			ResentmentDelta: 65,
			Time:            currentTime,
			Tags:            []string{"betrayal", "infidelity"},
		}
	}
	return nil
}

func (m *RelationshipManager) detectSaturationLocked(currentTime entities.GameTime) *RelationshipEvent {
	if m.poble.Personality.Extraversion >= 58 {
		return nil
	}

	limit := 4
	if m.poble.Personality.Extraversion < 32 {
		limit = 3
	}
	for targetID, contacts := range m.contactLog {
		if recentContactCount(contacts, currentTime, 24) < limit {
			continue
		}
		return &RelationshipEvent{
			Type:            RelationshipEventSaturation,
			TargetID:        targetID,
			TrustDelta:      -3,
			AttractionDelta: -6,
			RespectDelta:    -4,
			ResentmentDelta: 14,
			Time:            currentTime,
			Tags:            []string{"introvert_saturation"},
		}
	}
	return nil
}

func (m *RelationshipManager) detectNeglectLocked(currentTime entities.GameTime) *RelationshipEvent {
	bestTarget := ""
	bestPain := float32(0)
	for _, targetID := range m.sortedRelationshipIDsLocked() {
		relationship := m.poble.Relationships[targetID]
		hours := currentTime.Diff(relationship.LastInteraction)
		pain := neglectPain(m.poble.Needs.Belonging, relationship, hours)
		if pain > bestPain {
			bestTarget = targetID
			bestPain = pain
		}
	}
	if bestTarget == "" || bestPain < 20 {
		return nil
	}
	return &RelationshipEvent{
		Type:            RelationshipEventNeglect,
		TargetID:        bestTarget,
		TrustDelta:      -(bestPain * 0.18),
		AttractionDelta: -(bestPain * 0.04),
		RespectDelta:    -(bestPain * 0.08),
		ResentmentDelta: bestPain * 0.24,
		Time:            currentTime,
		Tags:            []string{"no_contact"},
	}
}

func (m *RelationshipManager) recordContactLocked(targetID string, at entities.GameTime) {
	contacts := append(m.contactLog[targetID], at)
	sort.SliceStable(contacts, func(i, j int) bool {
		return contacts[i].ToMinutes() < contacts[j].ToMinutes()
	})
	m.contactLog[targetID] = trimOldContacts(contacts, at, 72)
}

func (m *RelationshipManager) sortedRelationshipIDsLocked() []string {
	ids := make([]string, 0, len(m.poble.Relationships))
	for id := range m.poble.Relationships {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func (m *RelationshipManager) setPublicPerceptionLocked(targetID, observerID string, relationshipType entities.RelationshipType) {
	if m.publicPerception[targetID] == nil {
		m.publicPerception[targetID] = map[string]entities.RelationshipType{}
	}
	m.publicPerception[targetID][observerID] = relationshipType
}

func (m *RelationshipManager) perceptionLocked(targetID, observerID string) (entities.RelationshipType, bool) {
	byObserver, ok := m.publicPerception[targetID]
	if !ok {
		return entities.RelationshipStranger, false
	}
	perceived, ok := byObserver[observerID]
	return perceived, ok
}

func (m *RelationshipManager) derivePublicPerceptionLocked(targetID string) entities.RelationshipType {
	relationship, ok := m.poble.Relationships[targetID]
	if !ok || relationship.IsSecret {
		return entities.RelationshipAcquaintance
	}
	actual := m.inferRelationshipTypeLocked(relationship)
	switch actual {
	case entities.RelationshipToxicAttraction, entities.RelationshipSecretObsession:
		return entities.RelationshipRival
	case entities.RelationshipCodependent:
		return entities.RelationshipBestFriend
	case entities.RelationshipNemesis:
		return entities.RelationshipEnemy
	default:
		return actual
	}
}

func (m *RelationshipManager) defaultPublicTargetID() string {
	if m == nil || m.poble == nil {
		return ""
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	bestTarget := ""
	bestScore := float32(-1)
	for targetID, relationship := range m.poble.Relationships {
		score := relationship.Familiarity + relationship.Attraction + relationship.Resentment + relationship.Dependency
		if score > bestScore {
			bestTarget = targetID
			bestScore = score
		}
	}
	return bestTarget
}

func relationshipEventHasThirdParty(selfID, targetID string, event RelationshipEvent) bool {
	if event.ThirdPartyID != "" && event.ThirdPartyID != selfID && event.ThirdPartyID != targetID {
		return true
	}
	for _, tag := range event.Tags {
		if strings.EqualFold(tag, "third_party") || strings.HasPrefix(tag, "third_party:") {
			return true
		}
	}
	return false
}

func scaleNegative(delta, amplifier float32) float32 {
	if delta < 0 {
		return delta * amplifier
	}
	return delta
}

func scalePositive(delta, amplifier float32) float32 {
	if delta > 0 {
		return delta * amplifier
	}
	return delta
}

func affectionDelta(trust, attraction, resentment float32) float32 {
	return (trust * 0.18) + (attraction * 0.22) - (resentment * 0.12)
}

func familiarityDelta(event RelationshipEvent) float32 {
	switch event.Type {
	case RelationshipEventIntimacy, RelationshipEventBetrayal, RelationshipEventInfidelityDiscovered:
		return 8
	case RelationshipEventConflict, RelationshipEventSaturation:
		return 5
	default:
		return 3
	}
}

func updatedRelationshipTags(tags []string, event RelationshipEvent) []string {
	next := append([]string{}, tags...)
	next = append(next, event.Tags...)
	if event.Type == RelationshipEventInfidelityDiscovered {
		next = append(next, "handled_infidelity", "betrayal")
	}
	if event.IsPublic {
		next = append(next, "public")
	}
	return uniqueSortedStrings(next)
}

func shouldStoreRelationshipType(current, inferred entities.RelationshipType) bool {
	if isStructuralRelationship(current) && !isSevereRelationshipType(inferred) {
		return false
	}
	return true
}

func isStructuralRelationship(relationshipType entities.RelationshipType) bool {
	switch relationshipType {
	case entities.RelationshipParent, entities.RelationshipChild, entities.RelationshipSibling,
		entities.RelationshipFamily, entities.RelationshipMentor, entities.RelationshipStudent,
		entities.RelationshipBoss, entities.RelationshipEmployee:
		return true
	default:
		return false
	}
}

func isSevereRelationshipType(relationshipType entities.RelationshipType) bool {
	switch relationshipType {
	case entities.RelationshipNemesis, entities.RelationshipToxicAttraction,
		entities.RelationshipSecretObsession, entities.RelationshipCodependent,
		entities.RelationshipComplicated, entities.RelationshipEnemy:
		return true
	default:
		return false
	}
}

func isSecretObsession(relationship entities.Relationship) bool {
	return relationship.IsSecret &&
		(relationship.Type == entities.RelationshipObsession ||
			hasTag(relationship.Tags, "obsession") ||
			relationship.Attraction >= 72 ||
			relationship.Dependency >= 82)
}

func isContradictoryRelationship(relationship entities.Relationship) bool {
	highWarmth := relationship.Trust >= 58 || relationship.Affection >= 58 || relationship.Respect >= 65
	highPain := relationship.Resentment >= 45 || relationship.Fear >= 55
	desireWithDoubt := relationship.Attraction >= 55 && relationship.Trust < 45
	return (highWarmth && highPain) || desireWithDoubt
}

func signedTrust(relationship entities.Relationship) float32 {
	return (relationship.Trust * 2) - 100
}

func recentContactCount(contacts []entities.GameTime, currentTime entities.GameTime, hours int) int {
	count := 0
	for _, contact := range contacts {
		if currentTime.Diff(contact) <= hours {
			count++
		}
	}
	return count
}

func trimOldContacts(contacts []entities.GameTime, currentTime entities.GameTime, hours int) []entities.GameTime {
	kept := contacts[:0]
	for _, contact := range contacts {
		if currentTime.Diff(contact) <= hours {
			kept = append(kept, contact)
		}
	}
	return kept
}

func neglectPain(belonging float32, relationship entities.Relationship, hoursSinceContact int) float32 {
	if hoursSinceContact <= 0 {
		return 0
	}
	threshold := float32(96)
	if belonging >= 75 {
		threshold = 36
	} else if belonging >= 60 {
		threshold = 60
	}
	if float32(hoursSinceContact) <= threshold {
		return 0
	}

	bond := relationship.Affection*0.28 + relationship.Trust*0.22 + relationship.Attraction*0.18 + relationship.Dependency*0.22
	timePressure := minFloat32(float32(hoursSinceContact)-threshold, 120) / 120.0
	return clampPercent((bond*0.55)+(belonging*0.45)) * timePressure
}
