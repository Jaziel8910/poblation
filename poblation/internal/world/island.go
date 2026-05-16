package world

import (
	"fmt"
	"math/rand"

	"github.com/user/poblation/internal/entities"
)

// GameTime reuses the core simulation time type.
type GameTime = entities.GameTime

// Item reuses the core inventory item type.
type Item = entities.Item

// BiomeType identifies an island biome.
type BiomeType string

const (
	BiomeTropical BiomeType = "TROPICAL"
	BiomeCold     BiomeType = "COLD"
	BiomeVolcanic BiomeType = "VOLCANIC"
	BiomeForest   BiomeType = "FOREST"
	BiomeDesert   BiomeType = "DESERT"
	BiomeMystery  BiomeType = "MYSTERY"
)

// ResourceType identifies island resource categories.
type ResourceType string

const (
	ResourceFood      ResourceType = "FOOD"
	ResourceWater     ResourceType = "WATER"
	ResourceWood      ResourceType = "WOOD"
	ResourceStone     ResourceType = "STONE"
	ResourceMetal     ResourceType = "METAL"
	ResourceMedicine  ResourceType = "MEDICINE"
	ResourceLuxury    ResourceType = "LUXURY"
	ResourceFaith     ResourceType = "FAITH"
	ResourceKnowledge ResourceType = "KNOWLEDGE"
)

// BuildingType identifies functional building roles.
type BuildingType string

const (
	BuildingHome       BuildingType = "HOME"
	BuildingFarm       BuildingType = "FARM"
	BuildingWorkshop   BuildingType = "WORKSHOP"
	BuildingTemple     BuildingType = "TEMPLE"
	BuildingGovernment BuildingType = "GOVERNMENT"
	BuildingHospital   BuildingType = "HOSPITAL"
	BuildingPrison     BuildingType = "PRISON"
)

// IslandSize stores conceptual island dimensions.
type IslandSize struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

// DiaryEntry stores hidden or public diary fragments.
type DiaryEntry struct {
	AuthorID string   `json:"author_id"`
	Day      GameTime `json:"day"`
	Content  string   `json:"content"`
	IsSecret bool     `json:"is_secret"`
}

// Building stores island building state and private records.
type Building struct {
	ID                  string       `json:"id"`
	Name                string       `json:"name"`
	Type                BuildingType `json:"type"`
	OwnerID             string       `json:"owner_id"`
	Inhabitants         []string     `json:"inhabitants"`
	Inventory           []Item       `json:"inventory"`
	HasPrivateDiary     bool         `json:"has_private_diary"`
	PrivateDiaryEntries []DiaryEntry `json:"private_diary_entries"`
}

// Island stores a playable island and its simulation hooks.
type Island struct {
	ID                        string               `json:"id"`
	Name                      string               `json:"name"`
	Biome                     BiomeType            `json:"biome"`
	Size                      IslandSize           `json:"size"`
	Resources                 map[ResourceType]int `json:"resources"`
	Buildings                 []Building           `json:"buildings"`
	Pobles                    []string             `json:"pobles"`
	IsDiscovered              bool                 `json:"is_discovered"`
	ClimateModifier           float32              `json:"climate_modifier"`
	IsLocked                  bool                 `json:"is_locked"`
	IsMysterious              bool                 `json:"is_mysterious"`
	RequiresSpecialConditions bool                 `json:"requires_special_conditions"`
}

func newIsland(id string, biome BiomeType, size IslandSize, discovered bool, rng *rand.Rand) *Island {
	island := &Island{
		ID:              id,
		Name:            generateIslandName(biome, rng),
		Biome:           biome,
		Size:            size,
		Resources:       generateResources(biome, size),
		Buildings:       generateStarterBuildings(id, biome),
		Pobles:          []string{},
		IsDiscovered:    discovered,
		ClimateModifier: climateModifierForBiome(biome),
		IsLocked:        !discovered,
	}

	if biome == BiomeMystery {
		island.IsMysterious = true
		island.RequiresSpecialConditions = true
	}

	return island
}

func generateStarterBuildings(islandID string, biome BiomeType) []Building {
	buildings := []Building{
		{
			ID:                  fmt.Sprintf("%s_home_0", islandID),
			Name:                "Casa comun",
			Type:                BuildingHome,
			Inhabitants:         []string{},
			Inventory:           []Item{},
			HasPrivateDiary:     true,
			PrivateDiaryEntries: []DiaryEntry{},
		},
	}

	switch biome {
	case BiomeTropical, BiomeForest:
		buildings = append(buildings, Building{
			ID:                  fmt.Sprintf("%s_farm_0", islandID),
			Name:                "Huerto central",
			Type:                BuildingFarm,
			Inhabitants:         []string{},
			Inventory:           []Item{},
			HasPrivateDiary:     false,
			PrivateDiaryEntries: []DiaryEntry{},
		})
	case BiomeVolcanic:
		buildings = append(buildings, Building{
			ID:                  fmt.Sprintf("%s_workshop_0", islandID),
			Name:                "Forja de ceniza",
			Type:                BuildingWorkshop,
			Inhabitants:         []string{},
			Inventory:           []Item{},
			HasPrivateDiary:     true,
			PrivateDiaryEntries: []DiaryEntry{},
		})
	case BiomeCold:
		buildings = append(buildings, Building{
			ID:                  fmt.Sprintf("%s_temple_0", islandID),
			Name:                "Refugio del hielo",
			Type:                BuildingTemple,
			Inhabitants:         []string{},
			Inventory:           []Item{},
			HasPrivateDiary:     true,
			PrivateDiaryEntries: []DiaryEntry{},
		})
	case BiomeDesert:
		buildings = append(buildings, Building{
			ID:                  fmt.Sprintf("%s_hospital_0", islandID),
			Name:                "Pozo de cura",
			Type:                BuildingHospital,
			Inhabitants:         []string{},
			Inventory:           []Item{},
			HasPrivateDiary:     false,
			PrivateDiaryEntries: []DiaryEntry{},
		})
	}

	return buildings
}

func generateResources(biome BiomeType, size IslandSize) map[ResourceType]int {
	areaScale := maxInt(1, (size.Width*size.Height)/120)
	resources := map[ResourceType]int{
		ResourceFood:      40 * areaScale,
		ResourceWater:     35 * areaScale,
		ResourceWood:      20 * areaScale,
		ResourceStone:     20 * areaScale,
		ResourceMetal:     10 * areaScale,
		ResourceMedicine:  8 * areaScale,
		ResourceLuxury:    2 * areaScale,
		ResourceFaith:     6 * areaScale,
		ResourceKnowledge: 5 * areaScale,
	}

	switch biome {
	case BiomeTropical:
		resources[ResourceFood] += 25 * areaScale
		resources[ResourceWater] += 20 * areaScale
		resources[ResourceWood] += 15 * areaScale
		resources[ResourceLuxury] += 6 * areaScale
	case BiomeCold:
		resources[ResourceFood] -= 10 * areaScale
		resources[ResourceWater] += 10 * areaScale
		resources[ResourceStone] += 10 * areaScale
	case BiomeVolcanic:
		resources[ResourceMetal] += 30 * areaScale
		resources[ResourceStone] += 20 * areaScale
		resources[ResourceFood] -= 12 * areaScale
		resources[ResourceLuxury] += 4 * areaScale
	case BiomeForest:
		resources[ResourceWood] += 30 * areaScale
		resources[ResourceFood] += 12 * areaScale
		resources[ResourceMedicine] += 10 * areaScale
	case BiomeDesert:
		resources[ResourceWater] -= 18 * areaScale
		resources[ResourceStone] += 18 * areaScale
		resources[ResourceKnowledge] += 10 * areaScale
		resources[ResourceLuxury] += 8 * areaScale
	case BiomeMystery:
		resources[ResourceFaith] += 25 * areaScale
		resources[ResourceKnowledge] += 25 * areaScale
		resources[ResourceMetal] += 15 * areaScale
		resources[ResourceLuxury] += 10 * areaScale
	}

	for resource, amount := range resources {
		if amount < 0 {
			resources[resource] = 0
		}
	}

	return resources
}

func climateModifierForBiome(biome BiomeType) float32 {
	switch biome {
	case BiomeTropical:
		return 0.25
	case BiomeCold:
		return -0.35
	case BiomeVolcanic:
		return 0.45
	case BiomeForest:
		return 0.10
	case BiomeDesert:
		return 0.50
	case BiomeMystery:
		return 0.80
	default:
		return 0
	}
}

func generateIslandName(biome BiomeType, rng *rand.Rand) string {
	if rng == nil {
		rng = rand.New(rand.NewSource(0))
	}

	pools := map[BiomeType][]string{
		BiomeTropical: {"Sol de Marea", "Palma Roja", "Nido Azul", "Laguna Viva", "Jardin del Alba"},
		BiomeCold:     {"Velo Blanco", "Diente Boreal", "Nieve Callada", "Risco del Hielo", "Luz del Norte"},
		BiomeVolcanic: {"Sangre de Fuego", "Ceniza Madre", "Mandibula Negra", "Forja Salvaje", "Garganta Roja"},
		BiomeForest:   {"Bosque Hondo", "Raiz Cantante", "Sombra Verde", "Rama Antigua", "Corazon del Musgo"},
		BiomeDesert:   {"Arena Eterna", "Pozo del Viento", "Corona Seca", "Miraje Dorado", "Duna del Sol"},
		BiomeMystery:  {"Isla del Susurro", "Ojo Velado", "Origen Perdido", "Umbral Silente", "Bruma Sin Nombre"},
	}

	names := pools[biome]
	if len(names) == 0 {
		names = []string{"Isla Sin Nombre"}
	}

	return names[rng.Intn(len(names))]
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
