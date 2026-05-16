package entities

import "fmt"

// Sex identifies biological sex used by simulation data.
type Sex string

const (
	// Male identifies male characters.
	Male Sex = "Male"
	// Female identifies female characters.
	Female Sex = "Female"
	// Intersex identifies intersex characters.
	Intersex Sex = "Intersex"
)

func (s Sex) String() string {
	return string(s)
}

func (s Sex) IsValid() bool {
	switch s {
	case Male, Female, Intersex:
		return true
	default:
		return false
	}
}

// Orientation stores romantic and sexual attraction traits.
type Orientation struct {
	// Romantic represents romantic attraction from 0.0 hetero to 1.0 homo.
	Romantic float32 `json:"romantic"`
	// Sexual represents sexual attraction from 0.0 hetero to 1.0 homo.
	Sexual float32 `json:"sexual"`
	// Intensity represents libido from 0.0 asexual to 1.0 hypersexual.
	Intensity float32 `json:"intensity"`
	// Fluidity represents how much orientation shifts over time.
	Fluidity float32 `json:"fluidity"`
}

func NewOrientation() Orientation {
	return Orientation{
		Romantic:  0.5,
		Sexual:    0.5,
		Intensity: 0.5,
		Fluidity:  0.0,
	}
}

func (o Orientation) IsValid() bool {
	return isUnit(o.Romantic) && isUnit(o.Sexual) && isUnit(o.Intensity) && isUnit(o.Fluidity)
}

// ArchetypeID identifies a character narrative archetype.
type ArchetypeID string

const (
	// ArchetypeRuler identifies authority-focused characters.
	ArchetypeRuler ArchetypeID = "RULER"
	// ArchetypeLover identifies intimacy-focused characters.
	ArchetypeLover ArchetypeID = "LOVER"
	// ArchetypeJester identifies chaos and humor-focused characters.
	ArchetypeJester ArchetypeID = "JESTER"
	// ArchetypeSage identifies wisdom-focused characters.
	ArchetypeSage ArchetypeID = "SAGE"
	// ArchetypeRebel identifies rule-breaking characters.
	ArchetypeRebel ArchetypeID = "REBEL"
	// ArchetypeCaretaker identifies protective characters.
	ArchetypeCaretaker ArchetypeID = "CARETAKER"
	// ArchetypeVillain identifies antagonistic characters.
	ArchetypeVillain ArchetypeID = "VILLAIN"
	// ArchetypeGhost identifies withdrawn or haunted characters.
	ArchetypeGhost ArchetypeID = "GHOST"
	// ArchetypeAddict identifies compulsive characters.
	ArchetypeAddict ArchetypeID = "ADDICT"
	// ArchetypeProphet identifies visionary characters.
	ArchetypeProphet ArchetypeID = "PROPHET"
	// ArchetypeSchemer identifies manipulative planners.
	ArchetypeSchemer ArchetypeID = "SCHEMER"
	// ArchetypeInnocent identifies naive or pure characters.
	ArchetypeInnocent ArchetypeID = "INNOCENT"
	// ArchetypeWarrior identifies conflict-ready characters.
	ArchetypeWarrior ArchetypeID = "WARRIOR"
	// ArchetypeDrifter identifies rootless characters.
	ArchetypeDrifter ArchetypeID = "DRIFTER"
	// ArchetypeMirror identifies reflective characters.
	ArchetypeMirror ArchetypeID = "MIRROR"
	// ArchetypeCustom identifies user-defined archetypes.
	ArchetypeCustom ArchetypeID = "CUSTOM"
)

func (a ArchetypeID) String() string {
	return string(a)
}

func (a ArchetypeID) IsValid() bool {
	switch a {
	case ArchetypeRuler, ArchetypeLover, ArchetypeJester, ArchetypeSage, ArchetypeRebel,
		ArchetypeCaretaker, ArchetypeVillain, ArchetypeGhost, ArchetypeAddict, ArchetypeProphet,
		ArchetypeSchemer, ArchetypeInnocent, ArchetypeWarrior, ArchetypeDrifter, ArchetypeMirror,
		ArchetypeCustom:
		return true
	default:
		return false
	}
}

// Personality stores Big Five traits and POBLATION-specific modifiers.
type Personality struct {
	// Openness measures curiosity and imagination from 0 to 100.
	Openness float32 `json:"openness"`
	// Conscientiousness measures discipline and order from 0 to 100.
	Conscientiousness float32 `json:"conscientiousness"`
	// Extraversion measures social energy from 0 to 100.
	Extraversion float32 `json:"extraversion"`
	// Agreeableness measures empathy and cooperation from 0 to 100.
	Agreeableness float32 `json:"agreeableness"`
	// Neuroticism measures emotional volatility from 0 to 100.
	Neuroticism float32 `json:"neuroticism"`
	// Cruelty measures willingness to harm from 0 to 100.
	Cruelty float32 `json:"cruelty"`
	// Horniness measures sexual drive from 0 to 100.
	Horniness float32 `json:"horniness"`
	// Ambition measures desire for advancement from 0 to 100.
	Ambition float32 `json:"ambition"`
	// Jealousy measures possessiveness from 0 to 100.
	Jealousy float32 `json:"jealousy"`
	// Loyalty measures attachment and reliability from 0 to 100.
	Loyalty float32 `json:"loyalty"`
}

func NewPersonality() Personality {
	return Personality{
		Openness:          50,
		Conscientiousness: 50,
		Extraversion:      50,
		Agreeableness:     50,
		Neuroticism:       50,
		Cruelty:           0,
		Horniness:         50,
		Ambition:          50,
		Jealousy:          50,
		Loyalty:           50,
	}
}

func (p Personality) IsValid() bool {
	return isPercent(p.Openness) && isPercent(p.Conscientiousness) && isPercent(p.Extraversion) &&
		isPercent(p.Agreeableness) && isPercent(p.Neuroticism) && isPercent(p.Cruelty) &&
		isPercent(p.Horniness) && isPercent(p.Ambition) && isPercent(p.Jealousy) && isPercent(p.Loyalty)
}

// Needs stores survival, social, and psychological drives.
type Needs struct {
	// Hunger measures food need from 0 to 100.
	Hunger float32 `json:"hunger"`
	// Thirst measures water need from 0 to 100.
	Thirst float32 `json:"thirst"`
	// Sleep measures rest need from 0 to 100.
	Sleep float32 `json:"sleep"`
	// Safety measures security need from 0 to 100.
	Safety float32 `json:"safety"`
	// Belonging measures social connection need from 0 to 100.
	Belonging float32 `json:"belonging"`
	// Esteem measures status and respect need from 0 to 100.
	Esteem float32 `json:"esteem"`
	// Sex measures sexual need from 0 to 100.
	Sex float32 `json:"sex"`
	// Power measures control and influence need from 0 to 100.
	Power float32 `json:"power"`
	// Purpose measures meaning need from 0 to 100.
	Purpose float32 `json:"purpose"`
}

func NewNeeds() Needs {
	return Needs{
		Hunger:    0,
		Thirst:    0,
		Sleep:     0,
		Safety:    50,
		Belonging: 50,
		Esteem:    50,
		Sex:       0,
		Power:     50,
		Purpose:   50,
	}
}

func (n Needs) IsValid() bool {
	return isPercent(n.Hunger) && isPercent(n.Thirst) && isPercent(n.Sleep) && isPercent(n.Safety) &&
		isPercent(n.Belonging) && isPercent(n.Esteem) && isPercent(n.Sex) &&
		isPercent(n.Power) && isPercent(n.Purpose)
}

// MoodType identifies the dominant mood label.
type MoodType string

const (
	// MoodHappy indicates cheerful mood.
	MoodHappy MoodType = "HAPPY"
	// MoodContent indicates satisfied mood.
	MoodContent MoodType = "CONTENT"
	// MoodNeutral indicates baseline mood.
	MoodNeutral MoodType = "NEUTRAL"
	// MoodAnxious indicates anxious mood.
	MoodAnxious MoodType = "ANXIOUS"
	// MoodSad indicates sad mood.
	MoodSad MoodType = "SAD"
	// MoodAngry indicates angry mood.
	MoodAngry MoodType = "ANGRY"
	// MoodDepressed indicates depressed mood.
	MoodDepressed MoodType = "DEPRESSED"
	// MoodEuphoric indicates euphoric mood.
	MoodEuphoric MoodType = "EUPHORIC"
	// MoodObsessive indicates obsessive mood.
	MoodObsessive MoodType = "OBSESSIVE"
	// MoodNumb indicates emotionally numb mood.
	MoodNumb MoodType = "NUMB"
)

func (m MoodType) String() string {
	return string(m)
}

func (m MoodType) IsValid() bool {
	switch m {
	case MoodHappy, MoodContent, MoodNeutral, MoodAnxious, MoodSad, MoodAngry,
		MoodDepressed, MoodEuphoric, MoodObsessive, MoodNumb:
		return true
	default:
		return false
	}
}

// EmotionType identifies a specific active emotion.
type EmotionType string

const (
	// EmotionJoy identifies joy.
	EmotionJoy EmotionType = "joy"
	// EmotionAnger identifies anger.
	EmotionAnger EmotionType = "anger"
	// EmotionFear identifies fear.
	EmotionFear EmotionType = "fear"
	// EmotionDisgust identifies disgust.
	EmotionDisgust EmotionType = "disgust"
	// EmotionSurprise identifies surprise.
	EmotionSurprise EmotionType = "surprise"
	// EmotionLove identifies love.
	EmotionLove EmotionType = "love"
	// EmotionJealousy identifies jealousy.
	EmotionJealousy EmotionType = "jealousy"
	// EmotionShame identifies shame.
	EmotionShame EmotionType = "shame"
	// EmotionPride identifies pride.
	EmotionPride EmotionType = "pride"
	// EmotionGrief identifies grief.
	EmotionGrief EmotionType = "grief"
	// EmotionLust identifies lust.
	EmotionLust EmotionType = "lust"
	// EmotionResentment identifies resentment.
	EmotionResentment EmotionType = "resentment"
	// EmotionHope identifies hope.
	EmotionHope EmotionType = "hope"
	// EmotionGuilt identifies guilt.
	EmotionGuilt EmotionType = "guilt"
	// EmotionEnvy identifies envy.
	EmotionEnvy EmotionType = "envy"
	// EmotionTrust identifies trust.
	EmotionTrust EmotionType = "trust"
	// EmotionRelief identifies relief.
	EmotionRelief EmotionType = "relief"
	// EmotionBoredom identifies boredom.
	EmotionBoredom EmotionType = "boredom"
	// EmotionLoneliness identifies loneliness.
	EmotionLoneliness EmotionType = "loneliness"
	// EmotionCuriosity identifies curiosity.
	EmotionCuriosity EmotionType = "curiosity"
	// EmotionConfusion identifies confusion.
	EmotionConfusion EmotionType = "confusion"
	// EmotionAdmiration identifies admiration.
	EmotionAdmiration EmotionType = "admiration"
	// EmotionContempt identifies contempt.
	EmotionContempt EmotionType = "contempt"
	// EmotionAnxiety identifies anxiety.
	EmotionAnxiety EmotionType = "anxiety"
)

func (e EmotionType) String() string {
	return string(e)
}

func (e EmotionType) IsValid() bool {
	switch e {
	case EmotionJoy, EmotionAnger, EmotionFear, EmotionDisgust, EmotionSurprise, EmotionLove,
		EmotionJealousy, EmotionShame, EmotionPride, EmotionGrief, EmotionLust, EmotionResentment,
		EmotionHope, EmotionGuilt, EmotionEnvy, EmotionTrust, EmotionRelief, EmotionBoredom,
		EmotionLoneliness, EmotionCuriosity, EmotionConfusion, EmotionAdmiration, EmotionContempt,
		EmotionAnxiety:
		return true
	default:
		return false
	}
}

// EmotionalState stores PAD mood dimensions and active emotion labels.
type EmotionalState struct {
	// Valence measures pleasure versus displeasure.
	Valence float32 `json:"valence"`
	// Arousal measures activation versus calm.
	Arousal float32 `json:"arousal"`
	// Dominance measures control versus submission.
	Dominance float32 `json:"dominance"`
	// ActiveEmotions stores current emotion labels.
	ActiveEmotions []EmotionType `json:"active_emotions"`
	// CurrentMood stores the current dominant mood.
	CurrentMood MoodType `json:"current_mood"`
}

func NewEmotionalState() EmotionalState {
	return EmotionalState{
		Valence:        0,
		Arousal:        0,
		Dominance:      0,
		ActiveEmotions: []EmotionType{},
		CurrentMood:    MoodNeutral,
	}
}

func (e EmotionalState) IsValid() bool {
	if !isSignedUnit(e.Valence) || !isSignedUnit(e.Arousal) || !isSignedUnit(e.Dominance) || !e.CurrentMood.IsValid() {
		return false
	}
	for _, emotion := range e.ActiveEmotions {
		if !emotion.IsValid() {
			return false
		}
	}
	return true
}

// GameTime stores simulation time.
type GameTime struct {
	// Day stores the absolute simulation day.
	Day int `json:"day"`
	// Hour stores hour of day from 0 to 23.
	Hour int `json:"hour"`
	// Minute stores minute of hour from 0 to 59.
	Minute int `json:"minute"`
}

func NewGameTime(day, hour, minute int) GameTime {
	return GameTime{Day: day, Hour: hour, Minute: minute}
}

func (g GameTime) String() string {
	return fmt.Sprintf("Day %d %02d:%02d", g.Day, g.Hour, g.Minute)
}

func (g GameTime) Add(hours int) GameTime {
	totalMinutes := g.ToMinutes() + (hours * 60)
	if totalMinutes < 0 {
		totalMinutes = 0
	}

	return GameTime{
		Day:    totalMinutes / (24 * 60),
		Hour:   (totalMinutes / 60) % 24,
		Minute: totalMinutes % 60,
	}
}

func (g GameTime) Diff(other GameTime) int {
	return (g.ToMinutes() - other.ToMinutes()) / 60
}

func (g GameTime) IsSameDay(other GameTime) bool {
	return g.Day == other.Day
}

func (g GameTime) ToMinutes() int {
	return (g.Day * 24 * 60) + (g.Hour * 60) + g.Minute
}

func (g GameTime) IsValid() bool {
	return g.Day >= 0 && g.Hour >= 0 && g.Hour <= 23 && g.Minute >= 0 && g.Minute <= 59
}

// MemoryType identifies a memory category.
type MemoryType string

const (
	// MemoryPositive identifies positive memories.
	MemoryPositive MemoryType = "POSITIVE"
	// MemoryNegative identifies negative memories.
	MemoryNegative MemoryType = "NEGATIVE"
	// MemoryTraumatic identifies traumatic memories.
	MemoryTraumatic MemoryType = "TRAUMATIC"
	// MemoryErotic identifies erotic memories.
	MemoryErotic MemoryType = "EROTIC"
	// MemoryFunny identifies funny memories.
	MemoryFunny MemoryType = "FUNNY"
	// MemoryEmbarrassing identifies embarrassing memories.
	MemoryEmbarrassing MemoryType = "EMBARRASSING"
	// MemoryViolent identifies violent memories.
	MemoryViolent MemoryType = "VIOLENT"
	// MemoryRomantic identifies romantic memories.
	MemoryRomantic MemoryType = "ROMANTIC"
	// MemoryBetrayal identifies betrayal memories.
	MemoryBetrayal MemoryType = "BETRAYAL"
	// MemoryAchievement identifies achievement memories.
	MemoryAchievement MemoryType = "ACHIEVEMENT"
)

func (m MemoryType) String() string {
	return string(m)
}

func (m MemoryType) IsValid() bool {
	switch m {
	case MemoryPositive, MemoryNegative, MemoryTraumatic, MemoryErotic, MemoryFunny,
		MemoryEmbarrassing, MemoryViolent, MemoryRomantic, MemoryBetrayal, MemoryAchievement:
		return true
	default:
		return false
	}
}

// Memory stores an event remembered by a character.
type Memory struct {
	// ID stores unique memory identifier.
	ID string `json:"id"`
	// Timestamp stores when the memory happened.
	Timestamp GameTime `json:"timestamp"`
	// Type stores memory category.
	Type MemoryType `json:"type"`
	// Participants stores involved Poble IDs.
	Participants []string `json:"participants"`
	// EmotionIntensity stores emotional strength from 0 to 100.
	EmotionIntensity float32 `json:"emotion_intensity"`
	// IsRepressed marks memories hidden from normal recall.
	IsRepressed bool `json:"is_repressed"`
	// Tags stores searchable memory labels.
	Tags []string `json:"tags"`
	// Summary stores human-readable memory summary.
	Summary string `json:"summary"`
}

func NewMemory(id string, timestamp GameTime, memoryType MemoryType, summary string) Memory {
	return Memory{
		ID:               id,
		Timestamp:        timestamp,
		Type:             memoryType,
		Participants:     []string{},
		EmotionIntensity: 0,
		IsRepressed:      false,
		Tags:             []string{},
		Summary:          summary,
	}
}

func (m Memory) IsValid() bool {
	return m.ID != "" && m.Timestamp.IsValid() && m.Type.IsValid() && isPercent(m.EmotionIntensity)
}

// SecretType identifies a hidden information category.
type SecretType string

const (
	// SecretPastRelationship identifies hidden past relationships.
	SecretPastRelationship SecretType = "PAST_RELATIONSHIP"
	// SecretHiddenSkill identifies hidden skills.
	SecretHiddenSkill SecretType = "HIDDEN_SKILL"
	// SecretDarkDesire identifies hidden dark desires.
	SecretDarkDesire SecretType = "DARK_DESIRE"
	// SecretTrueOrientation identifies hidden orientation truth.
	SecretTrueOrientation SecretType = "TRUE_ORIENTATION"
	// SecretPlannedBetrayal identifies planned betrayal.
	SecretPlannedBetrayal SecretType = "PLANNED_BETRAYAL"
	// SecretTraumaEvent identifies hidden trauma events.
	SecretTraumaEvent SecretType = "TRAUMA_EVENT"
	// SecretObsession identifies hidden obsession.
	SecretObsession SecretType = "OBSESSION"
	// SecretChild identifies hidden children.
	SecretChild SecretType = "SECRET_CHILD"
	// SecretPhobia identifies hidden phobias.
	SecretPhobia SecretType = "PHOBIA"
	// SecretCriminalAct identifies hidden criminal acts.
	SecretCriminalAct SecretType = "CRIMINAL_ACT"
)

func (s SecretType) String() string {
	return string(s)
}

func (s SecretType) IsValid() bool {
	switch s {
	case SecretPastRelationship, SecretHiddenSkill, SecretDarkDesire, SecretTrueOrientation,
		SecretPlannedBetrayal, SecretTraumaEvent, SecretObsession, SecretChild, SecretPhobia,
		SecretCriminalAct:
		return true
	default:
		return false
	}
}

// Secret stores hidden character information.
type Secret struct {
	// ID stores unique secret identifier.
	ID string `json:"id"`
	// Type stores secret category.
	Type SecretType `json:"type"`
	// Content stores secret description.
	Content string `json:"content"`
	// IsRevealed marks whether the secret is public.
	IsRevealed bool `json:"is_revealed"`
	// KnownBy stores Poble IDs who know the secret.
	KnownBy []string `json:"known_by"`
	// RevealTrigger stores trigger key for revealing the secret.
	RevealTrigger string `json:"reveal_trigger"`
}

func NewSecret(id string, secretType SecretType, content string) Secret {
	return Secret{
		ID:            id,
		Type:          secretType,
		Content:       content,
		IsRevealed:    false,
		KnownBy:       []string{},
		RevealTrigger: "",
	}
}

func (s Secret) IsValid() bool {
	return s.ID != "" && s.Type.IsValid()
}

// RelationshipType identifies relationship category between two characters.
type RelationshipType string

const (
	// RelationshipStranger identifies no meaningful relationship.
	RelationshipStranger RelationshipType = "STRANGER"
	// RelationshipAcquaintance identifies light social familiarity.
	RelationshipAcquaintance RelationshipType = "ACQUAINTANCE"
	// RelationshipFriend identifies friendship.
	RelationshipFriend RelationshipType = "FRIEND"
	// RelationshipBestFriend identifies close friendship.
	RelationshipBestFriend RelationshipType = "BEST_FRIEND"
	// RelationshipRival identifies rivalry.
	RelationshipRival RelationshipType = "RIVAL"
	// RelationshipEnemy identifies hostility.
	RelationshipEnemy RelationshipType = "ENEMY"
	// RelationshipLover identifies romantic or sexual partnership.
	RelationshipLover RelationshipType = "LOVER"
	// RelationshipSpouse identifies marriage or equivalent bond.
	RelationshipSpouse RelationshipType = "SPOUSE"
	// RelationshipExSpouse identifies former spouse.
	RelationshipExSpouse RelationshipType = "EX_SPOUSE"
	// RelationshipParent identifies parent bond.
	RelationshipParent RelationshipType = "PARENT"
	// RelationshipChild identifies child bond.
	RelationshipChild RelationshipType = "CHILD"
	// RelationshipSibling identifies sibling bond.
	RelationshipSibling RelationshipType = "SIBLING"
	// RelationshipFamily identifies wider family bond.
	RelationshipFamily RelationshipType = "FAMILY"
	// RelationshipMentor identifies mentor bond.
	RelationshipMentor RelationshipType = "MENTOR"
	// RelationshipStudent identifies student bond.
	RelationshipStudent RelationshipType = "STUDENT"
	// RelationshipBoss identifies authority at work.
	RelationshipBoss RelationshipType = "BOSS"
	// RelationshipEmployee identifies subordinate at work.
	RelationshipEmployee RelationshipType = "EMPLOYEE"
	// RelationshipNeighbor identifies nearby resident bond.
	RelationshipNeighbor RelationshipType = "NEIGHBOR"
	// RelationshipAlly identifies cooperative bond.
	RelationshipAlly RelationshipType = "ALLY"
	// RelationshipBetrayer identifies betrayal-defined bond.
	RelationshipBetrayer RelationshipType = "BETRAYER"
	// RelationshipObsession identifies obsessive attachment.
	RelationshipObsession RelationshipType = "OBSESSION"
	// RelationshipCrush identifies one-sided attraction.
	RelationshipCrush RelationshipType = "CRUSH"
	// RelationshipFriendsWithBenefits identifies casual sexual bond.
	RelationshipFriendsWithBenefits RelationshipType = "FRIENDS_WITH_BENEFITS"
	// RelationshipCaretaker identifies care provider bond.
	RelationshipCaretaker RelationshipType = "CARETAKER"
	// RelationshipDependent identifies care receiver bond.
	RelationshipDependent RelationshipType = "DEPENDENT"
	// RelationshipNemesis identifies a relationship defined by lasting vendetta.
	RelationshipNemesis RelationshipType = "NEMESIS"
	// RelationshipToxicAttraction identifies desire mixed with active hatred.
	RelationshipToxicAttraction RelationshipType = "TOXIC_ATTRACTION"
	// RelationshipCodependent identifies attachment that disrupts functioning.
	RelationshipCodependent RelationshipType = "CODEPENDENT"
	// RelationshipSecretObsession identifies hidden one-sided obsession.
	RelationshipSecretObsession RelationshipType = "SECRET_OBSESSION"
	// RelationshipComplicated identifies unresolved contradictory bonds.
	RelationshipComplicated RelationshipType = "COMPLICATED"
)

func (r RelationshipType) String() string {
	return string(r)
}

func (r RelationshipType) IsValid() bool {
	switch r {
	case RelationshipStranger, RelationshipAcquaintance, RelationshipFriend, RelationshipBestFriend,
		RelationshipRival, RelationshipEnemy, RelationshipLover, RelationshipSpouse, RelationshipExSpouse,
		RelationshipParent, RelationshipChild, RelationshipSibling, RelationshipFamily, RelationshipMentor,
		RelationshipStudent, RelationshipBoss, RelationshipEmployee, RelationshipNeighbor, RelationshipAlly,
		RelationshipBetrayer, RelationshipObsession, RelationshipCrush, RelationshipFriendsWithBenefits,
		RelationshipCaretaker, RelationshipDependent, RelationshipNemesis, RelationshipToxicAttraction,
		RelationshipCodependent, RelationshipSecretObsession, RelationshipComplicated:
		return true
	default:
		return false
	}
}

// Relationship stores directed relationship state toward another character.
type Relationship struct {
	// TargetID stores the other Poble ID.
	TargetID string `json:"target_id"`
	// Type stores relationship category.
	Type RelationshipType `json:"type"`
	// Affection stores warmth from 0 to 100.
	Affection float32 `json:"affection"`
	// Trust stores confidence from 0 to 100.
	Trust float32 `json:"trust"`
	// Respect stores esteem from 0 to 100.
	Respect float32 `json:"respect"`
	// Fear stores fear from 0 to 100.
	Fear float32 `json:"fear"`
	// Attraction stores romantic or sexual attraction from 0 to 100.
	Attraction float32 `json:"attraction"`
	// Familiarity stores knowledge of the other character from 0 to 100.
	Familiarity float32 `json:"familiarity"`
	// Dependency stores practical or emotional reliance from 0 to 100.
	Dependency float32 `json:"dependency"`
	// Resentment stores stored bitterness from 0 to 100.
	Resentment float32 `json:"resentment"`
	// LastInteraction stores time of last interaction.
	LastInteraction GameTime `json:"last_interaction"`
	// SharedMemories stores memory IDs shared with this character.
	SharedMemories []string `json:"shared_memories"`
	// Tags stores custom relationship labels.
	Tags []string `json:"tags"`
	// IsSecret marks relationships hidden from others.
	IsSecret bool `json:"is_secret"`
}

func NewRelationship(targetID string, relationshipType RelationshipType) Relationship {
	return Relationship{
		TargetID:        targetID,
		Type:            relationshipType,
		Affection:       0,
		Trust:           50,
		Respect:         50,
		Fear:            0,
		Attraction:      0,
		Familiarity:     0,
		Dependency:      0,
		Resentment:      0,
		LastInteraction: NewGameTime(0, 0, 0),
		SharedMemories:  []string{},
		Tags:            []string{},
		IsSecret:        false,
	}
}

func (r Relationship) IsValid() bool {
	return r.TargetID != "" && r.Type.IsValid() && isPercent(r.Affection) && isPercent(r.Trust) &&
		isPercent(r.Respect) && isPercent(r.Fear) && isPercent(r.Attraction) &&
		isPercent(r.Familiarity) && isPercent(r.Dependency) && isPercent(r.Resentment) &&
		r.LastInteraction.IsValid()
}

// ConditionID identifies a physical health condition.
type ConditionID string

const (
	// ConditionHealthy identifies no major condition.
	ConditionHealthy ConditionID = "HEALTHY"
	// ConditionInjured identifies injury.
	ConditionInjured ConditionID = "INJURED"
	// ConditionSick identifies sickness.
	ConditionSick ConditionID = "SICK"
	// ConditionPregnant identifies pregnancy.
	ConditionPregnant ConditionID = "PREGNANT"
	// ConditionExhausted identifies exhaustion.
	ConditionExhausted ConditionID = "EXHAUSTED"
)

func (c ConditionID) String() string {
	return string(c)
}

func (c ConditionID) IsValid() bool {
	switch c {
	case ConditionHealthy, ConditionInjured, ConditionSick, ConditionPregnant, ConditionExhausted:
		return true
	default:
		return false
	}
}

// STIType identifies sexually transmitted infection data.
type STIType string

const (
	// STINone identifies no STI.
	STINone STIType = "NONE"
	// STIChlamydia identifies chlamydia.
	STIChlamydia STIType = "CHLAMYDIA"
	// STIGonorrhea identifies gonorrhea.
	STIGonorrhea STIType = "GONORRHEA"
	// STISyphilis identifies syphilis.
	STISyphilis STIType = "SYPHILIS"
	// STIHIV identifies HIV.
	STIHIV STIType = "HIV"
)

func (s STIType) String() string {
	return string(s)
}

func (s STIType) IsValid() bool {
	switch s {
	case STINone, STIChlamydia, STIGonorrhea, STISyphilis, STIHIV:
		return true
	default:
		return false
	}
}

// HealthState stores physical condition data.
type HealthState struct {
	// HP stores hit points from 0 to 100.
	HP int `json:"hp"`
	// Conditions stores active physical condition IDs.
	Conditions []ConditionID `json:"conditions"`
	// Age stores current biological age.
	Age int `json:"age"`
	// Fertility stores fertility from 0.0 to 1.0.
	Fertility float32 `json:"fertility"`
	// STIs stores active STI labels.
	STIs []STIType `json:"stis"`
}

func NewHealthState(age int) HealthState {
	return HealthState{
		HP:         100,
		Conditions: []ConditionID{},
		Age:        age,
		Fertility:  1.0,
		STIs:       []STIType{},
	}
}

func (h HealthState) IsValid() bool {
	if h.HP < 0 || h.HP > 100 || h.Age < 0 || !isUnit(h.Fertility) {
		return false
	}
	for _, condition := range h.Conditions {
		if !condition.IsValid() {
			return false
		}
	}
	for _, sti := range h.STIs {
		if !sti.IsValid() {
			return false
		}
	}
	return true
}

// MentalCondition identifies a mental health condition.
type MentalCondition string

const (
	// MentalStable identifies no major mental condition.
	MentalStable MentalCondition = "STABLE"
	// MentalAnxiety identifies anxiety condition.
	MentalAnxiety MentalCondition = "ANXIETY"
	// MentalDepression identifies depression condition.
	MentalDepression MentalCondition = "DEPRESSION"
	// MentalPTSD identifies trauma condition.
	MentalPTSD MentalCondition = "PTSD"
	// MentalAddiction identifies addiction condition.
	MentalAddiction MentalCondition = "ADDICTION"
	// MentalPsychosis identifies psychosis condition.
	MentalPsychosis MentalCondition = "PSYCHOSIS"
	// MentalObsessive identifies obsessive condition.
	MentalObsessive MentalCondition = "OBSESSIVE"
)

func (m MentalCondition) String() string {
	return string(m)
}

func (m MentalCondition) IsValid() bool {
	switch m {
	case MentalStable, MentalAnxiety, MentalDepression, MentalPTSD, MentalAddiction, MentalPsychosis, MentalObsessive:
		return true
	default:
		return false
	}
}

// MentalState stores psychological health data.
type MentalState struct {
	// Stability stores mental stability from 0 to 100.
	Stability int `json:"stability"`
	// Conditions stores active mental conditions.
	Conditions []MentalCondition `json:"conditions"`
	// Traumas stores memory IDs or trauma labels.
	Traumas []string `json:"traumas"`
	// TherapyLevel stores support level from 0 to 100.
	TherapyLevel int `json:"therapy_level"`
}

func NewMentalState() MentalState {
	return MentalState{
		Stability:    100,
		Conditions:   []MentalCondition{},
		Traumas:      []string{},
		TherapyLevel: 0,
	}
}

func (m MentalState) IsValid() bool {
	if m.Stability < 0 || m.Stability > 100 || m.TherapyLevel < 0 || m.TherapyLevel > 100 {
		return false
	}
	for _, condition := range m.Conditions {
		if !condition.IsValid() {
			return false
		}
	}
	return true
}

// VocabularyTier identifies how formal or informal a Poble speaks.
type VocabularyTier string

const (
	VocabPrimitive VocabularyTier = "PRIMITIVE"
	VocabSimple    VocabularyTier = "SIMPLE"
	VocabStandard  VocabularyTier = "STANDARD"
	VocabEducated  VocabularyTier = "EDUCATED"
	VocabEloquent  VocabularyTier = "ELOQUENT"
)

func (v VocabularyTier) String() string { return string(v) }

func (v VocabularyTier) IsValid() bool {
	switch v {
	case VocabPrimitive, VocabSimple, VocabStandard, VocabEducated, VocabEloquent:
		return true
	default:
		return false
	}
}

// VocabularyProfile stores speech pattern data for a Poble.
type VocabularyProfile struct {
	// Tier identifies formality level.
	Tier VocabularyTier `json:"tier"`
	// FillerWords stores habitual filler words this Poble uses.
	FillerWords []string `json:"filler_words"`
	// FavoriteExpressions stores signature phrases.
	FavoriteExpressions []string `json:"favorite_expressions"`
	// Verbosity from 0 (terse) to 100 (rambling).
	Verbosity float32 `json:"verbosity"`
}

func NewVocabularyProfile() VocabularyProfile {
	return VocabularyProfile{
		Tier:                VocabStandard,
		FillerWords:         []string{},
		FavoriteExpressions: []string{},
		Verbosity:           50,
	}
}

func (v VocabularyProfile) IsValid() bool {
	return v.Tier.IsValid() && isPercent(v.Verbosity)
}

// NicknameEntry stores one nickname and its origin context.
type NicknameEntry struct {
	// Nickname is the actual nickname text.
	Nickname string `json:"nickname"`
	// GivenBy is the Poble ID who coined it (empty = self-given).
	GivenBy string `json:"given_by"`
	// Reason describes why it was given.
	Reason string `json:"reason"`
	// IsPublic marks whether the nickname is widely known.
	IsPublic bool `json:"is_public"`
	// IsAccepted marks whether the Poble likes it.
	IsAccepted bool `json:"is_accepted"`
}

func NewNicknameEntry(nickname, givenBy, reason string) NicknameEntry {
	return NicknameEntry{
		Nickname:   nickname,
		GivenBy:    givenBy,
		Reason:     reason,
		IsPublic:   false,
		IsAccepted: true,
	}
}

// ProfanityLevel identifies how comfortable a Poble is with profanity.
type ProfanityLevel string

const (
	ProfanityNone     ProfanityLevel = "NONE"
	ProfanityMild     ProfanityLevel = "MILD"
	ProfanityModerate ProfanityLevel = "MODERATE"
	ProfanityHeavy    ProfanityLevel = "HEAVY"
	ProfanityExtreme  ProfanityLevel = "EXTREME"
)

func (p ProfanityLevel) String() string { return string(p) }

func (p ProfanityLevel) IsValid() bool {
	switch p {
	case ProfanityNone, ProfanityMild, ProfanityModerate, ProfanityHeavy, ProfanityExtreme:
		return true
	default:
		return false
	}
}

// ProfanityProfile stores swearing habits.
type ProfanityProfile struct {
	// Level identifies baseline profanity comfort.
	Level ProfanityLevel `json:"level"`
	// MildExpletives stores mild swear words this Poble uses.
	MildExpletives []string `json:"mild_expletives"`
	// StrongExpletives stores strong swear words this Poble uses.
	StrongExpletives []string `json:"strong_expletives"`
	// SignatureExpletive is the go-to swear word under stress.
	SignatureExpletive string `json:"signature_expletive"`
	// StressMultiplier scales profanity frequency under emotional pressure (1.0 = no change).
	StressMultiplier float32 `json:"stress_multiplier"`
}

func NewProfanityProfile() ProfanityProfile {
	return ProfanityProfile{
		Level:              ProfanityMild,
		MildExpletives:     []string{},
		StrongExpletives:   []string{},
		SignatureExpletive: "",
		StressMultiplier:   1.0,
	}
}

func (p ProfanityProfile) IsValid() bool {
	return p.Level.IsValid() && p.StressMultiplier >= 0
}

// PreferenceCategory identifies a hidden preference domain.
type PreferenceCategory string

const (
	PrefPhysical  PreferenceCategory = "PHYSICAL"
	PrefEmotional PreferenceCategory = "EMOTIONAL"
	PrefPower     PreferenceCategory = "POWER"
	PrefRitual    PreferenceCategory = "RITUAL"
	PrefTaboo     PreferenceCategory = "TABOO"
	PrefSensory   PreferenceCategory = "SENSORY"
)

func (p PreferenceCategory) String() string { return string(p) }

func (p PreferenceCategory) IsValid() bool {
	switch p {
	case PrefPhysical, PrefEmotional, PrefPower, PrefRitual, PrefTaboo, PrefSensory:
		return true
	default:
		return false
	}
}

// HiddenPreference stores one specific kink or preference.
type HiddenPreference struct {
	// Name describes the preference.
	Name string `json:"name"`
	// Category groups the preference.
	Category PreferenceCategory `json:"category"`
	// Intensity from 0 (curious) to 100 (defining).
	Intensity float32 `json:"intensity"`
	// IsDiscovered marks whether the Poble is aware of this preference.
	IsDiscovered bool `json:"is_discovered"`
	// IsSharedWith stores Poble IDs who know about this preference.
	IsSharedWith []string `json:"is_shared_with"`
}

// HiddenPreferences replaces the flat Kinks []string with structured data.
type HiddenPreferences struct {
	// Preferences stores all hidden preferences.
	Preferences []HiddenPreference `json:"preferences"`
	// DominantCategory is the most active preference domain.
	DominantCategory PreferenceCategory `json:"dominant_category"`
	// Openness measures willingness to explore from 0 to 100.
	Openness float32 `json:"openness"`
}

func NewHiddenPreferences() HiddenPreferences {
	return HiddenPreferences{
		Preferences:      []HiddenPreference{},
		DominantCategory: PrefEmotional,
		Openness:         50,
	}
}

func (h HiddenPreferences) IsValid() bool {
	return h.DominantCategory.IsValid() && isPercent(h.Openness)
}

// QuirkType identifies a behavioral quirk category.
type QuirkType string

const (
	QuirkPhysical  QuirkType = "PHYSICAL"
	QuirkVerbal    QuirkType = "VERBAL"
	QuirkEmotional QuirkType = "EMOTIONAL"
	QuirkSocial    QuirkType = "SOCIAL"
	QuirkHabit     QuirkType = "HABIT"
)

func (q QuirkType) String() string { return string(q) }

func (q QuirkType) IsValid() bool {
	switch q {
	case QuirkPhysical, QuirkVerbal, QuirkEmotional, QuirkSocial, QuirkHabit:
		return true
	default:
		return false
	}
}

// Quirk stores one behavioral quirk.
type Quirk struct {
	// ID uniquely identifies this quirk.
	ID string `json:"id"`
	// Type categorizes the quirk.
	Type QuirkType `json:"type"`
	// Description describes the quirk behavior.
	Description string `json:"description"`
	// Intensity from 0 to 100.
	Intensity float32 `json:"intensity"`
	// TriggeredByEmotion is the emotion that amplifies this quirk.
	TriggeredByEmotion EmotionType `json:"triggered_by_emotion"`
	// IsEndearing marks whether others find this quirk likeable.
	IsEndearing bool `json:"is_endearing"`
}

func NewQuirk(id string, quirkType QuirkType, description string) Quirk {
	return Quirk{
		ID:          id,
		Type:        quirkType,
		Description: description,
		Intensity:   50,
		IsEndearing: false,
	}
}

func (q Quirk) IsValid() bool {
	return q.ID != "" && q.Type.IsValid() && isPercent(q.Intensity)
}

// LoveLanguageType identifies how a Poble expresses and receives affection.
type LoveLanguageType string

const (
	LoveLangWordsOfAffirmation LoveLanguageType = "WORDS_OF_AFFIRMATION"
	LoveLangActsOfService      LoveLanguageType = "ACTS_OF_SERVICE"
	LoveLangGifts              LoveLanguageType = "GIFTS"
	LoveLangQualityTime        LoveLanguageType = "QUALITY_TIME"
	LoveLangPhysicalTouch      LoveLanguageType = "PHYSICAL_TOUCH"
)

func (l LoveLanguageType) String() string { return string(l) }

func (l LoveLanguageType) IsValid() bool {
	switch l {
	case LoveLangWordsOfAffirmation, LoveLangActsOfService, LoveLangGifts,
		LoveLangQualityTime, LoveLangPhysicalTouch:
		return true
	default:
		return false
	}
}

// LoveLanguage stores how a Poble gives and receives love.
type LoveLanguage struct {
	// Primary is the dominant way this Poble expresses love.
	Primary LoveLanguageType `json:"primary"`
	// Secondary is the secondary expression mode.
	Secondary LoveLanguageType `json:"secondary"`
	// ReceivePreference is how this Poble most wants to receive love.
	ReceivePreference LoveLanguageType `json:"receive_preference"`
}

func NewLoveLanguage() LoveLanguage {
	return LoveLanguage{
		Primary:           LoveLangQualityTime,
		Secondary:         LoveLangPhysicalTouch,
		ReceivePreference: LoveLangQualityTime,
	}
}

func (l LoveLanguage) IsValid() bool {
	return l.Primary.IsValid() && l.Secondary.IsValid() && l.ReceivePreference.IsValid()
}

// DefenseMechanismType identifies psychological defense strategies.
type DefenseMechanismType string

const (
	DefenseDenial          DefenseMechanismType = "DENIAL"
	DefenseProjection      DefenseMechanismType = "PROJECTION"
	DefenseRationalization DefenseMechanismType = "RATIONALIZATION"
	DefenseDisplacement    DefenseMechanismType = "DISPLACEMENT"
	DefenseRepression      DefenseMechanismType = "REPRESSION"
	DefenseHumor           DefenseMechanismType = "HUMOR"
	DefenseSublimation     DefenseMechanismType = "SUBLIMATION"
	DefenseIntellectualize DefenseMechanismType = "INTELLECTUALIZATION"
	DefenseRegression      DefenseMechanismType = "REGRESSION"
	DefenseDissociation    DefenseMechanismType = "DISSOCIATION"
)

func (d DefenseMechanismType) String() string { return string(d) }

func (d DefenseMechanismType) IsValid() bool {
	switch d {
	case DefenseDenial, DefenseProjection, DefenseRationalization, DefenseDisplacement,
		DefenseRepression, DefenseHumor, DefenseSublimation, DefenseIntellectualize,
		DefenseRegression, DefenseDissociation:
		return true
	default:
		return false
	}
}

// DefenseMechanism stores a Poble's primary psychological defense.
type DefenseMechanism struct {
	// Primary is the default defense under stress.
	Primary DefenseMechanismType `json:"primary"`
	// Secondary activates when Primary fails.
	Secondary DefenseMechanismType `json:"secondary"`
	// Threshold is the emotional intensity (0-100) that activates the defense.
	Threshold float32 `json:"threshold"`
	// Strength from 0 (weak, easily overwhelmed) to 100 (impenetrable).
	Strength float32 `json:"strength"`
}

func NewDefenseMechanism() DefenseMechanism {
	return DefenseMechanism{
		Primary:   DefenseDenial,
		Secondary: DefenseRationalization,
		Threshold: 60,
		Strength:  50,
	}
}

func (d DefenseMechanism) IsValid() bool {
	return d.Primary.IsValid() && d.Secondary.IsValid() && isPercent(d.Threshold) && isPercent(d.Strength)
}

// Superstition stores a belief in a non-rational cause-effect relationship.
type Superstition struct {
	// ID uniquely identifies this superstition.
	ID string `json:"id"`
	// Belief describes the superstitious belief.
	Belief string `json:"belief"`
	// Trigger describes what activates the behavior.
	Trigger string `json:"trigger"`
	// Ritual describes the protective behavior.
	Ritual string `json:"ritual"`
	// Strength from 0 (passing fancy) to 100 (absolute faith).
	Strength float32 `json:"strength"`
	// AffectsDecisions marks whether Strength > 70, meaning it alters behavior.
	AffectsDecisions bool `json:"affects_decisions"`
}

func NewSuperstition(id, belief, trigger, ritual string, strength float32) Superstition {
	return Superstition{
		ID:               id,
		Belief:           belief,
		Trigger:          trigger,
		Ritual:           ritual,
		Strength:         strength,
		AffectsDecisions: strength > 70,
	}
}

func (s Superstition) IsValid() bool {
	return s.ID != "" && s.Belief != "" && isPercent(s.Strength)
}

// SettlementLanguage stores the evolving vocabulary of a settlement.
type SettlementLanguage struct {
	// SlangWords stores settlement-specific slang.
	SlangWords []string `json:"slang_words"`
	// Greetings stores settlement-specific greetings.
	Greetings []string `json:"greetings"`
	// Curses stores settlement-specific curses.
	Curses []string `json:"curses"`
	// FounderInfluence stores the founding Pobles' vocabulary influence.
	FounderInfluence []string `json:"founder_influence"`
}

func NewSettlementLanguage() SettlementLanguage {
	return SettlementLanguage{
		SlangWords:       []string{},
		Greetings:        []string{},
		Curses:           []string{},
		FounderInfluence: []string{},
	}
}

// Item stores inventory item data.
type Item struct {
	// ID stores unique item identifier.
	ID string `json:"id"`
	// Name stores display name.
	Name string `json:"name"`
	// Type stores item category.
	Type string `json:"type"`
	// Quantity stores item count.
	Quantity int `json:"quantity"`
	// Value stores item money value.
	Value int `json:"value"`
	// Tags stores custom item labels.
	Tags []string `json:"tags"`
}

func NewItem(id, name, itemType string, quantity int) Item {
	return Item{
		ID:       id,
		Name:     name,
		Type:     itemType,
		Quantity: quantity,
		Value:    0,
		Tags:     []string{},
	}
}

func (i Item) IsValid() bool {
	return i.ID != "" && i.Name != "" && i.Quantity >= 0 && i.Value >= 0
}

// Poble stores complete character state.
type Poble struct {
	// ID stores unique character identifier.
	ID string `json:"id"`
	// Name stores character display name.
	Name string `json:"name"`
	// Age stores current character age.
	Age int `json:"age"`
	// Sex stores biological sex.
	Sex Sex `json:"sex"`
	// Orientation stores romantic and sexual orientation data.
	Orientation Orientation `json:"orientation"`
	// Archetype stores narrative archetype.
	Archetype ArchetypeID `json:"archetype"`
	// Personality stores trait data.
	Personality Personality `json:"personality"`
	// Appearance stores freeform physical description.
	Appearance string `json:"appearance"`
	// Secrets stores hidden facts.
	Secrets []Secret `json:"secrets"`
	// Memories stores remembered events.
	Memories []Memory `json:"memories"`
	// Relationships stores directed relationships by target Poble ID.
	Relationships map[string]Relationship `json:"relationships"`
	// Health stores physical health data.
	Health HealthState `json:"health"`
	// Mental stores mental health data.
	Mental MentalState `json:"mental"`
	// Needs stores current need data.
	Needs Needs `json:"needs"`
	// Inventory stores carried items.
	Inventory []Item `json:"inventory"`
	// Kinks stores private desire tags used only by behavior systems (legacy, see HiddenPrefs).
	Kinks []string `json:"kinks"`
	// Vocabulary stores speech pattern data.
	Vocabulary VocabularyProfile `json:"vocabulary"`
	// Nicknames stores all nicknames given to this Poble.
	Nicknames []NicknameEntry `json:"nicknames"`
	// Profanity stores swearing habits.
	Profanity ProfanityProfile `json:"profanity"`
	// HiddenPrefs stores structured hidden preferences (replaces flat Kinks).
	HiddenPrefs HiddenPreferences `json:"hidden_prefs"`
	// Quirks stores behavioral quirks.
	Quirks []Quirk `json:"quirks"`
	// LoveLang stores how this Poble gives and receives love.
	LoveLang LoveLanguage `json:"love_lang"`
	// Defense stores primary psychological defense mechanism.
	Defense DefenseMechanism `json:"defense"`
	// Superstitions stores irrational beliefs.
	Superstitions []Superstition `json:"superstitions"`
	// Money stores owned currency.
	Money int `json:"money"`
	// HomeID stores current home identifier.
	HomeID string `json:"home_id"`
	// Children stores child Poble IDs.
	Children []string `json:"children"`
	// Parents stores up to two parent Poble IDs.
	Parents [2]string `json:"parents"`
	// CurrentMood stores current mood label.
	CurrentMood MoodType `json:"current_mood"`
	// EmotionalState stores detailed emotional state.
	EmotionalState EmotionalState `json:"emotional_state"`
	// IsAlive marks whether character is alive.
	IsAlive bool `json:"is_alive"`
	// DayOfBirth stores simulation time of birth.
	DayOfBirth GameTime `json:"day_of_birth"`
}

func NewPoble(id, name string, age int, sex Sex) Poble {
	return Poble{
		ID:             id,
		Name:           name,
		Age:            age,
		Sex:            sex,
		Orientation:    NewOrientation(),
		Archetype:      ArchetypeCustom,
		Personality:    NewPersonality(),
		Appearance:     "",
		Secrets:        []Secret{},
		Memories:       []Memory{},
		Relationships:  map[string]Relationship{},
		Health:         NewHealthState(age),
		Mental:         NewMentalState(),
		Needs:          NewNeeds(),
		Inventory:      []Item{},
		Kinks:          []string{},
		Vocabulary:     NewVocabularyProfile(),
		Nicknames:      []NicknameEntry{},
		Profanity:      NewProfanityProfile(),
		HiddenPrefs:    NewHiddenPreferences(),
		Quirks:         []Quirk{},
		LoveLang:       NewLoveLanguage(),
		Defense:        NewDefenseMechanism(),
		Superstitions:  []Superstition{},
		Money:          0,
		HomeID:         "",
		Children:       []string{},
		Parents:        [2]string{},
		CurrentMood:    MoodNeutral,
		EmotionalState: NewEmotionalState(),
		IsAlive:        true,
		DayOfBirth:     NewGameTime(0, 0, 0),
	}
}

func (p Poble) IsValid() bool {
	return p.ID != "" && p.Name != "" && p.Age >= 0 && p.Sex.IsValid() && p.Orientation.IsValid() &&
		p.Archetype.IsValid() && p.Personality.IsValid() && p.Health.IsValid() && p.Mental.IsValid() &&
		p.Needs.IsValid() && p.CurrentMood.IsValid() && p.EmotionalState.IsValid() && p.DayOfBirth.IsValid()
}

// ThirdPartyType identifies outside help needed for reproduction.
type ThirdPartyType string

const (
	// ThirdPartyNone identifies no outside help.
	ThirdPartyNone ThirdPartyType = "NONE"
	// ThirdPartyDonor identifies a gamete donor path.
	ThirdPartyDonor ThirdPartyType = "DONOR"
	// ThirdPartySurrogate identifies a pregnancy carrier path.
	ThirdPartySurrogate ThirdPartyType = "SURROGATE"
	// ThirdPartyTech identifies assisted reproductive technology.
	ThirdPartyTech ThirdPartyType = "TECH"
)

func (t ThirdPartyType) String() string {
	return string(t)
}

func (t ThirdPartyType) IsValid() bool {
	switch t {
	case ThirdPartyNone, ThirdPartyDonor, ThirdPartySurrogate, ThirdPartyTech:
		return true
	default:
		return false
	}
}

// ReproductionPathType identifies possible routes toward parenthood.
type ReproductionPathType string

const (
	// ReproductionPathNatural identifies biological reproduction between two Pobles.
	ReproductionPathNatural ReproductionPathType = "NATURAL"
	// ReproductionPathDonorNeeded identifies reproduction requiring a donor.
	ReproductionPathDonorNeeded ReproductionPathType = "DONOR_NEEDED"
	// ReproductionPathSurrogateNeeded identifies reproduction requiring a surrogate.
	ReproductionPathSurrogateNeeded ReproductionPathType = "SURROGATE_NEEDED"
	// ReproductionPathAdoption identifies a non-biological adoption route.
	ReproductionPathAdoption ReproductionPathType = "ADOPTION"
	// ReproductionPathTechRequired identifies assisted technology as the route.
	ReproductionPathTechRequired ReproductionPathType = "TECH_REQUIRED"
)

func (r ReproductionPathType) String() string {
	return string(r)
}

func (r ReproductionPathType) IsValid() bool {
	switch r {
	case ReproductionPathNatural, ReproductionPathDonorNeeded, ReproductionPathSurrogateNeeded,
		ReproductionPathAdoption, ReproductionPathTechRequired:
		return true
	default:
		return false
	}
}

// ReproductionPath describes one biologically grounded route toward a child.
type ReproductionPath struct {
	// Type identifies the path category.
	Type ReproductionPathType `json:"type"`
	// Description stores the narrative hook for this path.
	Description string `json:"description"`
	// Availability marks whether this route exists in the current era.
	Availability bool `json:"availability"`
	// DramaScore measures narrative pressure from 0 to 100.
	DramaScore int `json:"drama_score"`
}

// ReproductionAnalysis stores biological and narrative reproduction checks.
type ReproductionAnalysis struct {
	// IsBiologicallyPossible marks whether these two Pobles can reproduce without outside help.
	IsBiologicallyPossible bool `json:"is_biologically_possible"`
	// RequiresThirdParty marks whether a donor, surrogate, or tech is needed.
	RequiresThirdParty bool `json:"requires_third_party"`
	// ThirdPartyType identifies which third-party route is biologically relevant.
	ThirdPartyType ThirdPartyType `json:"third_party_type"`
	// ConsanguinityLevel stores 0 for unrelated, 1 siblings, 2 cousins, 3+ distant kin.
	ConsanguinityLevel int `json:"consanguinity_level"`
	// ConsanguinityRisk stores genetic risk from close relation.
	ConsanguinityRisk float32 `json:"consanguinity_risk"`
	// FertilityChance stores chance from 0.0 to 1.0.
	FertilityChance float32 `json:"fertility_chance"`
	// AlternativePaths stores available narrative routes.
	AlternativePaths []ReproductionPath `json:"alternative_paths"`
}

// GeneticTrait identifies inherited physical and light epigenetic traits.
type GeneticTrait string

const (
	// GeneticTraitBuild identifies inherited body robustness.
	GeneticTraitBuild GeneticTrait = "BUILD"
	// GeneticTraitImmunity identifies inherited immune resilience.
	GeneticTraitImmunity GeneticTrait = "IMMUNITY"
	// GeneticTraitFertility identifies inherited fertility tendency.
	GeneticTraitFertility GeneticTrait = "FERTILITY"
	// GeneticTraitStressResponse identifies inherited stress response.
	GeneticTraitStressResponse GeneticTrait = "STRESS_RESPONSE"
	// GeneticTraitOpenness identifies light epigenetic openness tendency.
	GeneticTraitOpenness GeneticTrait = "OPENNESS"
	// GeneticTraitTemperament identifies inherited baseline temperament.
	GeneticTraitTemperament GeneticTrait = "TEMPERAMENT"
)

func (g GeneticTrait) String() string {
	return string(g)
}

func (g GeneticTrait) IsValid() bool {
	switch g {
	case GeneticTraitBuild, GeneticTraitImmunity, GeneticTraitFertility,
		GeneticTraitStressResponse, GeneticTraitOpenness, GeneticTraitTemperament:
		return true
	default:
		return false
	}
}

// GeneticRisk stores one possible descriptive inherited condition.
type GeneticRisk struct {
	// ID stores stable condition identifier.
	ID string `json:"id"`
	// Description explains the condition in narrative terms.
	Description string `json:"description"`
	// Probability stores risk from 0.0 to 1.0.
	Probability float32 `json:"probability"`
}

// Genetics stores inherited trait data.
type Genetics struct {
	// TraitMap stores inherited trait values from 0.0 to 1.0.
	TraitMap map[GeneticTrait]float32 `json:"trait_map"`
	// RecessiveRisks stores possible recessive conditions.
	RecessiveRisks []GeneticRisk `json:"recessive_risks"`
	// InbreedingCoefficient stores relatedness pressure from 0.0 upward.
	InbreedingCoefficient float32 `json:"inbreeding_coefficient"`
}

// PregnancyArc stores a reproduction storyline without importing event systems.
type PregnancyArc struct {
	// ParentIDs stores the intended family IDs.
	ParentIDs [2]string `json:"parent_ids"`
	// DonorID stores biological donor ID when relevant.
	DonorID string `json:"donor_id"`
	// RecipientID stores intended recipient or carrier ID.
	RecipientID string `json:"recipient_id"`
	// RequiresConsentDrama marks unresolved donor consent.
	RequiresConsentDrama bool `json:"requires_consent_drama"`
	// ChildMayDiscoverDonor marks future generational drama.
	ChildMayDiscoverDonor bool `json:"child_may_discover_donor"`
	// RelationshipPressure stores narrative pressure from 0 to 100.
	RelationshipPressure int `json:"relationship_pressure"`
}

// GameEvent stores a lightweight entity-level event.
type GameEvent struct {
	// ID uniquely identifies this event.
	ID string `json:"id"`
	// Type categorizes the event.
	Type string `json:"type"`
	// Timestamp records when the event happened.
	Timestamp GameTime `json:"timestamp"`
	// Participants stores involved Poble IDs.
	Participants []string `json:"participants"`
	// IsPublic marks whether the event is public knowledge.
	IsPublic bool `json:"is_public"`
	// Description stores narrative event text.
	Description string `json:"description"`
	// DramaScore measures narrative pressure from 0 to 100.
	DramaScore int `json:"drama_score"`
}

// World is the minimal entity-level world data reproduction needs.
type World struct {
	// State stores public world state.
	State WorldState `json:"state"`
	// Pobles stores living and known Pobles by ID.
	Pobles map[string]*Poble `json:"pobles"`
	// Events stores lightweight entity-level events.
	Events []GameEvent `json:"events"`
}

// Era identifies world progression phase.
type Era string

const (
	// EraZero identifies initial era.
	EraZero Era = "ERA_ZERO"
	// EraOne identifies first progression era.
	EraOne Era = "ERA_ONE"
	// EraTwo identifies second progression era.
	EraTwo Era = "ERA_TWO"
	// EraThree identifies third progression era.
	EraThree Era = "ERA_THREE"
	// EraFour identifies fourth progression era.
	EraFour Era = "ERA_FOUR"
)

func (e Era) String() string {
	return string(e)
}

func (e Era) IsValid() bool {
	switch e {
	case EraZero, EraOne, EraTwo, EraThree, EraFour:
		return true
	default:
		return false
	}
}

// TechID identifies technology unlocks.
type TechID string

func (t TechID) String() string {
	return string(t)
}

func (t TechID) IsValid() bool {
	return t != ""
}

// TechTree stores unlocked technologies.
type TechTree struct {
	// Unlocked stores technology unlock state by TechID.
	Unlocked map[TechID]bool `json:"unlocked"`
}

func NewTechTree() TechTree {
	return TechTree{Unlocked: map[TechID]bool{}}
}

func (t TechTree) IsValid() bool {
	return t.Unlocked != nil
}

// Settlement stores basic settlement state.
type Settlement struct {
	// ID stores unique settlement identifier.
	ID string `json:"id"`
	// Name stores settlement display name.
	Name string `json:"name"`
	// Population stores settlement population count.
	Population int `json:"population"`
	// IslandID stores parent island identifier.
	IslandID string `json:"island_id"`
	// LeaderID stores leader Poble ID.
	LeaderID string `json:"leader_id"`
	// Tags stores custom settlement labels.
	Tags []string `json:"tags"`
	// Language stores the evolving vocabulary of this settlement.
	Language SettlementLanguage `json:"language"`
}

func NewSettlement(id, name, islandID string) Settlement {
	return Settlement{
		ID:         id,
		Name:       name,
		Population: 0,
		IslandID:   islandID,
		LeaderID:   "",
		Tags:       []string{},
		Language:   NewSettlementLanguage(),
	}
}

func (s Settlement) IsValid() bool {
	return s.ID != "" && s.Name != "" && s.Population >= 0
}

// Island stores basic island state.
type Island struct {
	// ID stores unique island identifier.
	ID string `json:"id"`
	// Name stores island display name.
	Name string `json:"name"`
	// Size stores abstract island size.
	Size int `json:"size"`
	// Biome stores island biome label.
	Biome string `json:"biome"`
	// SettlementIDs stores settlement IDs on this island.
	SettlementIDs []string `json:"settlement_ids"`
	// Tags stores custom island labels.
	Tags []string `json:"tags"`
}

func NewIsland(id, name string) Island {
	return Island{
		ID:            id,
		Name:          name,
		Size:          0,
		Biome:         "",
		SettlementIDs: []string{},
		Tags:          []string{},
	}
}

func (i Island) IsValid() bool {
	return i.ID != "" && i.Name != "" && i.Size >= 0
}

// WorldState stores global save data.
type WorldState struct {
	// Era stores current world era.
	Era Era `json:"era"`
	// Population stores total living population.
	Population int `json:"population"`
	// Day stores current world time.
	Day GameTime `json:"day"`
	// Settlements stores known settlements.
	Settlements []Settlement `json:"settlements"`
	// Islands stores known islands.
	Islands []Island `json:"islands"`
	// TechTree stores technology progression.
	TechTree TechTree `json:"tech_tree"`
}

func NewWorldState() WorldState {
	return WorldState{
		Era:         EraZero,
		Population:  0,
		Day:         NewGameTime(0, 0, 0),
		Settlements: []Settlement{},
		Islands:     []Island{},
		TechTree:    NewTechTree(),
	}
}

func (w WorldState) IsValid() bool {
	return w.Era.IsValid() && w.Population >= 0 && w.Day.IsValid() && w.TechTree.IsValid()
}

func isUnit(value float32) bool {
	return value >= 0.0 && value <= 1.0
}

func isSignedUnit(value float32) bool {
	return value >= -1.0 && value <= 1.0
}

func isPercent(value float32) bool {
	return value >= 0.0 && value <= 100.0
}
