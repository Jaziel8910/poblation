package ai

import (
	"fmt"
	"math/rand"
	"strings"
	"sync"

	"github.com/user/poblation/internal/entities"
)

// RumourFactType identifies the real fact a rumour started from.
type RumourFactType string

const (
	RumourFactGeneric      RumourFactType = "GENERIC"
	RumourFactRelationship RumourFactType = "RELATIONSHIP"
	RumourFactSecret       RumourFactType = "SECRET"
	RumourFactBetrayal     RumourFactType = "BETRAYAL"
	RumourFactConflict     RumourFactType = "CONFLICT"
	RumourFactIntimacy     RumourFactType = "INTIMACY"
	RumourFactDeath        RumourFactType = "DEATH"
	RumourFactAchievement  RumourFactType = "ACHIEVEMENT"
)

// Rumour stores social knowledge that mutates as it travels.
type Rumour struct {
	ID               string         `json:"id"`
	OriginalFactType RumourFactType `json:"original_fact_type"`
	OriginalContent  string         `json:"original_content"`
	CurrentContent   string         `json:"current_content"`
	TruthScore       float32        `json:"truth_score"`
	Spreadings       int            `json:"spreadings"`
	KnownBy          []string       `json:"known_by"`
	IsSensitive      bool           `json:"is_sensitive"`
	SensitiveForID   string         `json:"sensitive_for_id"`
	SourceEventID    string         `json:"source_event_id"`
	OriginatorID     string         `json:"originator_id"`
	SubjectIDs       []string       `json:"subject_ids"`
	BelievedBy       []string       `json:"believed_by"`
	DoubtedBy        []string       `json:"doubted_by"`
	Tags             []string       `json:"tags"`
}

// Rumor is an alias for callers using US spelling.
type Rumor = Rumour

// ImpactEventType identifies drama produced by a rumour.
type ImpactEventType string

const (
	ImpactRumourSensitiveReached  ImpactEventType = "RUMOUR_SENSITIVE_REACHED"
	ImpactRumourConfrontation     ImpactEventType = "RUMOUR_CONFRONTATION"
	ImpactRumourRelationshipHit   ImpactEventType = "RUMOUR_RELATIONSHIP_HIT"
	ImpactRumourRevengeSeed       ImpactEventType = "RUMOUR_REVENGE_SEED"
	ImpactRumourPublicHumiliation ImpactEventType = "RUMOUR_PUBLIC_HUMILIATION"
)

// ImpactEvent is ready to be converted into the engine EventQueue.
type ImpactEvent struct {
	ID                 string             `json:"id"`
	Type               ImpactEventType    `json:"type"`
	RumourID           string             `json:"rumour_id"`
	ActorID            string             `json:"actor_id"`
	TargetID           string             `json:"target_id"`
	Severity           float32            `json:"severity"`
	RelationshipChange *RelationshipEvent `json:"relationship_change,omitempty"`
	Event              GameEvent          `json:"event"`
	Tags               []string           `json:"tags"`
}

// RumourSystem owns the rumour pool without importing internal/world.
type RumourSystem struct {
	mu      sync.RWMutex
	rumours map[string]*Rumour
	rng     *rand.Rand
}

// NewRumourSystem builds an empty procedural rumour system.
func NewRumourSystem(rng *rand.Rand) *RumourSystem {
	return &RumourSystem{
		rumours: map[string]*Rumour{},
		rng:     rng,
	}
}

// AddRumour stores a rumour and normalizes its public bookkeeping.
func (s *RumourSystem) AddRumour(rumour Rumour) {
	if s == nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	normalized := normalizeRumour(rumour)
	s.rumours[normalized.ID] = &normalized
}

// GetRumour returns a copy so callers cannot mutate the pool without the system.
func (s *RumourSystem) GetRumour(rumourID string) (Rumour, bool) {
	if s == nil || rumourID == "" {
		return Rumour{}, false
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	rumour, ok := s.rumours[rumourID]
	if !ok || rumour == nil {
		return Rumour{}, false
	}
	return copyRumour(*rumour), true
}

// SpreadRumour moves one rumour from one Poble to another and returns impacts.
func (s *RumourSystem) SpreadRumour(rumourID string, fromID string, toID string, world World) []ImpactEvent {
	if s == nil || rumourID == "" || toID == "" {
		return nil
	}

	from := findPobleInWorld(world, fromID)
	to := findPobleInWorld(world, toID)

	s.mu.Lock()
	rumour := s.rumours[rumourID]
	if rumour == nil {
		s.mu.Unlock()
		return nil
	}

	next := s.spreadMutation(rumour, from)
	next.KnownBy = uniqueSortedStrings(append(next.KnownBy, fromID, toID))
	if receiverBelievesRumour(&next, from, to, s.rng) {
		next.BelievedBy = uniqueSortedStrings(append(next.BelievedBy, toID))
	} else {
		next.DoubtedBy = uniqueSortedStrings(append(next.DoubtedBy, toID))
	}

	*rumour = normalizeRumour(next)
	snapshot := copyRumour(*rumour)
	s.mu.Unlock()

	return DetectRumourImpact(&snapshot, world)
}

// ShouldSpreadRumour estimates whether a Poble naturally repeats this rumour.
func ShouldSpreadRumour(rumour *Rumour, from *entities.Poble) bool {
	if rumour == nil || from == nil {
		return false
	}
	if from.Archetype == entities.ArchetypeJester {
		return true
	}
	return gossipTendency(from) >= 0.48 || rumour.IsSensitive && gossipTendency(from) >= 0.34
}

// MutateRumour returns a distorted copy of a rumour.
func MutateRumour(r *Rumour) *Rumour {
	if r == nil {
		return nil
	}

	mutated := copyRumour(*r)
	mutated.Spreadings++
	mutated.TruthScore = clampUnitValue(mutated.TruthScore - truthDrop(mutated.Spreadings))
	mutated.CurrentContent = dramaticRumourContent(mutated)
	mutated.Tags = uniqueSortedStrings(append(mutated.Tags, "mutated"))
	return &mutated
}

// CreateRumourFromEvent converts a real event into a pristine rumour.
func CreateRumourFromEvent(event GameEvent) *Rumour {
	content := strings.TrimSpace(event.Description)
	if content == "" {
		content = fallbackRumourContent(event)
	}

	rumour := Rumour{
		ID:               "rumour:" + fallbackEventID(event),
		OriginalFactType: rumourFactTypeForEvent(event),
		OriginalContent:  content,
		CurrentContent:   content,
		TruthScore:       1.0,
		KnownBy:          uniqueSortedStrings(eventParticipants(event)),
		IsSensitive:      eventCreatesSensitiveRumour(event),
		SensitiveForID:   sensitiveIDForEvent(event),
		SourceEventID:    event.ID,
		OriginatorID:     event.PrimaryActor,
		SubjectIDs:       uniqueSortedStrings(eventParticipants(event)),
		Tags:             uniqueSortedStrings(append([]string{"event:" + string(event.Type)}, event.Tags...)),
	}

	normalized := normalizeRumour(rumour)
	return &normalized
}

// DetectRumourImpact turns known rumours into concrete drama events.
func DetectRumourImpact(r *Rumour, world World) []ImpactEvent {
	if r == nil {
		return nil
	}

	impacts := make([]ImpactEvent, 0, 4)
	if r.IsSensitive && r.SensitiveForID != "" && containsString(r.KnownBy, r.SensitiveForID) {
		impacts = append(impacts, sensitiveReachedImpact(r))
	}

	for _, knowerID := range r.KnownBy {
		knower := findPobleInWorld(world, knowerID)
		if knower == nil {
			continue
		}
		impacts = append(impacts, rumourImpactsForKnower(r, knower)...)
	}

	return dedupeImpactEvents(impacts)
}

func (s *RumourSystem) spreadMutation(rumour *Rumour, from *entities.Poble) Rumour {
	distortionChance := rumourDistortionChance(rumour, from)
	if s.roll(distortionChance) {
		mutated := MutateRumour(rumour)
		if from != nil && from.Archetype == entities.ArchetypeJester && s.roll(0.45) {
			mutated.CurrentContent = jesterEmbellishment(mutated.CurrentContent)
			mutated.TruthScore = clampUnitValue(mutated.TruthScore - 0.07)
			mutated.Tags = uniqueSortedStrings(append(mutated.Tags, "jester_embellishment"))
		}
		return *mutated
	}

	advanced := copyRumour(*rumour)
	advanced.Spreadings++
	advanced.TruthScore = clampUnitValue(advanced.TruthScore - (truthDrop(advanced.Spreadings) * 0.45))
	return advanced
}

func (s *RumourSystem) roll(chance float32) bool {
	chance = clampUnitValue(chance)
	if s == nil || s.rng == nil {
		return chance >= 0.50
	}
	return s.rng.Float32() < chance
}

func normalizeRumour(rumour Rumour) Rumour {
	if rumour.ID == "" {
		rumour.ID = fallbackRumourID(rumour)
	}
	if rumour.CurrentContent == "" {
		rumour.CurrentContent = rumour.OriginalContent
	}
	if rumour.TruthScore == 0 && rumour.Spreadings == 0 {
		rumour.TruthScore = 1
	}
	rumour.TruthScore = clampUnitValue(rumour.TruthScore)
	rumour.KnownBy = uniqueSortedStrings(rumour.KnownBy)
	rumour.SubjectIDs = uniqueSortedStrings(rumour.SubjectIDs)
	rumour.BelievedBy = uniqueSortedStrings(rumour.BelievedBy)
	rumour.DoubtedBy = uniqueSortedStrings(rumour.DoubtedBy)
	rumour.Tags = uniqueSortedStrings(rumour.Tags)
	return rumour
}

func copyRumour(rumour Rumour) Rumour {
	copied := rumour
	copied.KnownBy = append([]string{}, rumour.KnownBy...)
	copied.SubjectIDs = append([]string{}, rumour.SubjectIDs...)
	copied.BelievedBy = append([]string{}, rumour.BelievedBy...)
	copied.DoubtedBy = append([]string{}, rumour.DoubtedBy...)
	copied.Tags = append([]string{}, rumour.Tags...)
	return copied
}

func rumourDistortionChance(rumour *Rumour, from *entities.Poble) float32 {
	chance := float32(0.10) + float32(rumour.Spreadings)*0.075
	if from != nil {
		chance += gossipTendency(from) * 0.18
		if from.Archetype == entities.ArchetypeJester {
			chance += 0.30
		}
	}
	if rumour.IsSensitive {
		chance += 0.08
	}
	return clampUnitValue(chance)
}

func gossipTendency(poble *entities.Poble) float32 {
	if poble == nil {
		return 0
	}
	extraversion := poble.Personality.Extraversion / 100.0
	carelessness := (100.0 - poble.Personality.Conscientiousness) / 100.0
	return clampUnitValue((extraversion * 0.58) + (carelessness * 0.42))
}

func receiverBelievesRumour(r *Rumour, from, to *entities.Poble, rng *rand.Rand) bool {
	if r == nil || to == nil {
		return false
	}

	chance := float32(0.25) + (r.TruthScore * 0.18)
	if from != nil {
		chance += trustInSpeaker(to, from.ID) * 0.0055
	}
	if r.IsSensitive {
		chance += 0.06
	}
	chance = clampUnitValue(chance)
	if rng == nil || chance >= 0.82 || chance <= 0.18 {
		return chance >= 0.50
	}
	return rng.Float32() < chance
}

func trustInSpeaker(receiver *entities.Poble, speakerID string) float32 {
	if receiver == nil || speakerID == "" {
		return 35
	}
	if relationship, ok := receiver.Relationships[speakerID]; ok {
		return relationship.Trust
	}
	return 35
}

func dramaticRumourContent(rumour Rumour) string {
	content := strings.TrimSpace(rumour.CurrentContent)
	if content == "" {
		content = strings.TrimSpace(rumour.OriginalContent)
	}

	switch rumour.OriginalFactType {
	case RumourFactConflict:
		return mutateConflictRumour(content, rumour.Spreadings)
	case RumourFactBetrayal:
		return mutateBetrayalRumour(content, rumour.Spreadings)
	case RumourFactIntimacy, RumourFactRelationship:
		return mutateIntimacyRumour(content, rumour.Spreadings)
	case RumourFactSecret:
		return mutateSecretRumour(content, rumour.Spreadings)
	default:
		return mutateGenericRumour(content, rumour.Spreadings)
	}
}

func mutateConflictRumour(content string, spreadings int) string {
	first, second, ok := firstTwoTitleWords(content)
	if spreadings >= 5 {
		if ok {
			return fmt.Sprintf("%s golpeo a %s", first, second)
		}
		return "La pelea termino en golpes"
	}
	if spreadings >= 3 {
		return content + "; dicen que hubo amenazas"
	}
	return content + "; no fue tan pequeno como lo cuentan"
}

func mutateBetrayalRumour(content string, spreadings int) string {
	if spreadings >= 5 {
		return content + "; fue planeado desde antes"
	}
	if spreadings >= 3 {
		return content + "; alguien lo esta tapando"
	}
	return content + "; eso no fue accidente"
}

func mutateIntimacyRumour(content string, spreadings int) string {
	if spreadings >= 5 {
		return content + "; llevan tiempo escondiendose"
	}
	if spreadings >= 3 {
		return content + "; nadie cree que fue solo una vez"
	}
	return content + "; habia demasiada confianza"
}

func mutateSecretRumour(content string, spreadings int) string {
	if spreadings >= 5 {
		return content + "; medio pueblo ya sabe la version fea"
	}
	if spreadings >= 3 {
		return content + "; falta lo peor"
	}
	return content + "; alguien esta mintiendo"
}

func mutateGenericRumour(content string, spreadings int) string {
	if spreadings >= 5 {
		return content + "; la version vieja ya no se parece"
	}
	if spreadings >= 3 {
		return content + "; hay partes que no cuadran"
	}
	return content + "; eso fue lo que se atrevieron a contar"
}

func jesterEmbellishment(content string) string {
	if content == "" {
		return "El chisme crecio con demasiados adornos"
	}
	return content + "; y segun el JESTER, fue peor y mas ridiculo"
}

func firstTwoTitleWords(content string) (string, string, bool) {
	words := strings.Fields(content)
	found := make([]string, 0, 2)
	for _, word := range words {
		cleaned := strings.Trim(word, ".,;:!?()[]{}\"'")
		if cleaned == "" || strings.ToLower(cleaned) == cleaned {
			continue
		}
		found = append(found, cleaned)
		if len(found) == 2 {
			return found[0], found[1], true
		}
	}
	return "", "", false
}

func rumourFactTypeForEvent(event GameEvent) RumourFactType {
	switch event.Type {
	case GameEventBetrayal:
		return RumourFactBetrayal
	case GameEventConflict, GameEventThreat, GameEventSocialNegative:
		return RumourFactConflict
	case GameEventIntimacy:
		return RumourFactIntimacy
	case GameEventDeath:
		return RumourFactDeath
	case GameEventGoalComplete:
		return RumourFactAchievement
	default:
		if containsString(event.Tags, "secret") {
			return RumourFactSecret
		}
		return RumourFactGeneric
	}
}

func eventCreatesSensitiveRumour(event GameEvent) bool {
	if event.IsTraumatic || event.Valence < -0.45 {
		return true
	}
	switch event.Type {
	case GameEventBetrayal, GameEventIntimacy, GameEventThreat, GameEventSocialNegative:
		return true
	default:
		return containsString(event.Tags, "secret") || containsString(event.Tags, "infidelity")
	}
}

func sensitiveIDForEvent(event GameEvent) string {
	for _, tag := range event.Tags {
		if strings.HasPrefix(tag, "sensitive_for:") {
			return strings.TrimPrefix(tag, "sensitive_for:")
		}
	}
	if event.TargetID != "" {
		return event.TargetID
	}
	return event.PrimaryActor
}

func fallbackRumourContent(event GameEvent) string {
	switch {
	case event.PrimaryActor != "" && event.TargetID != "":
		return fmt.Sprintf("%s and %s were involved in %s", event.PrimaryActor, event.TargetID, event.Type)
	case event.PrimaryActor != "":
		return fmt.Sprintf("%s was involved in %s", event.PrimaryActor, event.Type)
	default:
		return fmt.Sprintf("Something happened: %s", event.Type)
	}
}

func fallbackRumourID(rumour Rumour) string {
	base := rumour.OriginalContent
	if base == "" {
		base = rumour.CurrentContent
	}
	base = strings.ToLower(strings.TrimSpace(base))
	base = strings.ReplaceAll(base, " ", "-")
	if len(base) > 32 {
		base = base[:32]
	}
	if base == "" {
		base = "unknown"
	}
	return fmt.Sprintf("rumour:%s:%d", base, len(rumour.KnownBy)+rumour.Spreadings)
}

func sensitiveReachedImpact(r *Rumour) ImpactEvent {
	change := RelationshipEvent{
		Type:            RelationshipEventBetrayal,
		TargetID:        r.OriginatorID,
		TrustDelta:      -35,
		RespectDelta:    -18,
		ResentmentDelta: 42,
		IsPublic:        true,
		Tags:            []string{"rumour", "sensitive"},
	}
	return newImpactEvent(r, ImpactRumourSensitiveReached, r.SensitiveForID, r.OriginatorID, 78, &change)
}

func rumourImpactsForKnower(r *Rumour, knower *entities.Poble) []ImpactEvent {
	impacts := []ImpactEvent{}
	for _, subjectID := range r.SubjectIDs {
		if subjectID == "" || subjectID == knower.ID {
			continue
		}
		relationship, ok := knower.Relationships[subjectID]
		if !ok {
			continue
		}
		impacts = append(impacts, impactFromRelationship(r, knower.ID, subjectID, relationship)...)
	}
	return impacts
}

func impactFromRelationship(r *Rumour, knowerID, subjectID string, relationship entities.Relationship) []ImpactEvent {
	impacts := []ImpactEvent{}
	if r.IsSensitive && isIntimateRelationship(relationship) {
		change := rumourRelationshipHit(subjectID)
		impacts = append(impacts, newImpactEvent(r, ImpactRumourRelationshipHit, knowerID, subjectID, 68, &change))
	}
	if relationship.Resentment >= 70 && r.TruthScore >= 0.25 {
		impacts = append(impacts, newImpactEvent(r, ImpactRumourRevengeSeed, knowerID, subjectID, 60, nil))
	}
	if r.IsSensitive && containsString(r.KnownBy, subjectID) {
		impacts = append(impacts, newImpactEvent(r, ImpactRumourConfrontation, knowerID, subjectID, 64, nil))
	}
	return impacts
}

func rumourRelationshipHit(subjectID string) RelationshipEvent {
	return RelationshipEvent{
		Type:            RelationshipEventBetrayal,
		TargetID:        subjectID,
		TrustDelta:      -22,
		RespectDelta:    -12,
		ResentmentDelta: 26,
		IsPublic:        true,
		Tags:            []string{"rumour_impact"},
	}
}

func newImpactEvent(r *Rumour, impactType ImpactEventType, actorID, targetID string, severity float32, change *RelationshipEvent) ImpactEvent {
	eventType := GameEventSocialNegative
	if impactType == ImpactRumourRevengeSeed {
		eventType = GameEventBetrayal
	}
	return ImpactEvent{
		ID:                 fmt.Sprintf("impact:%s:%s:%s:%s", r.ID, impactType, actorID, targetID),
		Type:               impactType,
		RumourID:           r.ID,
		ActorID:            actorID,
		TargetID:           targetID,
		Severity:           clampPercent(severity),
		RelationshipChange: change,
		Event: GameEvent{
			ID:           fmt.Sprintf("event:%s:%s:%s", r.ID, actorID, targetID),
			Type:         eventType,
			PrimaryActor: actorID,
			TargetID:     targetID,
			Severity:     clampPercent(severity),
			Valence:      -0.65,
			Description:  r.CurrentContent,
			Tags:         []string{"rumour", string(impactType)},
		},
		Tags: []string{"rumour", string(impactType)},
	}
}

func isIntimateRelationship(relationship entities.Relationship) bool {
	switch relationship.Type {
	case entities.RelationshipLover, entities.RelationshipSpouse,
		entities.RelationshipFriendsWithBenefits, entities.RelationshipCrush,
		entities.RelationshipObsession, entities.RelationshipSecretObsession:
		return true
	default:
		return relationship.Attraction >= 65 || relationship.Dependency >= 72
	}
}

func dedupeImpactEvents(impacts []ImpactEvent) []ImpactEvent {
	seen := map[string]struct{}{}
	result := make([]ImpactEvent, 0, len(impacts))
	for _, impact := range impacts {
		if impact.ID == "" {
			continue
		}
		if _, ok := seen[impact.ID]; ok {
			continue
		}
		seen[impact.ID] = struct{}{}
		result = append(result, impact)
	}
	return result
}

func findPobleInWorld(world World, id string) *entities.Poble {
	if world == nil || id == "" {
		return nil
	}
	for _, poble := range world.GetAllPobles() {
		if poble != nil && poble.ID == id {
			return poble
		}
	}
	return nil
}

func truthDrop(spreadings int) float32 {
	return minFloat32(0.22, 0.045+float32(spreadings)*0.018)
}

func clampUnitValue(value float32) float32 {
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}
