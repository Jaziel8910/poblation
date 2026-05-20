package views

import (
	"fmt"
	"sort"
	"strings"
	"unicode"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/user/poblation/internal/ai"
	gameengine "github.com/user/poblation/internal/engine"
	"github.com/user/poblation/internal/events"
	"github.com/user/poblation/internal/templates"
	"github.com/user/poblation/internal/world"
)

// AppStateSnapshot is the shared root state the map view needs.
type AppStateSnapshot struct {
	World              *world.World
	EventFeed          []events.GameEvent
	SelectedPobleID    string
	SelectedBuildingID string
	Width              int
	Height             int
	Speed              float64
	IsPaused           bool
	TemplateEngine     *templates.TemplateEngine
	IsDirectorMode     bool
	Ending             *gameengine.Ending
	LastIntents        map[string]string
	IntentReasons      map[string]string
}

// ConsoleCommandMsg lets the root app process a command typed in the feed console.
type ConsoleCommandMsg struct {
	Command string
}

// OpenPobleDetailMsg asks the root app to open the poble detail screen.
type OpenPobleDetailMsg struct {
	PobleID string
}

// MapModel renders the main map layout.
type MapModel struct {
	state           AppStateSnapshot
	feed            FeedModel
	selectedPobleID string
	scrollX         int
	scrollY         int
	compactPanel    int
}

// FeedModel stores the event feed viewport plus the command console.
type FeedModel struct {
	viewport     viewport.Model
	input        textinput.Model
	events       []events.GameEvent
	commandLog   []string
	historyIndex int
}

type visiblePoble struct {
	ID       string
	Name     string
	Location world.Location
}

type tile struct {
	symbol string
	label  string
}

var (
	mapPanelStyle  = panelStyle("")
	feedPanelStyle = panelStyle("")
	mindPanelStyle = panelStyle("accent")

	mapHeaderStyle     = HeaderStyle
	feedHeaderStyle    = BodyStyle.Bold(true)
	speedStyle         = AccentStyle
	consolePromptStyle = AccentStyle

	mapCellWidth = 3

	selectedCellStyle = lipgloss.NewStyle().
				Width(mapCellWidth).
				Align(lipgloss.Center).
				Foreground(accentColor).
				Bold(true)

	plainCellStyle = lipgloss.NewStyle().
			Width(mapCellWidth).
			Align(lipgloss.Center)

	waterCellStyle  = plainCellStyle.Foreground(lipgloss.Color("#5DADEC"))
	landCellStyle   = plainCellStyle.Foreground(lipgloss.Color("#4A4A4A"))
	forestCellStyle = plainCellStyle.
			Foreground(lipgloss.Color("#6BCB77")).
			Bold(true)
	mountainCellStyle = plainCellStyle.
				Foreground(lipgloss.Color("#C9C9C9")).
				Bold(true)
	homeCellStyle = plainCellStyle.
			Foreground(lipgloss.Color("#F2C078")).
			Bold(true)
	pobleCellStyle = plainCellStyle.
			Foreground(lipgloss.Color("#FF6B6B")).
			Bold(true)
	eventCellStyle = plainCellStyle.
			Foreground(lipgloss.Color("#FFD93D")).
			Bold(true)
)

// NewMapModel returns the main map model with feed viewport and console ready.
func NewMapModel() MapModel {
	input := textinput.New()
	input.Prompt = "> "
	input.PromptStyle = consolePromptStyle
	input.TextStyle = lipgloss.NewStyle().Foreground(primaryColor)
	input.Placeholder = "pause, resume, tick, speed 2"
	input.PlaceholderStyle = lipgloss.NewStyle().Foreground(mutedColor)

	return MapModel{
		feed: FeedModel{
			viewport:     viewport.New(10, 8),
			input:        input,
			events:       []events.GameEvent{},
			commandLog:   []string{},
			historyIndex: 0,
		},
	}
}

// Init satisfies tea.Model.
func (m MapModel) Init() tea.Cmd {
	return nil
}

// SyncAppState updates the map with the latest root app snapshot.
func (m MapModel) SyncAppState(snapshot AppStateSnapshot) tea.Model {
	m.state = snapshot
	if snapshot.SelectedPobleID != "" {
		m.selectedPobleID = snapshot.SelectedPobleID
	}
	m.ensureSelectedPoble()
	m.clampScroll()
	width, height := m.feedViewportSize()
	m.feed = m.feed.Sync(snapshot.EventFeed, width, height)
	return m
}

func (m MapModel) Resize(width, height int) tea.Model {
	m.state.Width = width
	m.state.Height = height
	feedWidth, feedHeight := m.feedViewportSize()
	m.feed = m.feed.Sync(m.feed.events, feedWidth, feedHeight)
	m.clampScroll()
	return m
}

// Update handles map movement, feed scrolling, and console entry.
func (m MapModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch typed := msg.(type) {
	case tea.WindowSizeMsg:
		return m.Resize(typed.Width, typed.Height), nil
	case tea.KeyMsg:
		return m.handleKey(typed)
	default:
		return m, nil
	}
}

// View renders a responsive map shell.
func (m MapModel) View() string {
	layout := LayoutManager{Width: m.state.Width, Height: m.state.Height}
	height := maxInt(8, m.state.Height)
	if layout.IsSinglePanel() {
		return m.renderCompactView(layout, height)
	}
	if layout.IsTriplePanel() {
		mapWidth, mindWidth, feedWidth := layout.TriplePanelWidths()
		mapView := mapPanelStyle.Width(mapWidth).Height(height).Render(m.renderMapPanel(mapWidth))
		mindView := mindPanelStyle.Width(mindWidth).Height(height).Render(m.renderMindPanel(mindWidth))
		feedView := feedPanelStyle.Width(feedWidth).Height(height).Render(m.feed.View())
		return lipgloss.JoinHorizontal(lipgloss.Top, mapView, mindView, feedView)
	}
	mapWidth, feedWidth := m.panelWidths()
	mapView := mapPanelStyle.Width(mapWidth).Height(height).Render(m.renderMapPanel(mapWidth))
	feedView := feedPanelStyle.Width(feedWidth).Height(height).Render(m.feed.View())
	return lipgloss.JoinHorizontal(lipgloss.Top, mapView, feedView)
}

func (m MapModel) renderCompactView(layout LayoutManager, height int) string {
	panels := []string{"mapa", "mente", "feed"}
	switch m.compactPanel {
	case 1:
		return lipgloss.JoinVertical(
			lipgloss.Left,
			mindPanelStyle.Width(maxInt(26, m.state.Width)).Height(height).Render(m.renderMindPanel(maxInt(26, m.state.Width))),
			compactHint(layout, panels, m.compactPanel),
		)
	case 2:
		return lipgloss.JoinVertical(
			lipgloss.Left,
			feedPanelStyle.Width(maxInt(26, m.state.Width)).Height(height).Render(m.feed.View()),
			compactHint(layout, panels, m.compactPanel),
		)
	default:
		return lipgloss.JoinVertical(
			lipgloss.Left,
			mapPanelStyle.Width(maxInt(26, m.state.Width)).Height(height).Render(m.renderMapPanel(maxInt(26, m.state.Width))),
			compactHint(layout, panels, m.compactPanel),
		)
	}
}

func (m MapModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.feed.input.Focused() {
		return m.handleConsoleKey(msg)
	}

	switch msg.String() {
	case "enter":
		if m.selectedPobleID == "" {
			return m, nil
		}
		return m, func() tea.Msg {
			return OpenPobleDetailMsg{PobleID: m.selectedPobleID}
		}
	case "tab":
		if (LayoutManager{Width: m.state.Width, Height: m.state.Height}).IsSinglePanel() {
			m.compactPanel = (m.compactPanel + 1) % 3
			return m, nil
		}
		m.cycleSelectedPoble(1)
		return m, nil
	case "shift+tab":
		if (LayoutManager{Width: m.state.Width, Height: m.state.Height}).IsSinglePanel() {
			m.compactPanel = (m.compactPanel + 2) % 3
			return m, nil
		}
		m.cycleSelectedPoble(-1)
		return m, nil
	case "left":
		m.scrollX--
		m.clampScroll()
		return m, nil
	case "right":
		m.scrollX++
		m.clampScroll()
		return m, nil
	case "up":
		m.scrollY--
		m.clampScroll()
		return m, nil
	case "down":
		m.scrollY++
		m.clampScroll()
		return m, nil
	case "pgup":
		m.feed.viewport.ScrollUp(maxInt(1, m.feed.viewport.Height/2))
		return m, nil
	case "pgdown":
		m.feed.viewport.ScrollDown(maxInt(1, m.feed.viewport.Height/2))
		return m, nil
	case "home":
		m.feed.viewport.GotoTop()
		return m, nil
	case "end":
		m.feed.viewport.GotoBottom()
		return m, nil
	default:
		if isConsoleTypingKey(msg) {
			cmd := m.feed.input.Focus()
			var updateCmd tea.Cmd
			m.feed.input, updateCmd = m.feed.input.Update(msg)
			return m, tea.Batch(cmd, updateCmd)
		}
		return m, nil
	}
}

func (m MapModel) handleConsoleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.feed.input.Blur()
		return m, nil
	case "enter":
		command := strings.TrimSpace(m.feed.input.Value())
		m.feed.input.SetValue("")
		m.feed.input.CursorEnd()
		if command == "" {
			return m, nil
		}
		m.feed.pushHistory(command)
		return m, func() tea.Msg {
			return ConsoleCommandMsg{Command: command}
		}
	case "up":
		m.feed.restoreHistory(-1)
		return m, nil
	case "down":
		m.feed.restoreHistory(1)
		return m, nil
	default:
		var cmd tea.Cmd
		m.feed.input, cmd = m.feed.input.Update(msg)
		return m, cmd
	}
}

func (m MapModel) renderMapPanel(panelWidth int) string {
	island := m.currentIsland()
	if island == nil {
		return lipgloss.JoinVertical(
			lipgloss.Left,
			mapHeaderStyle.Render("MAPA"),
			mutedStyle.Render("No hay isla cargada."),
		)
	}

	header := lipgloss.JoinHorizontal(
		lipgloss.Left,
		mapHeaderStyle.Render("MAPA"),
		mutedStyle.Render("  "+island.Name),
		speedStyle.Render("  "+m.speedIndicator()),
	)

	innerWidth := maxInt(8, panelWidth-mapPanelStyle.GetHorizontalFrameSize())
	innerHeight := maxInt(5, m.state.Height-mapPanelStyle.GetVerticalFrameSize())
	cellCols := maxInt(4, innerWidth/mapCellWidth)
	cellRows := maxInt(1, innerHeight-6)

	grid := m.renderMapGrid(island, cellCols, cellRows)
	legend := mutedStyle.Render("~ agua . suelo t bosque ^ monte H casa AB poble")
	footer := lipgloss.JoinVertical(
		lipgloss.Left,
		legend,
		mutedStyle.Render(m.selectedSummary()),
		mutedStyle.Render(fmt.Sprintf("scroll %d,%d  isla %dx%d", m.scrollX, m.scrollY, island.Size.Width, island.Size.Height)),
	)
	return lipgloss.JoinVertical(lipgloss.Left, header, grid, footer)
}

func (m MapModel) renderMindPanel(panelWidth int) string {
	poble := selectedMindPoble(m.state)
	if poble == nil {
		return lipgloss.JoinVertical(
			lipgloss.Left,
			HeaderStyle.Render("MENTE"),
			MutedStyle.Render("No hay nadie seleccionado."),
		)
	}

	lines := []string{
		HeaderStyle.Render("MENTE"),
		SubheaderStyle.Render(fmt.Sprintf("%s %s", moodEmoji(poble.CurrentMood), poble.Name)),
		BodyStyle.Render(fmt.Sprintf("Mood: %s", poble.CurrentMood.String())),
		BodyStyle.Render(fmt.Sprintf("Estabilidad %d · Terapia %d", poble.Mental.Stability, poble.Mental.TherapyLevel)),
		BodyStyle.Render(fmt.Sprintf("Traumas %d · Secretos %d", len(poble.Mental.Traumas), len(poble.Secrets))),
	}
	if intent := m.state.LastIntents[poble.ID]; intent != "" {
		lines = append(lines, AccentStyle.Render("Ahora: "+humanIntent(intent)))
	}
	if reason := m.state.IntentReasons[poble.ID]; reason != "" {
		lines = append(lines, MutedStyle.Width(maxInt(18, panelWidth-4)).Render("Por que: "+humanReason(reason, *poble)))
	}
	if len(poble.Mental.Conditions) > 0 {
		conditions := make([]string, 0, len(poble.Mental.Conditions))
		for _, condition := range poble.Mental.Conditions {
			conditions = append(conditions, string(condition))
		}
		lines = append(lines, WarningStyle.Render("Condiciones: "+strings.Join(conditions, ", ")))
	}
	lines = append(lines, MutedStyle.Width(maxInt(18, panelWidth-4)).Render(m.selectedSummary()))
	return strings.Join(lines, "\n")
}

func (m MapModel) renderMapGrid(island *world.Island, cols, rows int) string {
	lines := make([]string, 0, rows)
	for y := 0; y < rows; y++ {
		worldY := m.scrollY + y
		line := strings.Builder{}
		for x := 0; x < cols; x++ {
			worldX := m.scrollX + x
			current := m.tileAt(island, worldX, worldY)
			selected := m.isSelectedAt(worldX, worldY)
			line.WriteString(renderCell(current.symbol, selected))
		}
		lines = append(lines, line.String())
	}
	return strings.Join(lines, "\n")
}

func (m MapModel) tileAt(island *world.Island, x, y int) tile {
	if x < 0 || y < 0 || x >= island.Size.Width || y >= island.Size.Height {
		return tile{symbol: " ", label: " "}
	}

	if poble, ok := m.pobleAt(island.ID, x, y); ok {
		return tile{
			symbol: m.pobleSymbol(poble),
			label:  "",
		}
	}
	if building, ok := derivedHomeAt(island, x, y); ok {
		return tile{
			symbol: "H",
			label:  abbreviateName(building.Name),
		}
	}
	return tile{symbol: terrainRune(island, x, y), label: " "}
}

func (m MapModel) pobleAt(islandID string, x, y int) (visiblePoble, bool) {
	if m.state.World == nil {
		return visiblePoble{}, false
	}
	for _, poble := range m.state.World.GetAllPobles() {
		location, ok := m.state.World.GetLocation(poble.ID)
		if !ok {
			continue
		}
		if location.IslandID == islandID && location.X == x && location.Y == y {
			return visiblePoble{ID: poble.ID, Name: poble.Name, Location: location}, true
		}
	}
	return visiblePoble{}, false
}

func (m MapModel) pobleSymbol(poble visiblePoble) string {
	if m.state.World == nil {
		return abbreviateName(poble.Name)
	}

	for _, event := range m.state.World.ActiveEvents {
		if !activeEventTouchesPoble(event, poble.ID) {
			continue
		}
		switch {
		case isViolentActiveEvent(event.Type):
			return "!!"
		case isRomanticActiveEvent(event.Type):
			return "<3"
		case isConversationActiveEvent(event.Type):
			return ".."
		}
	}

	fullPoble := m.state.World.GetPoble(poble.ID)
	if fullPoble != nil && fullPoble.Needs.Sleep >= 78 {
		return "zz"
	}
	return abbreviateName(poble.Name)
}

func (m MapModel) currentIsland() *world.Island {
	if m.state.World == nil {
		return nil
	}
	if island := m.state.World.GetIsland("island_0"); island != nil {
		return island
	}
	for _, island := range m.state.World.Islands {
		if island != nil && island.IsDiscovered {
			return island
		}
	}
	for _, island := range m.state.World.Islands {
		if island != nil {
			return island
		}
	}
	return nil
}

func (m *MapModel) ensureSelectedPoble() {
	if m.state.World == nil {
		m.selectedPobleID = ""
		return
	}
	if m.selectedPobleID != "" && m.state.World.GetPoble(m.selectedPobleID) != nil {
		m.ensureSelectedVisible()
		return
	}
	pobles := m.islandPobles()
	if len(pobles) == 0 {
		m.selectedPobleID = ""
		return
	}
	m.selectedPobleID = pobles[0].ID
	m.ensureSelectedVisible()
}

func (m MapModel) islandPobles() []visiblePoble {
	island := m.currentIsland()
	if island == nil || m.state.World == nil {
		return nil
	}

	pobles := make([]visiblePoble, 0, len(island.Pobles))
	for _, id := range island.Pobles {
		poble := m.state.World.GetPoble(id)
		location, ok := m.state.World.GetLocation(id)
		if poble == nil || !ok {
			continue
		}
		pobles = append(pobles, visiblePoble{ID: id, Name: poble.Name, Location: location})
	}
	sort.SliceStable(pobles, func(i, j int) bool {
		if pobles[i].Name == pobles[j].Name {
			return pobles[i].ID < pobles[j].ID
		}
		return pobles[i].Name < pobles[j].Name
	})
	return pobles
}

func (m *MapModel) cycleSelectedPoble(step int) {
	pobles := m.islandPobles()
	if len(pobles) == 0 {
		m.selectedPobleID = ""
		return
	}

	index := 0
	for i, poble := range pobles {
		if poble.ID == m.selectedPobleID {
			index = i
			break
		}
	}
	index = (index + step + len(pobles)) % len(pobles)
	m.selectedPobleID = pobles[index].ID
	m.ensureSelectedVisible()
}

func (m *MapModel) ensureSelectedVisible() {
	if m.state.World == nil || m.selectedPobleID == "" {
		return
	}
	location, ok := m.state.World.GetLocation(m.selectedPobleID)
	if !ok {
		return
	}

	cols, rows := m.visibleCellWindow()
	if location.X < m.scrollX {
		m.scrollX = location.X
	}
	if location.X >= m.scrollX+cols {
		m.scrollX = location.X - cols + 1
	}
	if location.Y < m.scrollY {
		m.scrollY = location.Y
	}
	if location.Y >= m.scrollY+rows {
		m.scrollY = location.Y - rows + 1
	}
	m.clampScroll()
}

func (m *MapModel) clampScroll() {
	island := m.currentIsland()
	if island == nil {
		m.scrollX = 0
		m.scrollY = 0
		return
	}

	cols, rows := m.visibleCellWindow()
	maxX := maxInt(0, island.Size.Width-cols)
	maxY := maxInt(0, island.Size.Height-rows)
	m.scrollX = clampInt(m.scrollX, 0, maxX)
	m.scrollY = clampInt(m.scrollY, 0, maxY)
}

func (m MapModel) visibleCellWindow() (int, int) {
	mapWidth, _ := m.panelWidths()
	innerWidth := maxInt(8, mapWidth-mapPanelStyle.GetHorizontalFrameSize())
	innerHeight := maxInt(5, m.state.Height-mapPanelStyle.GetVerticalFrameSize())
	return maxInt(4, innerWidth/mapCellWidth), maxInt(1, innerHeight-6)
}

func (m MapModel) feedViewportSize() (int, int) {
	_, feedWidth := m.panelWidths()
	innerWidth := maxInt(12, feedWidth-feedPanelStyle.GetHorizontalFrameSize())
	innerHeight := maxInt(6, m.state.Height-feedPanelStyle.GetVerticalFrameSize()-2)
	return innerWidth, innerHeight
}

func (m MapModel) panelWidths() (int, int) {
	return LayoutManager{Width: m.state.Width, Height: m.state.Height}.MainPanelWidths()
}

func (m MapModel) selectedSummary() string {
	if m.state.World == nil || m.selectedPobleID == "" {
		return "Sin poble seleccionado"
	}
	poble := m.state.World.GetPoble(m.selectedPobleID)
	location, ok := m.state.World.GetLocation(m.selectedPobleID)
	if poble == nil || !ok {
		return "Sin poble seleccionado"
	}
	return fmt.Sprintf("Seleccionado: %s (%d,%d)", poble.Name, location.X, location.Y)
}

func (m MapModel) speedIndicator() string {
	if m.state.IsPaused {
		return "PAUSA"
	}
	switch {
	case m.state.Speed >= 4:
		return "RUN x4"
	case m.state.Speed >= 2:
		return "RUN x2"
	default:
		return "RUN x1"
	}
}

func (m MapModel) isSelectedAt(x, y int) bool {
	if m.state.World == nil || m.selectedPobleID == "" {
		return false
	}
	location, ok := m.state.World.GetLocation(m.selectedPobleID)
	return ok && location.X == x && location.Y == y
}

func (f FeedModel) Sync(eventFeed []events.GameEvent, width, height int) FeedModel {
	f.events = append([]events.GameEvent(nil), eventFeed...)
	f.viewport.Width = maxInt(10, width)
	f.viewport.Height = maxInt(3, height)
	f.viewport.SetContent(renderFeedEvents(f.events, maxInt(8, width-1)))
	if f.viewport.YOffset == 0 {
		f.viewport.GotoBottom()
	}
	f.input.Width = maxInt(8, width-4)
	if f.historyIndex > len(f.commandLog) {
		f.historyIndex = len(f.commandLog)
	}
	return f
}

func (f FeedModel) View() string {
	content := f.viewport.View()
	if strings.TrimSpace(content) == "" {
		content = mutedStyle.Render("Aun no hay eventos.")
	}
	return lipgloss.JoinVertical(
		lipgloss.Left,
		feedHeaderStyle.Render("FEED"),
		content,
		f.input.View(),
	)
}

func (f *FeedModel) pushHistory(command string) {
	f.commandLog = append(f.commandLog, command)
	f.historyIndex = len(f.commandLog)
}

func (f *FeedModel) restoreHistory(step int) {
	if len(f.commandLog) == 0 {
		return
	}

	next := clampInt(f.historyIndex+step, 0, len(f.commandLog))
	f.historyIndex = next
	if next == len(f.commandLog) {
		f.input.SetValue("")
		f.input.CursorEnd()
		return
	}
	f.input.SetValue(f.commandLog[next])
	f.input.CursorEnd()
}

func renderFeedEvents(eventFeed []events.GameEvent, width int) string {
	if len(eventFeed) == 0 {
		return ""
	}

	lines := make([]string, 0, len(eventFeed))
	for _, event := range eventFeed {
		if !isMapFeedEventVisible(event) {
			continue
		}
		prefix := eventIcon(event.Type) + " "
		timestamp := fmt.Sprintf("[%03d %02d:%02d]", event.Timestamp.Day, event.Timestamp.Hour, event.Timestamp.Minute)
		description := strings.TrimSpace(event.Description)
		if description == "" {
			description = humanizeEventType(event.Type)
		}
		line := truncateRunes(prefix+timestamp+" "+description, width)
		lines = append(lines, eventStyle(event.Type).Render(line))
	}
	return strings.Join(lines, "\n")
}

func isMapFeedEventVisible(event events.GameEvent) bool {
	if strings.TrimSpace(event.Description) != "" {
		return true
	}
	switch event.Type {
	case events.EventBirth, events.EventBirthday, events.EventPregnancy, events.EventAdoption:
		return true
	case events.EventDeathNatural, events.EventDeathAccident, events.EventDeathMurder, events.EventSuicide,
		events.EventFuneral:
		return true
	case events.EventMarriage, events.EventDivorce, events.EventAffairStart, events.EventAffairEnd,
		events.EventBetrayalRevealed, events.EventForgiveness:
		return true
	case events.EventFightVerbal, events.EventFightPhysical, events.EventWarDeclaration,
		events.EventPeaceTreaty, events.EventCoup, events.EventRevolution, events.EventExile:
		return true
	case events.EventElection, events.EventEraChange, events.EventTechDiscovered,
		events.EventTradeEstablished, events.EventMonopolyFormed:
		return true
	case events.EventEarthquake, events.EventStorm, events.EventDrought, events.EventPlague,
		events.EventFire, events.EventFlood, events.EventIslandDiscovery:
		return true
	default:
		return false
	}
}

func terrainRune(island *world.Island, x, y int) string {
	if island == nil {
		return "."
	}
	if x == 0 || y == 0 || x == island.Size.Width-1 || y == island.Size.Height-1 {
		return "~"
	}
	score := tileNoise(island.ID, x, y)
	switch {
	case score%13 == 0:
		return "^"
	case score%7 == 0 || island.Biome == world.BiomeForest:
		return "t"
	default:
		return "."
	}
}

func derivedHomeAt(island *world.Island, x, y int) (world.Building, bool) {
	if island == nil {
		return world.Building{}, false
	}
	for index, building := range island.Buildings {
		if building.Type != world.BuildingHome {
			continue
		}
		posX, posY := buildingPosition(island, index)
		if posX == x && posY == y {
			return building, true
		}
	}
	return world.Building{}, false
}

func buildingPosition(island *world.Island, index int) (int, int) {
	if island == nil {
		return 0, 0
	}
	spanX := maxInt(1, island.Size.Width-4)
	spanY := maxInt(1, island.Size.Height-4)
	return 2 + ((index * 5) % spanX), 2 + ((index * 3) % spanY)
}

func tileNoise(seed string, x, y int) int {
	sum := 0
	for _, r := range seed {
		sum += int(r)
	}
	return sum + (x * 17) + (y * 31)
}

func renderCell(value string, selected bool) string {
	if selected {
		return selectedCellStyle.Render(value)
	}
	switch value {
	case "~":
		return waterCellStyle.Render(value)
	case ".":
		return landCellStyle.Render(value)
	case "t":
		return forestCellStyle.Render(value)
	case "^":
		return mountainCellStyle.Render(value)
	case "H":
		return homeCellStyle.Render(value)
	case "!!", "<3", "..", "zz":
		return eventCellStyle.Render(value)
	default:
		if strings.TrimSpace(value) != "" {
			return pobleCellStyle.Render(value)
		}
		return plainCellStyle.Render(value)
	}
}

func abbreviateName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "?"
	}
	runes := []rune(strings.ToUpper(name))
	if len(runes) == 1 {
		return string(runes)
	}
	return string(runes[:2])
}

func humanizeEventType(eventType events.EventType) string {
	switch eventType {
	case events.EventFightVerbal:
		return "Discusion fuerte"
	case events.EventFightPhysical:
		return "Pelea fisica"
	case events.EventElection:
		return "Eleccion"
	case events.EventTradeEstablished:
		return "Intercambio cerrado"
	case events.EventTechDiscovered:
		return "Tecnologia descubierta"
	case events.EventBetrayalRevealed:
		return "Traicion revelada"
	case events.EventBirth:
		return "Nacimiento"
	case events.EventPregnancy:
		return "Embarazo"
	case events.EventDeathNatural, events.EventDeathAccident, events.EventDeathMurder, events.EventSuicide:
		return "Muerte"
	case events.EventMarriage:
		return "Matrimonio"
	case events.EventDivorce:
		return "Ruptura"
	case events.EventFire:
		return "Incendio"
	case events.EventStorm:
		return "Tormenta"
	case events.EventPlague:
		return "Plaga"
	case events.EventEraChange:
		return "Cambio de era"
	}
	parts := strings.Split(strings.ToLower(string(eventType)), "_")
	for i := range parts {
		if parts[i] == "" {
			continue
		}
		runes := []rune(parts[i])
		runes[0] = unicode.ToUpper(runes[0])
		parts[i] = string(runes)
	}
	return strings.Join(parts, " ")
}

func activeEventTouchesPoble(event ai.GameEvent, pobleID string) bool {
	if event.PrimaryActor == pobleID || event.TargetID == pobleID {
		return true
	}
	for _, participant := range event.Participants {
		if participant == pobleID {
			return true
		}
	}
	return false
}

func isViolentActiveEvent(eventType ai.GameEventType) bool {
	return eventType == ai.GameEventDeath || eventType == ai.GameEventThreat || eventType == ai.GameEventConflict
}

func isRomanticActiveEvent(eventType ai.GameEventType) bool {
	return eventType == ai.GameEventIntimacy
}

func isConversationActiveEvent(eventType ai.GameEventType) bool {
	return eventType == ai.GameEventSocialPositive || eventType == ai.GameEventSocialNegative || eventType == ai.GameEventBetrayal
}

func isImportantEvent(eventType events.EventType) bool {
	return isViolentEvent(eventType) || eventType == events.EventEraChange || eventType == events.EventPopulationMilestone
}

func isViolentEvent(eventType events.EventType) bool {
	switch eventType {
	case events.EventFightVerbal, events.EventFightPhysical, events.EventDeathAccident,
		events.EventDeathMurder, events.EventDeathNatural, events.EventSuicide,
		events.EventWarDeclaration, events.EventCoup, events.EventRevolution:
		return true
	default:
		return false
	}
}

func isPositiveEvent(eventType events.EventType) bool {
	switch eventType {
	case events.EventBirth, events.EventMarriage, events.EventForgiveness,
		events.EventRecovery, events.EventTechDiscovered, events.EventPopulationMilestone:
		return true
	default:
		return false
	}
}

func isDramaEvent(eventType events.EventType) bool {
	switch eventType {
	case events.EventPregnancy, events.EventRevelation, events.EventDivorce,
		events.EventAffairStart, events.EventAffairEnd, events.EventBetrayalRevealed,
		events.EventRumourSpread, events.EventGossipChain, events.EventEraChange:
		return true
	default:
		return false
	}
}

func isConsoleTypingKey(msg tea.KeyMsg) bool {
	switch msg.Type {
	case tea.KeyRunes, tea.KeySpace, tea.KeyBackspace, tea.KeyDelete:
		return true
	default:
		return false
	}
}

func truncateRunes(text string, limit int) string {
	if limit <= 0 {
		return ""
	}
	runes := []rune(text)
	if len(runes) <= limit {
		return text
	}
	if limit == 1 {
		return string(runes[:1])
	}
	return string(runes[:limit-1]) + "."
}

func clampInt(value, low, high int) int {
	if value < low {
		return low
	}
	if value > high {
		return high
	}
	return value
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
