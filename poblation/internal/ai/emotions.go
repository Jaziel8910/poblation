package ai

import (
	"sort"

	"github.com/user/poblation/internal/entities"
)

// GameEventType identifies broad event categories for emotional processing.
type GameEventType string

const (
	GameEventGeneric        GameEventType = "GENERIC"
	GameEventDeath          GameEventType = "DEATH"
	GameEventThreat         GameEventType = "THREAT"
	GameEventConflict       GameEventType = "CONFLICT"
	GameEventBetrayal       GameEventType = "BETRAYAL"
	GameEventSocialPositive GameEventType = "SOCIAL_POSITIVE"
	GameEventSocialNegative GameEventType = "SOCIAL_NEGATIVE"
	GameEventGoalComplete   GameEventType = "GOAL_COMPLETE"
	GameEventIntimacy       GameEventType = "INTIMACY"
)

// GameEvent describes a world event interpreted by the emotion system.
type GameEvent struct {
	ID           string            `json:"id"`
	Type         GameEventType     `json:"type"`
	Time         entities.GameTime `json:"time"`
	PrimaryActor string            `json:"primary_actor"`
	TargetID     string            `json:"target_id"`
	Participants []string          `json:"participants"`
	Severity     float32           `json:"severity"`
	Valence      float32           `json:"valence"`
	IsTraumatic  bool              `json:"is_traumatic"`
	Description  string            `json:"description"`
	Tags         []string          `json:"tags"`
}

// EmotionChange stores one generated emotional response.
type EmotionChange struct {
	Emotion       entities.EmotionType `json:"emotion"`
	Intensity     float32              `json:"intensity"`
	DurationHours int                  `json:"duration_hours"`
	SourceEventID string               `json:"source_event_id"`
	Reason        string               `json:"reason"`
}

// ActiveEmotion tracks one emotion over time.
type ActiveEmotion struct {
	Emotion        entities.EmotionType `json:"emotion"`
	Intensity      float32              `json:"intensity"`
	RemainingHours float32              `json:"remaining_hours"`
	SourceEventID  string               `json:"source_event_id"`
}

// EmotionConflict stores contradictory emotional pressures.
type EmotionConflict struct {
	First   entities.EmotionType `json:"first"`
	Second  entities.EmotionType `json:"second"`
	Tension float32              `json:"tension"`
	Thought string               `json:"thought"`
}

type recentEventImpact struct {
	AgeHours float32
	Valence  float32
	Weight   float32
}

// EmotionSystem processes events and maintains emotional state for one Poble.
type EmotionSystem struct {
	Poble                    *entities.Poble
	Active                   []ActiveEmotion
	BaseMoodValence          float32
	RecentEvents             []recentEventImpact
	EmotionConflicts         []EmotionConflict
	InternalConflictThoughts []string
	EmotionallyInteresting   bool
}

// NewEmotionSystem builds a new emotion system bound to one Poble.
func NewEmotionSystem(poble *entities.Poble) *EmotionSystem {
	system := &EmotionSystem{Poble: poble}
	if poble == nil {
		return system
	}

	system.BaseMoodValence = poble.EmotionalState.Valence
	for _, emotion := range poble.EmotionalState.ActiveEmotions {
		system.Active = append(system.Active, ActiveEmotion{
			Emotion:        emotion,
			Intensity:      30,
			RemainingHours: 6,
		})
	}

	system.rebuildState()
	return system
}

// ProcessEvent generates and applies emotional changes from one event.
func (s *EmotionSystem) ProcessEvent(event GameEvent) []EmotionChange {
	if s == nil || s.Poble == nil {
		return nil
	}

	changes := s.eventToChanges(event)
	for _, change := range changes {
		s.applyChange(change)
	}

	if len(changes) > 0 {
		s.RecentEvents = append(s.RecentEvents, recentEventImpact{
			AgeHours: 0,
			Valence:  s.eventValence(changes),
			Weight:   s.eventWeight(changes),
		})
	}

	s.DetectEmotionConflict()
	s.rebuildState()
	return changes
}

// Update decays emotions, refreshes mood, and tracks contradictions.
func (s *EmotionSystem) Update(deltaHours int) {
	if s == nil || s.Poble == nil || deltaHours <= 0 {
		return
	}

	hours := float32(deltaHours)
	decayed := s.Active[:0]
	for _, emotion := range s.Active {
		emotion.RemainingHours -= hours
		emotion.Intensity = clampPercent(emotion.Intensity - (s.decayRate(emotion.Emotion) * hours))
		if emotion.RemainingHours > 0 && emotion.Intensity > 1.0 {
			decayed = append(decayed, emotion)
		}
	}
	s.Active = decayed

	filteredEvents := s.RecentEvents[:0]
	for _, item := range s.RecentEvents {
		item.AgeHours += hours
		if item.AgeHours <= 24.0 {
			filteredEvents = append(filteredEvents, item)
		}
	}
	s.RecentEvents = filteredEvents

	s.MoodDrift(deltaHours)
	s.DetectEmotionConflict()
	s.rebuildState()
}

// GetCurrentValence returns current signed valence for template selection.
func (s *EmotionSystem) GetCurrentValence() float32 {
	if s == nil || s.Poble == nil {
		return 0
	}
	return s.Poble.EmotionalState.Valence
}

// DetectEmotionConflict finds contradictory active emotions and marks drama.
func (s *EmotionSystem) DetectEmotionConflict() []EmotionConflict {
	if s == nil || s.Poble == nil {
		return nil
	}

	pairs := [][2]entities.EmotionType{
		{entities.EmotionLove, entities.EmotionAnger},
		{entities.EmotionLove, entities.EmotionResentment},
		{entities.EmotionTrust, entities.EmotionFear},
		{entities.EmotionPride, entities.EmotionShame},
		{entities.EmotionJoy, entities.EmotionGrief},
		{entities.EmotionLust, entities.EmotionDisgust},
		{entities.EmotionHope, entities.EmotionAnxiety},
	}

	intensities := s.intensityByEmotion()
	conflicts := make([]EmotionConflict, 0, len(pairs))
	thoughts := make([]string, 0, len(pairs))

	for _, pair := range pairs {
		first := intensities[pair[0]]
		second := intensities[pair[1]]
		if first < 18.0 || second < 18.0 {
			continue
		}

		conflict := EmotionConflict{
			First:   pair[0],
			Second:  pair[1],
			Tension: clampPercent((first + second) / 2.0),
			Thought: conflictThought(pair[0], pair[1]),
		}
		conflicts = append(conflicts, conflict)
		thoughts = append(thoughts, conflict.Thought)
	}

	sort.SliceStable(conflicts, func(i, j int) bool {
		return conflicts[i].Tension > conflicts[j].Tension
	})

	s.EmotionConflicts = conflicts
	s.InternalConflictThoughts = thoughts
	s.EmotionallyInteresting = len(conflicts) > 0
	return conflicts
}

// MoodDrift moves the daily mood baseline gradually using needs and fresh events.
func (s *EmotionSystem) MoodDrift(deltaHours int) {
	if s == nil || s.Poble == nil || deltaHours <= 0 {
		return
	}

	var recentValence float32
	var recentWeight float32
	for _, item := range s.RecentEvents {
		weight := maxFloat32(0.1, item.Weight*(1.0-item.AgeHours/24.0))
		recentValence += item.Valence * weight
		recentWeight += weight
	}
	if recentWeight > 0 {
		recentValence /= recentWeight
	}

	needs := s.Poble.Needs
	needsPenalty := ((needs.Hunger + needs.Thirst + needs.Sleep) * 0.004) +
		((needs.Belonging + needs.Esteem + needs.Purpose) * 0.0015)
	baseline := ((s.Poble.Personality.Agreeableness - s.Poble.Personality.Neuroticism) / 160.0) - needsPenalty
	target := clampSignedUnit(baseline + recentValence)

	inertia := minFloat32(float32(deltaHours)*0.08, 0.45)
	s.BaseMoodValence = clampSignedUnit(s.BaseMoodValence + ((target - s.BaseMoodValence) * inertia))
}

func (s *EmotionSystem) eventToChanges(event GameEvent) []EmotionChange {
	severity := clampPercent(event.Severity)
	if severity == 0 {
		severity = 50
	}

	referenceID := s.referenceTarget(event)
	sentiment := s.socialSentiment(referenceID)
	memoryBias := s.memoryBias(referenceID)
	neuroticAmplifier := 1.0 + (s.Poble.Personality.Neuroticism / 140.0)
	changes := make([]EmotionChange, 0, 4)

	add := func(emotion entities.EmotionType, intensity float32, traumatic bool, reason string) {
		intensity = clampPercent(intensity * neuroticAmplifier)
		if intensity < 8.0 {
			return
		}
		changes = append(changes, EmotionChange{
			Emotion:       emotion,
			Intensity:     intensity,
			DurationHours: s.durationFor(emotion, severity, traumatic),
			SourceEventID: event.ID,
			Reason:        reason,
		})
	}

	switch event.Type {
	case GameEventDeath:
		griefBond := maxFloat32(0, sentiment+memoryBias)
		hostility := maxFloat32(0, -(sentiment + memoryBias))
		add(entities.EmotionGrief, severity*0.55+griefBond*0.65, event.IsTraumatic, "loss touched bond and memory")
		add(entities.EmotionFear, severity*0.30+maxFloat32(0, memoryBias), event.IsTraumatic, "death reminds body that world unsafe")
		add(entities.EmotionJoy, hostility*0.55, false, "enemy gone can feel good")
		add(entities.EmotionRelief, hostility*0.45, false, "threat removed")
	case GameEventThreat:
		add(entities.EmotionFear, severity*0.75+maxFloat32(0, memoryBias), true, "danger nearby")
		add(entities.EmotionAnxiety, severity*0.55, true, "future feels unstable")
		if s.Poble.Archetype == entities.ArchetypeWarrior || s.Poble.Personality.Cruelty > 60 {
			add(entities.EmotionAnger, severity*0.35, false, "threat triggers counter-force")
		}
	case GameEventConflict:
		add(entities.EmotionAnger, severity*0.65+maxFloat32(0, -sentiment)*0.25, false, "open conflict stings pride and safety")
		add(entities.EmotionContempt, maxFloat32(0, -sentiment)*0.45, false, "respect drops")
		add(entities.EmotionFear, severity*0.25, event.IsTraumatic, "conflict can escalate")
	case GameEventBetrayal:
		add(entities.EmotionAnger, severity*0.60+maxFloat32(0, sentiment)*0.20, false, "trust broken")
		add(entities.EmotionResentment, severity*0.70+maxFloat32(0, sentiment)*0.25, event.IsTraumatic, "hurt lingers")
		add(entities.EmotionGrief, severity*0.40+maxFloat32(0, sentiment)*0.35, event.IsTraumatic, "bond damaged")
	case GameEventSocialPositive:
		add(entities.EmotionJoy, severity*0.55+maxFloat32(0, sentiment)*0.20, false, "social contact feels good")
		add(entities.EmotionTrust, severity*0.45+maxFloat32(0, sentiment)*0.30, false, "interaction supports trust")
		if sentiment > 20 {
			add(entities.EmotionLove, severity*0.35+memoryBias*0.20, false, "warmth deepens attachment")
		}
	case GameEventSocialNegative:
		add(entities.EmotionLoneliness, severity*0.45, false, "social wound leaves distance")
		add(entities.EmotionShame, severity*0.35, false, "social hit turns inward")
		add(entities.EmotionAnger, severity*0.25+maxFloat32(0, -sentiment)*0.20, false, "rejection can provoke retaliation")
	case GameEventGoalComplete:
		add(entities.EmotionPride, severity*0.70, false, "goal completion raises self-worth")
		add(entities.EmotionJoy, severity*0.45, false, "effort paid off")
		add(entities.EmotionHope, severity*0.35, false, "future opens")
	case GameEventIntimacy:
		add(entities.EmotionLove, severity*0.45+maxFloat32(0, sentiment)*0.30, false, "closeness strengthens bond")
		add(entities.EmotionLust, severity*0.35+(s.Poble.Personality.Horniness*0.35), false, "body responds to intimacy")
		add(entities.EmotionTrust, severity*0.30, false, "vulnerability shared")
	default:
		if event.Valence >= 0 {
			add(entities.EmotionJoy, (severity*0.35)+(event.Valence*30), false, "event reads as positive")
			add(entities.EmotionHope, (severity*0.20)+(event.Valence*20), false, "positive signal shapes outlook")
		} else {
			add(entities.EmotionAnxiety, (severity*0.30)+(-event.Valence*30), event.IsTraumatic, "negative uncertainty spikes")
			add(entities.EmotionGrief, (severity*0.25)+(-event.Valence*20), event.IsTraumatic, "negative signal hurts")
		}
	}

	return changes
}

func (s *EmotionSystem) applyChange(change EmotionChange) {
	for i := range s.Active {
		if s.Active[i].Emotion != change.Emotion {
			continue
		}

		current := &s.Active[i]
		total := current.Intensity + change.Intensity
		current.Intensity = clampPercent(total)
		current.RemainingHours = maxFloat32(current.RemainingHours, float32(change.DurationHours))
		if change.SourceEventID != "" {
			current.SourceEventID = change.SourceEventID
		}
		return
	}

	s.Active = append(s.Active, ActiveEmotion{
		Emotion:        change.Emotion,
		Intensity:      clampPercent(change.Intensity),
		RemainingHours: float32(change.DurationHours),
		SourceEventID:  change.SourceEventID,
	})
}

func (s *EmotionSystem) decayRate(emotion entities.EmotionType) float32 {
	neuroticism := s.Poble.Personality.Neuroticism / 100.0
	slowNegative := 1.0 - (neuroticism * 0.35)

	switch emotion {
	case entities.EmotionGrief:
		return 0.35 * slowNegative
	case entities.EmotionResentment:
		return 0.85 * slowNegative
	case entities.EmotionLove:
		return 1.20
	case entities.EmotionTrust:
		return 1.30
	case entities.EmotionAnxiety, entities.EmotionFear:
		return 2.20 * slowNegative
	case entities.EmotionJoy, entities.EmotionRelief:
		return 4.50
	case entities.EmotionAnger, entities.EmotionContempt:
		return 3.20 * slowNegative
	case entities.EmotionLust:
		return 3.80
	case entities.EmotionPride, entities.EmotionHope:
		return 2.40
	default:
		return 2.60
	}
}

func (s *EmotionSystem) durationFor(emotion entities.EmotionType, severity float32, traumatic bool) int {
	base := 6 + int(severity/12.0)
	switch emotion {
	case entities.EmotionGrief:
		base = 72 + int(severity)
	case entities.EmotionResentment:
		base = 48 + int(severity*0.6)
	case entities.EmotionLove, entities.EmotionTrust:
		base = 24 + int(severity*0.4)
	case entities.EmotionFear, entities.EmotionAnxiety:
		base = 16 + int(severity*0.5)
	}
	if traumatic && (emotion == entities.EmotionGrief || emotion == entities.EmotionFear || emotion == entities.EmotionAnxiety || emotion == entities.EmotionResentment) {
		base *= 2
	}
	if base < 2 {
		base = 2
	}
	return base
}

func (s *EmotionSystem) socialSentiment(targetID string) float32 {
	if targetID == "" {
		return 0
	}

	relationship, ok := s.Poble.Relationships[targetID]
	if !ok {
		return 0
	}

	score := (relationship.Affection * 0.35) +
		(relationship.Trust * 0.25) +
		(relationship.Respect * 0.10) +
		(relationship.Attraction * 0.12) -
		(relationship.Resentment * 0.32) -
		(relationship.Fear * 0.12)

	switch relationship.Type {
	case entities.RelationshipFriend, entities.RelationshipBestFriend, entities.RelationshipFamily, entities.RelationshipLover, entities.RelationshipSpouse:
		score += 18
	case entities.RelationshipEnemy, entities.RelationshipBetrayer:
		score -= 30
	case entities.RelationshipRival:
		score -= 12
	case entities.RelationshipCrush:
		score += 10
	}

	return clampRange(score, -100, 100)
}

func (s *EmotionSystem) memoryBias(targetID string) float32 {
	if targetID == "" {
		return 0
	}

	var positive float32
	var negative float32
	for _, memory := range s.Poble.Memories {
		if !containsString(memory.Participants, targetID) {
			continue
		}

		weight := maxFloat32(10.0, memory.EmotionIntensity) / 100.0
		switch memory.Type {
		case entities.MemoryPositive, entities.MemoryFunny, entities.MemoryRomantic, entities.MemoryAchievement:
			positive += 22.0 * weight
		case entities.MemoryNegative, entities.MemoryTraumatic, entities.MemoryBetrayal, entities.MemoryViolent, entities.MemoryEmbarrassing:
			negative += 24.0 * weight
		}
	}

	return clampRange(positive-negative, -35, 35)
}

func (s *EmotionSystem) rebuildState() {
	if s == nil || s.Poble == nil {
		return
	}

	if len(s.Active) == 0 {
		s.Poble.EmotionalState.ActiveEmotions = nil
		s.Poble.EmotionalState.Valence = clampSignedUnit(s.BaseMoodValence)
		s.Poble.EmotionalState.Arousal = 0
		s.Poble.EmotionalState.Dominance = 0
		s.Poble.CurrentMood = s.moodFromCore(s.Poble.EmotionalState.Valence, 0, 0)
		s.Poble.EmotionalState.CurrentMood = s.Poble.CurrentMood
		return
	}

	sort.SliceStable(s.Active, func(i, j int) bool {
		return s.Active[i].Intensity > s.Active[j].Intensity
	})

	var valence float32
	var arousal float32
	var dominance float32
	var total float32
	activeLabels := make([]entities.EmotionType, 0, len(s.Active))

	for _, emotion := range s.Active {
		weight := emotion.Intensity / 100.0
		v, a, d := emotionVector(emotion.Emotion)
		valence += v * weight
		arousal += a * weight
		dominance += d * weight
		total += weight
		if emotion.Intensity >= 10.0 {
			activeLabels = append(activeLabels, emotion.Emotion)
		}
	}

	if total > 0 {
		valence /= total
		arousal /= total
		dominance /= total
	}

	finalValence := clampSignedUnit((s.BaseMoodValence * 0.4) + (valence * 0.6))
	finalArousal := clampSignedUnit(arousal)
	finalDominance := clampSignedUnit(dominance)

	s.Poble.EmotionalState.Valence = finalValence
	s.Poble.EmotionalState.Arousal = finalArousal
	s.Poble.EmotionalState.Dominance = finalDominance
	s.Poble.EmotionalState.ActiveEmotions = activeLabels
	s.Poble.CurrentMood = s.moodFromCore(finalValence, finalArousal, finalDominance)
	s.Poble.EmotionalState.CurrentMood = s.Poble.CurrentMood
}

func (s *EmotionSystem) moodFromCore(valence, arousal, dominance float32) entities.MoodType {
	switch {
	case valence > 0.65 && arousal > 0.35:
		return entities.MoodEuphoric
	case valence > 0.25 && arousal >= 0:
		return entities.MoodHappy
	case valence > 0.10:
		return entities.MoodContent
	case valence < -0.70 && arousal < -0.15:
		return entities.MoodDepressed
	case valence < -0.45 && arousal > 0.30 && dominance < 0:
		return entities.MoodAnxious
	case valence < -0.35 && arousal > 0.35 && dominance >= 0:
		return entities.MoodAngry
	case valence < -0.25:
		return entities.MoodSad
	default:
		return entities.MoodNeutral
	}
}

func (s *EmotionSystem) referenceTarget(event GameEvent) string {
	if event.TargetID != "" {
		return event.TargetID
	}
	if event.PrimaryActor != "" && event.PrimaryActor != s.Poble.ID {
		return event.PrimaryActor
	}
	for _, participant := range event.Participants {
		if participant != s.Poble.ID {
			return participant
		}
	}
	return ""
}

func (s *EmotionSystem) intensityByEmotion() map[entities.EmotionType]float32 {
	values := make(map[entities.EmotionType]float32, len(s.Active))
	for _, emotion := range s.Active {
		if emotion.Intensity > values[emotion.Emotion] {
			values[emotion.Emotion] = emotion.Intensity
		}
	}
	return values
}

func (s *EmotionSystem) eventValence(changes []EmotionChange) float32 {
	var total float32
	var weight float32
	for _, change := range changes {
		v, _, _ := emotionVector(change.Emotion)
		total += v * change.Intensity
		weight += change.Intensity
	}
	if weight == 0 {
		return 0
	}
	return clampSignedUnit(total / weight)
}

func (s *EmotionSystem) eventWeight(changes []EmotionChange) float32 {
	var total float32
	for _, change := range changes {
		total += change.Intensity
	}
	if total == 0 {
		return 0
	}
	return total / float32(len(changes))
}

func emotionVector(emotion entities.EmotionType) (float32, float32, float32) {
	switch emotion {
	case entities.EmotionJoy:
		return 0.90, 0.55, 0.30
	case entities.EmotionLove:
		return 0.80, 0.35, 0.10
	case entities.EmotionTrust:
		return 0.50, -0.15, 0.10
	case entities.EmotionHope:
		return 0.55, 0.20, 0.10
	case entities.EmotionPride:
		return 0.60, 0.30, 0.55
	case entities.EmotionRelief:
		return 0.45, -0.20, 0.05
	case entities.EmotionLust:
		return 0.30, 0.70, 0.10
	case entities.EmotionAnger:
		return -0.60, 0.80, 0.35
	case entities.EmotionFear:
		return -0.75, 0.75, -0.60
	case entities.EmotionAnxiety:
		return -0.65, 0.60, -0.45
	case entities.EmotionGrief:
		return -0.90, -0.10, -0.40
	case entities.EmotionResentment:
		return -0.70, 0.35, 0.05
	case entities.EmotionContempt:
		return -0.55, 0.20, 0.30
	case entities.EmotionShame:
		return -0.75, 0.10, -0.60
	case entities.EmotionLoneliness:
		return -0.55, -0.15, -0.35
	case entities.EmotionConfusion:
		return -0.20, 0.15, -0.10
	case entities.EmotionDisgust:
		return -0.65, 0.30, 0.10
	default:
		return 0, 0, 0
	}
}

func conflictThought(first, second entities.EmotionType) string {
	switch {
	case first == entities.EmotionLove && second == entities.EmotionResentment:
		return "Quiere acercarse y castigar al mismo tiempo."
	case first == entities.EmotionLove && second == entities.EmotionAnger:
		return "Ama a quien también le provoca rabia."
	case first == entities.EmotionTrust && second == entities.EmotionFear:
		return "Parte suya confia; parte suya espera daño."
	case first == entities.EmotionJoy && second == entities.EmotionGrief:
		return "Siente alivio y perdida mezclados."
	case first == entities.EmotionPride && second == entities.EmotionShame:
		return "Quiere celebrar y esconderse a la vez."
	default:
		return "Siente impulsos emocionales en direcciones opuestas."
	}
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
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

func clampSignedPercent(value float32) float32 {
	if value < -100 {
		return -100
	}
	if value > 100 {
		return 100
	}
	return value
}

func clampSignedUnit(value float32) float32 {
	if value < -1 {
		return -1
	}
	if value > 1 {
		return 1
	}
	return value
}

func clampRange(value, minValue, maxValue float32) float32 {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

func minFloat32(a, b float32) float32 {
	if a < b {
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

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
