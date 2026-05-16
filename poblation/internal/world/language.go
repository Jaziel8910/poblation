package world

import (
	"math/rand"

	"github.com/user/poblation/internal/entities"
)

// SeedSettlementLanguage initializes a settlement's language from its founders.
func SeedSettlementLanguage(settlement *entities.Settlement, founders []*entities.Poble, rng *rand.Rand) {
	if settlement == nil || len(founders) == 0 || rng == nil {
		return
	}

	lang := entities.NewSettlementLanguage()

	// Seed founder influence from vocabulary profiles.
	for _, founder := range founders {
		if founder == nil {
			continue
		}
		for _, filler := range founder.Vocabulary.FillerWords {
			lang.FounderInfluence = appendUnique(lang.FounderInfluence, filler)
		}
		for _, expr := range founder.Vocabulary.FavoriteExpressions {
			lang.FounderInfluence = appendUnique(lang.FounderInfluence, expr)
		}
	}

	// Generate settlement-specific slang from founder characteristics.
	slangPool := []string{
		"torchado", "chiflado", "chamba", "piola", "grasa",
		"fierro", "chante", "pila", "crudo", "onda",
		"tusa", "bravo", "mazo", "rola", "guacho",
	}
	greetingPool := []string{
		"que onda", "como va", "firme", "arriba",
		"dale", "venga", "presente", "aqui andamos",
	}
	cursePool := []string{
		"hijo del fuego", "maldito polvo", "que se pudra",
		"rata de ceniza", "tragahumo", "comesombras",
	}

	slangCount := 2 + rng.Intn(3)
	for i := 0; i < slangCount; i++ {
		word := slangPool[rng.Intn(len(slangPool))]
		lang.SlangWords = appendUnique(lang.SlangWords, word)
	}

	greetingCount := 1 + rng.Intn(2)
	for i := 0; i < greetingCount; i++ {
		greeting := greetingPool[rng.Intn(len(greetingPool))]
		lang.Greetings = appendUnique(lang.Greetings, greeting)
	}

	curseCount := 1 + rng.Intn(2)
	for i := 0; i < curseCount; i++ {
		curse := cursePool[rng.Intn(len(cursePool))]
		lang.Curses = appendUnique(lang.Curses, curse)
	}

	settlement.Language = lang
}

// EvolveSettlementLanguage adds new words over time based on events.
func EvolveSettlementLanguage(settlement *entities.Settlement, newWord string) {
	if settlement == nil || newWord == "" {
		return
	}
	settlement.Language.SlangWords = appendUnique(settlement.Language.SlangWords, newWord)
}

// GetSettlementSlang returns a random slang word for template use.
func GetSettlementSlang(settlement *entities.Settlement, rng *rand.Rand) string {
	if settlement == nil || len(settlement.Language.SlangWords) == 0 {
		return ""
	}
	return settlement.Language.SlangWords[rng.Intn(len(settlement.Language.SlangWords))]
}

// GetSettlementGreeting returns a random greeting for template use.
func GetSettlementGreeting(settlement *entities.Settlement, rng *rand.Rand) string {
	if settlement == nil || len(settlement.Language.Greetings) == 0 {
		return ""
	}
	return settlement.Language.Greetings[rng.Intn(len(settlement.Language.Greetings))]
}

func appendUnique(slice []string, value string) []string {
	for _, existing := range slice {
		if existing == value {
			return slice
		}
	}
	return append(slice, value)
}
