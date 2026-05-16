package entities

import (
	"math/rand"
	"testing"
)

// --- Vocabulary System Tests ---

func TestGenerateVocabulary_SageGetsEloquent(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	p := Personality{Openness: 80, Extraversion: 50, Conscientiousness: 50}
	v := generateVocabulary(p, ArchetypeSage, rng)
	if v.Tier != VocabEloquent {
		t.Errorf("sage with high openness: got tier %s, want ELOQUENT", v.Tier)
	}
	if !v.IsValid() {
		t.Error("vocabulary profile invalid")
	}
}

func TestGenerateVocabulary_LowOpenessGetsSimple(t *testing.T) {
	rng := rand.New(rand.NewSource(99))
	p := Personality{Openness: 20, Extraversion: 40, Conscientiousness: 50}
	v := generateVocabulary(p, ArchetypeWarrior, rng)
	if v.Tier != VocabSimple {
		t.Errorf("low openness: got tier %s, want SIMPLE", v.Tier)
	}
}

func TestGenerateVocabulary_HasFillers(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	p := Personality{Openness: 55, Extraversion: 60, Conscientiousness: 50}
	v := generateVocabulary(p, ArchetypeJester, rng)
	if len(v.FillerWords) == 0 {
		t.Error("expected at least one filler word")
	}
	if len(v.FavoriteExpressions) == 0 {
		t.Error("expected at least one favorite expression")
	}
}

// --- Nickname System Tests ---

func TestGenerateNicknames_HighExtraversion(t *testing.T) {
	// Run many times to verify trend: high extraversion = more nicknames.
	withNickname := 0
	for seed := int64(0); seed < 200; seed++ {
		rng := rand.New(rand.NewSource(seed))
		p := Personality{Extraversion: 90}
		nicks := generateNicknames(p, rng)
		if len(nicks) > 0 {
			withNickname++
		}
	}
	// Should have nicknames more than 40% of the time with high extraversion.
	if withNickname < 80 {
		t.Errorf("high extraversion: only %d/200 pobles got nicknames, want >= 80", withNickname)
	}
}

func TestNicknameEntry_Defaults(t *testing.T) {
	nick := NewNicknameEntry("Sombra", "giver_1", "always hides")
	if nick.Nickname != "Sombra" || nick.GivenBy != "giver_1" || !nick.IsAccepted {
		t.Errorf("nickname defaults: %+v", nick)
	}
	if nick.IsPublic {
		t.Error("new nicknames should not be public by default")
	}
}

func TestNicknameEntry_SelfAdopted(t *testing.T) {
	nick := NewNicknameEntry("Ceniza", "", "self-chosen")
	if nick.GivenBy != "" {
		t.Error("self-adopted nickname should have empty GivenBy")
	}
}

// --- Profanity System Tests ---

func TestGenerateProfanity_InnocentGetsNone(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	p := Personality{Cruelty: 10, Agreeableness: 80, Conscientiousness: 60}
	prof := generateProfanity(p, ArchetypeInnocent, rng)
	if prof.Level != ProfanityNone {
		t.Errorf("innocent archetype: got profanity %s, want NONE", prof.Level)
	}
}

func TestGenerateProfanity_HighCrueltyGetsHeavy(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	p := Personality{Cruelty: 75, Neuroticism: 75, Conscientiousness: 35}
	prof := generateProfanity(p, ArchetypeWarrior, rng)
	if prof.Level != ProfanityHeavy {
		t.Errorf("high cruelty poble: got profanity %s, want HEAVY", prof.Level)
	}
}

func TestGenerateProfanity_HasExpletives(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	p := Personality{Cruelty: 50, Conscientiousness: 50}
	prof := generateProfanity(p, ArchetypeDrifter, rng)
	if len(prof.MildExpletives) == 0 || len(prof.StrongExpletives) == 0 || prof.SignatureExpletive == "" {
		t.Error("profanity profile missing expletives")
	}
	if !prof.IsValid() {
		t.Error("profanity profile invalid")
	}
}

// --- Hidden Preferences Tests ---

func TestGenerateHiddenPreferences_MigratesKinks(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	p := Personality{Openness: 60, Horniness: 60}
	kinks := []string{"quiet dominance", "voice fixation", "gentle restraint"}
	hp := generateHiddenPreferences(p, kinks, rng)
	if len(hp.Preferences) != 3 {
		t.Errorf("expected 3 preferences migrated from kinks, got %d", len(hp.Preferences))
	}
	if !hp.IsValid() {
		t.Error("hidden preferences invalid")
	}
}

func TestCategorizeKink_PowerCategory(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	cat := categorizeKink("quiet dominance", rng)
	if cat != PrefPower {
		t.Errorf("expected PrefPower for 'quiet dominance', got %s", cat)
	}
}

func TestCategorizeKink_TabooCategory(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	cat := categorizeKink("forbidden tenderness", rng)
	if cat != PrefTaboo {
		t.Errorf("expected PrefTaboo for 'forbidden tenderness', got %s", cat)
	}
}

// --- Quirks Tests ---

func TestGenerateQuirks_AlwaysHasOne(t *testing.T) {
	for seed := int64(0); seed < 50; seed++ {
		rng := rand.New(rand.NewSource(seed))
		p := Personality{Agreeableness: 50}
		quirks := generateQuirks(p, rng)
		if len(quirks) == 0 {
			t.Errorf("seed %d: expected at least one quirk", seed)
		}
		for _, q := range quirks {
			if !q.IsValid() {
				t.Errorf("seed %d: quirk invalid: %+v", seed, q)
			}
		}
	}
}

func TestGenerateQuirks_UniqueDescriptions(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	p := Personality{Agreeableness: 50}
	quirks := generateQuirks(p, rng)
	seen := map[string]bool{}
	for _, q := range quirks {
		if seen[q.Description] {
			t.Errorf("duplicate quirk description: %s", q.Description)
		}
		seen[q.Description] = true
	}
}

func TestQuirkPool_AllValid(t *testing.T) {
	for _, q := range quirkPool() {
		if q.Description == "" {
			t.Error("quirk pool entry with empty description")
		}
		if !q.Type.IsValid() {
			t.Errorf("quirk pool entry with invalid type: %s", q.Type)
		}
	}
}

// --- Love Language Tests ---

func TestGenerateLoveLanguage_Valid(t *testing.T) {
	for seed := int64(0); seed < 50; seed++ {
		rng := rand.New(rand.NewSource(seed))
		p := Personality{Extraversion: 50, Conscientiousness: 50, Agreeableness: 50, Horniness: 50}
		ll := generateLoveLanguage(p, ArchetypeDrifter, rng)
		if !ll.IsValid() {
			t.Errorf("seed %d: love language invalid: %+v", seed, ll)
		}
		if ll.Primary == ll.Secondary {
			t.Errorf("seed %d: primary == secondary: %s", seed, ll.Primary)
		}
	}
}

func TestGenerateLoveLanguage_CaretakerPreference(t *testing.T) {
	actsCount := 0
	for seed := int64(0); seed < 200; seed++ {
		rng := rand.New(rand.NewSource(seed))
		p := Personality{Conscientiousness: 80, Agreeableness: 70}
		ll := generateLoveLanguage(p, ArchetypeCaretaker, rng)
		if ll.Primary == LoveLangActsOfService {
			actsCount++
		}
	}
	// CARETAKER should lean toward Acts of Service.
	if actsCount < 40 {
		t.Errorf("caretaker: only %d/200 got ActsOfService primary, want >= 40", actsCount)
	}
}

func TestLoveLanguageType_AllValid(t *testing.T) {
	langs := []LoveLanguageType{
		LoveLangWordsOfAffirmation, LoveLangActsOfService,
		LoveLangGifts, LoveLangQualityTime, LoveLangPhysicalTouch,
	}
	for _, l := range langs {
		if !l.IsValid() {
			t.Errorf("love language type %s should be valid", l)
		}
	}
	if LoveLanguageType("INVALID").IsValid() {
		t.Error("INVALID love language should not be valid")
	}
}

// --- Defense Mechanism Tests ---

func TestGenerateDefenseMechanism_JesterGetsHumor(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	p := Personality{Neuroticism: 50, Conscientiousness: 50}
	d := generateDefenseMechanism(p, ArchetypeJester, rng)
	if d.Primary != DefenseHumor {
		t.Errorf("jester: got defense %s, want HUMOR", d.Primary)
	}
	if !d.IsValid() {
		t.Error("defense mechanism invalid")
	}
}

func TestGenerateDefenseMechanism_PrimaryNeverEqualSecondary(t *testing.T) {
	for seed := int64(0); seed < 50; seed++ {
		rng := rand.New(rand.NewSource(seed))
		p := Personality{Neuroticism: 50, Conscientiousness: 50}
		d := generateDefenseMechanism(p, ArchetypeDrifter, rng)
		if d.Primary == d.Secondary {
			t.Errorf("seed %d: primary == secondary: %s", seed, d.Primary)
		}
	}
}

func TestDefenseMechanismType_AllValid(t *testing.T) {
	types := []DefenseMechanismType{
		DefenseDenial, DefenseProjection, DefenseRationalization, DefenseDisplacement,
		DefenseRepression, DefenseHumor, DefenseSublimation, DefenseIntellectualize,
		DefenseRegression, DefenseDissociation,
	}
	for _, dt := range types {
		if !dt.IsValid() {
			t.Errorf("defense type %s should be valid", dt)
		}
	}
}

// --- Superstitions Tests ---

func TestGenerateSuperstitions_HighNeuroticismMoreLikely(t *testing.T) {
	withSuperstition := 0
	for seed := int64(0); seed < 200; seed++ {
		rng := rand.New(rand.NewSource(seed))
		p := Personality{Neuroticism: 90, Openness: 70}
		sups := generateSuperstitions(p, rng)
		if len(sups) > 0 {
			withSuperstition++
		}
	}
	// High neuroticism + openness should give superstitions > 50% of the time.
	if withSuperstition < 100 {
		t.Errorf("high neuroticism: only %d/200 got superstitions, want >= 100", withSuperstition)
	}
}

func TestGenerateSuperstitions_AllValid(t *testing.T) {
	for seed := int64(0); seed < 50; seed++ {
		rng := rand.New(rand.NewSource(seed))
		p := Personality{Neuroticism: 80, Openness: 60}
		sups := generateSuperstitions(p, rng)
		for _, s := range sups {
			if !s.IsValid() {
				t.Errorf("seed %d: superstition invalid: %+v", seed, s)
			}
		}
	}
}

func TestSuperstitionPool_Complete(t *testing.T) {
	pool := superstitionPool()
	if len(pool) < 10 {
		t.Errorf("expected at least 10 superstition entries, got %d", len(pool))
	}
	for i, entry := range pool {
		if entry[0] == "" || entry[1] == "" || entry[2] == "" {
			t.Errorf("superstition pool entry %d has empty field", i)
		}
	}
}

// --- VocabularyProfile Validation ---

func TestVocabularyProfile_Validation(t *testing.T) {
	v := NewVocabularyProfile()
	if !v.IsValid() {
		t.Error("default vocabulary should be valid")
	}
	v.Verbosity = -5
	if v.IsValid() {
		t.Error("negative verbosity should be invalid")
	}
	v.Verbosity = 50
	v.Tier = VocabularyTier("BROKEN")
	if v.IsValid() {
		t.Error("invalid tier should be invalid")
	}
}

// --- ProfanityProfile Validation ---

func TestProfanityProfile_Validation(t *testing.T) {
	p := NewProfanityProfile()
	if !p.IsValid() {
		t.Error("default profanity should be valid")
	}
	p.StressMultiplier = -1
	if p.IsValid() {
		t.Error("negative stress multiplier should be invalid")
	}
}

// --- HiddenPreferences Validation ---

func TestHiddenPreferences_Validation(t *testing.T) {
	h := NewHiddenPreferences()
	if !h.IsValid() {
		t.Error("default hidden preferences should be valid")
	}
	h.Openness = 150
	if h.IsValid() {
		t.Error("openness > 100 should be invalid")
	}
}

// --- Full GeneratePople Integration ---

func TestGeneratePople_AllNewSystemsPresent(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	config := PoblConfig{AgeRange: [2]int{20, 40}}
	poble, err := GeneratePople(config, rng)
	if err != nil {
		t.Fatalf("GeneratePople failed: %v", err)
	}
	if !poble.Vocabulary.IsValid() {
		t.Error("generated poble has invalid vocabulary")
	}
	if !poble.Profanity.IsValid() {
		t.Error("generated poble has invalid profanity")
	}
	if !poble.HiddenPrefs.IsValid() {
		t.Error("generated poble has invalid hidden preferences")
	}
	if len(poble.Quirks) == 0 {
		t.Error("generated poble has no quirks")
	}
	if !poble.LoveLang.IsValid() {
		t.Error("generated poble has invalid love language")
	}
	if !poble.Defense.IsValid() {
		t.Error("generated poble has invalid defense mechanism")
	}
}
