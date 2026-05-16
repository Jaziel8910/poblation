package world

import (
	"fmt"
	"math/rand"
	"sort"

	"github.com/user/poblation/internal/entities"
)

// TechID reuses the core string-based tech identifier.
type TechID = entities.TechID

const (
	TechFireControl      TechID = "FIRE_CONTROL"
	TechBasicShelter     TechID = "BASIC_SHELTER"
	TechStoneTools       TechID = "STONE_TOOLS"
	TechWriting          TechID = "WRITING"
	TechOralLaw          TechID = "ORAL_LAW"
	TechCurrency         TechID = "CURRENCY"
	TechAgriculture      TechID = "AGRICULTURE"
	TechBasicMedicine    TechID = "BASIC_MEDICINE"
	TechContraception    TechID = "CONTRACEPTION"
	TechSurgery          TechID = "SURGERY"
	TechWaterPipes       TechID = "WATER_PIPES"
	TechElectricity      TechID = "ELECTRICITY"
	TechCommunication    TechID = "COMMUNICATION"
	TechNavigation       TechID = "NAVIGATION"
	TechReproductionTech TechID = "REPRODUCTION_TECH"
	TechComputers        TechID = "COMPUTERS"
	TechAIResearch       TechID = "AI_RESEARCH"
	TechFishingNets      TechID = "FISHING_NETS"
	TechMasonry          TechID = "MASONRY"
	TechMetalworking     TechID = "METALWORKING"
	TechIrrigation       TechID = "IRRIGATION"
	TechPrinting         TechID = "PRINTING"
	TechAstronomy        TechID = "ASTRONOMY"
)

// DiscoveryMethod identifies how a technology is found.
type DiscoveryMethod string

const (
	DiscoveryResearch DiscoveryMethod = "RESEARCH"
	DiscoveryEvent    DiscoveryMethod = "EVENT"
	DiscoveryAccident DiscoveryMethod = "ACCIDENT"
)

// ResearchProject stores ongoing research work.
type ResearchProject struct {
	TechID            TechID   `json:"tech_id"`
	StartedAt         GameTime `json:"started_at"`
	LeadResearcherID  string   `json:"lead_researcher_id"`
	Progress          float32  `json:"progress"`
	DailyProgress     float32  `json:"daily_progress"`
	SupportingFactors []string `json:"supporting_factors"`
}

// TechNode stores one discoverable technology.
type TechNode struct {
	ID              TechID          `json:"id"`
	Name            string          `json:"name"`
	Description     string          `json:"description"`
	Prerequisites   []TechID        `json:"prerequisites"`
	EraRequired     Era             `json:"era_required"`
	DiscoveryMethod DiscoveryMethod `json:"discovery_method"`
	DiscoveryChance float32         `json:"discovery_chance"`
	UnlocksFeatures []string        `json:"unlocks_features"`
}

// TechTree stores discovered and ongoing technology.
type TechTree struct {
	Unlocked   map[TechID]bool             `json:"unlocked"`
	Discovered map[TechID]GameTime         `json:"discovered"`
	InProgress map[TechID]*ResearchProject `json:"in_progress"`
}

// NewTechTree creates an empty runtime tech tree.
func NewTechTree() TechTree {
	return TechTree{
		Unlocked:   map[TechID]bool{},
		Discovered: map[TechID]GameTime{},
		InProgress: map[TechID]*ResearchProject{},
	}
}

// ToEntityTechTree exports only the serialized unlocked state expected by other packages.
func (t TechTree) ToEntityTechTree() entities.TechTree {
	exported := entities.NewTechTree()
	for id, unlocked := range t.Unlocked {
		exported.Unlocked[id] = unlocked
	}
	return exported
}

// AttemptDiscovery rolls once for a new technology discovery.
func AttemptDiscovery(world *World) *TechNode {
	if world == nil {
		return nil
	}
	if world.rng == nil {
		world.rng = rand.New(rand.NewSource(int64(world.Calendar.ToMinutes() + 1)))
	}

	candidates := discoverableTechs(world)
	if len(candidates) == 0 {
		return nil
	}

	for _, node := range candidates {
		chance := discoveryChanceFor(world, node)
		if world.rng.Float32() > chance {
			continue
		}
		recordDiscovery(world, node)
		return node
	}
	return nil
}

// GetUnlockedFeatures returns all currently enabled feature keys.
func (t TechTree) GetUnlockedFeatures() []string {
	features := map[string]bool{}
	for _, node := range allTechNodes() {
		if !t.Unlocked[node.ID] {
			continue
		}
		for _, feature := range node.UnlocksFeatures {
			if feature != "" {
				features[feature] = true
			}
		}
	}

	result := make([]string, 0, len(features))
	for feature := range features {
		result = append(result, feature)
	}
	sort.Strings(result)
	return result
}

func discoverableTechs(world *World) []*TechNode {
	all := allTechNodes()
	candidates := make([]*TechNode, 0, len(all))
	for i := range all {
		node := &all[i]
		if world.TechTree.Unlocked[node.ID] {
			continue
		}
		if eraIndex(world.Era) < eraIndex(node.EraRequired) {
			continue
		}
		if !hasTechPrerequisites(world.TechTree, node.Prerequisites) {
			continue
		}
		candidates = append(candidates, node)
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		return discoveryChanceFor(world, candidates[i]) > discoveryChanceFor(world, candidates[j])
	})
	return candidates
}

func discoveryChanceFor(world *World, node *TechNode) float32 {
	chance := node.DiscoveryChance
	chance *= eraDiscoveryMultiplier(world.Era)
	chance *= opennessDiscoveryMultiplier(world)
	chance *= idleResearchMultiplier(world)
	chance *= resourceDiscoveryMultiplier(world, node)
	if node.DiscoveryMethod == DiscoveryAccident {
		chance *= accidentBias(world, node)
	}
	return clampWorldFloat(chance, 0, 0.92)
}

func recordDiscovery(world *World, node *TechNode) {
	now := world.Calendar
	world.TechTree.Unlocked[node.ID] = true
	world.TechTree.Discovered[node.ID] = now
	delete(world.TechTree.InProgress, node.ID)
	world.State.TechTree = world.TechTree.ToEntityTechTree()
	world.ActiveEvents = append(world.ActiveEvents, GameEvent{
		ID:           fmt.Sprintf("tech_%s_%d", node.ID, now.ToMinutes()),
		Type:         "GOAL_COMPLETE",
		Time:         now,
		PrimaryActor: chooseResearchLeader(world),
		Severity:     0.72,
		Valence:      0.9,
		Description:  node.Name + " entered the world and changed what is possible",
		Tags:         append([]string{"technology", string(node.ID)}, node.UnlocksFeatures...),
	})
}

func hasTechPrerequisites(tree TechTree, prerequisites []TechID) bool {
	for _, prerequisite := range prerequisites {
		if !tree.Unlocked[prerequisite] {
			return false
		}
	}
	return true
}

func eraDiscoveryMultiplier(era Era) float32 {
	return 1 + float32(eraIndex(era))*0.18
}

func opennessDiscoveryMultiplier(world *World) float32 {
	pobles := world.GetAllPobles()
	if len(pobles) == 0 {
		return 0.55
	}
	high := 0
	for _, poble := range pobles {
		if poble != nil && poble.Personality.Openness >= 68 {
			high++
		}
	}
	return 0.8 + float32(high)*0.09
}

func idleResearchMultiplier(world *World) float32 {
	idle := 0
	for _, poble := range world.GetAllPobles() {
		if poble == nil {
			continue
		}
		if poble.Personality.Openness >= 60 &&
			poble.Needs.Hunger < 45 &&
			poble.Needs.Thirst < 45 &&
			poble.Needs.Sleep < 45 {
			idle++
		}
	}
	return 0.9 + float32(idle)*0.06
}

func resourceDiscoveryMultiplier(world *World, node *TechNode) float32 {
	resources := totalResources(world)
	multiplier := float32(1)

	switch node.ID {
	case TechFireControl, TechStoneTools, TechMasonry:
		if resources[ResourceWood] > 60 && resources[ResourceStone] > 40 {
			multiplier += 0.22
		}
	case TechAgriculture, TechIrrigation:
		if resources[ResourceFood] > 120 && resources[ResourceWater] > 80 {
			multiplier += 0.28
		}
	case TechBasicMedicine, TechSurgery, TechContraception, TechReproductionTech:
		if resources[ResourceMedicine] > 24 {
			multiplier += 0.30
		}
	case TechNavigation, TechAstronomy:
		if resources[ResourceKnowledge] > 20 {
			multiplier += 0.20
		}
	case TechElectricity, TechCommunication, TechComputers, TechAIResearch:
		if resources[ResourceMetal] > 80 && resources[ResourceKnowledge] > 45 {
			multiplier += 0.25
		}
	}

	return multiplier
}

func accidentBias(world *World, node *TechNode) float32 {
	if node.ID != TechFireControl {
		return 0.75
	}
	pobles := world.GetPopulation()
	if pobles <= 2 {
		return 2.2
	}
	return 1.2
}

func totalResources(world *World) map[ResourceType]int {
	totals := map[ResourceType]int{}
	if world == nil {
		return totals
	}
	for _, island := range world.Islands {
		if island == nil || !island.IsDiscovered {
			continue
		}
		for resource, amount := range island.Resources {
			totals[resource] += amount
		}
	}
	return totals
}

func chooseResearchLeader(world *World) string {
	bestID := ""
	bestScore := float32(-1)
	for _, poble := range world.GetAllPobles() {
		if poble == nil {
			continue
		}
		score := poble.Personality.Openness + poble.Personality.Conscientiousness + poble.Needs.Purpose
		if score > bestScore {
			bestScore = score
			bestID = poble.ID
		}
	}
	return bestID
}

func technologyLevel(world *World) int {
	if world == nil {
		return 0
	}
	count := len(world.TechTree.Discovered)
	hasEraThree := false
	hasEraFour := false
	for _, node := range allTechNodes() {
		if !world.TechTree.Unlocked[node.ID] {
			continue
		}
		if node.EraRequired == entities.EraThree {
			hasEraThree = true
		}
		if node.EraRequired == entities.EraFour {
			hasEraFour = true
		}
	}

	switch {
	case hasEraFour && count >= 16:
		return 4
	case hasEraThree && count >= 12:
		return 3
	case count >= 7:
		return 2
	case count >= 3:
		return 1
	default:
		return 0
	}
}

func allTechNodes() []TechNode {
	return []TechNode{
		{ID: TechFireControl, Name: "Fire Control", Description: "keep flame alive instead of fearing it every night", EraRequired: entities.EraZero, DiscoveryMethod: DiscoveryAccident, DiscoveryChance: 0.24, UnlocksFeatures: []string{"cooking", "warmth", "night_light"}},
		{ID: TechBasicShelter, Name: "Basic Shelter", Description: "intentional windbreaks and communal refuge", Prerequisites: []TechID{TechFireControl}, EraRequired: entities.EraZero, DiscoveryMethod: DiscoveryResearch, DiscoveryChance: 0.18, UnlocksFeatures: []string{"stable_housing"}},
		{ID: TechStoneTools, Name: "Stone Tools", Description: "sharp edges become repeatable instead of lucky", EraRequired: entities.EraZero, DiscoveryMethod: DiscoveryAccident, DiscoveryChance: 0.16, UnlocksFeatures: []string{"toolmaking", "butchering"}},
		{ID: TechFishingNets, Name: "Fishing Nets", Description: "the sea starts feeding more than chance", Prerequisites: []TechID{TechStoneTools}, EraRequired: entities.EraZero, DiscoveryMethod: DiscoveryResearch, DiscoveryChance: 0.10, UnlocksFeatures: []string{"coastal_food"}},
		{ID: TechAgriculture, Name: "Agriculture", Description: "food stops being only a hunt and starts becoming a plan", Prerequisites: []TechID{TechBasicShelter, TechStoneTools}, EraRequired: entities.EraOne, DiscoveryMethod: DiscoveryResearch, DiscoveryChance: 0.10, UnlocksFeatures: []string{"farming", "food_surplus"}},
		{ID: TechMasonry, Name: "Masonry", Description: "stone becomes structure instead of debris", Prerequisites: []TechID{TechStoneTools}, EraRequired: entities.EraOne, DiscoveryMethod: DiscoveryResearch, DiscoveryChance: 0.09, UnlocksFeatures: []string{"durable_buildings"}},
		{ID: TechWriting, Name: "Writing", Description: "memory leaves the body and starts surviving on purpose", Prerequisites: []TechID{TechBasicShelter}, EraRequired: entities.EraOne, DiscoveryMethod: DiscoveryResearch, DiscoveryChance: 0.08, UnlocksFeatures: []string{"archives", "formal_records"}},
		{ID: TechOralLaw, Name: "Oral Law", Description: "custom turns into something people can point to", EraRequired: entities.EraOne, DiscoveryMethod: DiscoveryEvent, DiscoveryChance: 0.07, UnlocksFeatures: []string{"laws", "dispute_resolution"}},
		{ID: TechBasicMedicine, Name: "Basic Medicine", Description: "care becomes technique instead of hope alone", Prerequisites: []TechID{TechFireControl}, EraRequired: entities.EraOne, DiscoveryMethod: DiscoveryResearch, DiscoveryChance: 0.09, UnlocksFeatures: []string{"infection_care", "midwifery"}},
		{ID: TechCurrency, Name: "Currency", Description: "value can move even when goods do not", Prerequisites: []TechID{TechWriting, TechAgriculture}, EraRequired: entities.EraTwo, DiscoveryMethod: DiscoveryEvent, DiscoveryChance: 0.07, UnlocksFeatures: []string{"currency", "trade_prices"}},
		{ID: TechIrrigation, Name: "Irrigation", Description: "water starts obeying routes instead of weather", Prerequisites: []TechID{TechAgriculture}, EraRequired: entities.EraTwo, DiscoveryMethod: DiscoveryResearch, DiscoveryChance: 0.07, UnlocksFeatures: []string{"crop_stability"}},
		{ID: TechMetalworking, Name: "Metalworking", Description: "heat and ore become leverage", Prerequisites: []TechID{TechFireControl, TechStoneTools}, EraRequired: entities.EraTwo, DiscoveryMethod: DiscoveryResearch, DiscoveryChance: 0.08, UnlocksFeatures: []string{"metal_tools", "weapons"}},
		{ID: TechContraception, Name: "Contraception", Description: "sex and reproduction stop being a single button", Prerequisites: []TechID{TechBasicMedicine}, EraRequired: entities.EraTwo, DiscoveryMethod: DiscoveryResearch, DiscoveryChance: 0.06, UnlocksFeatures: []string{"family_planning"}},
		{ID: TechSurgery, Name: "Surgery", Description: "cutting to save becomes thinkable", Prerequisites: []TechID{TechBasicMedicine, TechMetalworking}, EraRequired: entities.EraTwo, DiscoveryMethod: DiscoveryResearch, DiscoveryChance: 0.05, UnlocksFeatures: []string{"advanced_medicine"}},
		{ID: TechPrinting, Name: "Printing", Description: "ideas stop traveling one mouth at a time", Prerequisites: []TechID{TechWriting}, EraRequired: entities.EraTwo, DiscoveryMethod: DiscoveryResearch, DiscoveryChance: 0.05, UnlocksFeatures: []string{"mass_text"}},
		{ID: TechNavigation, Name: "Navigation", Description: "the horizon becomes a route instead of a wall", Prerequisites: []TechID{TechWriting, TechFishingNets}, EraRequired: entities.EraThree, DiscoveryMethod: DiscoveryResearch, DiscoveryChance: 0.05, UnlocksFeatures: []string{"island_travel"}},
		{ID: TechWaterPipes, Name: "Water Pipes", Description: "clean water can be sent where people actually are", Prerequisites: []TechID{TechMasonry, TechIrrigation}, EraRequired: entities.EraThree, DiscoveryMethod: DiscoveryResearch, DiscoveryChance: 0.05, UnlocksFeatures: []string{"sanitation"}},
		{ID: TechElectricity, Name: "Electricity", Description: "light and power stop depending on fire and muscle", Prerequisites: []TechID{TechMetalworking}, EraRequired: entities.EraThree, DiscoveryMethod: DiscoveryResearch, DiscoveryChance: 0.04, UnlocksFeatures: []string{"powered_tools", "electric_light"}},
		{ID: TechCommunication, Name: "Communication", Description: "distance loses some of its cruelty", Prerequisites: []TechID{TechElectricity, TechWriting}, EraRequired: entities.EraThree, DiscoveryMethod: DiscoveryResearch, DiscoveryChance: 0.04, UnlocksFeatures: []string{"long_distance_messages"}},
		{ID: TechAstronomy, Name: "Astronomy", Description: "the sky becomes measurable and useful", Prerequisites: []TechID{TechWriting, TechNavigation}, EraRequired: entities.EraThree, DiscoveryMethod: DiscoveryResearch, DiscoveryChance: 0.03, UnlocksFeatures: []string{"celestial_calendars"}},
		{ID: TechReproductionTech, Name: "Reproduction Tech", Description: "assisted reproduction enters the moral vocabulary of the world", Prerequisites: []TechID{TechContraception, TechSurgery, TechBasicMedicine}, EraRequired: entities.EraFour, DiscoveryMethod: DiscoveryResearch, DiscoveryChance: 0.03, UnlocksFeatures: []string{"assisted_reproduction"}},
		{ID: TechComputers, Name: "Computers", Description: "symbolic work gains machine speed", Prerequisites: []TechID{TechElectricity, TechCommunication}, EraRequired: entities.EraFour, DiscoveryMethod: DiscoveryResearch, DiscoveryChance: 0.02, UnlocksFeatures: []string{"computation"}},
		{ID: TechAIResearch, Name: "AI Research", Description: "thinking about thinking becomes an institution", Prerequisites: []TechID{TechComputers}, EraRequired: entities.EraFour, DiscoveryMethod: DiscoveryResearch, DiscoveryChance: 0.015, UnlocksFeatures: []string{"artificial_intelligence"}},
	}
}

func eraIndex(era Era) int {
	switch era {
	case entities.EraZero:
		return 0
	case entities.EraOne:
		return 1
	case entities.EraTwo:
		return 2
	case entities.EraThree:
		return 3
	case entities.EraFour:
		return 4
	default:
		return 0
	}
}

func clampWorldFloat(value, min, max float32) float32 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}
