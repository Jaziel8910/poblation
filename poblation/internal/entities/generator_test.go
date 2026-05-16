package entities

import (
	"math/rand"
	"reflect"
	"testing"
)

func TestGeneratedPobleValuesStayInRange(t *testing.T) {
	rng := rand.New(rand.NewSource(99))

	for i := 0; i < 500; i++ {
		poble, err := GeneratePople(PoblConfig{AgeRange: [2]int{18, 70}}, rng)
		if err != nil {
			t.Fatalf("GeneratePople returned error: %v", err)
		}
		if !poble.IsValid() {
			t.Fatalf("generated invalid poble: %+v", poble)
		}
		if len(poble.Secrets) < 1 || len(poble.Secrets) > 3 {
			t.Fatalf("expected 1-3 secrets, got %d", len(poble.Secrets))
		}
		if len(poble.Kinks) < 1 || len(poble.Kinks) > 3 {
			t.Fatalf("expected 1-3 kinks, got %d", len(poble.Kinks))
		}
		if !poble.Needs.IsValid() {
			t.Fatalf("generated invalid needs: %+v", poble.Needs)
		}
		if !poble.EmotionalState.IsValid() {
			t.Fatalf("generated invalid emotional state: %+v", poble.EmotionalState)
		}
		for _, secret := range poble.Secrets {
			if !secret.IsValid() {
				t.Fatalf("generated invalid secret: %+v", secret)
			}
		}
	}
}

func TestPersonalityCorrelationsApplyStatistically(t *testing.T) {
	rng := rand.New(rand.NewSource(1234))

	var highNeuroJealousy, lowNeuroJealousy float32
	var highNeuroCount, lowNeuroCount float32
	var highAgreeCruelty, lowAgreeCruelty float32
	var highAgreeCount, lowAgreeCount float32

	for i := 0; i < 2000; i++ {
		poble, err := GeneratePople(PoblConfig{AgeRange: [2]int{18, 65}}, rng)
		if err != nil {
			t.Fatalf("GeneratePople returned error: %v", err)
		}
		p := poble.Personality
		if p.Neuroticism >= 65 {
			highNeuroJealousy += p.Jealousy
			highNeuroCount++
		}
		if p.Neuroticism <= 35 {
			lowNeuroJealousy += p.Jealousy
			lowNeuroCount++
		}
		if p.Agreeableness >= 65 {
			highAgreeCruelty += p.Cruelty
			highAgreeCount++
		}
		if p.Agreeableness <= 35 {
			lowAgreeCruelty += p.Cruelty
			lowAgreeCount++
		}
	}

	if highNeuroCount == 0 || lowNeuroCount == 0 || highAgreeCount == 0 || lowAgreeCount == 0 {
		t.Fatal("not enough generated samples for correlation test")
	}
	if highNeuroJealousy/highNeuroCount <= lowNeuroJealousy/lowNeuroCount+8 {
		t.Fatalf("expected high neuroticism to produce meaningfully higher jealousy")
	}
	if highAgreeCruelty/highAgreeCount >= lowAgreeCruelty/lowAgreeCount-8 {
		t.Fatalf("expected high agreeableness to produce meaningfully lower cruelty")
	}
}

func TestSameSeedProducesIdenticalPobles(t *testing.T) {
	config := PoblConfig{AgeRange: [2]int{21, 45}, InspirationNotes: "keeps old radio"}

	first, err := GeneratePople(config, rand.New(rand.NewSource(777)))
	if err != nil {
		t.Fatalf("GeneratePople returned error: %v", err)
	}
	second, err := GeneratePople(config, rand.New(rand.NewSource(777)))
	if err != nil {
		t.Fatalf("GeneratePople returned error: %v", err)
	}

	if !reflect.DeepEqual(first, second) {
		t.Fatalf("expected same seed to generate identical pobles")
	}
}

func TestGenerateStartingPairAlwaysHasDramaPotential(t *testing.T) {
	for seed := int64(1); seed <= 100; seed++ {
		pair, err := generateStartingPairWithRand(
			PoblConfig{AgeRange: [2]int{20, 50}},
			PoblConfig{AgeRange: [2]int{20, 50}},
			rand.New(rand.NewSource(seed)),
		)
		if err != nil {
			t.Fatalf("generateStartingPairWithRand returned error: %v", err)
		}

		score := assessChemistry(pair[0], pair[1])
		if score.DramaPotential <= 50 {
			t.Fatalf("expected drama potential > 50 for seed %d, got %.2f", seed, score.DramaPotential)
		}
		if len(pair[0].Secrets)+len(pair[1].Secrets) == 0 {
			t.Fatalf("expected pair to have at least one secret")
		}
		if len(pair[0].Relationships) == 0 || len(pair[1].Relationships) == 0 {
			t.Fatalf("expected pair relationships to be linked")
		}
	}
}
