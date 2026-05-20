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
		Description:  fmt.Sprintf("%s entro al mundo y cambio lo posible", node.Name),
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
		{ID: TechFireControl, Name: "Control del fuego", Description: "mantener la llama viva en vez de temerle cada noche", EraRequired: entities.EraZero, DiscoveryMethod: DiscoveryAccident, DiscoveryChance: 0.24, UnlocksFeatures: []string{"cooking", "warmth", "night_light"}},
		{ID: TechBasicShelter, Name: "Refugio basico", Description: "rompevientos y refugio comunal hechos con intencion", Prerequisites: []TechID{TechFireControl}, EraRequired: entities.EraZero, DiscoveryMethod: DiscoveryResearch, DiscoveryChance: 0.18, UnlocksFeatures: []string{"stable_housing"}},
		{ID: TechStoneTools, Name: "Herramientas de piedra", Description: "los filos dejan de ser suerte y empiezan a repetirse", EraRequired: entities.EraZero, DiscoveryMethod: DiscoveryAccident, DiscoveryChance: 0.16, UnlocksFeatures: []string{"toolmaking", "butchering"}},
		{ID: TechFishingNets, Name: "Redes de pesca", Description: "el mar empieza a alimentar mas que el azar", Prerequisites: []TechID{TechStoneTools}, EraRequired: entities.EraZero, DiscoveryMethod: DiscoveryResearch, DiscoveryChance: 0.10, UnlocksFeatures: []string{"coastal_food"}},
		{ID: TechAgriculture, Name: "Agricultura", Description: "la comida deja de ser solo caza y empieza a ser plan", Prerequisites: []TechID{TechBasicShelter, TechStoneTools}, EraRequired: entities.EraOne, DiscoveryMethod: DiscoveryResearch, DiscoveryChance: 0.10, UnlocksFeatures: []string{"farming", "food_surplus"}},
		{ID: TechMasonry, Name: "Mamposteria", Description: "la piedra se vuelve estructura en vez de escombro", Prerequisites: []TechID{TechStoneTools}, EraRequired: entities.EraOne, DiscoveryMethod: DiscoveryResearch, DiscoveryChance: 0.09, UnlocksFeatures: []string{"durable_buildings"}},
		{ID: TechWriting, Name: "Escritura", Description: "la memoria sale del cuerpo y empieza a sobrevivir a proposito", Prerequisites: []TechID{TechBasicShelter}, EraRequired: entities.EraOne, DiscoveryMethod: DiscoveryResearch, DiscoveryChance: 0.08, UnlocksFeatures: []string{"archives", "formal_records"}},
		{ID: TechOralLaw, Name: "Ley oral", Description: "la costumbre se vuelve algo que la gente puede senalar", EraRequired: entities.EraOne, DiscoveryMethod: DiscoveryEvent, DiscoveryChance: 0.07, UnlocksFeatures: []string{"laws", "dispute_resolution"}},
		{ID: TechBasicMedicine, Name: "Medicina basica", Description: "cuidar se vuelve tecnica y no solo esperanza", Prerequisites: []TechID{TechFireControl}, EraRequired: entities.EraOne, DiscoveryMethod: DiscoveryResearch, DiscoveryChance: 0.09, UnlocksFeatures: []string{"infection_care", "midwifery"}},
		{ID: TechCurrency, Name: "Moneda", Description: "el valor puede moverse incluso cuando los bienes no", Prerequisites: []TechID{TechWriting, TechAgriculture}, EraRequired: entities.EraTwo, DiscoveryMethod: DiscoveryEvent, DiscoveryChance: 0.07, UnlocksFeatures: []string{"currency", "trade_prices"}},
		{ID: TechIrrigation, Name: "Riego", Description: "el agua empieza a obedecer rutas en vez del clima", Prerequisites: []TechID{TechAgriculture}, EraRequired: entities.EraTwo, DiscoveryMethod: DiscoveryResearch, DiscoveryChance: 0.07, UnlocksFeatures: []string{"crop_stability"}},
		{ID: TechMetalworking, Name: "Metalurgia", Description: "el calor y el mineral se vuelven ventaja", Prerequisites: []TechID{TechFireControl, TechStoneTools}, EraRequired: entities.EraTwo, DiscoveryMethod: DiscoveryResearch, DiscoveryChance: 0.08, UnlocksFeatures: []string{"metal_tools", "weapons"}},
		{ID: TechContraception, Name: "Anticoncepcion", Description: "sexo y reproduccion dejan de ser el mismo boton", Prerequisites: []TechID{TechBasicMedicine}, EraRequired: entities.EraTwo, DiscoveryMethod: DiscoveryResearch, DiscoveryChance: 0.06, UnlocksFeatures: []string{"family_planning"}},
		{ID: TechSurgery, Name: "Cirugia", Description: "cortar para salvar se vuelve pensable", Prerequisites: []TechID{TechBasicMedicine, TechMetalworking}, EraRequired: entities.EraTwo, DiscoveryMethod: DiscoveryResearch, DiscoveryChance: 0.05, UnlocksFeatures: []string{"advanced_medicine"}},
		{ID: TechPrinting, Name: "Imprenta", Description: "las ideas dejan de viajar una boca a la vez", Prerequisites: []TechID{TechWriting}, EraRequired: entities.EraTwo, DiscoveryMethod: DiscoveryResearch, DiscoveryChance: 0.05, UnlocksFeatures: []string{"mass_text"}},
		{ID: TechNavigation, Name: "Navegacion", Description: "el horizonte se vuelve ruta en vez de muro", Prerequisites: []TechID{TechWriting, TechFishingNets}, EraRequired: entities.EraThree, DiscoveryMethod: DiscoveryResearch, DiscoveryChance: 0.05, UnlocksFeatures: []string{"island_travel"}},
		{ID: TechWaterPipes, Name: "Tuberias de agua", Description: "el agua limpia puede llegar a donde vive la gente", Prerequisites: []TechID{TechMasonry, TechIrrigation}, EraRequired: entities.EraThree, DiscoveryMethod: DiscoveryResearch, DiscoveryChance: 0.05, UnlocksFeatures: []string{"sanitation"}},
		{ID: TechElectricity, Name: "Electricidad", Description: "la luz y la fuerza dejan de depender del fuego y el musculo", Prerequisites: []TechID{TechMetalworking}, EraRequired: entities.EraThree, DiscoveryMethod: DiscoveryResearch, DiscoveryChance: 0.04, UnlocksFeatures: []string{"powered_tools", "electric_light"}},
		{ID: TechCommunication, Name: "Comunicacion", Description: "la distancia pierde parte de su crueldad", Prerequisites: []TechID{TechElectricity, TechWriting}, EraRequired: entities.EraThree, DiscoveryMethod: DiscoveryResearch, DiscoveryChance: 0.04, UnlocksFeatures: []string{"long_distance_messages"}},
		{ID: TechAstronomy, Name: "Astronomia", Description: "el cielo se vuelve medible y util", Prerequisites: []TechID{TechWriting, TechNavigation}, EraRequired: entities.EraThree, DiscoveryMethod: DiscoveryResearch, DiscoveryChance: 0.03, UnlocksFeatures: []string{"celestial_calendars"}},
		{ID: TechReproductionTech, Name: "Reproduccion asistida", Description: "la reproduccion asistida entra al vocabulario moral del mundo", Prerequisites: []TechID{TechContraception, TechSurgery, TechBasicMedicine}, EraRequired: entities.EraFour, DiscoveryMethod: DiscoveryResearch, DiscoveryChance: 0.03, UnlocksFeatures: []string{"assisted_reproduction"}},
		{ID: TechComputers, Name: "Computadoras", Description: "el trabajo simbolico gana velocidad de maquina", Prerequisites: []TechID{TechElectricity, TechCommunication}, EraRequired: entities.EraFour, DiscoveryMethod: DiscoveryResearch, DiscoveryChance: 0.02, UnlocksFeatures: []string{"computation"}},
		{ID: TechAIResearch, Name: "Investigacion IA", Description: "pensar sobre pensar se vuelve una institucion", Prerequisites: []TechID{TechComputers}, EraRequired: entities.EraFour, DiscoveryMethod: DiscoveryResearch, DiscoveryChance: 0.015, UnlocksFeatures: []string{"artificial_intelligence"}},
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
