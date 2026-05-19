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
	Legitimacy    int            `json:"legitimacy"`
	Stability     int            `json:"stability"`
	EstablishedAt GameTime       `json:"established_at"`
	LastElection  GameTime       `json:"last_election"`
	LastCrisis    GameTime       `json:"last_crisis"`
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

// DailyUpdate advances the long-running civilization systems once per day.
func (m CivilizationManager) DailyUpdate(world *World) []GameEvent {
	if world == nil {
		return nil
	}
	beforeHistory := len(world.EventHistory)
	beforeActive := len(world.ActiveEvents)

	DailyResourceUpdate(world)
	_ = AttemptDiscovery(world)
	_ = MaintainGovernment(world)
	for _, event := range runInstitutionalPulse(world) {
		world.ActiveEvents = append(world.ActiveEvents, event)
	}
	_ = runMarketPulse(world)

	events := make([]GameEvent, 0, len(world.EventHistory)-beforeHistory+len(world.ActiveEvents)-beforeActive)
	events = append(events, world.EventHistory[beforeHistory:]...)
	events = append(events, world.ActiveEvents[beforeActive:]...)
	return events
}

// MaintainGovernment keeps the emergent government aligned with population and history.
func MaintainGovernment(world *World) *GameEvent {
	if world == nil || world.GetPopulation() < 21 {
		return nil
	}
	previous := governmentSignature(world.Government)
	if world.Government == nil {
		world.Government = buildGovernment(world)
	} else {
		world.Government.Type = DetermineGovernmentType(world)
		world.Government.Leader = chooseGovernmentLeader(world, world.Government.Type)
		world.Government.Laws = emergentLaws(world)
	}
	refreshGovernmentMetrics(world)
	if world.Government == nil || governmentSignature(world.Government) == previous {
		world.syncState()
		return nil
	}
	event := GameEvent{
		ID:           fmt.Sprintf("government_%s_%d", strings.ToLower(string(world.Government.Type)), world.Calendar.ToMinutes()),
		Type:         ai.GameEventSocialPositive,
		Time:         world.Calendar,
		PrimaryActor: governmentLeaderID(world.Government),
		Participants: livingPobleIDs(world),
		Severity:     0.74,
		Valence:      0.38,
		Description:  governmentDescription(world.Government),
		Tags:         []string{"government", "institution", strings.ToLower(string(world.Government.Type))},
	}
	world.ActiveEvents = append(world.ActiveEvents, event)
	world.syncState()
	return &event
}

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

func governmentSignature(government *GovernmentSystem) string {
	if government == nil {
		return ""
	}
	return fmt.Sprintf("%s|%s|%d", government.Type, governmentLeaderID(government), len(government.Laws))
}

func governmentLeaderID(government *GovernmentSystem) string {
	if government == nil || government.Leader == nil {
		return ""
	}
	return *government.Leader
}

func governmentDescription(government *GovernmentSystem) string {
	if government == nil {
		return "the settlement failed to turn power into a shape people could name"
	}
	leader := governmentLeaderID(government)
	if leader == "" {
		leader = "no single leader"
	}
	return fmt.Sprintf("%s became the settlement shape of power under %s with %d legitimacy and %d stability", government.Type, leader, government.Legitimacy, government.Stability)
}

func refreshGovernmentMetrics(world *World) {
	if world == nil || world.Government == nil {
		return
	}
	world.Government.Legitimacy = governmentLegitimacy(world, world.Government)
	world.Government.Stability = governmentStability(world, world.Government)
}

func governmentLegitimacy(world *World, government *GovernmentSystem) int {
	if world == nil || government == nil {
		return 0
	}
	score := 48
	if leader := world.GetPoble(governmentLeaderID(government)); leader != nil {
		score += int(leader.Personality.Conscientiousness-50) / 4
		score += int(leader.Personality.Agreeableness-50) / 5
		score += int(leader.Personality.Loyalty-50) / 5
		score -= int(leader.Personality.Cruelty) / 8
		if leader.Archetype == entities.ArchetypeRuler || leader.Archetype == entities.ArchetypeCaretaker || leader.Archetype == entities.ArchetypeProphet {
			score += 8
		}
	}
	score -= int(violencePressure(world) * 34)
	score -= resourcePressure(world)
	score -= int(averageResentment(world) / 4)
	if len(government.Laws) > 1 {
		score += minInt(10, len(government.Laws)*3)
	}
	switch government.Type {
	case GovernmentDictatorship, GovernmentOligarchy:
		score -= 6
	case GovernmentDemocracy, GovernmentMatriarchy:
		score += 6
	case GovernmentAnarchism:
		score -= 2
	}
	return clampWorldInt(score, 0, 100)
}

func governmentStability(world *World, government *GovernmentSystem) int {
	if world == nil || government == nil {
		return 0
	}
	score := 44 + government.Legitimacy/3
	score -= int(violencePressure(world) * 40)
	score -= resourcePressure(world) / 2
	score -= int(averageResentment(world) / 5)
	if world.GetPopulation() >= 80 {
		score += 8
	}
	if hasBuildingType(world, BuildingGovernment) {
		score += 10
	}
	switch government.Type {
	case GovernmentDictatorship, GovernmentMonarchy:
		score += 5
	case GovernmentAnarchism:
		score -= 8
	case GovernmentDemocracy:
		score += 3
	}
	return clampWorldInt(score, 0, 100)
}

func runInstitutionalPulse(world *World) []GameEvent {
	if world == nil || world.Government == nil || world.GetPopulation() < 21 {
		return nil
	}
	refreshGovernmentMetrics(world)
	events := []GameEvent{}
	if event := runGovernmentCrisis(world); event != nil {
		events = append(events, *event)
		refreshGovernmentMetrics(world)
	}
	if event := runElection(world); event != nil {
		events = append(events, *event)
		refreshGovernmentMetrics(world)
	}
	if event := runLawEnforcement(world); event != nil {
		events = append(events, *event)
	}
	return events
}

func runElection(world *World) *GameEvent {
	government := world.Government
	if government == nil || !governmentHasElections(government.Type) {
		return nil
	}
	daysSince := world.Calendar.Day - government.LastElection.Day
	if daysSince < 30 {
		return nil
	}
	oldLeader := governmentLeaderID(government)
	newLeader := chooseGovernmentLeader(world, government.Type)
	if newLeader == nil {
		return nil
	}
	government.Leader = newLeader
	government.LastElection = world.Calendar
	government.Legitimacy = clampWorldInt(government.Legitimacy+9, 0, 100)
	if *newLeader == oldLeader {
		government.Stability = clampWorldInt(government.Stability+5, 0, 100)
	} else {
		government.Stability = clampWorldInt(government.Stability-4, 0, 100)
	}
	return &GameEvent{
		ID:           fmt.Sprintf("election_%d_%s", world.Calendar.Day, *newLeader),
		Type:         ai.GameEventSocialPositive,
		Time:         world.Calendar,
		PrimaryActor: *newLeader,
		Participants: livingPobleIDs(world),
		Severity:     0.58,
		Valence:      0.34,
		Description:  electionDescription(oldLeader, *newLeader),
		Tags:         []string{"government", "institution", "election"},
	}
}

func runGovernmentCrisis(world *World) *GameEvent {
	government := world.Government
	if government == nil {
		return nil
	}
	if world.Calendar.Day-government.LastCrisis.Day < 14 {
		return nil
	}
	resentment := averageResentment(world)
	if government.Stability >= 25 || (violencePressure(world) < 0.22 && resentment < 58 && government.Legitimacy >= 20) {
		return nil
	}
	oldLeader := governmentLeaderID(government)
	oldType := government.Type
	crisisTag := "coup"
	newType := GovernmentDictatorship
	if resentment >= 58 || government.Legitimacy < 18 {
		crisisTag = "revolution"
		if violencePressure(world) > 0.38 {
			newType = GovernmentAnarchism
		} else {
			newType = GovernmentDemocracy
		}
	}
	government.Type = newType
	government.Leader = chooseGovernmentLeader(world, newType)
	government.Laws = emergentLaws(world)
	government.LastCrisis = world.Calendar
	government.Legitimacy = clampWorldInt(government.Legitimacy+16, 0, 100)
	government.Stability = clampWorldInt(government.Stability+18, 0, 100)
	newLeader := governmentLeaderID(government)
	eventType := ai.GameEventConflict
	valence := float32(-0.62)
	if crisisTag == "revolution" && newType == GovernmentDemocracy {
		eventType = ai.GameEventSocialPositive
		valence = 0.18
	}
	return &GameEvent{
		ID:           fmt.Sprintf("%s_%d_%s", crisisTag, world.Calendar.Day, newLeader),
		Type:         eventType,
		Time:         world.Calendar,
		PrimaryActor: newLeader,
		TargetID:     oldLeader,
		Participants: livingPobleIDs(world),
		Severity:     0.9,
		Valence:      valence,
		IsTraumatic:  eventType == ai.GameEventConflict,
		Description:  crisisDescription(crisisTag, oldType, newType, oldLeader, newLeader),
		Tags:         []string{"government", "institution", crisisTag, strings.ToLower(string(newType))},
	}
}

func runLawEnforcement(world *World) *GameEvent {
	if world == nil || world.Government == nil || len(world.Government.Laws) == 0 {
		return nil
	}
	if world.Calendar.Day%7 != 0 {
		return nil
	}
	recent := recentInstitutionalOffense(world)
	if recent == nil {
		return nil
	}
	law := matchingLaw(world.Government.Laws, *recent)
	if law == nil {
		return nil
	}
	world.Government.Legitimacy = clampWorldInt(world.Government.Legitimacy+4, 0, 100)
	world.Government.Stability = clampWorldInt(world.Government.Stability+3, 0, 100)
	actor := recent.PrimaryActor
	if actor == "" && len(recent.Participants) > 0 {
		actor = recent.Participants[0]
	}
	return &GameEvent{
		ID:           fmt.Sprintf("law_enforcement_%d_%s", world.Calendar.Day, law.ID),
		Type:         ai.GameEventSocialNegative,
		Time:         world.Calendar,
		PrimaryActor: actor,
		Participants: livingPobleIDs(world),
		Severity:     0.5,
		Valence:      -0.22,
		Description:  fmt.Sprintf("the settlement enforced %s after %s", law.ID, recent.Description),
		Tags:         []string{"government", "institution", "law", "enforcement", law.ID},
	}
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
	government := &GovernmentSystem{
		Type:          governmentType,
		Leader:        leader,
		Laws:          emergentLaws(world),
		EstablishedAt: world.Calendar,
		LastElection:  world.Calendar,
		LastCrisis:    world.Calendar,
	}
	refreshGovernmentMetricsFor(world, government)
	return government
}

func refreshGovernmentMetricsFor(world *World, government *GovernmentSystem) {
	if world == nil || government == nil {
		return
	}
	government.Legitimacy = governmentLegitimacy(world, government)
	government.Stability = governmentStability(world, government)
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
		score := candidateScoreForGovernment(poble, governmentType)
		if world.Government != nil && world.Government.Leader != nil && *world.Government.Leader == poble.ID {
			score += 6
		}
		score -= averageResentmentToward(world, poble.ID) / 3
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

func candidateScoreForGovernment(poble *entities.Poble, governmentType GovernmentType) float32 {
	if poble == nil {
		return -1
	}
	score := poble.Personality.Ambition + poble.Personality.Loyalty + poble.Personality.Conscientiousness
	score += poble.Personality.Extraversion / 2
	score += poble.Personality.Openness / 4
	switch governmentType {
	case GovernmentDemocracy, GovernmentMatriarchy:
		score += poble.Personality.Agreeableness / 2
		score -= poble.Personality.Cruelty / 2
	case GovernmentDictatorship, GovernmentOligarchy:
		score += poble.Personality.Cruelty / 3
	case GovernmentTheocracy:
		score += poble.Personality.Openness / 3
	case GovernmentAnarchism:
		score += poble.Personality.Agreeableness / 3
		score -= poble.Personality.Ambition / 4
	}
	return score
}

func governmentHasElections(governmentType GovernmentType) bool {
	switch governmentType {
	case GovernmentDemocracy, GovernmentMatriarchy, GovernmentTribal, GovernmentAnarchism:
		return true
	default:
		return false
	}
}

func electionDescription(oldLeader, newLeader string) string {
	if newLeader == "" {
		return "the settlement gathered for an election but nobody could hold the room"
	}
	if oldLeader == "" || oldLeader == newLeader {
		return fmt.Sprintf("the settlement renewed its trust in %s through an open vote", newLeader)
	}
	return fmt.Sprintf("the settlement chose %s over %s and power moved without blood", newLeader, oldLeader)
}

func crisisDescription(kind string, oldType, newType GovernmentType, oldLeader, newLeader string) string {
	if kind == "revolution" {
		return fmt.Sprintf("a revolution broke the old %s under %s and forced a new %s around %s", oldType, emptyLeader(oldLeader), newType, emptyLeader(newLeader))
	}
	return fmt.Sprintf("a coup turned the old %s under %s into a %s led by %s", oldType, emptyLeader(oldLeader), newType, emptyLeader(newLeader))
}

func emptyLeader(id string) string {
	if id == "" {
		return "no one"
	}
	return id
}

func resourcePressure(world *World) int {
	if world == nil || world.GetPopulation() == 0 {
		return 0
	}
	resources := totalResources(world)
	people := maxInt(1, world.GetPopulation())
	pressure := 0
	if resources[ResourceFood] < people*2 {
		pressure += 14
	}
	if resources[ResourceWater] < people*2 {
		pressure += 18
	}
	if resources[ResourceWood]+resources[ResourceStone] < people {
		pressure += 8
	}
	return pressure
}

func averageResentment(world *World) float32 {
	if world == nil {
		return 0
	}
	total := float32(0)
	count := 0
	for _, poble := range world.GetAllPobles() {
		if poble == nil {
			continue
		}
		for _, rel := range poble.Relationships {
			total += rel.Resentment
			count++
		}
	}
	if count == 0 {
		return 0
	}
	return total / float32(count)
}

func averageResentmentToward(world *World, targetID string) float32 {
	if world == nil || targetID == "" {
		return 0
	}
	total := float32(0)
	count := 0
	for _, poble := range world.GetAllPobles() {
		if poble == nil {
			continue
		}
		if rel, ok := poble.Relationships[targetID]; ok {
			total += rel.Resentment
			count++
		}
	}
	if count == 0 {
		return 0
	}
	return total / float32(count)
}

func hasBuildingType(world *World, buildingType BuildingType) bool {
	if world == nil {
		return false
	}
	for _, island := range world.Islands {
		if island == nil {
			continue
		}
		if islandHasBuildingType(island, buildingType) {
			return true
		}
	}
	return false
}

func recentInstitutionalOffense(world *World) *GameEvent {
	if world == nil {
		return nil
	}
	thresholdMinutes := world.Calendar.ToMinutes() - 24*60
	for i := len(world.EventHistory) - 1; i >= 0; i-- {
		event := world.EventHistory[i]
		if event.Time.ToMinutes() < thresholdMinutes {
			break
		}
		if event.Type == ai.GameEventDeath || event.Type == ai.GameEventConflict || event.Type == ai.GameEventThreat || hasAnyTag(event.Tags, "theft", "steal", "murder", "violence") {
			return &event
		}
	}
	return nil
}

func matchingLaw(laws []Law, event GameEvent) *Law {
	for i := range laws {
		law := laws[i]
		if !law.IsEnforced {
			continue
		}
		if law.ID == "law_no_murder" && (event.Type == ai.GameEventDeath || hasAnyTag(event.Tags, "murder", "violence")) {
			return &law
		}
		if law.ID == "law_property" && hasAnyTag(event.Tags, "theft", "steal") {
			return &law
		}
		if law.ID == "law_public_order" && (event.Type == ai.GameEventConflict || event.Type == ai.GameEventThreat) {
			return &law
		}
	}
	return nil
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
