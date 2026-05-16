package world

import (
	"fmt"
	"sort"
	"strings"

	"github.com/user/poblation/internal/ai"
	"github.com/user/poblation/internal/entities"
)

// LocationType identifies one world space where Pobles can gather.
type LocationType string

const (
	LocationLeafShelter        LocationType = "LEAF_SHELTER"
	LocationCampfire           LocationType = "CAMPFIRE"
	LocationBeach              LocationType = "BEACH"
	LocationCave               LocationType = "CAVE"
	LocationWoodenCabin        LocationType = "WOODEN_CABIN"
	LocationCommunalFire       LocationType = "COMMUNAL_FIRE"
	LocationFarm               LocationType = "FARM"
	LocationRiver              LocationType = "RIVER"
	LocationCemetery           LocationType = "CEMETERY"
	LocationWorkshop           LocationType = "WORKSHOP"
	LocationFamilyHome         LocationType = "FAMILY_HOME"
	LocationMarket             LocationType = "MARKET"
	LocationTavern             LocationType = "TAVERN"
	LocationPrimarySchool      LocationType = "PRIMARY_SCHOOL"
	LocationTownSquare         LocationType = "TOWN_SQUARE"
	LocationClinic             LocationType = "CLINIC"
	LocationTemple             LocationType = "TEMPLE"
	LocationJailPrimitive      LocationType = "JAIL_PRIMITIVE"
	LocationApartment          LocationType = "APARTMENT"
	LocationHighSchool         LocationType = "HIGH_SCHOOL"
	LocationHospital           LocationType = "HOSPITAL"
	LocationCourthouse         LocationType = "COURTHOUSE"
	LocationGovernmentOffice   LocationType = "GOVERNMENT_OFFICE"
	LocationLibrary            LocationType = "LIBRARY"
	LocationDiveBar            LocationType = "DIVE_BAR"
	LocationHotel              LocationType = "HOTEL"
	LocationNewspaperOffice    LocationType = "NEWSPAPER_OFFICE"
	LocationPark               LocationType = "PARK"
	LocationPort               LocationType = "PORT"
	LocationUniversity         LocationType = "UNIVERSITY"
	LocationMentalHealthClinic LocationType = "MENTAL_HEALTH_CLINIC"
	LocationEmbassy            LocationType = "EMBASSY"
	LocationPrisonFormal       LocationType = "PRISON_FORMAL"
	LocationStadium            LocationType = "STADIUM"
)

func (l LocationType) String() string { return string(l) }

// LocationMemory stores collective memories tied to a place.
type LocationMemory struct {
	ID               string           `json:"id"`
	CreatedAt        GameTime         `json:"created_at"`
	Summary          string           `json:"summary"`
	Participants     []string         `json:"participants"`
	EmotionalResidue int              `json:"emotional_residue"`
	Tags             []string         `json:"tags"`
	EventType        ai.GameEventType `json:"event_type"`
}

// Atmosphere stores the current emotional weather of a location.
type Atmosphere struct {
	TimeOfDay map[string]float32 `json:"time_of_day"`
	Weather   map[string]float32 `json:"weather"`
	Occupancy map[string]float32 `json:"occupancy"`
}

// Location stores one narratively meaningful place in the world.
type Location struct {
	ID                string           `json:"id"`
	Name              string           `json:"name"`
	Type              LocationType     `json:"type"`
	Era               Era              `json:"era"`
	Capacity          int              `json:"capacity"`
	PrivacyLevel      int              `json:"privacy_level"`
	RumourMultiplier  float32          `json:"rumour_multiplier"`
	TemplateModifiers []string         `json:"template_modifiers"`
	ActiveMemories    []LocationMemory `json:"active_memories"`
	CurrentOccupants  []string         `json:"current_occupants"`
	Owner             string           `json:"owner"`
	AccessRules       map[string]bool  `json:"access_rules"`
	Atmosphere        Atmosphere       `json:"atmosphere"`
	Events            []ai.GameEvent   `json:"events"`
	IslandID          string           `json:"island_id"`
	BuildingID        string           `json:"building_id"`
	X                 int              `json:"x"`
	Y                 int              `json:"y"`
}

func newLocation(id, name string, locationType LocationType, era Era, islandID string, x, y int) *Location {
	return &Location{
		ID:               id,
		Name:             name,
		Type:             locationType,
		Era:              era,
		Capacity:         defaultLocationCapacity(locationType),
		PrivacyLevel:     defaultPrivacyLevel(locationType),
		RumourMultiplier: defaultRumourMultiplier(locationType),
		TemplateModifiers: []string{
			"type:" + strings.ToLower(locationType.String()),
		},
		ActiveMemories:   []LocationMemory{},
		CurrentOccupants: []string{},
		Owner:            "",
		AccessRules:      defaultAccessRules(locationType),
		Atmosphere: Atmosphere{
			TimeOfDay: map[string]float32{},
			Weather:   map[string]float32{},
			Occupancy: map[string]float32{},
		},
		Events:   []ai.GameEvent{},
		IslandID: islandID,
		X:        x,
		Y:        y,
	}
}

func generateLocationTemplateModifiers(loc Location, time GameTime, occupants []*Poble) []string {
	tags := map[string]struct{}{}
	addLocationTag(tags, "type:"+strings.ToLower(loc.Type.String()))
	addLocationTag(tags, "privacy:"+privacyLabel(loc.PrivacyLevel))
	addLocationTag(tags, "time:"+locationTimeBucket(time))

	switch {
	case len(occupants) == 0:
		addLocationTag(tags, "occupancy:empty")
	case len(occupants) == 1:
		addLocationTag(tags, "occupancy:alone")
	case len(occupants) == 2:
		addLocationTag(tags, "occupancy:pair")
	default:
		addLocationTag(tags, "occupancy:crowded")
	}

	if len(occupants) >= maxInt(1, loc.Capacity-1) {
		addLocationTag(tags, "overcrowded")
	}

	for _, memory := range lastLocationMemories(loc.ActiveMemories, 3) {
		for _, tag := range memory.Tags {
			addLocationTag(tags, "memory:"+strings.ToLower(strings.TrimSpace(tag)))
		}
		if memory.EmotionalResidue >= 70 {
			addLocationTag(tags, "emotion:charged")
		}
	}

	if len(occupants) >= 2 {
		for i := 0; i < len(occupants); i++ {
			for j := i + 1; j < len(occupants); j++ {
				if occupants[i] == nil || occupants[j] == nil {
					continue
				}
				relationship, ok := occupants[i].Relationships[occupants[j].ID]
				if !ok {
					continue
				}
				addLocationTag(tags, "relationship:"+strings.ToLower(relationship.Type.String()))
				if relationship.Resentment >= 60 {
					addLocationTag(tags, "resentment")
				}
				if relationship.Attraction >= 65 {
					addLocationTag(tags, "attraction")
				}
				if relationship.Trust >= 70 {
					addLocationTag(tags, "trust")
				}
			}
		}
	}

	result := make([]string, 0, len(tags))
	for tag := range tags {
		result = append(result, tag)
	}
	sort.Strings(result)
	return result
}

// GetDramaScore estimates how close a place is to producing interesting trouble.
func (w *World) GetDramaScore(locID string) int {
	if w == nil || locID == "" {
		return 0
	}
	loc, ok := w.locationByID(locID)
	if !ok {
		return 0
	}

	score := len(loc.CurrentOccupants) * 8
	if loc.PrivacyLevel >= 70 {
		score += 10
	}
	if loc.RumourMultiplier >= 1.4 {
		score += 8
	}
	switch locationTimeBucket(w.Calendar) {
	case "night":
		score += 12
	case "late_night":
		score += 16
	}

	occupants := w.occupantsForLocationLocked(loc)
	for i := 0; i < len(occupants); i++ {
		for j := i + 1; j < len(occupants); j++ {
			relationship, ok := occupants[i].Relationships[occupants[j].ID]
			if !ok {
				score += 2
				continue
			}
			score += int(relationship.Resentment / 10)
			score += int(relationship.Attraction / 12)
			score += int(relationship.Dependency / 18)
			if relationship.Type == entities.RelationshipEnemy || relationship.Type == entities.RelationshipNemesis {
				score += 12
			}
			if relationship.Type == entities.RelationshipSecretObsession || relationship.Type == entities.RelationshipComplicated {
				score += 15
			}
			if relationship.IsSecret {
				score += 10
			}
		}
	}

	for _, memory := range lastLocationMemories(loc.ActiveMemories, 4) {
		score += memory.EmotionalResidue / 8
	}

	if score > 100 {
		return 100
	}
	return maxInt(0, score)
}

// MovePoble changes one Poble's active place and records any dramatic residue.
func (w *World) MovePoble(poblID string, fromID string, toID string) {
	if w == nil || poblID == "" || toID == "" {
		return
	}

	currentID := fromID
	if currentID == "" {
		currentID = w.occupancy[poblID]
	}
	if currentID == toID {
		return
	}

	poble := w.GetPoble(poblID)
	if poble == nil {
		return
	}

	if currentID != "" {
		if loc, ok := w.locationByID(currentID); ok {
			loc.CurrentOccupants = removeString(loc.CurrentOccupants, poblID)
			loc.Atmosphere.Occupancy = occupancyDescriptor(len(loc.CurrentOccupants), loc.Capacity)
		}
	}

	to, ok := w.locationByID(toID)
	if !ok {
		return
	}
	if !containsID(to.CurrentOccupants, poblID) {
		to.CurrentOccupants = append(to.CurrentOccupants, poblID)
	}
	w.occupancy[poblID] = toID
	to.Atmosphere.TimeOfDay = map[string]float32{locationTimeBucket(w.Calendar): 1}
	to.Atmosphere.Weather = map[string]float32{weatherBucketForWorld(w): 1}
	to.Atmosphere.Occupancy = occupancyDescriptor(len(to.CurrentOccupants), to.Capacity)
	to.TemplateModifiers = generateLocationTemplateModifiers(*to, w.Calendar, w.occupantsForLocationLocked(to))

	if score := w.GetDramaScore(toID); score >= 55 {
		event := ai.GameEvent{
			ID:           fmt.Sprintf("location_shift_%s_%s_%d", poblID, toID, w.Calendar.ToMinutes()),
			Type:         ai.GameEventConflict,
			Time:         w.Calendar,
			PrimaryActor: poblID,
			Severity:     clampWorldFloat(float32(score)/100, 0, 1),
			Valence:      clampSignedWorld((float32(score) / 100) - 0.25),
			Description:  locationDramaDescription(to, score),
			Tags:         append([]string{"location", strings.ToLower(to.Type.String())}, to.TemplateModifiers...),
		}
		to.Events = append(to.Events, event)
		w.ActiveEvents = append(w.ActiveEvents, event)
		w.rememberLocationEvent(to, event)
	}
}

// SpawnLocationForEra creates any era-specific places missing from the world.
func SpawnLocationForEra(era Era, world *World) []Location {
	if world == nil {
		return nil
	}
	specs := locationSpecsForEra(era)
	created := make([]Location, 0, len(specs))
	origin := primaryDiscoveredIsland(world)
	if origin == nil {
		return nil
	}

	for index, spec := range specs {
		if world.hasLocationType(spec.Type) {
			continue
		}
		name := world.proceduralLocationName(spec.Type, index)
		id := fmt.Sprintf("%s_%s", strings.ToLower(spec.Type.String()), strings.ReplaceAll(strings.ToLower(name), " ", "_"))
		loc := newLocation(id, name, spec.Type, era, origin.ID, spec.X, spec.Y)
		loc.BuildingID = spec.BuildingID
		loc.Owner = spec.Owner
		world.ensureLocation(loc)
		created = append(created, *loc)
	}
	return created
}

func (w *World) locationByID(id string) (*Location, bool) {
	if w == nil || id == "" || w.Locations == nil {
		return nil, false
	}
	loc, ok := w.Locations[id]
	return loc, ok && loc != nil
}

func (w *World) occupantsForLocationLocked(loc *Location) []*Poble {
	if w == nil || loc == nil {
		return nil
	}
	result := make([]*Poble, 0, len(loc.CurrentOccupants))
	for _, id := range loc.CurrentOccupants {
		if poble := w.GetPoble(id); poble != nil {
			result = append(result, poble)
		}
	}
	return result
}

func (w *World) ensureLocation(loc *Location) *Location {
	if w == nil || loc == nil {
		return nil
	}
	if w.Locations == nil {
		w.Locations = map[string]*Location{}
	}
	if existing, ok := w.Locations[loc.ID]; ok && existing != nil {
		return existing
	}
	loc.TemplateModifiers = generateLocationTemplateModifiers(*loc, w.Calendar, nil)
	w.Locations[loc.ID] = loc
	return loc
}

func (w *World) rememberLocationEvent(loc *Location, event ai.GameEvent) {
	if w == nil || loc == nil {
		return
	}
	memory := LocationMemory{
		ID:               fmt.Sprintf("loc_memory_%s_%d", loc.ID, event.Time.ToMinutes()),
		CreatedAt:        event.Time,
		Summary:          event.Description,
		Participants:     append([]string{}, loc.CurrentOccupants...),
		EmotionalResidue: int(event.Severity * 100),
		Tags:             append([]string{}, event.Tags...),
		EventType:        event.Type,
	}
	loc.ActiveMemories = append(loc.ActiveMemories, memory)
	if len(loc.ActiveMemories) > 10 {
		loc.ActiveMemories = loc.ActiveMemories[len(loc.ActiveMemories)-10:]
	}
}

func (w *World) proceduralLocationName(locationType LocationType, index int) string {
	settlementName := "Orilla"
	var slang string
	if len(w.State.Settlements) > 0 {
		settlementName = w.State.Settlements[0].Name
		if len(w.State.Settlements[0].Language.SlangWords) > 0 {
			slang = w.State.Settlements[0].Language.SlangWords[index%len(w.State.Settlements[0].Language.SlangWords)]
		}
	}
	base := locationTypeBaseName(locationType)
	if slang == "" {
		return fmt.Sprintf("%s de %s", base, settlementName)
	}
	return fmt.Sprintf("%s %s", base, strings.Title(slang))
}

func (w *World) hasLocationType(locationType LocationType) bool {
	for _, loc := range w.Locations {
		if loc != nil && loc.Type == locationType {
			return true
		}
	}
	return false
}

func lastLocationMemories(memories []LocationMemory, limit int) []LocationMemory {
	if len(memories) <= limit {
		return memories
	}
	return memories[len(memories)-limit:]
}

func addLocationTag(tags map[string]struct{}, tag string) {
	tag = strings.TrimSpace(strings.ToLower(tag))
	if tag != "" {
		tags[tag] = struct{}{}
	}
}

func locationTimeBucket(time GameTime) string {
	switch {
	case time.Hour >= 22 || time.Hour < 4:
		return "late_night"
	case time.Hour >= 18:
		return "night"
	case time.Hour >= 12:
		return "afternoon"
	case time.Hour >= 6:
		return "morning"
	default:
		return "dawn"
	}
}

func weatherBucketForWorld(world *World) string {
	if world == nil {
		return "clear"
	}
	switch world.Calendar.Day % 5 {
	case 0:
		return "humid"
	case 1:
		return "clear"
	case 2:
		return "windy"
	case 3:
		return "stormy"
	default:
		return "heavy_air"
	}
}

func occupancyDescriptor(count, capacity int) map[string]float32 {
	result := map[string]float32{}
	switch {
	case count == 0:
		result["empty"] = 1
	case count == 1:
		result["quiet"] = 1
	case capacity > 0 && count >= capacity:
		result["packed"] = 1
	default:
		result["active"] = 1
	}
	return result
}

func locationDramaDescription(loc *Location, score int) string {
	if loc == nil {
		return "A place turned charged."
	}
	switch {
	case score >= 80:
		return fmt.Sprintf("%s feels one bad sentence away from a story nobody can retract.", loc.Name)
	case score >= 65:
		return fmt.Sprintf("%s is holding too much chemistry and too little room to hide it.", loc.Name)
	default:
		return fmt.Sprintf("%s keeps the kind of silence that usually breaks on its own.", loc.Name)
	}
}

func locationTypeBaseName(locationType LocationType) string {
	switch locationType {
	case LocationCampfire, LocationCommunalFire:
		return "Fogata"
	case LocationBeach:
		return "Playa"
	case LocationCave:
		return "Cueva"
	case LocationWoodenCabin:
		return "Cabana"
	case LocationFarm:
		return "Granja"
	case LocationRiver:
		return "Rio"
	case LocationCemetery:
		return "Cementerio"
	case LocationWorkshop:
		return "Taller"
	case LocationFamilyHome:
		return "Casa"
	case LocationMarket:
		return "Mercado"
	case LocationTavern, LocationDiveBar:
		return "Bar"
	case LocationTemple:
		return "Templo"
	case LocationTownSquare:
		return "Plaza"
	case LocationClinic, LocationHospital, LocationMentalHealthClinic:
		return "Clinica"
	case LocationLibrary:
		return "Biblioteca"
	case LocationPort:
		return "Puerto"
	case LocationUniversity, LocationHighSchool, LocationPrimarySchool:
		return "Escuela"
	case LocationEmbassy:
		return "Embajada"
	case LocationStadium:
		return "Estadio"
	default:
		return "Lugar"
	}
}

func defaultLocationCapacity(locationType LocationType) int {
	switch locationType {
	case LocationLeafShelter, LocationCave, LocationWoodenCabin, LocationFamilyHome:
		return 4
	case LocationCampfire, LocationCommunalFire, LocationTemple, LocationWorkshop:
		return 8
	case LocationMarket, LocationTownSquare, LocationPort, LocationStadium:
		return 18
	default:
		return 10
	}
}

func defaultPrivacyLevel(locationType LocationType) int {
	switch locationType {
	case LocationLeafShelter, LocationCave, LocationWoodenCabin, LocationFamilyHome, LocationHotel, LocationApartment:
		return 80
	case LocationMentalHealthClinic, LocationClinic, LocationHospital:
		return 70
	case LocationTemple, LocationLibrary, LocationPark:
		return 45
	default:
		return 25
	}
}

func defaultRumourMultiplier(locationType LocationType) float32 {
	switch locationType {
	case LocationMarket, LocationTavern, LocationDiveBar, LocationTownSquare, LocationNewspaperOffice:
		return 1.75
	case LocationTemple, LocationPort:
		return 1.35
	default:
		if isSchoolLocation(locationType) {
			return 1.35
		}
		return 1.0
	}
}

func isSchoolLocation(locationType LocationType) bool {
	return locationType == LocationPrimarySchool || locationType == LocationHighSchool || locationType == LocationUniversity
}

func defaultAccessRules(locationType LocationType) map[string]bool {
	rules := map[string]bool{"public": true}
	switch locationType {
	case LocationJailPrimitive, LocationPrisonFormal, LocationGovernmentOffice, LocationCourthouse:
		rules["public"] = false
		rules["officials"] = true
	case LocationFamilyHome, LocationApartment, LocationHotel:
		rules["public"] = false
		rules["guests"] = true
	}
	return rules
}

func privacyLabel(level int) string {
	switch {
	case level >= 75:
		return "private"
	case level >= 45:
		return "semi_private"
	default:
		return "public"
	}
}

type locationSpec struct {
	Type       LocationType
	X          int
	Y          int
	BuildingID string
	Owner      string
}

func locationSpecsForEra(era Era) []locationSpec {
	switch era {
	case entities.EraZero:
		return []locationSpec{
			{Type: LocationLeafShelter, X: 8, Y: 7},
			{Type: LocationCampfire, X: 12, Y: 9},
			{Type: LocationBeach, X: 4, Y: 18},
			{Type: LocationCave, X: 22, Y: 6},
		}
	case entities.EraOne:
		return []locationSpec{
			{Type: LocationWoodenCabin, X: 10, Y: 8},
			{Type: LocationCommunalFire, X: 14, Y: 11},
			{Type: LocationFarm, X: 18, Y: 15},
			{Type: LocationRiver, X: 6, Y: 14},
			{Type: LocationCemetery, X: 26, Y: 8},
		}
	case entities.EraTwo:
		return []locationSpec{
			{Type: LocationWorkshop, X: 17, Y: 10},
			{Type: LocationFamilyHome, X: 12, Y: 6},
			{Type: LocationMarket, X: 19, Y: 13},
			{Type: LocationTavern, X: 15, Y: 14},
			{Type: LocationPrimarySchool, X: 9, Y: 12},
			{Type: LocationTownSquare, X: 14, Y: 12},
			{Type: LocationClinic, X: 21, Y: 8},
			{Type: LocationTemple, X: 24, Y: 10},
			{Type: LocationJailPrimitive, X: 23, Y: 14},
		}
	case entities.EraThree:
		return []locationSpec{
			{Type: LocationApartment, X: 11, Y: 5},
			{Type: LocationHighSchool, X: 8, Y: 11},
			{Type: LocationHospital, X: 22, Y: 7},
			{Type: LocationCourthouse, X: 20, Y: 11},
			{Type: LocationGovernmentOffice, X: 19, Y: 9},
			{Type: LocationLibrary, X: 10, Y: 13},
			{Type: LocationDiveBar, X: 16, Y: 15},
			{Type: LocationHotel, X: 7, Y: 16},
			{Type: LocationNewspaperOffice, X: 18, Y: 8},
			{Type: LocationPark, X: 13, Y: 16},
			{Type: LocationPort, X: 3, Y: 19},
		}
	case entities.EraFour:
		return []locationSpec{
			{Type: LocationUniversity, X: 7, Y: 9},
			{Type: LocationMentalHealthClinic, X: 23, Y: 9},
			{Type: LocationEmbassy, X: 25, Y: 11},
			{Type: LocationPrisonFormal, X: 27, Y: 13},
			{Type: LocationStadium, X: 16, Y: 18},
		}
	default:
		return nil
	}
}

func clampSignedWorld(value float32) float32 {
	if value < -1 {
		return -1
	}
	if value > 1 {
		return 1
	}
	return value
}

func removeString(values []string, target string) []string {
	result := values[:0]
	for _, value := range values {
		if value != target {
			result = append(result, value)
		}
	}
	return result
}
