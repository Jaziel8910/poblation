package world

import (
	"math/rand"
	"testing"

	"github.com/user/poblation/internal/entities"
)

func TestSeedSettlementLanguage_FromFounders(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	settlement := &entities.Settlement{
		ID:   "settlement_1",
		Name: "El Refugio",
	}
	founder1 := &entities.Poble{
		ID:      "founder_1",
		IsAlive: true,
		Vocabulary: entities.VocabularyProfile{
			Tier:                entities.VocabEducated,
			FillerWords:         []string{"bueno", "pues"},
			FavoriteExpressions: []string{"la cosa es que"},
			Verbosity:           50,
		},
	}
	founder2 := &entities.Poble{
		ID:      "founder_2",
		IsAlive: true,
		Vocabulary: entities.VocabularyProfile{
			Tier:                entities.VocabStandard,
			FillerWords:         []string{"mira"},
			FavoriteExpressions: []string{"en fin"},
			Verbosity:           60,
		},
	}

	SeedSettlementLanguage(settlement, []*entities.Poble{founder1, founder2}, rng)

	if len(settlement.Language.SlangWords) == 0 {
		t.Error("settlement should have slang words after seeding")
	}
	if len(settlement.Language.Greetings) == 0 {
		t.Error("settlement should have greetings after seeding")
	}
	if len(settlement.Language.FounderInfluence) == 0 {
		t.Error("settlement should have founder influence")
	}
}

func TestSeedSettlementLanguage_NilSafety(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	// Should not panic.
	SeedSettlementLanguage(nil, nil, rng)
	SeedSettlementLanguage(&entities.Settlement{}, nil, rng)
	SeedSettlementLanguage(&entities.Settlement{}, []*entities.Poble{}, rng)
}

func TestEvolveSettlementLanguage_AddsWord(t *testing.T) {
	settlement := &entities.Settlement{
		ID:       "settlement_1",
		Language: entities.NewSettlementLanguage(),
	}
	EvolveSettlementLanguage(settlement, "chiflado")
	if len(settlement.Language.SlangWords) != 1 || settlement.Language.SlangWords[0] != "chiflado" {
		t.Error("evolve should add new slang word")
	}
	// No duplicates.
	EvolveSettlementLanguage(settlement, "chiflado")
	if len(settlement.Language.SlangWords) != 1 {
		t.Error("evolve should not duplicate existing slang")
	}
}

func TestGetSettlementSlang_ReturnsSlang(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	settlement := &entities.Settlement{
		Language: entities.SettlementLanguage{
			SlangWords: []string{"torchado", "piola"},
		},
	}
	slang := GetSettlementSlang(settlement, rng)
	if slang != "torchado" && slang != "piola" {
		t.Errorf("unexpected slang: %s", slang)
	}
}

func TestGetSettlementSlang_EmptySettlement(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	slang := GetSettlementSlang(nil, rng)
	if slang != "" {
		t.Error("nil settlement should return empty string")
	}
}

func TestGetSettlementGreeting_ReturnsGreeting(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	settlement := &entities.Settlement{
		Language: entities.SettlementLanguage{
			Greetings: []string{"que onda", "firme"},
		},
	}
	greeting := GetSettlementGreeting(settlement, rng)
	if greeting != "que onda" && greeting != "firme" {
		t.Errorf("unexpected greeting: %s", greeting)
	}
}
