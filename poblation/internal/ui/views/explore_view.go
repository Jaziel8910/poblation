package views

import (
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/user/poblation/internal/entities"
	"github.com/user/poblation/internal/events"
	"github.com/user/poblation/internal/world"
)

// OpenHouseMsg asks the root app to open one specific house.
type OpenHouseMsg struct {
	BuildingID string
	OwnerID    string
}

type exploreFocus string

const (
	exploreFocusLocations    exploreFocus = "locations"
	exploreFocusInspectables exploreFocus = "inspectables"
)

type exploreLocationKind string

const (
	exploreLocationIsland   exploreLocationKind = "island"
	exploreLocationArea     exploreLocationKind = "area"
	exploreLocationBuilding exploreLocationKind = "building"
)

type exploreInspectableKind string

const (
	exploreInspectablePoble    exploreInspectableKind = "poble"
	exploreInspectableBuilding exploreInspectableKind = "building"
	exploreInspectableResource exploreInspectableKind = "resource"
)

type ExploreLocation struct {
	Kind       exploreLocationKind
	ID         string
	Name       string
	Subtitle   string
	IslandID   string
	BuildingID string
	X          int
	Y          int
	Distance   int
}

type ExploreInspectable struct {
	Kind        exploreInspectableKind
	Label       string
	Detail      string
	PobleID     string
	BuildingID  string
	IsPrivate   bool
	ResourceKey string
}

// ExploreModel renders the world exploration screen.
type ExploreModel struct {
	Locations           []ExploreLocation
	SelectedLocation    int
	SelectedInspectable int
	Focus               exploreFocus
	state               AppStateSnapshot
}

var (
	explorePanelStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(borderColor).
				Background(backgroundColor).
				Foreground(primaryColor).
				Padding(0, 1)

	exploreHeaderStyle = lipgloss.NewStyle().
				Foreground(secondaryColor).
				Bold(true)

	exploreSelectedStyle = lipgloss.NewStyle().
				Foreground(accentColor).
				Bold(true)

	exploreSectionStyle = lipgloss.NewStyle().
				Foreground(primaryColor).
				Bold(true)
)

// NewExploreModel returns the exploration view model.
func NewExploreModel() ExploreModel {
	return ExploreModel{
		Locations:           []ExploreLocation{},
		SelectedLocation:    0,
		SelectedInspectable: 0,
		Focus:               exploreFocusLocations,
	}
}

// Init satisfies tea.Model.
func (m ExploreModel) Init() tea.Cmd {
	return nil
}

// SyncAppState refreshes location data from the world snapshot.
func (m ExploreModel) SyncAppState(snapshot AppStateSnapshot) tea.Model {
	m.state = snapshot
	m.Locations = buildExploreLocations(snapshot.World)
	if len(m.Locations) == 0 {
		m.SelectedLocation = 0
		m.SelectedInspectable = 0
		return m
	}
	m.SelectedLocation = clampInt(m.SelectedLocation, 0, len(m.Locations)-1)
	m.SelectedInspectable = clampInt(m.SelectedInspectable, 0, maxInt(0, len(m.currentInspectables())-1))
	return m
}

func (m ExploreModel) Resize(width, height int) tea.Model {
	m.state.Width = width
	m.state.Height = height
	return m
}

// Update handles exploration navigation and view changes.
func (m ExploreModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch typed := msg.(type) {
	case tea.WindowSizeMsg:
		return m.Resize(typed.Width, typed.Height), nil
	case tea.KeyMsg:
		return m.handleKey(typed)
	default:
		return m, nil
	}
}

// View renders map art on the left and contextual details on the right.
func (m ExploreModel) View() string {
	leftWidth, rightWidth := m.panelWidths()
	left := explorePanelStyle.Width(leftWidth).Height(maxInt(10, m.state.Height-1)).Render(m.renderMapPanel(leftWidth))
	right := explorePanelStyle.Width(rightWidth).Height(maxInt(10, m.state.Height-1)).Render(m.renderDetailPanel(rightWidth))
	return lipgloss.JoinHorizontal(lipgloss.Top, left, right)
}

func (m ExploreModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "tab":
		if m.Focus == exploreFocusLocations {
			m.Focus = exploreFocusInspectables
		} else {
			m.Focus = exploreFocusLocations
		}
		return m, nil
	case "up":
		if m.Focus == exploreFocusLocations {
			m.SelectedLocation--
			m.SelectedLocation = clampInt(m.SelectedLocation, 0, maxInt(0, len(m.Locations)-1))
			m.SelectedInspectable = 0
			return m, nil
		}
		m.SelectedInspectable--
		m.SelectedInspectable = clampInt(m.SelectedInspectable, 0, maxInt(0, len(m.currentInspectables())-1))
		return m, nil
	case "down":
		if m.Focus == exploreFocusLocations {
			m.SelectedLocation++
			m.SelectedLocation = clampInt(m.SelectedLocation, 0, maxInt(0, len(m.Locations)-1))
			m.SelectedInspectable = 0
			return m, nil
		}
		m.SelectedInspectable++
		m.SelectedInspectable = clampInt(m.SelectedInspectable, 0, maxInt(0, len(m.currentInspectables())-1))
		return m, nil
	case "enter":
		return m.activateInspectable()
	default:
		return m, nil
	}
}

func (m ExploreModel) activateInspectable() (tea.Model, tea.Cmd) {
	inspectables := m.currentInspectables()
	if len(inspectables) == 0 || m.Focus != exploreFocusInspectables {
		return m, nil
	}
	selected := inspectables[clampInt(m.SelectedInspectable, 0, len(inspectables)-1)]
	switch selected.Kind {
	case exploreInspectablePoble:
		return m, func() tea.Msg {
			return OpenPobleDetailMsg{PobleID: selected.PobleID}
		}
	case exploreInspectableBuilding:
		if !selected.IsPrivate {
			return m, nil
		}
		ownerID := ""
		if building, _, ok := findBuildingByID(m.state.World, selected.BuildingID); ok {
			ownerID = building.OwnerID
			if ownerID == "" && len(building.Inhabitants) > 0 {
				ownerID = building.Inhabitants[0]
			}
		}
		return m, func() tea.Msg {
			return OpenHouseMsg{BuildingID: selected.BuildingID, OwnerID: ownerID}
		}
	default:
		return m, nil
	}
}

func (m ExploreModel) renderMapPanel(panelWidth int) string {
	island := currentIslandForExplore(m.state.World)
	if island == nil {
		return lipgloss.JoinVertical(
			lipgloss.Left,
			exploreHeaderStyle.Render("MAPA"),
			mutedStyle.Render("No hay isla activa."),
		)
	}

	highlightX, highlightY := exploreHighlightCoords(island, m.currentLocation())
	lines := []string{
		exploreHeaderStyle.Render("MAPA DE EXPLORACION"),
		mutedStyle.Render(fmt.Sprintf("%s · %s", island.Name, strings.ToLower(string(island.Biome)))),
		renderExploreIslandMap(m.state.World, island, highlightX, highlightY, maxInt(12, panelWidth-4)),
	}
	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func (m ExploreModel) renderDetailPanel(panelWidth int) string {
	location := m.currentLocation()
	if location == nil {
		return lipgloss.JoinVertical(
			lipgloss.Left,
			exploreHeaderStyle.Render("EXPLORAR"),
			mutedStyle.Render("No hay lugares para visitar."),
		)
	}

	inspectables := m.currentInspectables()
	lines := []string{
		exploreHeaderStyle.Render("EXPLORAR"),
		m.renderLocationsList(maxInt(18, panelWidth-4)),
		"",
		exploreSectionStyle.Render(location.Name),
		mutedStyle.Render(location.Subtitle),
		"",
		exploreSectionStyle.Render("Quien esta aqui"),
		renderExplorePeople(m.currentPobles()),
		"",
		exploreSectionStyle.Render("Recursos"),
		renderExploreResources(m.currentResources()),
		"",
		exploreSectionStyle.Render("Eventos recientes"),
		renderExploreEvents(m.currentEvents()),
		"",
		exploreSectionStyle.Render("Inspeccionar"),
		renderExploreInspectables(inspectables, m.SelectedInspectable, m.Focus == exploreFocusInspectables),
		"",
		mutedStyle.Render("TAB cambia foco · ENTER abre detalle/casa"),
	}
	return strings.Join(lines, "\n")
}

func (m ExploreModel) renderLocationsList(width int) string {
	lines := make([]string, 0, len(m.Locations))
	for index, location := range m.Locations {
		line := fmt.Sprintf("%s · %s", location.Name, location.Subtitle)
		line = truncateRunes(line, width)
		if index == m.SelectedLocation {
			prefix := ">"
			if m.Focus != exploreFocusLocations {
				prefix = "•"
			}
			lines = append(lines, exploreSelectedStyle.Render(prefix+" "+line))
			continue
		}
		lines = append(lines, mutedStyle.Render("  "+line))
	}
	return strings.Join(lines, "\n")
}

func (m ExploreModel) currentLocation() *ExploreLocation {
	if len(m.Locations) == 0 {
		return nil
	}
	index := clampInt(m.SelectedLocation, 0, len(m.Locations)-1)
	return &m.Locations[index]
}

func (m ExploreModel) currentInspectables() []ExploreInspectable {
	location := m.currentLocation()
	if location == nil || m.state.World == nil {
		return nil
	}
	return buildExploreInspectables(m.state.World, *location)
}

func (m ExploreModel) currentPobles() []*entities.Poble {
	location := m.currentLocation()
	if location == nil || m.state.World == nil {
		return nil
	}
	return poblesForExploreLocation(m.state.World, *location)
}

func (m ExploreModel) currentResources() map[string]int {
	location := m.currentLocation()
	if location == nil || m.state.World == nil {
		return nil
	}
	return resourcesForExploreLocation(m.state.World, *location)
}

func (m ExploreModel) currentEvents() []events.GameEvent {
	location := m.currentLocation()
	if location == nil || m.state.World == nil {
		return nil
	}
	return eventsForExploreLocation(m.state.EventFeed, poblesForExploreLocation(m.state.World, *location))
}

func (m ExploreModel) panelWidths() (int, int) {
	totalWidth := maxInt(58, m.state.Width)
	leftWidth := int(float64(totalWidth) * 0.42)
	if leftWidth < 24 {
		leftWidth = 24
	}
	if leftWidth > totalWidth-24 {
		leftWidth = totalWidth - 24
	}
	return leftWidth, totalWidth - leftWidth
}

func buildExploreLocations(w *world.World) []ExploreLocation {
	if w == nil {
		return nil
	}
	current := currentIslandForExplore(w)
	if current == nil {
		return nil
	}

	locations := []ExploreLocation{
		{
			Kind:     exploreLocationIsland,
			ID:       current.ID,
			Name:     current.Name,
			Subtitle: fmt.Sprintf("%s · distancia 0", strings.ToLower(string(current.Biome))),
			IslandID: current.ID,
			X:        current.Size.Width / 2,
			Y:        current.Size.Height / 2,
			Distance: 0,
		},
	}

	locations = append(locations, derivedAreaLocations(current)...)
	for index := range current.Buildings {
		building := current.Buildings[index]
		posX, posY := buildingPosition(current, index)
		subtitle := humanizeBuildingType(building.Type)
		if building.Type == world.BuildingHome {
			subtitle += " privada"
		}
		locations = append(locations, ExploreLocation{
			Kind:       exploreLocationBuilding,
			ID:         building.ID,
			Name:       building.Name,
			Subtitle:   subtitle,
			IslandID:   current.ID,
			BuildingID: building.ID,
			X:          posX,
			Y:          posY,
		})
	}

	for _, island := range sortedDiscoveredIslands(w) {
		if island == nil || island.ID == current.ID {
			continue
		}
		locations = append(locations, ExploreLocation{
			Kind:     exploreLocationIsland,
			ID:       island.ID,
			Name:     island.Name,
			Subtitle: fmt.Sprintf("%s · distancia %d", strings.ToLower(string(island.Biome)), exploreDistance(current, island)),
			IslandID: island.ID,
			X:        island.Size.Width / 2,
			Y:        island.Size.Height / 2,
			Distance: exploreDistance(current, island),
		})
	}

	return locations
}

func derivedAreaLocations(island *world.Island) []ExploreLocation {
	if island == nil {
		return nil
	}
	areas := []ExploreLocation{
		{
			Kind:     exploreLocationArea,
			ID:       island.ID + ":beach",
			Name:     "Playa",
			Subtitle: "borde de agua y restos del mar",
			IslandID: island.ID,
			X:        1,
			Y:        maxInt(1, island.Size.Height/2),
		},
		{
			Kind:     exploreLocationArea,
			ID:       island.ID + ":forest",
			Name:     "Bosque",
			Subtitle: "sombra, recursos y secretos faciles de esconder",
			IslandID: island.ID,
			X:        maxInt(2, island.Size.Width-3),
			Y:        maxInt(2, island.Size.Height/3),
		},
		{
			Kind:     exploreLocationArea,
			ID:       island.ID + ":center",
			Name:     "Centro del pueblo",
			Subtitle: "donde todo termina siendo asunto de todos",
			IslandID: island.ID,
			X:        island.Size.Width / 2,
			Y:        island.Size.Height / 2,
		},
	}
	return areas
}

func buildExploreInspectables(w *world.World, location ExploreLocation) []ExploreInspectable {
	inspectables := []ExploreInspectable{}

	for _, poble := range poblesForExploreLocation(w, location) {
		if poble == nil {
			continue
		}
		inspectables = append(inspectables, ExploreInspectable{
			Kind:    exploreInspectablePoble,
			Label:   "Poble · " + poble.Name,
			Detail:  fmt.Sprintf("%s · mood %s", strings.ToLower(poble.Archetype.String()), strings.ToLower(poble.CurrentMood.String())),
			PobleID: poble.ID,
		})
	}

	for _, building := range buildingsForExploreLocation(w, location) {
		inspectables = append(inspectables, ExploreInspectable{
			Kind:       exploreInspectableBuilding,
			Label:      "Edificio · " + building.Name,
			Detail:     humanizeBuildingType(building.Type),
			BuildingID: building.ID,
			IsPrivate:  building.Type == world.BuildingHome,
		})
	}

	resources := resourcesForExploreLocation(w, location)
	resourceKeys := make([]string, 0, len(resources))
	for resource := range resources {
		resourceKeys = append(resourceKeys, resource)
	}
	sort.Strings(resourceKeys)
	for _, resource := range resourceKeys {
		inspectables = append(inspectables, ExploreInspectable{
			Kind:        exploreInspectableResource,
			Label:       "Recurso · " + strings.ToLower(resource),
			Detail:      fmt.Sprintf("%d disponibles", resources[resource]),
			ResourceKey: resource,
		})
	}

	return inspectables
}

func renderExploreIslandMap(w *world.World, island *world.Island, selectedX, selectedY, panelWidth int) string {
	if island == nil {
		return ""
	}
	cols := clampInt((panelWidth-2)/2, 8, island.Size.Width)
	rows := clampInt(12, 4, island.Size.Height)
	offsetX := clampInt(selectedX-(cols/2), 0, maxInt(0, island.Size.Width-cols))
	offsetY := clampInt(selectedY-(rows/2), 0, maxInt(0, island.Size.Height-rows))

	lines := make([]string, 0, rows)
	for y := 0; y < rows; y++ {
		worldY := offsetY + y
		var line strings.Builder
		for x := 0; x < cols; x++ {
			worldX := offsetX + x
			if worldX == selectedX && worldY == selectedY {
				line.WriteString(exploreSelectedStyle.Render("@"))
				continue
			}
			if building, ok := exploreBuildingAt(island, worldX, worldY); ok {
				symbol := "B"
				if building.Type == world.BuildingHome {
					symbol = "H"
				}
				line.WriteString(symbol)
				continue
			}
			if w != nil {
				if poble, ok := explorePobleAt(w, island.ID, worldX, worldY); ok && poble != nil {
					line.WriteString("P")
					continue
				}
			}
			line.WriteString(terrainRune(island, worldX, worldY))
		}
		lines = append(lines, line.String())
	}
	return strings.Join(lines, "\n")
}

func renderExplorePeople(pobles []*entities.Poble) string {
	if len(pobles) == 0 {
		return mutedStyle.Render("Nadie visible ahora.")
	}
	lines := make([]string, 0, len(pobles))
	for _, poble := range pobles {
		if poble == nil {
			continue
		}
		lines = append(lines, fmt.Sprintf("• %s (%s)", poble.Name, strings.ToLower(poble.CurrentMood.String())))
	}
	return strings.Join(lines, "\n")
}

func renderExploreResources(resources map[string]int) string {
	if len(resources) == 0 {
		return mutedStyle.Render("Nada destacado.")
	}
	keys := make([]string, 0, len(resources))
	for key := range resources {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	lines := make([]string, 0, len(keys))
	for _, key := range keys {
		lines = append(lines, fmt.Sprintf("• %s: %d", strings.ToLower(key), resources[key]))
	}
	return strings.Join(lines, "\n")
}

func renderExploreEvents(eventList []events.GameEvent) string {
	if len(eventList) == 0 {
		return mutedStyle.Render("No hay eco reciente en este lugar.")
	}
	lines := make([]string, 0, len(eventList))
	for _, event := range eventList {
		label := strings.TrimSpace(event.Description)
		if label == "" {
			label = humanizeEventType(event.Type)
		}
		lines = append(lines, "• "+truncateRunes(label, 64))
	}
	return strings.Join(lines, "\n")
}

func renderExploreInspectables(items []ExploreInspectable, selected int, focused bool) string {
	if len(items) == 0 {
		return mutedStyle.Render("Nada por abrir aqui.")
	}
	lines := make([]string, 0, len(items))
	for index, item := range items {
		line := item.Label + " · " + item.Detail
		if index == selected {
			prefix := ">"
			if !focused {
				prefix = "•"
			}
			lines = append(lines, exploreSelectedStyle.Render(prefix+" "+truncateRunes(line, 72)))
			continue
		}
		lines = append(lines, "  "+truncateRunes(line, 72))
	}
	return strings.Join(lines, "\n")
}

func currentIslandForExplore(w *world.World) *world.Island {
	if w == nil {
		return nil
	}
	if island := w.GetIsland("island_0"); island != nil {
		return island
	}
	discovered := sortedDiscoveredIslands(w)
	if len(discovered) > 0 {
		return discovered[0]
	}
	for _, island := range w.Islands {
		if island != nil {
			return island
		}
	}
	return nil
}

func sortedDiscoveredIslands(w *world.World) []*world.Island {
	if w == nil {
		return nil
	}
	list := make([]*world.Island, 0, len(w.Islands))
	for _, island := range w.Islands {
		if island != nil && island.IsDiscovered {
			list = append(list, island)
		}
	}
	sort.SliceStable(list, func(i, j int) bool {
		return list[i].Name < list[j].Name
	})
	return list
}

func exploreHighlightCoords(island *world.Island, location *ExploreLocation) (int, int) {
	if island == nil || location == nil {
		return 0, 0
	}
	return clampInt(location.X, 0, maxInt(0, island.Size.Width-1)), clampInt(location.Y, 0, maxInt(0, island.Size.Height-1))
}

func buildingsForExploreLocation(w *world.World, location ExploreLocation) []world.Building {
	if w == nil {
		return nil
	}
	island := w.GetIsland(location.IslandID)
	if island == nil {
		return nil
	}

	switch location.Kind {
	case exploreLocationBuilding:
		if building, _, ok := findBuildingByID(w, location.BuildingID); ok {
			return []world.Building{building}
		}
	case exploreLocationArea:
		buildings := []world.Building{}
		for index := range island.Buildings {
			building := island.Buildings[index]
			posX, posY := buildingPosition(island, index)
			if exploreWithinArea(location, posX, posY) {
				buildings = append(buildings, building)
			}
		}
		return buildings
	default:
		return append([]world.Building(nil), island.Buildings...)
	}
	return nil
}

func poblesForExploreLocation(w *world.World, location ExploreLocation) []*entities.Poble {
	if w == nil {
		return nil
	}
	island := w.GetIsland(location.IslandID)
	if island == nil {
		return nil
	}

	result := []*entities.Poble{}
	for _, poble := range w.GetAllPobles() {
		if poble == nil {
			continue
		}
		loc, ok := w.GetLocation(poble.ID)
		if !ok || loc.IslandID != island.ID {
			continue
		}

		switch location.Kind {
		case exploreLocationBuilding:
			if loc.BuildingID == location.BuildingID || poble.HomeID == location.BuildingID {
				result = append(result, poble)
			}
		case exploreLocationArea:
			if exploreWithinArea(location, loc.X, loc.Y) {
				result = append(result, poble)
			}
		default:
			result = append(result, poble)
		}
	}
	sort.SliceStable(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result
}

func resourcesForExploreLocation(w *world.World, location ExploreLocation) map[string]int {
	if w == nil {
		return nil
	}
	island := w.GetIsland(location.IslandID)
	if island == nil {
		return nil
	}

	base := map[string]int{}
	switch location.Kind {
	case exploreLocationArea:
		switch strings.ToLower(location.Name) {
		case "playa":
			base["water"] = island.Resources[world.ResourceWater] / 2
			base["food"] = island.Resources[world.ResourceFood] / 4
		case "bosque":
			base["wood"] = island.Resources[world.ResourceWood] / 2
			base["medicine"] = island.Resources[world.ResourceMedicine] / 2
		default:
			base["food"] = island.Resources[world.ResourceFood] / 5
			base["knowledge"] = island.Resources[world.ResourceKnowledge] / 2
		}
	case exploreLocationBuilding:
		if building, _, ok := findBuildingByID(w, location.BuildingID); ok {
			for _, item := range building.Inventory {
				if item.Quantity > 0 {
					base[item.Name] += item.Quantity
				}
			}
			if len(base) == 0 {
				base["shelter"] = 1
			}
		}
	default:
		for resource, amount := range island.Resources {
			base[strings.ToLower(string(resource))] = amount
		}
	}
	return base
}

func eventsForExploreLocation(feed []events.GameEvent, pobles []*entities.Poble) []events.GameEvent {
	if len(feed) == 0 {
		return nil
	}
	participantSet := map[string]struct{}{}
	for _, poble := range pobles {
		if poble != nil {
			participantSet[poble.ID] = struct{}{}
		}
	}

	result := make([]events.GameEvent, 0, 3)
	for _, event := range feed {
		if len(result) >= 3 {
			break
		}
		if len(participantSet) == 0 {
			result = append(result, event)
			continue
		}
		for _, participant := range event.Participants {
			if _, ok := participantSet[participant]; ok {
				result = append(result, event)
				break
			}
		}
	}
	return result
}

func exploreDistance(current, other *world.Island) int {
	if current == nil || other == nil {
		return 0
	}
	return absInt(current.Size.Width-other.Size.Width) + absInt(current.Size.Height-other.Size.Height) + 1
}

func exploreWithinArea(location ExploreLocation, x, y int) bool {
	switch strings.ToLower(location.Name) {
	case "playa":
		return x <= 2 || y <= 2 || x >= location.X+2
	case "bosque":
		return x >= maxInt(0, location.X-3) && y >= maxInt(0, location.Y-2)
	default:
		return absInt(x-location.X) <= 3 && absInt(y-location.Y) <= 2
	}
}

func exploreBuildingAt(island *world.Island, x, y int) (world.Building, bool) {
	if island == nil {
		return world.Building{}, false
	}
	for index := range island.Buildings {
		building := island.Buildings[index]
		posX, posY := buildingPosition(island, index)
		if posX == x && posY == y {
			return building, true
		}
	}
	return world.Building{}, false
}

func explorePobleAt(w *world.World, islandID string, x, y int) (*entities.Poble, bool) {
	if w == nil {
		return nil, false
	}
	for _, poble := range w.GetAllPobles() {
		if poble == nil {
			continue
		}
		location, ok := w.GetLocation(poble.ID)
		if ok && location.IslandID == islandID && location.X == x && location.Y == y {
			return poble, true
		}
	}
	return nil, false
}

func findBuildingByID(w *world.World, buildingID string) (world.Building, *world.Island, bool) {
	if w == nil || buildingID == "" {
		return world.Building{}, nil, false
	}
	for _, island := range w.Islands {
		if island == nil {
			continue
		}
		for index := range island.Buildings {
			if island.Buildings[index].ID == buildingID {
				return island.Buildings[index], island, true
			}
		}
	}
	return world.Building{}, nil, false
}

func humanizeBuildingType(buildingType world.BuildingType) string {
	return strings.ToLower(strings.ReplaceAll(string(buildingType), "_", " "))
}

func absInt(value int) int {
	if value < 0 {
		return -value
	}
	return value
}
