package world

import (
	"fmt"
	"sort"
	"strings"

	"github.com/user/poblation/internal/ai"
	"github.com/user/poblation/internal/entities"
)

// GovernmentType identifies emergent governing structure.
type GovernmentType string

const (
	GovernmentNone         GovernmentType = "NONE"
	GovernmentTribal       GovernmentType = "TRIBAL"
	GovernmentTheocracy    GovernmentType = "THEOCRACY"
	GovernmentDemocracy    GovernmentType = "DEMOCRACY"
	GovernmentOligarchy    GovernmentType = "OLIGARCHY"
	GovernmentDictatorship GovernmentType = "DICTATORSHIP"
	GovernmentMatriarchy   GovernmentType = "MATRIARCHY"
	GovernmentAnarchism    GovernmentType = "ANARCHISM"
	GovernmentMonarchy     GovernmentType = "MONARCHY"
)

// Law stores one emergent law.
type Law struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	Penalty     string `json:"penalty"`
	IsEnforced  bool   `json:"is_enforced"`
}

// GovernmentSystem stores emergent government state.
type GovernmentSystem struct {
	Type          GovernmentType `json:"type"`
	Leader        *string        `json:"leader"`
	Laws          []Law          `json:"laws"`
	EstablishedAt GameTime       `json:"established_at"`
}

// EraTransition describes a civilization era shift.
type EraTransition struct {
	FromEra          Era       `json:"from_era"`
	ToEra            Era       `json:"to_era"`
	UnlockedFeatures []string  `json:"unlocked_features"`
	NarrativeEvent   GameEvent `json:"narrative_event"`
}

// CivilizationManager owns era and government checks.
type CivilizationManager struct{}

// CheckEraTransition checks whether the world is ready to move into the next era.
func (m CivilizationManager) CheckEraTransition(world *World) *EraTransition {
	if world == nil {
		return nil
	}

	ensureEraInfrastructure(world)
	if world.Government == nil && world.GetPopulation() >= 21 {
		world.Government = buildGovernment(world)
	}

	from := world.Era
	var next Era
	var unlocked []string

	switch world.Era {
	case entities.EraZero:
		if world.GetPopulation() >= 5 {
			next = entities.EraOne
			unlocked = []string{"settlement_identity", "basic_roles", "era_one_dialogues"}
		}
	case entities.EraOne:
		if world.GetPopulation() >= 21 && hasBasicBuildingSet(world) {
			next = entities.EraTwo
			unlocked = []string{"organized_trade", "formal_workshops", "era_two_relationship_drama"}
		}
	case entities.EraTwo:
		if world.GetPopulation() >= 101 && governmentEstablished(world) && hasCurrency(world) {
			next = entities.EraThree
			unlocked = []string{"state_power", "inter_island_ambition", "institutional_memory"}
		}
	case entities.EraThree:
		if world.GetPopulation() >= 501 && technologyLevel(world) >= 3 {
			next = entities.EraFour
			unlocked = []string{"advanced_infrastructure", "assisted_reproduction", "mass_communication"}
		}
	}

	if next == "" || next == from {
		return nil
	}

	world.Era = next
	world.State.Era = next
	createdLocations := SpawnLocationForEra(next, world)
	event := GameEvent{
		ID:           fmt.Sprintf("era_%s_%s_%d", from, next, world.Calendar.ToMinutes()),
		Type:         ai.GameEventGoalComplete,
		Time:         world.Calendar,
		Severity:     0.95,
		Valence:      0.88,
		Description:  eraAnnouncement(from, next),
		Participants: livingPobleIDs(world),
		Tags:         append([]string{"era_transition", string(from), string(next)}, locationTransitionTags(createdLocations)...),
	}
	world.ActiveEvents = append(world.ActiveEvents, event)
	world.syncState()

	return &EraTransition{
		FromEra:          from,
		ToEra:            next,
		UnlockedFeatures: unlocked,
		NarrativeEvent:   event,
	}
}

func locationTransitionTags(created []Location) []string {
	if len(created) == 0 {
		return nil
	}
	tags := make([]string, 0, len(created))
	for _, loc := range created {
		tags = append(tags, "location:"+strings.ToLower(loc.Type.String()))
	}
	return tags
}

// GetEraDescription returns the narrative framing for an era.
func GetEraDescription(era Era) string {
	switch era {
	case entities.EraZero:
		return "Two lives against extinction. Every choice still feels like weather and blood."
	case entities.EraOne:
		return "A fragile village exists now. People stop being only survivors and start becoming neighbors."
	case entities.EraTwo:
		return "The settlement learns roles, trade, and memory. Drama stops being private and starts becoming civic."
	case entities.EraThree:
		return "Institutions rise. Power, law, faith, and ambition all want a permanent home."
	case entities.EraFour:
		return "A nation breathes here now. Technology, ideology, and legacy can reach farther than any one family."
	default:
		return "The era is undefined, which usually means the world got ahead of itself."
	}
}

// DetermineGovernmentType infers government from population character and history.
func DetermineGovernmentType(world *World) GovernmentType {
	if world == nil || world.GetPopulation() == 0 {
		return GovernmentNone
	}

	counts := archetypeCounts(world)
	population := float32(world.GetPopulation())
	violence := violencePressure(world)
	femaleLeadBias := femaleLeadershipBias(world)

	switch {
	case counts[entities.ArchetypeProphet] >= maxArchetypeCount(counts):
		return GovernmentTheocracy
	case counts[entities.ArchetypeRuler]+counts[entities.ArchetypeSchemer] >= int(population*0.32) && violence >= 0.45:
		return GovernmentDictatorship
	case counts[entities.ArchetypeRuler] >= int(population*0.22) && violence < 0.45:
		return GovernmentMonarchy
	case counts[entities.ArchetypeInnocent]+counts[entities.ArchetypeCaretaker] >= int(population*0.45):
		return GovernmentDemocracy
	case femaleLeadBias >= 0.60 && counts[entities.ArchetypeCaretaker] >= int(population*0.18):
		return GovernmentMatriarchy
	case counts[entities.ArchetypeVillain]+counts[entities.ArchetypeWarrior] >= int(population*0.28):
		return GovernmentOligarchy
	case violence < 0.12 && counts[entities.ArchetypeRuler] == 0 && counts[entities.ArchetypeProphet] == 0:
		return GovernmentAnarchism
	default:
		return GovernmentTribal
	}
}

func buildGovernment(world *World) *GovernmentSystem {
	governmentType := DetermineGovernmentType(world)
	if governmentType == GovernmentNone {
		return nil
	}
	leader := chooseGovernmentLeader(world, governmentType)
	return &GovernmentSystem{
		Type:          governmentType,
		Leader:        leader,
		Laws:          emergentLaws(world),
		EstablishedAt: world.Calendar,
	}
}

func governmentEstablished(world *World) bool {
	if world == nil {
		return false
	}
	if world.Government == nil {
		world.Government = buildGovernment(world)
	}
	return world.Government != nil && world.Government.Type != GovernmentNone
}

func hasCurrency(world *World) bool {
	if world == nil {
		return false
	}
	if world.TechTree.Unlocked[TechCurrency] {
		return true
	}

	minted := 0
	for _, poble := range world.GetAllPobles() {
		if poble != nil && poble.Money > 0 {
			minted++
		}
	}
	return minted >= maxInt(3, world.GetPopulation()/4)
}

func hasBasicBuildingSet(world *World) bool {
	required := map[BuildingType]bool{
		BuildingHome:     false,
		BuildingFarm:     false,
		BuildingWorkshop: false,
		BuildingTemple:   false,
	}
	for _, island := range world.Islands {
		if island == nil || !island.IsDiscovered {
			continue
		}
		for _, building := range island.Buildings {
			if _, ok := required[building.Type]; ok {
				required[building.Type] = true
			}
		}
	}
	for _, present := range required {
		if !present {
			return false
		}
	}
	return true
}

func ensureEraInfrastructure(world *World) {
	if world == nil || world.GetPopulation() == 0 {
		return
	}

	island := primaryDiscoveredIsland(world)
	if island == nil {
		return
	}

	resources := totalResources(world)
	population := world.GetPopulation()

	if population >= 8 && !islandHasBuildingType(island, BuildingFarm) && resources[ResourceFood] >= 20 {
		island.Buildings = append(island.Buildings, emergentBuilding(island.ID, BuildingFarm, "Huerto comun"))
	}
	if population >= 12 && !islandHasBuildingType(island, BuildingWorkshop) && resources[ResourceWood]+resources[ResourceStone] >= 30 {
		island.Buildings = append(island.Buildings, emergentBuilding(island.ID, BuildingWorkshop, "Taller comun"))
	}
	if population >= 15 && !islandHasBuildingType(island, BuildingTemple) {
		island.Buildings = append(island.Buildings, emergentBuilding(island.ID, BuildingTemple, "Casa de rito"))
	}
	if population >= 80 && !islandHasBuildingType(island, BuildingGovernment) && governmentEstablished(world) {
		island.Buildings = append(island.Buildings, emergentBuilding(island.ID, BuildingGovernment, "Casa de gobierno"))
	}
}

func primaryDiscoveredIsland(world *World) *Island {
	for _, island := range world.Islands {
		if island != nil && island.IsDiscovered {
			return island
		}
	}
	return nil
}

func islandHasBuildingType(island *Island, buildingType BuildingType) bool {
	if island == nil {
		return false
	}
	for _, building := range island.Buildings {
		if building.Type == buildingType {
			return true
		}
	}
	return false
}

func emergentBuilding(islandID string, buildingType BuildingType, name string) Building {
	return Building{
		ID:                  fmt.Sprintf("%s_%s_%d", islandID, strings.ToLower(string(buildingType)), len(name)),
		Name:                name,
		Type:                buildingType,
		Inhabitants:         []string{},
		Inventory:           []Item{},
		HasPrivateDiary:     buildingType == BuildingTemple || buildingType == BuildingGovernment || buildingType == BuildingWorkshop,
		PrivateDiaryEntries: []DiaryEntry{},
	}
}

func emergentLaws(world *World) []Law {
	laws := []Law{}
	if world == nil {
		return laws
	}

	murderSeen := false
	theftSeen := false
	publicChaos := false
	for _, event := range append([]GameEvent{}, world.EventHistory...) {
		if event.Type == ai.GameEventDeath && hasAnyTag(event.Tags, "murder", "violence") {
			murderSeen = true
		}
		if hasAnyTag(event.Tags, "theft", "steal") {
			theftSeen = true
		}
		if event.Type == ai.GameEventConflict || event.Type == ai.GameEventThreat {
			publicChaos = true
		}
	}

	if murderSeen {
		laws = append(laws, Law{ID: "law_no_murder", Description: "killing within the settlement becomes a punishable offense", Penalty: "exile or execution", IsEnforced: true})
	}
	if theftSeen || hasCurrency(world) {
		laws = append(laws, Law{ID: "law_property", Description: "taking stored goods or money without consent becomes theft", Penalty: "restitution or public punishment", IsEnforced: true})
	}
	if publicChaos {
		laws = append(laws, Law{ID: "law_public_order", Description: "violent disorder in shared spaces draws intervention", Penalty: "detention or forced service", IsEnforced: true})
	}
	if len(laws) == 0 {
		laws = append(laws, Law{ID: "law_custom", Description: "the settlement begins by enforcing custom before codifying detail", Penalty: "shame, labor, or exclusion", IsEnforced: false})
	}
	return laws
}

func chooseGovernmentLeader(world *World, governmentType GovernmentType) *string {
	bestID := ""
	bestScore := float32(-1)
	for _, poble := range world.GetAllPobles() {
		if poble == nil {
			continue
		}
		score := poble.Personality.Ambition + poble.Personality.Loyalty + poble.Personality.Conscientiousness
		switch governmentType {
		case GovernmentTheocracy:
			if poble.Archetype == entities.ArchetypeProphet {
				score += 40
			}
		case GovernmentDictatorship, GovernmentMonarchy:
			if poble.Archetype == entities.ArchetypeRuler || poble.Archetype == entities.ArchetypeSchemer {
				score += 35
			}
		case GovernmentDemocracy, GovernmentMatriarchy:
			if poble.Archetype == entities.ArchetypeCaretaker || poble.Archetype == entities.ArchetypeInnocent {
				score += 20
			}
		}
		if score > bestScore {
			bestScore = score
			bestID = poble.ID
		}
	}
	if bestID == "" {
		return nil
	}
	return &bestID
}

func eraAnnouncement(from Era, to Era) string {
	return fmt.Sprintf("%s gives way to %s. The world stops improvising and starts remembering itself.", from, to)
}

func livingPobleIDs(world *World) []string {
	ids := make([]string, 0, world.GetPopulation())
	for _, poble := range world.GetAllPobles() {
		if poble != nil {
			ids = append(ids, poble.ID)
		}
	}
	sort.Strings(ids)
	return ids
}

func archetypeCounts(world *World) map[entities.ArchetypeID]int {
	counts := map[entities.ArchetypeID]int{}
	for _, poble := range world.GetAllPobles() {
		if poble != nil {
			counts[poble.Archetype]++
		}
	}
	return counts
}

func maxArchetypeCount(counts map[entities.ArchetypeID]int) int {
	best := 0
	for _, count := range counts {
		if count > best {
			best = count
		}
	}
	return best
}

func violencePressure(world *World) float32 {
	if world == nil || len(world.EventHistory) == 0 {
		return 0
	}
	violent := 0
	for _, event := range world.EventHistory {
		if event.Type == ai.GameEventConflict || event.Type == ai.GameEventThreat || event.Type == ai.GameEventDeath {
			violent++
		}
	}
	return float32(violent) / float32(len(world.EventHistory))
}

func femaleLeadershipBias(world *World) float32 {
	if world == nil || world.GetPopulation() == 0 {
		return 0
	}
	weighted := float32(0)
	for _, poble := range world.GetAllPobles() {
		if poble == nil {
			continue
		}
		switch poble.Sex {
		case entities.Female:
			weighted += 1
		case entities.Intersex:
			weighted += 0.6
		}
	}
	return weighted / float32(world.GetPopulation())
}

func hasAnyTag(tags []string, targets ...string) bool {
	for _, tag := range tags {
		for _, target := range targets {
			if tag == target {
				return true
			}
		}
	}
	return false
}
