package world

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"

	"github.com/user/poblation/internal/ai"
	"github.com/user/poblation/internal/entities"
)

// WorldState reuses the core serialized world state type.
type WorldState = entities.WorldState

// Era reuses the core era type.
type Era = entities.Era

// Poble reuses the core Poble type.
type Poble = entities.Poble

// GameEvent reuses the AI event type for active world events.
type GameEvent = ai.GameEvent

const techNavigation TechID = TechNavigation

// Rumour stores loose social knowledge circulating in the world.
type Rumour struct {
	ID          string   `json:"id"`
	SourceID    string   `json:"source_id"`
	AboutID     string   `json:"about_id"`
	Content     string   `json:"content"`
	Truthiness  float32  `json:"truthiness"`
	SpreadLevel float32  `json:"spread_level"`
	Tags        []string `json:"tags"`
}

// World stores runtime world state plus rich island data.
type World struct {
	State        WorldState           `json:"state"`
	Islands      map[string]*Island   `json:"islands"`
	Locations    map[string]*Location `json:"locations"`
	ActiveEvents []GameEvent          `json:"active_events"`
	EventHistory []GameEvent          `json:"event_history"`
	RumourPool   []Rumour             `json:"rumour_pool"`
	TechTree     TechTree             `json:"tech_tree"`
	Government   *GovernmentSystem    `json:"government"`
	Era          Era                  `json:"era"`
	Calendar     GameTime             `json:"calendar"`
	pobles       map[string]*Poble
	occupancy    map[string]string
	rng          *rand.Rand
}

type serializableWorld struct {
	State        WorldState           `json:"state"`
	Islands      map[string]*Island   `json:"islands"`
	Locations    map[string]*Location `json:"locations"`
	ActiveEvents []GameEvent          `json:"active_events"`
	EventHistory []GameEvent          `json:"event_history"`
	RumourPool   []Rumour             `json:"rumour_pool"`
	TechTree     TechTree             `json:"tech_tree"`
	Government   *GovernmentSystem    `json:"government"`
	Era          Era                  `json:"era"`
	Calendar     GameTime             `json:"calendar"`
	Pobles       []*Poble             `json:"pobles"`
	Occupancy    map[string]string    `json:"occupancy"`
}

// NewWorld builds the starting world with one discovered island and future islands locked.
func NewWorld(seed int64) *World {
	rng := rand.New(rand.NewSource(seed))
	world := &World{
		State:        entities.NewWorldState(),
		Islands:      map[string]*Island{},
		Locations:    map[string]*Location{},
		ActiveEvents: []GameEvent{},
		EventHistory: []GameEvent{},
		RumourPool:   []Rumour{},
		TechTree:     NewTechTree(),
		Government:   nil,
		Era:          entities.EraZero,
		Calendar:     entities.NewGameTime(0, 0, 0),
		pobles:       map[string]*Poble{},
		occupancy:    map[string]string{},
		rng:          rng,
	}

	origin := newIsland("island_0", BiomeTropical, IslandSize{Width: 60, Height: 40}, true, rng)
	origin.Name = "El Origen"
	origin.IsLocked = false
	world.Islands[origin.ID] = origin

	world.State.Islands = append(world.State.Islands, entities.NewIsland(origin.ID, origin.Name))
	world.State.Islands[0].Biome = string(origin.Biome)
	world.State.Islands[0].Size = origin.Size.Width * origin.Size.Height

	biomes := []BiomeType{BiomeForest, BiomeCold, BiomeVolcanic, BiomeDesert}
	sizes := []IslandSize{
		{Width: 52, Height: 34},
		{Width: 55, Height: 36},
		{Width: 48, Height: 32},
		{Width: 58, Height: 30},
	}

	for i := 0; i < 4; i++ {
		id := fmt.Sprintf("island_%d", i+1)
		island := newIsland(id, biomes[i], sizes[i], false, rng)
		island.IsLocked = true
		island.IsDiscovered = false
		island.Name = ensureUniqueName(world.Islands, island.Name, i+1)
		world.Islands[id] = island

		stateIsland := entities.NewIsland(island.ID, island.Name)
		stateIsland.Biome = string(island.Biome)
		stateIsland.Size = island.Size.Width * island.Size.Height
		world.State.Islands = append(world.State.Islands, stateIsland)
	}

	if rng.Float64() < 0.05 {
		mystery := newIsland("island_mystery", BiomeMystery, IslandSize{Width: 44, Height: 28}, false, rng)
		mystery.IsLocked = true
		mystery.IsDiscovered = false
		mystery.Name = ensureUniqueName(world.Islands, mystery.Name, 99)
		world.Islands[mystery.ID] = mystery

		stateIsland := entities.NewIsland(mystery.ID, mystery.Name)
		stateIsland.Biome = string(mystery.Biome)
		stateIsland.Size = mystery.Size.Width * mystery.Size.Height
		world.State.Islands = append(world.State.Islands, stateIsland)
	}

	world.State.TechTree = world.TechTree.ToEntityTechTree()
	world.State.Era = world.Era
	world.State.Day = world.Calendar
	world.ensureDefaultSettlement()
	SpawnLocationForEra(world.Era, world)
	world.syncState()
	return world
}

// MarshalJSON persists runtime-only world indexes needed for save/load.
func (w *World) MarshalJSON() ([]byte, error) {
	if w == nil {
		return json.Marshal(serializableWorld{})
	}

	pobles := make([]*Poble, 0, len(w.pobles))
	for _, poble := range w.pobles {
		if poble != nil {
			pobles = append(pobles, poble)
		}
	}

	payload := serializableWorld{
		State:        w.State,
		Islands:      w.Islands,
		Locations:    w.Locations,
		ActiveEvents: w.ActiveEvents,
		EventHistory: w.EventHistory,
		RumourPool:   w.RumourPool,
		TechTree:     w.TechTree,
		Government:   w.Government,
		Era:          w.Era,
		Calendar:     w.Calendar,
		Pobles:       pobles,
		Occupancy:    w.occupancy,
	}
	return json.Marshal(payload)
}

// UnmarshalJSON restores runtime indexes omitted by default JSON handling.
func (w *World) UnmarshalJSON(data []byte) error {
	var payload serializableWorld
	if err := json.Unmarshal(data, &payload); err != nil {
		return err
	}

	w.State = payload.State
	w.Islands = payload.Islands
	w.Locations = payload.Locations
	w.ActiveEvents = payload.ActiveEvents
	w.EventHistory = payload.EventHistory
	w.RumourPool = payload.RumourPool
	w.TechTree = payload.TechTree
	w.Government = payload.Government
	w.Era = payload.Era
	w.Calendar = payload.Calendar
	w.pobles = map[string]*Poble{}
	for _, poble := range payload.Pobles {
		if poble != nil {
			w.pobles[poble.ID] = poble
		}
	}
	if payload.Occupancy != nil {
		w.occupancy = payload.Occupancy
	} else {
		w.occupancy = map[string]string{}
	}
	if w.Locations == nil {
		w.Locations = map[string]*Location{}
	}
	w.rng = rand.New(rand.NewSource(int64(w.Calendar.ToMinutes() + 1)))
	w.ensureDefaultSettlement()
	w.syncState()
	return nil
}

// GetIsland returns island data by ID.
func (w *World) GetIsland(id string) *Island {
	if w == nil {
		return nil
	}
	return w.Islands[id]
}

// DiscoverIsland reveals one blocked island if navigation exists and chance succeeds.
func (w *World) DiscoverIsland(explorerID string) (*Island, bool) {
	if w == nil || !w.hasNavigationTech() {
		return nil, false
	}

	_ = explorerID
	candidates := make([]*Island, 0, len(w.Islands))
	for _, island := range w.Islands {
		if island.IsDiscovered {
			continue
		}
		if island.RequiresSpecialConditions {
			if !(w.Era == entities.EraThree || w.Era == entities.EraFour) {
				continue
			}
		}
		candidates = append(candidates, island)
	}

	if len(candidates) == 0 {
		return nil, false
	}

	chance := eraDiscoveryChance(w.Era)
	if w.rng.Float32() > chance {
		return nil, false
	}

	selected := candidates[w.rng.Intn(len(candidates))]
	selected.IsDiscovered = true
	selected.IsLocked = false
	w.syncState()
	return selected, true
}

// GetPopulation returns living population count.
func (w *World) GetPopulation() int {
	return len(w.GetAllPobles())
}

// GetWorldState returns the synchronized public world snapshot.
func (w *World) GetWorldState() entities.WorldState {
	if w == nil {
		return entities.NewWorldState()
	}
	w.syncState()
	return w.State
}

// GetAllPobles returns all living pobles known to the world.
func (w *World) GetAllPobles() []*Poble {
	if w == nil {
		return nil
	}

	pobles := make([]*Poble, 0, len(w.pobles))
	for _, poble := range w.pobles {
		if poble != nil && poble.IsAlive {
			pobles = append(pobles, poble)
		}
	}
	return pobles
}

// GetAllKnownPobles returns every known Poble, alive or dead.
func (w *World) GetAllKnownPobles() []*Poble {
	if w == nil {
		return nil
	}

	pobles := make([]*Poble, 0, len(w.pobles))
	for _, poble := range w.pobles {
		if poble != nil {
			pobles = append(pobles, poble)
		}
	}
	return pobles
}

// GetPoble returns one living Poble by ID.
func (w *World) GetPoble(id string) *Poble {
	if w == nil || id == "" {
		return nil
	}
	poble := w.pobles[id]
	if poble == nil || !poble.IsAlive {
		return nil
	}
	return poble
}

// GetLocation returns the coarse location of a living Poble, if known.
func (w *World) GetLocation(id string) (Location, bool) {
	if w == nil || id == "" {
		return Location{}, false
	}
	poble := w.pobles[id]
	if poble == nil || !poble.IsAlive {
		return Location{}, false
	}
	locationID, ok := w.occupancy[id]
	if !ok {
		return Location{}, false
	}
	location, ok := w.locationByID(locationID)
	if !ok {
		return Location{}, false
	}
	return *location, true
}

// GetPoblAt returns the living Poble at a coarse location, if any.
func (w *World) GetPoblAt(location Location) *Poble {
	if w == nil {
		return nil
	}

	for id, candidate := range w.pobles {
		if candidate == nil || !candidate.IsAlive {
			continue
		}
		if loc, ok := w.GetLocation(id); ok && sameLocationSpot(loc, location) {
			return candidate
		}
	}

	return nil
}

// AddPoble places a Poble into the runtime world index.
func (w *World) AddPoble(poble *Poble, location Location) bool {
	if w == nil || poble == nil || poble.ID == "" {
		return false
	}

	island := w.GetIsland(location.IslandID)
	if island == nil {
		return false
	}

	if strings.TrimSpace(location.ID) == "" {
		location.ID = defaultLocationIDForSpawn(location)
	}
	location.IslandID = island.ID
	stored := w.ensureLocation(&location)

	w.pobles[poble.ID] = poble
	w.occupancy[poble.ID] = stored.ID
	if !containsID(stored.CurrentOccupants, poble.ID) {
		stored.CurrentOccupants = append(stored.CurrentOccupants, poble.ID)
	}
	stored.Atmosphere.TimeOfDay = map[string]float32{locationTimeBucket(w.Calendar): 1}
	stored.Atmosphere.Weather = map[string]float32{weatherBucketForWorld(w): 1}
	stored.Atmosphere.Occupancy = occupancyDescriptor(len(stored.CurrentOccupants), stored.Capacity)
	stored.TemplateModifiers = generateLocationTemplateModifiers(*stored, w.Calendar, w.occupantsForLocationLocked(stored))
	if !containsID(island.Pobles, poble.ID) {
		island.Pobles = append(island.Pobles, poble.ID)
	}
	w.seedSettlementLanguageIfNeeded()

	w.syncState()
	return true
}

func (w *World) hasNavigationTech() bool {
	if w == nil {
		return false
	}
	return w.TechTree.Unlocked[techNavigation]
}

func (w *World) syncState() {
	if w == nil {
		return
	}

	w.State.TechTree = w.TechTree.ToEntityTechTree()
	w.State.Era = w.Era
	w.State.Day = w.Calendar
	w.State.Population = w.GetPopulation()
	for index := range w.State.Settlements {
		w.State.Settlements[index].Population = w.State.Population
	}
}

func eraDiscoveryChance(era Era) float32 {
	switch era {
	case entities.EraZero:
		return 0.18
	case entities.EraOne:
		return 0.35
	case entities.EraTwo:
		return 0.55
	case entities.EraThree:
		return 0.72
	case entities.EraFour:
		return 0.90
	default:
		return 0.25
	}
}

func ensureUniqueName(existing map[string]*Island, name string, suffix int) string {
	for _, island := range existing {
		if island.Name == name {
			return fmt.Sprintf("%s %d", name, suffix)
		}
	}
	return name
}

func containsID(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func sameLocationSpot(a, b Location) bool {
	if a.ID != "" && b.ID != "" {
		return a.ID == b.ID
	}
	return a.IslandID == b.IslandID && a.BuildingID == b.BuildingID && a.X == b.X && a.Y == b.Y
}

func defaultLocationIDForSpawn(location Location) string {
	base := strings.TrimSpace(strings.ToLower(location.BuildingID))
	if base == "" {
		base = fmt.Sprintf("%s_%d_%d", strings.ToLower(location.IslandID), location.X, location.Y)
	}
	return base
}

func (w *World) ensureDefaultSettlement() {
	if w == nil || len(w.State.Settlements) > 0 {
		return
	}
	settlement := entities.NewSettlement("settlement_0", "El Origen", "island_0")
	w.State.Settlements = append(w.State.Settlements, settlement)
	if len(w.State.Islands) > 0 {
		w.State.Islands[0].SettlementIDs = append(w.State.Islands[0].SettlementIDs, settlement.ID)
	}
}

func (w *World) seedSettlementLanguageIfNeeded() {
	if w == nil || len(w.State.Settlements) == 0 || w.rng == nil {
		return
	}
	founders := w.GetAllPobles()
	if len(founders) == 0 {
		return
	}
	settlement := &w.State.Settlements[0]
	if len(settlement.Language.FounderInfluence) > 0 {
		return
	}
	SeedSettlementLanguage(settlement, founders, w.rng)
}
