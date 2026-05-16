package entities

import (
	"errors"
	"fmt"
	"math"
	"math/rand"
	"strings"
	"time"
)

// PoblConfig stores player input for procedural Poble generation.
type PoblConfig struct {
	// Name stores optional player-provided name.
	Name string `json:"name"`
	// Sex stores optional player-selected sex.
	Sex *Sex `json:"sex"`
	// Archetype stores optional player-selected archetype.
	Archetype *ArchetypeID `json:"archetype"`
	// RomanticOrientation stores optional romantic orientation value.
	RomanticOrientation *float32 `json:"romantic_orientation"`
	// SexualOrientation stores optional sexual orientation value.
	SexualOrientation *float32 `json:"sexual_orientation"`
	// InspirationNotes stores freeform player notes.
	InspirationNotes string `json:"inspiration_notes"`
	// AgeRange stores inclusive min and max age.
	AgeRange [2]int `json:"age_range"`
}

// ChemistryScore describes narrative chemistry between two Pobles.
type ChemistryScore struct {
	// AttractionPotential measures likely attraction from 0 to 100.
	AttractionPotential float32 `json:"attraction_potential"`
	// ConflictPotential measures likely friction from 0 to 100.
	ConflictPotential float32 `json:"conflict_potential"`
	// LongTermCompatibility measures durable compatibility from 0 to 100.
	LongTermCompatibility float32 `json:"long_term_compatibility"`
	// DramaPotential measures useful narrative tension from 0 to 100.
	DramaPotential float32 `json:"drama_potential"`
	// Description summarizes the relationship dynamic.
	Description string `json:"description"`
}

type weightedArchetype struct {
	id     ArchetypeID
	weight float64
}

type weightedSecret struct {
	id     SecretType
	weight float64
}

// GeneratePople creates a procedural Poble with correlated traits.
func GeneratePople(config PoblConfig, rng *rand.Rand) (*Poble, error) {
	if rng == nil {
		rng = rand.New(rand.NewSource(time.Now().UnixNano()))
	}

	if err := validatePoblConfig(config); err != nil {
		return nil, err
	}

	sex := chooseSex(config, rng)
	archetype := chooseArchetype(config, rng)
	age := chooseAge(config, rng)
	opennessSeed := trait(rng)
	orientation := generateOrientation(config, rng, opennessSeed)
	personality := generatePersonality(archetype, rng, opennessSeed)
	orientation.Fluidity = clampUnit((personality.Openness/100)*0.65 + float32(rng.NormFloat64()*0.08))
	if archetype == ArchetypeLover {
		orientation.Intensity = clampUnit(orientation.Intensity + 0.15)
	}

	name := strings.TrimSpace(config.Name)
	if name == "" {
		name = generateName(personality, rng)
	}

	poble := NewPoble(generateID("poble", rng), name, age, sex)
	poble.Orientation = orientation
	poble.Archetype = archetype
	poble.Personality = personality
	poble.Appearance = generateAppearance(sex, age, personality, rng)
	poble.Secrets = generateSecrets(archetype, personality, config.InspirationNotes, rng)
	poble.Needs = generateNeeds(archetype, rng)
	poble.EmotionalState = generateEmotionalState(personality, rng)
	poble.CurrentMood = poble.EmotionalState.CurrentMood
	poble.Kinks = generateKinks(personality, rng)
	poble.Vocabulary = generateVocabulary(personality, archetype, rng)
	poble.Nicknames = generateNicknames(personality, rng)
	poble.Profanity = generateProfanity(personality, archetype, rng)
	poble.HiddenPrefs = generateHiddenPreferences(personality, poble.Kinks, rng)
	poble.Quirks = generateQuirks(personality, rng)
	poble.LoveLang = generateLoveLanguage(personality, archetype, rng)
	poble.Defense = generateDefenseMechanism(personality, archetype, rng)
	poble.Superstitions = generateSuperstitions(personality, rng)
	poble.DayOfBirth = NewGameTime(0, 0, 0).Add(-age * 365 * 24)

	return &poble, nil
}

// GenerateStartingPair creates two Pobles with useful narrative chemistry.
func GenerateStartingPair(config1, config2 PoblConfig) ([2]*Poble, error) {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	return generateStartingPairWithRand(config1, config2, rng)
}

func generateStartingPairWithRand(config1, config2 PoblConfig, rng *rand.Rand) ([2]*Poble, error) {
	first, err := GeneratePople(config1, rng)
	if err != nil {
		return [2]*Poble{}, err
	}

	second, err := GeneratePople(config2, rng)
	if err != nil {
		return [2]*Poble{}, err
	}

	score := assessChemistry(first, second)
	adjustPairForDrama(first, second, score, rng)
	ensurePairSecret(first, second, rng)
	linkStartingPair(first, second)

	return [2]*Poble{first, second}, nil
}

func assessChemistry(a, b *Poble) ChemistryScore {
	if a == nil || b == nil {
		return ChemistryScore{Description: "missing chemistry data"}
	}

	attractionA := attractionToward(a, b)
	attractionB := attractionToward(b, a)
	attraction := (attractionA + attractionB) / 2

	conflict := clampPercent(abs32(a.Personality.Agreeableness-b.Personality.Agreeableness)*0.25 +
		abs32(a.Personality.Ambition-b.Personality.Ambition)*0.2 +
		(a.Personality.Cruelty+b.Personality.Cruelty)*0.18 +
		(a.Personality.Jealousy+b.Personality.Jealousy)*0.18 +
		abs32(a.Personality.Conscientiousness-b.Personality.Conscientiousness)*0.14)

	compatibility := clampPercent(100 -
		abs32(a.Personality.Loyalty-b.Personality.Loyalty)*0.22 -
		abs32(a.Personality.Agreeableness-b.Personality.Agreeableness)*0.2 -
		abs32(a.Personality.Openness-b.Personality.Openness)*0.12 -
		(a.Personality.Cruelty+b.Personality.Cruelty)*0.12 +
		(a.Personality.Conscientiousness+b.Personality.Conscientiousness)*0.08)

	drama := clampPercent((attraction * 0.35) + (conflict * 0.45) + ((100 - abs32(attraction-conflict)) * 0.2))
	description := describeChemistry(attraction, conflict, compatibility)

	return ChemistryScore{
		AttractionPotential:   attraction,
		ConflictPotential:     conflict,
		LongTermCompatibility: compatibility,
		DramaPotential:        drama,
		Description:           description,
	}
}

func validatePoblConfig(config PoblConfig) error {
	minAge, maxAge := config.AgeRange[0], config.AgeRange[1]
	if minAge == 0 && maxAge == 0 {
		return nil
	}
	if minAge < 0 || maxAge < 0 || minAge > maxAge {
		return fmt.Errorf("invalid age range: [%d, %d]", minAge, maxAge)
	}
	if config.Sex != nil && !config.Sex.IsValid() {
		return fmt.Errorf("invalid sex: %s", config.Sex.String())
	}
	if config.Archetype != nil && (!config.Archetype.IsValid() || *config.Archetype == ArchetypeCustom) {
		return fmt.Errorf("invalid generated archetype: %s", config.Archetype.String())
	}
	if config.RomanticOrientation != nil && !isUnit(*config.RomanticOrientation) {
		return errors.New("romantic orientation must be between 0.0 and 1.0")
	}
	if config.SexualOrientation != nil && !isUnit(*config.SexualOrientation) {
		return errors.New("sexual orientation must be between 0.0 and 1.0")
	}
	return nil
}

func chooseSex(config PoblConfig, rng *rand.Rand) Sex {
	if config.Sex != nil {
		return *config.Sex
	}

	roll := rng.Float64()
	switch {
	case roll < 0.485:
		return Male
	case roll < 0.97:
		return Female
	default:
		return Intersex
	}
}

func chooseAge(config PoblConfig, rng *rand.Rand) int {
	minAge, maxAge := config.AgeRange[0], config.AgeRange[1]
	if minAge == 0 && maxAge == 0 {
		minAge, maxAge = 18, 55
	}
	return minAge + rng.Intn(maxAge-minAge+1)
}

func chooseArchetype(config PoblConfig, rng *rand.Rand) ArchetypeID {
	if config.Archetype != nil {
		return *config.Archetype
	}

	weights := []weightedArchetype{
		{ArchetypeRuler, 7}, {ArchetypeLover, 8}, {ArchetypeJester, 7},
		{ArchetypeSage, 7}, {ArchetypeRebel, 7}, {ArchetypeCaretaker, 7},
		{ArchetypeVillain, 5}, {ArchetypeGhost, 6}, {ArchetypeAddict, 6},
		{ArchetypeProphet, 5}, {ArchetypeSchemer, 6}, {ArchetypeInnocent, 7},
		{ArchetypeWarrior, 8}, {ArchetypeDrifter, 7}, {ArchetypeMirror, 7},
	}

	total := 0.0
	for i := range weights {
		weights[i].weight *= 0.85 + rng.Float64()*0.3
		total += weights[i].weight
	}

	roll := rng.Float64() * total
	for _, weighted := range weights {
		roll -= weighted.weight
		if roll <= 0 {
			return weighted.id
		}
	}
	return ArchetypeInnocent
}

func generateOrientation(config PoblConfig, rng *rand.Rand, opennessSeed float32) Orientation {
	romantic := float32(0.0)
	if rng.Float64() < 0.10 {
		romantic = clampUnit(float32(0.45 + rng.NormFloat64()*0.28))
	} else {
		romantic = clampUnit(float32(math.Abs(rng.NormFloat64()) * 0.08))
	}

	sexual := romantic
	if rng.Float64() < 0.25 {
		sexual = clampUnit(romantic + float32(rng.NormFloat64()*0.32))
	}

	if config.RomanticOrientation != nil {
		romantic = *config.RomanticOrientation
	}
	if config.SexualOrientation != nil {
		sexual = *config.SexualOrientation
	}

	return Orientation{
		Romantic:  romantic,
		Sexual:    sexual,
		Intensity: clampUnit(float32(0.45 + rng.NormFloat64()*0.22)),
		Fluidity:  clampUnit((opennessSeed/100)*0.65 + float32(rng.NormFloat64()*0.08)),
	}
}

func generatePersonality(archetype ArchetypeID, rng *rand.Rand, opennessSeed float32) Personality {
	openness := opennessSeed
	neuroticism := trait(rng)
	ambition := trait(rng)
	agreeableness := clampPercent(trait(rng) - (ambition-50)*0.18)
	conscientiousness := clampPercent(trait(rng) - (neuroticism-50)*0.25)
	extraversion := trait(rng)
	horniness := trait(rng)
	cruelty := clampPercent(24 + (50-agreeableness)*0.45 + (ambition-50)*0.22 + float32(rng.NormFloat64()*12))
	jealousy := clampPercent(34 + neuroticism*0.48 + (horniness-50)*0.08 + float32(rng.NormFloat64()*10))
	loyalty := clampPercent(55 + (agreeableness-50)*0.42 - (cruelty-45)*0.18 + float32(rng.NormFloat64()*9))

	personality := Personality{
		Openness:          openness,
		Conscientiousness: conscientiousness,
		Extraversion:      extraversion,
		Agreeableness:     agreeableness,
		Neuroticism:       neuroticism,
		Cruelty:           cruelty,
		Horniness:         horniness,
		Ambition:          ambition,
		Jealousy:          jealousy,
		Loyalty:           loyalty,
	}

	applyArchetypePull(&personality, archetype, rng)
	enforceCorrelations(&personality)
	return personality
}

func applyArchetypePull(p *Personality, archetype ArchetypeID, rng *rand.Rand) {
	pull := func(value *float32, amount float32, probability float64) {
		if rng.Float64() <= probability {
			*value = clampPercent(*value + amount + float32(rng.NormFloat64()*4))
		}
	}

	switch archetype {
	case ArchetypeRuler:
		pull(&p.Ambition, 20, 0.9)
		pull(&p.Cruelty, 15, 0.65)
		pull(&p.Agreeableness, -10, 0.75)
	case ArchetypeLover:
		pull(&p.Horniness, 25, 0.9)
		pull(&p.Neuroticism, 20, 0.7)
		pull(&p.Agreeableness, 10, 0.55)
	case ArchetypeJester:
		pull(&p.Extraversion, 20, 0.85)
		pull(&p.Conscientiousness, -15, 0.75)
		pull(&p.Neuroticism, 10, 0.5)
	case ArchetypeSage:
		pull(&p.Openness, 25, 0.9)
		pull(&p.Conscientiousness, 12, 0.65)
		pull(&p.Extraversion, -12, 0.65)
	case ArchetypeRebel:
		pull(&p.Openness, 16, 0.7)
		pull(&p.Conscientiousness, -20, 0.8)
		pull(&p.Agreeableness, -8, 0.55)
	case ArchetypeCaretaker:
		pull(&p.Agreeableness, 25, 0.9)
		pull(&p.Loyalty, 20, 0.85)
		pull(&p.Ambition, -8, 0.55)
	case ArchetypeVillain:
		pull(&p.Cruelty, 30, 0.95)
		pull(&p.Ambition, 25, 0.9)
		pull(&p.Agreeableness, -35, 0.95)
	case ArchetypeGhost:
		pull(&p.Extraversion, -25, 0.9)
		pull(&p.Neuroticism, 14, 0.65)
		pull(&p.Openness, 10, 0.55)
	case ArchetypeAddict:
		pull(&p.Neuroticism, 20, 0.85)
		pull(&p.Conscientiousness, -22, 0.85)
		pull(&p.Horniness, 12, 0.45)
	case ArchetypeProphet:
		pull(&p.Openness, 20, 0.8)
		pull(&p.Ambition, 16, 0.75)
		pull(&p.Agreeableness, -10, 0.55)
	case ArchetypeSchemer:
		pull(&p.Ambition, 18, 0.85)
		pull(&p.Conscientiousness, 18, 0.75)
		pull(&p.Agreeableness, -16, 0.75)
	case ArchetypeInnocent:
		pull(&p.Agreeableness, 28, 0.9)
		pull(&p.Cruelty, -25, 0.95)
		pull(&p.Neuroticism, 10, 0.55)
	case ArchetypeWarrior:
		pull(&p.Loyalty, 20, 0.85)
		pull(&p.Cruelty, 10, 0.45)
		pull(&p.Conscientiousness, 15, 0.7)
	case ArchetypeDrifter:
		pull(&p.Openness, 18, 0.75)
		pull(&p.Conscientiousness, -25, 0.9)
		pull(&p.Loyalty, -10, 0.6)
	case ArchetypeMirror:
		pull(&p.Agreeableness, 10, 0.5)
		pull(&p.Openness, 18, 0.75)
		pull(&p.Neuroticism, 12, 0.6)
	}
}

func enforceCorrelations(p *Personality) {
	p.Jealousy = clampPercent(p.Jealousy + (p.Neuroticism-50)*0.18)
	p.Conscientiousness = clampPercent(p.Conscientiousness - (p.Neuroticism-50)*0.08)
	p.Cruelty = clampPercent(p.Cruelty + (50-p.Agreeableness)*0.22 + (p.Ambition-50)*0.12)
	p.Loyalty = clampPercent(p.Loyalty + (p.Agreeableness-50)*0.18 - (p.Cruelty-50)*0.08)
	p.Agreeableness = clampPercent(p.Agreeableness - (p.Ambition-50)*0.06)
}

func generateSecrets(archetype ArchetypeID, p Personality, notes string, rng *rand.Rand) []Secret {
	count := 1 + rng.Intn(3)
	secrets := make([]Secret, 0, count)
	used := map[SecretType]bool{}

	for len(secrets) < count {
		secretType := chooseSecretType(archetype, p, rng)
		if used[secretType] {
			continue
		}
		used[secretType] = true
		content := secretContent(secretType, archetype, p, notes, rng)
		secrets = append(secrets, NewSecret(generateID("secret", rng), secretType, content))
	}

	return secrets
}

func chooseSecretType(archetype ArchetypeID, p Personality, rng *rand.Rand) SecretType {
	options := []weightedSecret{
		{SecretPastRelationship, 8}, {SecretHiddenSkill, 8}, {SecretDarkDesire, 8},
		{SecretTrueOrientation, 7}, {SecretPlannedBetrayal, 6}, {SecretTraumaEvent, 8},
		{SecretObsession, 7}, {SecretChild, 3}, {SecretPhobia, 6}, {SecretCriminalAct, 5},
	}

	if p.Cruelty > 60 && p.Agreeableness < 40 {
		boostSecret(options, SecretCriminalAct, 18)
		boostSecret(options, SecretDarkDesire, 16)
	}
	if p.Neuroticism > 60 {
		boostSecret(options, SecretTraumaEvent, 18)
		boostSecret(options, SecretPhobia, 10)
	}
	if p.Horniness > 65 && p.Conscientiousness < 45 {
		boostSecret(options, SecretTrueOrientation, 18)
		boostSecret(options, SecretPastRelationship, 8)
	}

	switch archetype {
	case ArchetypeJester:
		boostSecret(options, SecretTraumaEvent, 20)
	case ArchetypeSchemer:
		boostSecret(options, SecretPlannedBetrayal, 18)
	case ArchetypeLover:
		boostSecret(options, SecretPastRelationship, 12)
		boostSecret(options, SecretObsession, 12)
	case ArchetypeAddict:
		boostSecret(options, SecretDarkDesire, 16)
	case ArchetypeVillain:
		boostSecret(options, SecretCriminalAct, 25)
	case ArchetypeProphet:
		boostSecret(options, SecretObsession, 14)
	case ArchetypeCaretaker:
		boostSecret(options, SecretHiddenSkill, 10)
	case ArchetypeGhost:
		boostSecret(options, SecretTraumaEvent, 16)
	}

	return weightedSecretPick(options, rng)
}

func boostSecret(options []weightedSecret, target SecretType, amount float64) {
	for i := range options {
		if options[i].id == target {
			options[i].weight += amount
			return
		}
	}
}

func weightedSecretPick(options []weightedSecret, rng *rand.Rand) SecretType {
	total := 0.0
	for _, option := range options {
		total += option.weight
	}

	roll := rng.Float64() * total
	for _, option := range options {
		roll -= option.weight
		if roll <= 0 {
			return option.id
		}
	}
	return SecretTraumaEvent
}

func secretContent(secretType SecretType, archetype ArchetypeID, p Personality, notes string, rng *rand.Rand) string {
	noteHint := ""
	if strings.TrimSpace(notes) != "" && rng.Float64() < 0.35 {
		noteHint = " tied to something the player hinted at"
	}

	contents := map[SecretType][]string{
		SecretPastRelationship: {"keeps a name from before the end hidden", "still writes letters to someone gone", "recognizes an old lover in every silence"},
		SecretHiddenSkill:      {"knows field medicine but avoids admitting it", "can repair radios and pretends it was luck", "understands maps better than anyone suspects"},
		SecretDarkDesire:       {"wants control badly enough to fear themself", "imagines being obeyed and hates how good it feels", "keeps a cruel wish polished in private"},
		SecretTrueOrientation:  {"performs a safer version of desire in public", "has attraction patterns they do not want named", "knows their wanting is less simple than they say"},
		SecretPlannedBetrayal:  {"has a backup plan that leaves someone behind", "is quietly testing who can be used", "keeps an exit route no one else knows"},
		SecretTraumaEvent:      {"froze when someone needed help before the collapse", "survived a room they still smells in dreams", "remembers the exact sound of the last evacuation"},
		SecretObsession:        {"tracks one person with frightening precision", "has assigned meaning to a coincidence", "keeps returning to the same impossible idea"},
		SecretChild:            {"believes a child may still be alive somewhere", "lost proof of a child and cannot stop searching", "keeps a child's name folded inside clothing"},
		SecretPhobia:           {"cannot stand locked doors", "panics when the lights cut out", "lies about why deep water makes them shake"},
		SecretCriminalAct:      {"did something unforgivable to survive", "stole medicine from someone weaker", "knows exactly where a body was left"},
	}

	pool := contents[secretType]
	content := pool[rng.Intn(len(pool))]
	if archetype == ArchetypeJester && secretType == SecretTraumaEvent {
		content = "uses jokes to keep old grief from speaking clearly"
	}
	if p.Cruelty > 75 && secretType == SecretCriminalAct {
		content = "felt less remorse than they expected after hurting someone"
	}
	return content + noteHint
}

func generateNeeds(archetype ArchetypeID, rng *rand.Rand) Needs {
	needs := Needs{
		Hunger:    needMid(rng),
		Thirst:    needMid(rng),
		Sleep:     needMid(rng),
		Safety:    needMid(rng),
		Belonging: needMid(rng),
		Esteem:    needMid(rng),
		Sex:       needMid(rng),
		Power:     needMid(rng),
		Purpose:   needMid(rng),
	}

	switch archetype {
	case ArchetypeRuler:
		needs.Power = 60 + rng.Float32()*20
		needs.Esteem = clampPercent(needs.Esteem + 15)
	case ArchetypeLover:
		needs.Belonging = clampPercent(needs.Belonging + 20)
		needs.Sex = clampPercent(needs.Sex + 18)
	case ArchetypeCaretaker:
		needs.Belonging = clampPercent(needs.Belonging + 18)
		needs.Purpose = clampPercent(needs.Purpose + 16)
	case ArchetypeVillain:
		needs.Power = clampPercent(needs.Power + 25)
	case ArchetypeGhost:
		needs.Belonging = clampPercent(needs.Belonging - 18)
		needs.Safety = clampPercent(needs.Safety + 18)
	case ArchetypeAddict:
		setAddictNeed(&needs, rng)
	case ArchetypeProphet:
		needs.Purpose = clampPercent(needs.Purpose + 28)
	case ArchetypeWarrior:
		needs.Safety = clampPercent(needs.Safety + 20)
		needs.Purpose = clampPercent(needs.Purpose + 12)
	}

	return needs
}

func needMid(rng *rand.Rand) float32 {
	return clampPercent(float32(45 + rng.NormFloat64()*13))
}

func setAddictNeed(needs *Needs, rng *rand.Rand) {
	value := float32(90 + rng.Float32()*10)
	switch rng.Intn(5) {
	case 0:
		needs.Sex = value
	case 1:
		needs.Power = value
	case 2:
		needs.Belonging = value
	case 3:
		needs.Esteem = value
	default:
		needs.Purpose = value
	}
}

func generateEmotionalState(p Personality, rng *rand.Rand) EmotionalState {
	state := NewEmotionalState()
	state.Valence = clampSignedUnit(-0.25 - (p.Neuroticism / 100 * 0.35) + float32(rng.NormFloat64()*0.12))
	state.Arousal = clampSignedUnit(-0.05 + (p.Neuroticism / 100 * 0.75) + float32(rng.NormFloat64()*0.18))
	state.Dominance = clampSignedUnit((p.Ambition-p.Neuroticism)/120 + float32(rng.NormFloat64()*0.12))
	state.ActiveEmotions = []EmotionType{EmotionGrief}

	if p.Neuroticism > 65 {
		state.ActiveEmotions = append(state.ActiveEmotions, EmotionFear, EmotionAnxiety)
		state.CurrentMood = MoodAnxious
	} else if p.Agreeableness > 70 {
		state.ActiveEmotions = append(state.ActiveEmotions, EmotionHope)
		state.CurrentMood = MoodSad
	} else if p.Cruelty > 70 {
		state.ActiveEmotions = append(state.ActiveEmotions, EmotionResentment)
		state.CurrentMood = MoodAngry
	} else {
		state.ActiveEmotions = append(state.ActiveEmotions, EmotionLoneliness)
		state.CurrentMood = MoodNeutral
	}

	return state
}

func generateAppearance(sex Sex, age int, p Personality, rng *rand.Rand) string {
	builds := []string{"compact build", "long-limbed frame", "soft shoulders", "broad hands", "narrow posture", "steady stance", "restless posture"}
	faces := []string{"uneven eyebrows", "watchful eyes", "a chipped front tooth", "sun-darkened skin", "a careful half-smile", "tired eyelids", "freckled cheeks", "a crooked nose"}
	hair := []string{"close-cropped hair", "loose dark curls", "silver at the temples", "a shaved side part", "tangled shoulder-length hair", "a practical braid", "thin hair tied back"}
	marks := []string{"small burn scar on one wrist", "ink stain under the nails", "old cut across the knuckles", "salt-cracked lips", "faint limp when tired", "weathered forearms", "nervous bite marks on the thumb"}

	ageText := "adult"
	if age < 25 {
		ageText = "young adult"
	} else if age > 50 {
		ageText = "older adult"
	}

	presence := "measured"
	if p.Extraversion > 70 {
		presence = "bright"
	} else if p.Neuroticism > 70 {
		presence = "tense"
	} else if p.Cruelty > 70 {
		presence = "hard"
	}

	return fmt.Sprintf("%s %s, %s, %s, %s, %s presence",
		ageText, strings.ToLower(sex.String()), pick(builds, rng), pick(faces, rng), pick(hair, rng), presenceWithMark(pick(marks, rng), presence))
}

func presenceWithMark(mark, presence string) string {
	return fmt.Sprintf("%s and %s", mark, presence)
}

func generateKinks(p Personality, rng *rand.Rand) []string {
	count := 1
	if p.Horniness > 55 || p.Openness > 60 {
		count++
	}
	if p.Horniness > 75 && p.Openness > 65 {
		count++
	}

	pool := kinkPool()
	kinks := make([]string, 0, count)
	used := map[string]bool{}
	for len(kinks) < count {
		candidate := pool[rng.Intn(len(pool))]
		if used[candidate] {
			continue
		}
		used[candidate] = true
		kinks = append(kinks, candidate)
	}
	return kinks
}

func kinkPool() []string {
	return []string{
		"affectionate praise", "quiet dominance", "ritualized consent", "emotional vulnerability",
		"slow courtship", "danger proximity", "forbidden tenderness", "power exchange",
		"jealous attention", "protective restraint", "intellectual seduction", "competence crush",
		"public composure", "private softness", "devotional service", "being chosen",
		"being challenged", "gentle teasing", "confession intimacy", "role reversal",
		"caretaking", "rival tension", "secret meetings", "slow trust", "mutual survival",
		"authority tension", "surrender of control", "earned praise", "dangerous honesty",
		"possessive language", "soft aftercare", "formal ritual", "messy desire",
		"touch starvation", "being witnessed", "obedience games", "command voice",
		"protective jealousy", "vulnerability after conflict", "competent hands", "rough honesty",
		"gentle restraint", "shared secrecy", "being needed", "being forgiven",
		"devotion under pressure", "mutual corruption", "clean boundaries", "slow dancing",
		"voice fixation", "hands fixation", "scar tenderness", "survival bonding",
		"dangerous loyalty", "forbidden praise", "controlled risk", "tender rivalry",
	}
}

func generateName(p Personality, rng *rand.Rand) string {
	names := []string{
		"Amina", "Mateo", "Sofia", "Ilya", "Noor", "Kenji", "Leila", "Tomas",
		"Zara", "Maya", "Omar", "Anika", "Rafael", "Nadia", "Elias", "Yara",
		"Priya", "Dante", "Mei", "Samir", "Ines", "Luca", "Farah", "Niko",
		"Amara", "Diego", "Sana", "Kaito", "Mara", "Jonas", "Lina", "Tariq",
	}
	nicknames := []string{"Ash", "Blue", "Moth", "Saint", "Nox", "Rue", "Patch", "Vega", "Sol", "Echo", "Kit", "Vale"}

	if p.Extraversion < 35 && rng.Float64() < 0.45 {
		return nicknames[rng.Intn(len(nicknames))]
	}
	return names[rng.Intn(len(names))]
}

func adjustPairForDrama(a, b *Poble, score ChemistryScore, rng *rand.Rand) {
	if score.LongTermCompatibility > 82 && score.ConflictPotential < 35 {
		a.Personality.Jealousy = clampPercent(a.Personality.Jealousy + 18)
		b.Personality.Ambition = clampPercent(b.Personality.Ambition + 15)
	}
	if score.ConflictPotential > 88 && score.AttractionPotential < 20 {
		a.Personality.Agreeableness = clampPercent(a.Personality.Agreeableness + 10)
		b.Personality.Loyalty = clampPercent(b.Personality.Loyalty + 12)
	}

	final := assessChemistry(a, b)
	for final.DramaPotential <= 50 {
		target := a
		if rng.Intn(2) == 0 {
			target = b
		}
		target.Personality.Jealousy = clampPercent(target.Personality.Jealousy + 8)
		target.Personality.Neuroticism = clampPercent(target.Personality.Neuroticism + 5)
		target.Personality.Loyalty = clampPercent(target.Personality.Loyalty + 4)
		final = assessChemistry(a, b)
	}
}

func ensurePairSecret(a, b *Poble, rng *rand.Rand) {
	secretType := SecretObsession
	content := fmt.Sprintf("feels an unsafe pull toward %s and has told no one", b.Name)
	if rng.Float64() < 0.35 {
		secretType = SecretPlannedBetrayal
		content = fmt.Sprintf("has considered abandoning %s if survival demands it", b.Name)
	}

	a.Secrets = append(a.Secrets, NewSecret(generateID("secret", rng), secretType, content))
}

func linkStartingPair(a, b *Poble) {
	score := assessChemistry(a, b)
	relAB := NewRelationship(b.ID, RelationshipAcquaintance)
	relAB.Attraction = score.AttractionPotential
	relAB.Resentment = score.ConflictPotential * 0.35
	relAB.Trust = clampPercent(score.LongTermCompatibility * 0.6)
	relAB.Familiarity = 45
	relAB.Tags = []string{"starting_pair", "last_humans"}

	relBA := NewRelationship(a.ID, RelationshipAcquaintance)
	relBA.Attraction = score.AttractionPotential
	relBA.Resentment = score.ConflictPotential * 0.35
	relBA.Trust = clampPercent(score.LongTermCompatibility * 0.6)
	relBA.Familiarity = 45
	relBA.Tags = []string{"starting_pair", "last_humans"}

	a.Relationships[b.ID] = relAB
	b.Relationships[a.ID] = relBA
}

func attractionToward(source, target *Poble) float32 {
	sameSex := source.Sex == target.Sex
	preferred := float32(0)
	if sameSex {
		preferred = 1
	}

	orientationFit := 100 * (1 - abs32(source.Orientation.Sexual-preferred))
	intensity := 35 + source.Orientation.Intensity*65
	personalityFit := 100 - abs32(source.Personality.Openness-target.Personality.Openness)*0.18 -
		abs32(source.Personality.Horniness-target.Personality.Horniness)*0.12 -
		target.Personality.Cruelty*0.08

	return clampPercent((orientationFit * 0.45) + (intensity * 0.35) + (personalityFit * 0.2))
}

func describeChemistry(attraction, conflict, compatibility float32) string {
	switch {
	case attraction > 65 && conflict > 60:
		return "magnetic and volatile"
	case attraction > 65 && compatibility > 60:
		return "warm but not simple"
	case conflict > 70:
		return "useful friction with survival pressure"
	case compatibility > 75:
		return "stable bond with hidden pressure points"
	default:
		return "uneven alliance with room to turn"
	}
}

func trait(rng *rand.Rand) float32 {
	return clampPercent(float32(50 + rng.NormFloat64()*18))
}

func generateID(prefix string, rng *rand.Rand) string {
	return fmt.Sprintf("%s_%016x", prefix, rng.Uint64())
}

func pick(values []string, rng *rand.Rand) string {
	return values[rng.Intn(len(values))]
}

func abs32(value float32) float32 {
	if value < 0 {
		return -value
	}
	return value
}

func clampUnit(value float32) float32 {
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
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

func clampPercent(value float32) float32 {
	if value < 0 {
		return 0
	}
	if value > 100 {
		return 100
	}
	return value
}

func generateVocabulary(p Personality, archetype ArchetypeID, rng *rand.Rand) VocabularyProfile {
	tier := VocabStandard
	switch {
	case p.Openness > 75 && (archetype == ArchetypeSage || archetype == ArchetypeProphet):
		tier = VocabEloquent
	case p.Openness > 60:
		tier = VocabEducated
	case p.Openness < 30:
		tier = VocabSimple
	case p.Conscientiousness < 25 && p.Extraversion < 35:
		tier = VocabPrimitive
	}

	fillers := [][]string{
		{"bueno", "o sea", "pues", "digamos"},
		{"mira", "la verdad", "como te digo"},
		{"es que", "no se", "sabes"},
	}
	expressions := [][]string{
		{"en fin", "a ver", "la cosa es que"},
		{"eso ya lo veremos", "al final todo da igual"},
		{"uno nunca sabe", "peor seria no intentarlo"},
	}

	filler := fillers[rng.Intn(len(fillers))]
	count := 1 + rng.Intn(2)
	if p.Extraversion > 70 {
		count++
	}
	selectedFillers := make([]string, 0, count)
	for i := 0; i < count && i < len(filler); i++ {
		selectedFillers = append(selectedFillers, filler[i])
	}

	expr := expressions[rng.Intn(len(expressions))]
	selectedExprs := []string{expr[rng.Intn(len(expr))]}

	verbosity := clampPercent(p.Extraversion*0.55 + p.Openness*0.25 + float32(rng.NormFloat64()*12))
	return VocabularyProfile{
		Tier:                tier,
		FillerWords:         selectedFillers,
		FavoriteExpressions: selectedExprs,
		Verbosity:           verbosity,
	}
}

func generateNicknames(p Personality, rng *rand.Rand) []NicknameEntry {
	// Most pobles start with 0-1 nicknames. High extraversion = more likely.
	if rng.Float64() > 0.35+(float64(p.Extraversion)/300) {
		return []NicknameEntry{}
	}
	pool := []string{"Sombra", "Roca", "Chispa", "Polvo", "Sal", "Humo", "Garras", "Ceniza", "Brasa", "Viento"}
	nickname := pool[rng.Intn(len(pool))]
	return []NicknameEntry{NewNicknameEntry(nickname, "", "self-adopted")}
}

func generateProfanity(p Personality, archetype ArchetypeID, rng *rand.Rand) ProfanityProfile {
	level := ProfanityMild
	switch {
	case archetype == ArchetypeInnocent || archetype == ArchetypeProphet:
		level = ProfanityNone
	case p.Cruelty > 70 || (p.Neuroticism > 70 && p.Conscientiousness < 40):
		level = ProfanityHeavy
	case archetype == ArchetypeJester || archetype == ArchetypeWarrior:
		level = ProfanityModerate
	case p.Agreeableness > 75:
		level = ProfanityMild
	}
	if p.Cruelty > 85 && p.Conscientiousness < 30 {
		level = ProfanityExtreme
	}

	mild := []string{"mierda", "joder", "carajo", "demonios", "rayos"}
	strong := []string{"hijo de puta", "la puta madre", "maldita sea", "me cago en todo"}
	expletives := []string{"mierda", "joder", "carajo", "coño", "hostia"}

	mildPick := []string{mild[rng.Intn(len(mild))]}
	strongPick := []string{strong[rng.Intn(len(strong))]}
	signature := expletives[rng.Intn(len(expletives))]

	stress := clampUnit(1.0 + (p.Neuroticism-50)/200)
	return ProfanityProfile{
		Level:              level,
		MildExpletives:     mildPick,
		StrongExpletives:   strongPick,
		SignatureExpletive: signature,
		StressMultiplier:   stress,
	}
}

func generateHiddenPreferences(p Personality, legacyKinks []string, rng *rand.Rand) HiddenPreferences {
	prefs := make([]HiddenPreference, 0, len(legacyKinks))
	categories := []PreferenceCategory{PrefPhysical, PrefEmotional, PrefPower, PrefRitual, PrefTaboo, PrefSensory}
	categoryCounts := map[PreferenceCategory]int{}

	for _, kink := range legacyKinks {
		cat := categorizeKink(kink, rng)
		categoryCounts[cat]++
		prefs = append(prefs, HiddenPreference{
			Name:         kink,
			Category:     cat,
			Intensity:    clampPercent(40 + float32(rng.NormFloat64()*18)),
			IsDiscovered: rng.Float64() < 0.6,
			IsSharedWith: []string{},
		})
	}

	dominant := PrefEmotional
	bestCount := 0
	for _, cat := range categories {
		if categoryCounts[cat] > bestCount {
			dominant = cat
			bestCount = categoryCounts[cat]
		}
	}

	openness := clampPercent(p.Openness*0.6 + p.Horniness*0.3 + float32(rng.NormFloat64()*10))
	return HiddenPreferences{
		Preferences:      prefs,
		DominantCategory: dominant,
		Openness:         openness,
	}
}

func categorizeKink(kink string, rng *rand.Rand) PreferenceCategory {
	lower := strings.ToLower(kink)
	switch {
	case strings.Contains(lower, "touch") || strings.Contains(lower, "hands") || strings.Contains(lower, "physical"):
		return PrefPhysical
	case strings.Contains(lower, "power") || strings.Contains(lower, "dominance") || strings.Contains(lower, "control") || strings.Contains(lower, "command") || strings.Contains(lower, "obedience"):
		return PrefPower
	case strings.Contains(lower, "ritual") || strings.Contains(lower, "formal") || strings.Contains(lower, "devotion"):
		return PrefRitual
	case strings.Contains(lower, "forbidden") || strings.Contains(lower, "secret") || strings.Contains(lower, "taboo") || strings.Contains(lower, "dangerous"):
		return PrefTaboo
	case strings.Contains(lower, "voice") || strings.Contains(lower, "scar") || strings.Contains(lower, "sensory"):
		return PrefSensory
	default:
		return PrefEmotional
	}
}

func generateQuirks(p Personality, rng *rand.Rand) []Quirk {
	count := 1 + rng.Intn(3) // 1-3 quirks
	pool := quirkPool()
	quirks := make([]Quirk, 0, count)
	used := map[string]bool{}

	for len(quirks) < count {
		entry := pool[rng.Intn(len(pool))]
		if used[entry.Description] {
			continue
		}
		used[entry.Description] = true
		q := NewQuirk(generateID("quirk", rng), entry.Type, entry.Description)
		q.Intensity = clampPercent(35 + float32(rng.NormFloat64()*22))
		q.TriggeredByEmotion = entry.TriggeredByEmotion
		q.IsEndearing = rng.Float64() < 0.45+(float64(p.Agreeableness)/400)
		quirks = append(quirks, q)
	}
	return quirks
}

type quirkTemplate struct {
	Type               QuirkType
	Description        string
	TriggeredByEmotion EmotionType
}

func quirkPool() []quirkTemplate {
	return []quirkTemplate{
		{QuirkPhysical, "cracks knuckles when thinking", EmotionAnxiety},
		{QuirkPhysical, "taps foot during conversations", EmotionBoredom},
		{QuirkPhysical, "touches face when lying", EmotionGuilt},
		{QuirkPhysical, "paces back and forth under pressure", EmotionFear},
		{QuirkVerbal, "finishes other people's sentences", EmotionAnxiety},
		{QuirkVerbal, "hums under their breath when concentrating", EmotionCuriosity},
		{QuirkVerbal, "talks to themselves when alone", EmotionLoneliness},
		{QuirkVerbal, "laughs at inappropriate moments", EmotionFear},
		{QuirkEmotional, "tears up during arguments", EmotionAnger},
		{QuirkEmotional, "goes completely silent when hurt", EmotionGrief},
		{QuirkEmotional, "smiles wider when more upset", EmotionResentment},
		{QuirkSocial, "stands too close during conversation", EmotionLove},
		{QuirkSocial, "avoids eye contact with authority", EmotionFear},
		{QuirkSocial, "always positions near exits", EmotionAnxiety},
		{QuirkSocial, "mirrors the posture of whoever they talk to", EmotionTrust},
		{QuirkHabit, "collects small objects from important moments", EmotionHope},
		{QuirkHabit, "counts steps when walking long distances", EmotionAnxiety},
		{QuirkHabit, "sleeps with one eye on the door", EmotionFear},
		{QuirkHabit, "organizes food before eating", EmotionAnxiety},
		{QuirkHabit, "checks locks multiple times", EmotionFear},
	}
}

func generateLoveLanguage(p Personality, archetype ArchetypeID, rng *rand.Rand) LoveLanguage {
	allLangs := []LoveLanguageType{
		LoveLangWordsOfAffirmation, LoveLangActsOfService,
		LoveLangGifts, LoveLangQualityTime, LoveLangPhysicalTouch,
	}

	weights := [5]float64{20, 20, 20, 20, 20}
	// Personality correlations.
	weights[0] += float64(p.Extraversion) * 0.3  // Words: extraverts talk more.
	weights[1] += float64(p.Conscientiousness) * 0.25
	weights[3] += float64(p.Agreeableness) * 0.2
	weights[4] += float64(p.Horniness) * 0.15

	switch archetype {
	case ArchetypeCaretaker:
		weights[1] += 25
	case ArchetypeLover:
		weights[4] += 20
		weights[0] += 15
	case ArchetypeSage:
		weights[3] += 20
	case ArchetypeGhost:
		weights[3] += 25
		weights[4] -= 10
	case ArchetypeJester:
		weights[0] += 20
	}

	primary := weightedLangPick(allLangs, weights[:], rng)
	secondary := primary
	for secondary == primary {
		secondary = allLangs[rng.Intn(len(allLangs))]
	}
	receive := primary
	if rng.Float64() < 0.4 {
		receive = secondary
	}

	return LoveLanguage{Primary: primary, Secondary: secondary, ReceivePreference: receive}
}

func weightedLangPick(langs []LoveLanguageType, weights []float64, rng *rand.Rand) LoveLanguageType {
	total := 0.0
	for _, w := range weights {
		total += math.Max(w, 1)
	}
	roll := rng.Float64() * total
	for i, w := range weights {
		roll -= math.Max(w, 1)
		if roll <= 0 {
			return langs[i]
		}
	}
	return langs[0]
}

func generateDefenseMechanism(p Personality, archetype ArchetypeID, rng *rand.Rand) DefenseMechanism {
	allDefenses := []DefenseMechanismType{
		DefenseDenial, DefenseProjection, DefenseRationalization, DefenseDisplacement,
		DefenseRepression, DefenseHumor, DefenseSublimation, DefenseIntellectualize,
		DefenseRegression, DefenseDissociation,
	}

	primary := DefenseDenial
	switch archetype {
	case ArchetypeJester:
		primary = DefenseHumor
	case ArchetypeSage:
		primary = DefenseIntellectualize
	case ArchetypeSchemer:
		primary = DefenseRationalization
	case ArchetypeGhost:
		primary = DefenseDissociation
	case ArchetypeWarrior:
		primary = DefenseDisplacement
	case ArchetypeAddict:
		primary = DefenseRegression
	case ArchetypeVillain:
		primary = DefenseProjection
	case ArchetypeCaretaker:
		primary = DefenseSublimation
	case ArchetypeInnocent:
		primary = DefenseDenial
	case ArchetypeProphet:
		primary = DefenseSublimation
	case ArchetypeRuler:
		primary = DefenseRationalization
	default:
		primary = allDefenses[rng.Intn(len(allDefenses))]
	}

	secondary := primary
	for secondary == primary {
		secondary = allDefenses[rng.Intn(len(allDefenses))]
	}

	threshold := clampPercent(60 - p.Neuroticism*0.25 + float32(rng.NormFloat64()*10))
	strength := clampPercent(50 + (p.Conscientiousness-50)*0.3 + float32(rng.NormFloat64()*12))

	return DefenseMechanism{
		Primary:   primary,
		Secondary: secondary,
		Threshold: threshold,
		Strength:  strength,
	}
}

func generateSuperstitions(p Personality, rng *rand.Rand) []Superstition {
	// 0-2 superstitions. Higher neuroticism/openness = more likely to have any.
	chance := 0.3 + float64(p.Neuroticism)/400 + float64(p.Openness)/500
	if rng.Float64() > chance {
		return []Superstition{}
	}

	count := 1
	if rng.Float64() < 0.35+(float64(p.Neuroticism)/500) {
		count = 2
	}

	pool := superstitionPool()
	superstitions := make([]Superstition, 0, count)
	used := map[string]bool{}

	for len(superstitions) < count {
		entry := pool[rng.Intn(len(pool))]
		if used[entry[0]] {
			continue
		}
		used[entry[0]] = true
		strength := clampPercent(35 + p.Neuroticism*0.35 + float32(rng.NormFloat64()*18))
		s := NewSuperstition(generateID("superstition", rng), entry[0], entry[1], entry[2], strength)
		superstitions = append(superstitions, s)
	}
	return superstitions
}

func superstitionPool() [][3]string {
	return [][3]string{
		{"the dead watch from water", "passing still water", "avoid looking at own reflection"},
		{"naming the dead draws them back", "saying a dead person's name", "refer to the dead by relationship only"},
		{"sleeping facing north invites sickness", "choosing bed orientation", "always sleep facing east or south"},
		{"spilling salt opens a wound somewhere", "spilling salt", "throw a pinch over the left shoulder"},
		{"whistling after dark calls something hungry", "night whistling", "hum instead of whistling at night"},
		{"counting stars means counting days left alive", "counting stars", "look away from the night sky"},
		{"a bird inside the shelter means death is choosing", "bird entering indoors", "open all exits and stay still"},
		{"odd numbers protect from harm", "arranging objects", "always arrange important things in odd numbers"},
		{"rain during a birth means the child carries luck", "rain during childbirth", "save the first rainwater after birth"},
		{"broken tools mean broken trust nearby", "a tool breaking", "bury the broken tool and replace it silently"},
		{"fire that won't light means a lie was told today", "fire difficulty", "confess something small before trying again"},
		{"left hand first through a door brings conflict", "entering doorways", "always lead with right hand"},
	}
}
