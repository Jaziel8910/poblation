package entities

import (
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"time"
)

// ReproductionSystem analyzes fertility, lineage risk, and non-magical paths.
type ReproductionSystem struct {
	world *World
	rng   *rand.Rand
}

// NewReproductionSystem builds a reproduction system with deterministic RNG when provided.
func NewReproductionSystem(world *World, rng *rand.Rand) *ReproductionSystem {
	if rng == nil {
		rng = rand.New(rand.NewSource(time.Now().UnixNano()))
	}
	return &ReproductionSystem{world: world, rng: rng}
}

// CanReproduce checks whether two Pobles can have a biological child together.
func (s *ReproductionSystem) CanReproduce(a, b *Poble) ReproductionAnalysis {
	analysis := ReproductionAnalysis{ThirdPartyType: ThirdPartyNone}
	if a == nil || b == nil || !a.IsAlive || !b.IsAlive {
		analysis.AlternativePaths = []ReproductionPath{unavailableAdoptionPath()}
		return analysis
	}

	level := s.consanguinityLevel(a, b)
	analysis.ConsanguinityLevel = level
	analysis.ConsanguinityRisk = consanguinityRisk(level)
	carrier, source := reproductiveRoles(a, b)
	if carrier != nil && source != nil {
		analysis.IsBiologicallyPossible = true
		analysis.FertilityChance = fertilityChance(carrier, source)
		analysis.AlternativePaths = s.naturalPaths(a, b, analysis)
		return analysis
	}

	analysis.RequiresThirdParty = true
	analysis.ThirdPartyType = requiredThirdParty(a, b)
	analysis.FertilityChance = 0
	analysis.AlternativePaths = s.thirdPartyPaths(a, b, analysis.ThirdPartyType, level)
	return analysis
}

// CanReproduce is a convenience helper when no world state is available.
func CanReproduce(a, b *Poble) ReproductionAnalysis {
	return NewReproductionSystem(nil, nil).CanReproduce(a, b)
}

// InheritTraits blends physical and light epigenetic traits from two parents.
func InheritTraits(parentA, parentB *Poble) Genetics {
	coefficient := inbreedingCoefficient(parentA, parentB)
	genetics := Genetics{
		TraitMap: map[GeneticTrait]float32{
			GeneticTraitBuild:          blendTrait(parentA, parentB, healthScore),
			GeneticTraitImmunity:       blendTrait(parentA, parentB, immuneScore),
			GeneticTraitFertility:      blendTrait(parentA, parentB, fertilityScore),
			GeneticTraitStressResponse: blendTrait(parentA, parentB, stressResponseScore),
			GeneticTraitOpenness:       blendTrait(parentA, parentB, opennessScore),
			GeneticTraitTemperament:    blendTrait(parentA, parentB, temperamentScore),
		},
		RecessiveRisks:        []GeneticRisk{},
		InbreedingCoefficient: coefficient,
	}
	if coefficient > 0.25 {
		genetics.RecessiveRisks = append(genetics.RecessiveRisks, GeneticRisk{
			ID:          "recessive_fragility",
			Description: "a quiet inherited fragility that doctors can describe but not neatly solve",
			Probability: clampUnit(coefficient * 0.75),
		})
	}
	return genetics
}

// AdoptionEvent creates a baby who arrived through a survival event.
func AdoptionEvent(world *World) (*Poble, GameEvent) {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	now := worldTime(world)
	baby, err := GeneratePople(PoblConfig{AgeRange: [2]int{0, 0}}, rng)
	if err != nil || baby == nil {
		fallback := NewPoble(reproductionID("adopted_baby", rng), "Llegado", 0, Female)
		baby = &fallback
	}

	baby.ID = reproductionID("adopted_baby", rng)
	baby.Age = 0
	baby.DayOfBirth = now
	baby.Health = NewHealthState(0)
	baby.Health.Fertility = 0
	baby.Mental = NewMentalState()
	baby.Needs = NewNeeds()
	baby.Secrets = []Secret{NewSecret(reproductionID("secret", rng), SecretChild, adoptionSecret(rng))}
	baby.Memories = []Memory{}
	baby.Relationships = map[string]Relationship{}
	baby.Children = []string{}
	baby.Parents = [2]string{}
	baby.IsAlive = true

	if world != nil {
		if world.Pobles == nil {
			world.Pobles = map[string]*Poble{}
		}
		world.Pobles[baby.ID] = baby
		world.State.Population = livingPopulation(world.Pobles)
	}

	event := GameEvent{
		ID:           reproductionID("adoption", rng),
		Type:         "ADOPTION",
		Timestamp:    now,
		Participants: []string{baby.ID},
		IsPublic:     true,
		Description:  "a baby arrived with no clean answer about where they came from",
		DramaScore:   72,
	}
	if world != nil {
		world.Events = append(world.Events, event)
	}
	return baby, event
}

// DonorProcess starts a donor reproduction arc with consent and future identity drama.
func DonorProcess(donorID string, recipientID string, world *World) (*PregnancyArc, error) {
	if strings.TrimSpace(donorID) == "" || strings.TrimSpace(recipientID) == "" {
		return nil, errors.New("DonorProcess: donorID and recipientID are required")
	}
	if world == nil || world.Pobles == nil {
		return nil, errors.New("DonorProcess: world has no pobles")
	}

	donor := world.Pobles[donorID]
	recipient := world.Pobles[recipientID]
	if donor == nil || recipient == nil {
		return nil, fmt.Errorf("DonorProcess: missing donor or recipient: donor=%s recipient=%s", donorID, recipientID)
	}

	consented := donorConsentKnown(donor, recipient)
	pressure := donorDramaPressure(donor, recipient, consented)
	arc := &PregnancyArc{
		ParentIDs:             [2]string{recipientID, ""},
		DonorID:               donorID,
		RecipientID:           recipientID,
		RequiresConsentDrama:  !consented,
		ChildMayDiscoverDonor: true,
		RelationshipPressure:  pressure,
	}
	applyDonorRelationshipPressure(donor, recipient, pressure, consented)
	if !consented {
		recipient.Secrets = append(recipient.Secrets, NewSecret(
			fmt.Sprintf("donor_secret_%s_%s", recipientID, donorID),
			SecretChild,
			"donor identity is not settled publicly",
		))
	}
	world.Events = append(world.Events, donorEvent(donor, recipient, worldTime(world), pressure, consented))
	return arc, nil
}

func (s *ReproductionSystem) naturalPaths(a, b *Poble, analysis ReproductionAnalysis) []ReproductionPath {
	drama := 18 + int(analysis.ConsanguinityRisk*100)
	available := analysis.FertilityChance > 0
	paths := []ReproductionPath{{
		Type:         ReproductionPathNatural,
		Description:  "pregnancy can happen naturally, with health and timing still mattering",
		Availability: available,
		DramaScore:   clampIntLocal(drama, 0, 100),
	}}
	if analysis.ConsanguinityLevel > 0 {
		paths = append(paths, s.consanguinityPath(analysis.ConsanguinityLevel))
	}
	return paths
}

func (s *ReproductionSystem) thirdPartyPaths(a, b *Poble, thirdParty ThirdPartyType, level int) []ReproductionPath {
	era := s.era()
	paths := make([]ReproductionPath, 0, 4)
	paths = append(paths, earlyAdoptionPath(era))
	if thirdParty == ThirdPartyDonor {
		paths = append(paths, donorPath(era))
	} else {
		paths = append(paths, surrogatePath(era))
	}
	paths = append(paths, techPath(era))
	if s.lastTwoLiving(a, b) {
		paths = append(paths, lastLovePath())
	}
	if level > 0 {
		paths = append(paths, s.consanguinityPath(level))
	}
	return paths
}

func (s *ReproductionSystem) consanguinityPath(level int) ReproductionPath {
	era := s.era()
	if level == 1 && era == EraZero {
		return ReproductionPath{
			Type:         ReproductionPathNatural,
			Description:  "survival can force the choice, and the world remembers the cost",
			Availability: true,
			DramaScore:   95,
		}
	}
	if era == EraTwo || era == EraThree || era == EraFour {
		return ReproductionPath{
			Type:         ReproductionPathNatural,
			Description:  "possible, but it becomes a public scandal if discovered",
			Availability: true,
			DramaScore:   88,
		}
	}
	return ReproductionPath{
		Type:         ReproductionPathNatural,
		Description:  "possible with private fear and genetic risk",
		Availability: true,
		DramaScore:   70,
	}
}

func (s *ReproductionSystem) consanguinityLevel(a, b *Poble) int {
	level := directConsanguinityLevel(a, b)
	if level > 0 {
		return level
	}
	if s == nil || s.world == nil {
		return relationshipConsanguinityLevel(a, b)
	}
	if parentsAreSiblings(a, b, s.world.Pobles) || parentsShareParent(a, b, s.world.Pobles) {
		return 2
	}
	return relationshipConsanguinityLevel(a, b)
}

func (s *ReproductionSystem) era() Era {
	if s == nil || s.world == nil {
		return EraZero
	}
	if s.world.State.Era.IsValid() {
		return s.world.State.Era
	}
	return EraZero
}

func (s *ReproductionSystem) lastTwoLiving(a, b *Poble) bool {
	if s == nil || s.world == nil || len(s.world.Pobles) == 0 {
		return false
	}
	count := 0
	for _, poble := range s.world.Pobles {
		if poble != nil && poble.IsAlive {
			count++
		}
	}
	return count == 2 && a != nil && b != nil && a.IsAlive && b.IsAlive
}

func reproductiveRoles(a, b *Poble) (*Poble, *Poble) {
	if canCarryPregnancyEntity(a) && canImpregnateEntity(b) {
		return a, b
	}
	if canCarryPregnancyEntity(b) && canImpregnateEntity(a) {
		return b, a
	}
	return nil, nil
}

func canCarryPregnancyEntity(poble *Poble) bool {
	return poble != nil && (poble.Sex == Female || poble.Sex == Intersex) && poble.Age >= 16 && poble.Age <= 50
}

func canImpregnateEntity(poble *Poble) bool {
	return poble != nil && (poble.Sex == Male || poble.Sex == Intersex) && poble.Age >= 16 && poble.Health.Fertility >= 0.05
}

func requiredThirdParty(a, b *Poble) ThirdPartyType {
	aCarrier, bCarrier := canCarryPregnancyEntity(a), canCarryPregnancyEntity(b)
	aSource, bSource := canImpregnateEntity(a), canImpregnateEntity(b)
	switch {
	case aSource && bSource && !aCarrier && !bCarrier:
		return ThirdPartySurrogate
	case aCarrier || bCarrier:
		return ThirdPartyDonor
	default:
		return ThirdPartyTech
	}
}

func fertilityChance(carrier, source *Poble) float32 {
	if carrier == nil || source == nil {
		return 0
	}
	chance := carrier.Health.Fertility * source.Health.Fertility
	chance *= ageFertilityModifier(carrier, true)
	chance *= ageFertilityModifier(source, false)
	chance *= healthFertilityModifier(carrier) * healthFertilityModifier(source)
	chance *= stressFertilityModifier(carrier) * stressFertilityModifier(source)
	return clampUnit(chance)
}

func ageFertilityModifier(poble *Poble, carrier bool) float32 {
	if poble == nil || poble.Age < 16 {
		return 0
	}
	age := poble.Age
	if age >= 20 && age <= 35 {
		return 1
	}
	if carrier {
		switch {
		case age < 20:
			return 0.65
		case age <= 40:
			return 0.78
		case age <= 45:
			return 0.35
		case age <= 50:
			return 0.08
		default:
			return 0
		}
	}
	switch {
	case age < 20:
		return 0.75
	case age <= 45:
		return 0.9
	case age <= 60:
		return 0.7
	case age <= 75:
		return 0.35
	default:
		return 0.12
	}
}

func healthFertilityModifier(poble *Poble) float32 {
	if poble == nil {
		return 0
	}
	modifier := float32(poble.Health.HP) / 100
	if len(poble.Health.Conditions) > 0 {
		modifier -= float32(len(poble.Health.Conditions)) * 0.08
	}
	return clampFloatLocal(modifier, 0.05, 1)
}

func stressFertilityModifier(poble *Poble) float32 {
	if poble == nil {
		return 0
	}
	stress := float32(100-poble.Mental.Stability) / 140
	if poble.EmotionalState.Arousal > 0 {
		stress += poble.EmotionalState.Arousal * 0.18
	}
	for _, emotion := range poble.EmotionalState.ActiveEmotions {
		if emotion == EmotionFear || emotion == EmotionAnxiety || emotion == EmotionGrief {
			stress += 0.08
		}
	}
	return clampFloatLocal(1-stress, 0.15, 1)
}

func directConsanguinityLevel(a, b *Poble) int {
	if a == nil || b == nil {
		return 0
	}
	if a.ID == b.ID {
		return 1
	}
	if isParentChild(a, b) || shareKnownParent(a, b) {
		return 1
	}
	return 0
}

func relationshipConsanguinityLevel(a, b *Poble) int {
	if a == nil || b == nil {
		return 0
	}
	if rel, ok := a.Relationships[b.ID]; ok {
		switch rel.Type {
		case RelationshipSibling:
			return 1
		case RelationshipFamily:
			return 3
		}
	}
	if rel, ok := b.Relationships[a.ID]; ok {
		switch rel.Type {
		case RelationshipSibling:
			return 1
		case RelationshipFamily:
			return 3
		}
	}
	return 0
}

func isParentChild(a, b *Poble) bool {
	for _, parentID := range a.Parents {
		if parentID != "" && parentID == b.ID {
			return true
		}
	}
	for _, parentID := range b.Parents {
		if parentID != "" && parentID == a.ID {
			return true
		}
	}
	return false
}

func shareKnownParent(a, b *Poble) bool {
	for _, parentA := range a.Parents {
		if parentA == "" {
			continue
		}
		for _, parentB := range b.Parents {
			if parentB != "" && parentA == parentB {
				return true
			}
		}
	}
	return false
}

func parentsAreSiblings(a, b *Poble, pobles map[string]*Poble) bool {
	for _, parentAID := range a.Parents {
		parentA := pobles[parentAID]
		if parentA == nil {
			continue
		}
		for _, parentBID := range b.Parents {
			parentB := pobles[parentBID]
			if parentB != nil && relationshipConsanguinityLevel(parentA, parentB) == 1 {
				return true
			}
		}
	}
	return false
}

func parentsShareParent(a, b *Poble, pobles map[string]*Poble) bool {
	for _, parentAID := range a.Parents {
		parentA := pobles[parentAID]
		if parentA == nil {
			continue
		}
		for _, parentBID := range b.Parents {
			parentB := pobles[parentBID]
			if parentB != nil && shareKnownParent(parentA, parentB) {
				return true
			}
		}
	}
	return false
}

func consanguinityRisk(level int) float32 {
	switch {
	case level <= 0:
		return 0
	case level == 1:
		return 0.25
	case level == 2:
		return 0.08
	default:
		return 0.015
	}
}

func inbreedingCoefficient(a, b *Poble) float32 {
	switch directConsanguinityLevel(a, b) {
	case 1:
		if a != nil && b != nil && a.ID == b.ID {
			return 0.5
		}
		return 0.25
	default:
		if relationshipConsanguinityLevel(a, b) >= 3 {
			return 0.015
		}
		if a != nil && b != nil && shareKnownParent(a, b) {
			return 0.25
		}
		return 0
	}
}

func earlyAdoptionPath(era Era) ReproductionPath {
	return ReproductionPath{
		Type:         ReproductionPathAdoption,
		Description:  "a child can arrive through an event, with their origin left dangerous",
		Availability: era == EraZero || era == EraOne || era == EraTwo || era == EraThree || era == EraFour,
		DramaScore:   76,
	}
}

func unavailableAdoptionPath() ReproductionPath {
	return ReproductionPath{
		Type:         ReproductionPathAdoption,
		Description:  "adoption needs a living world event before it can happen",
		Availability: false,
		DramaScore:   20,
	}
}

func donorPath(era Era) ReproductionPath {
	return ReproductionPath{
		Type:         ReproductionPathDonorNeeded,
		Description:  "a donor makes biology possible, but not emotionally clean",
		Availability: era == EraTwo || era == EraThree || era == EraFour,
		DramaScore:   82,
	}
}

func surrogatePath(era Era) ReproductionPath {
	return ReproductionPath{
		Type:         ReproductionPathSurrogateNeeded,
		Description:  "a surrogate means another body joins the family story",
		Availability: era == EraTwo || era == EraThree || era == EraFour,
		DramaScore:   88,
	}
}

func techPath(era Era) ReproductionPath {
	return ReproductionPath{
		Type:         ReproductionPathTechRequired,
		Description:  "assisted reproduction needs enough civilization to exist",
		Availability: era == EraThree || era == EraFour,
		DramaScore:   55,
	}
}

func lastLovePath() ReproductionPath {
	return ReproductionPath{
		Type:         ReproductionPathTechRequired,
		Description:  "END_LOVE: El Ultimo Camino, love survives even if the species does not",
		Availability: true,
		DramaScore:   100,
	}
}

func blendTrait(a, b *Poble, score func(*Poble) float32) float32 {
	if a == nil && b == nil {
		return 0.5
	}
	if a == nil {
		return clampUnit(score(b))
	}
	if b == nil {
		return clampUnit(score(a))
	}
	return clampUnit((score(a) + score(b)) / 2)
}

func healthScore(p *Poble) float32 {
	if p == nil {
		return 0.5
	}
	return clampUnit(float32(p.Health.HP) / 100)
}

func immuneScore(p *Poble) float32 {
	if p == nil {
		return 0.5
	}
	penalty := float32(len(p.Health.Conditions)+len(p.Health.STIs)) * 0.08
	return clampUnit(float32(p.Health.HP)/100 - penalty)
}

func fertilityScore(p *Poble) float32 {
	if p == nil {
		return 0.5
	}
	return clampUnit(p.Health.Fertility)
}

func stressResponseScore(p *Poble) float32 {
	if p == nil {
		return 0.5
	}
	return clampUnit((100 - p.Personality.Neuroticism + float32(p.Mental.Stability)) / 200)
}

func opennessScore(p *Poble) float32 {
	if p == nil {
		return 0.5
	}
	return clampUnit(p.Personality.Openness / 100)
}

func temperamentScore(p *Poble) float32 {
	if p == nil {
		return 0.5
	}
	value := (p.Personality.Agreeableness + p.Personality.Loyalty + (100 - p.Personality.Cruelty)) / 300
	return clampUnit(value)
}

func donorConsentKnown(donor, recipient *Poble) bool {
	if donor == nil || recipient == nil {
		return false
	}
	rel, ok := donor.Relationships[recipient.ID]
	if !ok {
		return false
	}
	return rel.Trust >= 55 && rel.Respect >= 35 && rel.Resentment < 65
}

func donorDramaPressure(donor, recipient *Poble, consented bool) int {
	pressure := 54
	if !consented {
		pressure += 28
	}
	if donor != nil && recipient != nil {
		if rel, ok := donor.Relationships[recipient.ID]; ok {
			pressure += int(rel.Attraction*0.12 + rel.Resentment*0.18 - rel.Trust*0.08)
		}
		if donor.Personality.Jealousy > 65 || recipient.Personality.Jealousy > 65 {
			pressure += 10
		}
	}
	return clampIntLocal(pressure, 0, 100)
}

func applyDonorRelationshipPressure(donor, recipient *Poble, pressure int, consented bool) {
	if donor == nil || recipient == nil {
		return
	}
	ensureRelationshipMap(donor)
	ensureRelationshipMap(recipient)
	donorRel := relationOrDefault(donor, recipient.ID, RelationshipComplicated)
	recipientRel := relationOrDefault(recipient, donor.ID, RelationshipComplicated)
	delta := float32(pressure) / 8
	donorRel.Familiarity = clampPercent(donorRel.Familiarity + 12)
	recipientRel.Familiarity = clampPercent(recipientRel.Familiarity + 12)
	if consented {
		donorRel.Trust = clampPercent(donorRel.Trust + 6)
		recipientRel.Trust = clampPercent(recipientRel.Trust + 6)
	} else {
		donorRel.Resentment = clampPercent(donorRel.Resentment + delta)
		recipientRel.Resentment = clampPercent(recipientRel.Resentment + delta)
	}
	donor.Relationships[recipient.ID] = donorRel
	recipient.Relationships[donor.ID] = recipientRel
}

func relationOrDefault(owner *Poble, targetID string, relType RelationshipType) Relationship {
	if rel, ok := owner.Relationships[targetID]; ok {
		return rel
	}
	return NewRelationship(targetID, relType)
}

func ensureRelationshipMap(poble *Poble) {
	if poble.Relationships == nil {
		poble.Relationships = map[string]Relationship{}
	}
}

func donorEvent(donor, recipient *Poble, now GameTime, pressure int, consented bool) GameEvent {
	eventType := "DONOR_CONSENT"
	description := "donor process began with consent clear enough to become a relationship problem later"
	if !consented {
		eventType = "DONOR_SECRET"
		description = "donor process began with consent unclear enough to become a future scandal"
	}
	return GameEvent{
		ID:           fmt.Sprintf("donor_%s_%s_%d", donor.ID, recipient.ID, now.ToMinutes()),
		Type:         eventType,
		Timestamp:    now,
		Participants: []string{donor.ID, recipient.ID},
		IsPublic:     consented,
		Description:  description,
		DramaScore:   pressure,
	}
}

func adoptionSecret(rng *rand.Rand) string {
	options := []string{
		"arrived with a marked cloth no one recognizes",
		"was left where the tide could erase every footprint",
		"carries a name scratched onto something that should not have survived",
	}
	return options[rng.Intn(len(options))]
}

func worldTime(world *World) GameTime {
	if world == nil {
		return NewGameTime(0, 0, 0)
	}
	if world.State.Day.IsValid() {
		return world.State.Day
	}
	return NewGameTime(0, 0, 0)
}

func livingPopulation(pobles map[string]*Poble) int {
	count := 0
	for _, poble := range pobles {
		if poble != nil && poble.IsAlive {
			count++
		}
	}
	return count
}

func reproductionID(prefix string, rng *rand.Rand) string {
	return fmt.Sprintf("%s_%016x", prefix, rng.Uint64())
}

func clampFloatLocal(value, min, max float32) float32 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func clampIntLocal(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}
