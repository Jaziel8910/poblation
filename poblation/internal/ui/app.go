package ui

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/user/poblation/internal/ai"
	"github.com/user/poblation/internal/config"
	"github.com/user/poblation/internal/engine"
	"github.com/user/poblation/internal/entities"
	"github.com/user/poblation/internal/events"
	"github.com/user/poblation/internal/save"
	"github.com/user/poblation/internal/templates"
	uiviews "github.com/user/poblation/internal/ui/views"
	"github.com/user/poblation/internal/world"
)

// ViewType identifies every top-level Bubbletea screen in the app shell.
type ViewType int

const (
	VIEW_MENU ViewType = iota
	VIEW_MAIN_MAP
	VIEW_MIND
	VIEW_DIALOGUE
	VIEW_WORLD_EXPLORE
	VIEW_HOUSE_EXPLORE
	VIEW_POBLES_LIST
	VIEW_POBLE_DETAIL
	VIEW_EVENTS_FEED
	VIEW_SETTLEMENT
	VIEW_MINIGAME_SEX
	VIEW_MINIGAME_FIGHT
	VIEW_NEWSPAPER
	VIEW_ENDINGS
	VIEW_CREATE_POBLE
	VIEW_SETTINGS
	VIEW_DEBUG
)

// NotificationType controls notification color and urgency.
type NotificationType string

const (
	NotificationInfo  NotificationType = "info"
	NotificationDeath NotificationType = "death"
	NotificationBirth NotificationType = "birth"
	NotificationDrama NotificationType = "drama"
)

// Notification is a short-lived floating event notice.
type Notification struct {
	ID        string
	Type      NotificationType
	Message   string
	CreatedAt time.Time
	ExpiresAt time.Time
}

// GameTickMsg adapts engine ticks into Bubbletea messages.
type GameTickMsg struct {
	Tick engine.GameTick
}

// GameEventMsg adapts processed game events into Bubbletea messages.
type GameEventMsg struct {
	Event events.GameEvent
}

// ViewChangeMsg lets child models ask the app shell to switch screens.
type ViewChangeMsg struct {
	View               ViewType
	SelectedPobleID    string
	SelectedBuildingID string
}

type appInitializedMsg struct {
	tickSub chan engine.GameTick
}

type templatesLoadedMsg struct {
	err error
}

type templateLoadStartedMsg struct {
	updates <-chan templateLoadUpdateMsg
}

type templateLoadUpdateMsg struct {
	Category         string
	LoadedTemplates  int
	TotalTemplates   int
	LoadedCategories int
	TotalCategories  int
	CategoryLoaded   map[string]int
	CategoryTotals   map[string]int
	Done             bool
	Err              error
}

type loadingPulseMsg struct {
	at time.Time
}

type notificationExpiredMsg struct {
	id string
}

type appStateAwareModel interface {
	SyncAppState(uiviews.AppStateSnapshot) tea.Model
}

type resizableModel interface {
	Resize(width, height int) tea.Model
}

type viewEnterAwareModel interface {
	OnEnter() (tea.Model, tea.Cmd)
}

type globalNavigationBlockingModel interface {
	BlocksGlobalNavigation() bool
}

// KeyMap stores the global app bindings.
type KeyMap struct {
	Mind       key.Binding
	World      key.Binding
	Pobles     key.Binding
	Events     key.Binding
	Settlement key.Binding
	Pause      key.Binding
	SpeedUp    key.Binding
	SpeedDown  key.Binding
	Select     key.Binding
	Back       key.Binding
	Debug      key.Binding
	Quit       key.Binding
}

// AppModel is the Bubbletea root model. It owns view routing and shared state.
type AppModel struct {
	CurrentView        ViewType
	PreviousView       ViewType
	World              *world.World
	Engine             *engine.TimeEngine
	SelectedPobleID    string
	SelectedBuildingID string
	EventFeed          []events.GameEvent
	SubModels          map[ViewType]tea.Model
	KeyMap             KeyMap
	Width              int
	Height             int
	IsDebugMode        bool
	ConsoleInput       string
	NotificationQueue  []Notification
	Console            *engine.ConsoleSystem
	ActiveEnding       *engine.Ending
	Orchestrator       *engine.Orchestrator
	Mode               config.GameMode

	ctx             context.Context
	cancel          context.CancelFunc
	eventQueue      *events.EventQueue
	templateEngine  *templates.TemplateEngine
	rng             *rand.Rand
	tickSub         chan engine.GameTick
	lastIntents     map[string]string
	intentReasons   map[string]string
	speed           float64
	isPaused        bool
	templateLoadSub <-chan templateLoadUpdateMsg
	loading         templateLoadingState
}

// NewAppModel builds the app shell with safe defaults for standalone startup.
func NewAppModel(w *world.World, clock *engine.TimeEngine) AppModel {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	if w == nil {
		w = world.NewWorld(time.Now().UnixNano())
	}
	if clock == nil {
		clock = engine.NewTimeEngine()
	}

	ctx, cancel := context.WithCancel(context.Background())
	return AppModel{
		CurrentView:    VIEW_MENU,
		PreviousView:   VIEW_MENU,
		World:          w,
		Engine:         clock,
		EventFeed:      []events.GameEvent{},
		SubModels:      defaultSubModels(),
		KeyMap:         DefaultKeyMap(),
		Width:          80,
		Height:         24,
		ctx:            ctx,
		cancel:         cancel,
		eventQueue:     events.NewEventQueue(rng),
		templateEngine: templates.NewTemplateEngine(rng),
		rng:            rng,
		lastIntents:    map[string]string{},
		intentReasons:  map[string]string{},
		speed:          1.0,
		Console:        engine.NewConsoleSystem(w, clock, rng),
		loading:        newTemplateLoadingState(),
		Mode:           config.LoadOrDefault().Settings.Gameplay.ActiveMode,
	}
}

// NewAppModelWithOrchestrator builds the UI shell on top of the engine runtime.
func NewAppModelWithOrchestrator(orchestrator *engine.Orchestrator) AppModel {
	if orchestrator == nil {
		return NewAppModel(nil, nil)
	}
	model := NewAppModel(orchestrator.World(), orchestrator.TimeEngine())
	snapshot := orchestrator.GetWorldSnapshot()
	model.Orchestrator = orchestrator
	model.templateEngine = orchestrator.TemplateEngine()
	model.EventFeed = snapshot.EventFeed
	model.ActiveEnding = snapshot.ActiveEnding
	model.IsDebugMode = snapshot.Debug
	model.speed = snapshot.Speed
	model.isPaused = snapshot.IsPaused
	model.lastIntents, model.intentReasons = intentMapsFromSnapshot(snapshot)
	model.Mode = config.LoadOrDefault().Settings.Gameplay.ActiveMode
	model.Console = engine.NewConsoleSystem(model.World, model.Engine, model.rng)
	return model
}

// DefaultKeyMap returns the global app shortcuts.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Mind:       key.NewBinding(key.WithKeys("m"), key.WithHelp("M", "mente")),
		World:      key.NewBinding(key.WithKeys("w"), key.WithHelp("W", "mundo")),
		Pobles:     key.NewBinding(key.WithKeys("p"), key.WithHelp("P", "pobles")),
		Events:     key.NewBinding(key.WithKeys("e"), key.WithHelp("E", "eventos")),
		Settlement: key.NewBinding(key.WithKeys("s"), key.WithHelp("S", "settlement")),
		Pause:      key.NewBinding(key.WithKeys(" "), key.WithHelp("SPACE", "pausa")),
		SpeedUp:    key.NewBinding(key.WithKeys("+", "="), key.WithHelp("+", "velocidad")),
		SpeedDown:  key.NewBinding(key.WithKeys("-"), key.WithHelp("-", "lento")),
		Select:     key.NewBinding(key.WithKeys("enter"), key.WithHelp("ENTER", "seleccionar")),
		Back:       key.NewBinding(key.WithKeys("esc"), key.WithHelp("ESC", "atras")),
		Debug:      key.NewBinding(key.WithKeys("`"), key.WithHelp("`", "debug")),
		Quit:       key.NewBinding(key.WithKeys("ctrl+c", "q"), key.WithHelp("Q", "salir")),
	}
}

// Init starts runtime subscriptions and loads templates.
func (m AppModel) Init() tea.Cmd {
	return tea.Batch(
		initializeRuntimeCmd(m.Engine, m.ctx),
		startTemplateLoadingCmd(m.templateEngine, templateRoot()),
		loadingPulseCmd(),
	)
}

// Update routes messages through the root model and current sub-model.
func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch typed := msg.(type) {
	case appInitializedMsg:
		m.tickSub = typed.tickSub
		return m, waitForGameTick(m.tickSub)
	case templateLoadStartedMsg:
		m.templateLoadSub = typed.updates
		return m, waitForTemplateLoad(m.templateLoadSub)
	case templateLoadUpdateMsg:
		return m.handleTemplateLoadUpdate(typed)
	case templatesLoadedMsg:
		return m.handleTemplateLoad(typed)
	case loadingPulseMsg:
		return m.handleLoadingPulse(typed)
	case tea.WindowSizeMsg:
		return m.handleWindowSize(typed)
	case tea.KeyMsg:
		return m.handleKey(typed)
	case GameTickMsg:
		return m.handleGameTick(typed)
	case GameEventMsg:
		return m.handleGameEvent(typed)
	case ViewChangeMsg:
		return m.switchToView(typed.View, typed.SelectedPobleID, typed.SelectedBuildingID)
	case uiviews.OpenPobleDetailMsg:
		return m.switchToView(VIEW_POBLE_DETAIL, typed.PobleID, "")
	case uiviews.OpenHouseMsg:
		return m.switchToView(VIEW_HOUSE_EXPLORE, typed.OwnerID, typed.BuildingID)
	case uiviews.ExitEncounterMsg:
		return m.switchToView(VIEW_MAIN_MAP, "", "")
	case uiviews.ExitFightMsg:
		return m.switchToView(VIEW_MAIN_MAP, "", "")
	case uiviews.ConsoleCommandMsg:
		return m.handleConsoleCommand(typed)
	case uiviews.MenuNewCivilizationMsg:
		return m.beginNewCivilization()
	case uiviews.MenuOpenSettingsMsg:
		return m.switchToView(VIEW_SETTINGS, "", "")
	case uiviews.MenuLoadSaveMsg:
		return m.loadSaveData(typed.Data)
	case uiviews.MenuQuitMsg:
		if m.cancel != nil {
			m.cancel()
		}
		if m.Engine != nil {
			m.Engine.Stop()
		}
		return m, tea.Quit
	case uiviews.CloseSettingsMsg:
		return m.switchToView(VIEW_MENU, "", "")
	case uiviews.CreatePobleCancelMsg:
		return m.switchToView(VIEW_MENU, "", "")
	case uiviews.CreatePobleCompleteMsg:
		return m.finishPobleCreation(typed)
	case uiviews.CreatePobleAddAnotherMsg:
		return m.openNextFounderCreation()
	case uiviews.CreatePobleStartWorldMsg:
		return m.startCreatedCivilization()
	case uiviews.RestartCivilizationMsg:
		return m.beginNewCivilization()
	case notificationExpiredMsg:
		return m.expireNotification(typed.id), nil
	default:
		return m.updateCurrentSubModel(msg)
	}
}

// View renders the active screen plus always-on chrome.
func (m AppModel) View() string {
	width := safeWidth(m.Width)
	status := m.renderStatusBar(width)
	body := m.renderBody(width)
	overlay := m.renderNotificationOverlay(width)
	nav := m.renderNavigationBar(width)

	parts := []string{status}
	if overlay != "" {
		parts = append(parts, overlay)
	}
	parts = append(parts, body, nav)
	return appBackgroundStyle.Width(width).Render(lipgloss.JoinVertical(lipgloss.Left, parts...))
}

func initializeRuntimeCmd(clock *engine.TimeEngine, ctx context.Context) tea.Cmd {
	return func() tea.Msg {
		if clock == nil {
			return appInitializedMsg{}
		}
		sub := clock.Subscribe()
		clock.Start(ctx)
		return appInitializedMsg{tickSub: sub}
	}
}

func startTemplateLoadingCmd(engine *templates.TemplateEngine, root string) tea.Cmd {
	return func() tea.Msg {
		if engine == nil || strings.TrimSpace(root) == "" {
			updates := make(chan templateLoadUpdateMsg, 1)
			updates <- templateLoadUpdateMsg{Done: true, Err: fmt.Errorf("ui.loadTemplates: loader unavailable")}
			close(updates)
			return templateLoadStartedMsg{updates: updates}
		}

		updates := make(chan templateLoadUpdateMsg, 16)
		go loadTemplateCategories(engine, root, updates)
		return templateLoadStartedMsg{updates: updates}
	}
}

func waitForTemplateLoad(sub <-chan templateLoadUpdateMsg) tea.Cmd {
	return func() tea.Msg {
		if sub == nil {
			return nil
		}
		msg, ok := <-sub
		if !ok {
			return templateLoadUpdateMsg{Done: true}
		}
		return msg
	}
}

func waitForGameTick(sub <-chan engine.GameTick) tea.Cmd {
	return func() tea.Msg {
		if sub == nil {
			return nil
		}
		tick, ok := <-sub
		if !ok {
			return nil
		}
		return GameTickMsg{Tick: tick}
	}
}

func expireNotificationCmd(id string, delay time.Duration) tea.Cmd {
	return func() tea.Msg {
		time.Sleep(delay)
		return notificationExpiredMsg{id: id}
	}
}

func eventMsgCmd(event events.GameEvent) tea.Cmd {
	return func() tea.Msg {
		return GameEventMsg{Event: event}
	}
}

func (m AppModel) handleTemplateLoad(msg templatesLoadedMsg) (tea.Model, tea.Cmd) {
	if msg.err == nil {
		return m, nil
	}
	note := NewNotification(NotificationDrama, "Plantillas: "+msg.err.Error())
	m.NotificationQueue = append(m.NotificationQueue, note)
	return m, expireNotificationCmd(note.ID, 3*time.Second)
}

func (m AppModel) handleTemplateLoadUpdate(msg templateLoadUpdateMsg) (tea.Model, tea.Cmd) {
	m.loading.UpdatedAt = time.Now()
	if msg.CategoryTotals != nil {
		m.loading.CategoryTotals = msg.CategoryTotals
	}
	if msg.CategoryLoaded != nil {
		m.loading.CategoryLoaded = msg.CategoryLoaded
	}
	m.loading.CurrentCategory = msg.Category
	m.loading.LoadedTemplates = msg.LoadedTemplates
	m.loading.TotalTemplates = msg.TotalTemplates
	m.loading.LoadedCategories = msg.LoadedCategories
	m.loading.TotalCategories = msg.TotalCategories

	if msg.Done {
		m.loading.Active = false
		m.loading.Err = msg.Err
		if msg.Err != nil {
			note := NewNotification(NotificationDrama, "Plantillas: "+msg.Err.Error())
			m.NotificationQueue = append(m.NotificationQueue, note)
			return m, expireNotificationCmd(note.ID, 3*time.Second)
		}
		return m, nil
	}

	return m, waitForTemplateLoad(m.templateLoadSub)
}

func (m AppModel) handleLoadingPulse(msg loadingPulseMsg) (tea.Model, tea.Cmd) {
	if !m.loading.Active {
		return m, nil
	}
	m.loading.TaglineFrame++
	m.loading.UpdatedAt = msg.at
	return m, loadingPulseCmd()
}

func (m AppModel) handleWindowSize(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	m.Width = msg.Width
	m.Height = msg.Height
	return m.updateAllSubModels(msg)
}

func (m AppModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if key.Matches(msg, m.KeyMap.Quit) {
		if m.cancel != nil {
			m.cancel()
		}
		if m.Engine != nil {
			m.Engine.Stop()
		}
		return m, tea.Quit
	}
	if m.CurrentView == VIEW_DEBUG && m.IsDebugMode {
		if updated, cmd, handled := m.handleDebugInput(msg); handled {
			return updated, cmd
		}
	}
	return m.handleGlobalNavigation(msg)
}

func (m AppModel) handleGlobalNavigation(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.currentViewBlocksGlobalNavigation() {
		return m.updateCurrentSubModel(msg)
	}
	switch {
	case key.Matches(msg, m.KeyMap.Mind):
		return m.switchToAllowedView(VIEW_MIND, "", "")
	case key.Matches(msg, m.KeyMap.World):
		return m.switchToAllowedView(VIEW_WORLD_EXPLORE, "", "")
	case key.Matches(msg, m.KeyMap.Pobles):
		return m.switchToAllowedView(VIEW_POBLES_LIST, "", "")
	case key.Matches(msg, m.KeyMap.Events):
		return m.switchToAllowedView(VIEW_EVENTS_FEED, "", "")
	case key.Matches(msg, m.KeyMap.Settlement):
		return m.switchToAllowedView(VIEW_SETTLEMENT, "", "")
	case key.Matches(msg, m.KeyMap.Pause):
		if !modeCanControlTime(m.Mode) {
			return m.modeBlockedNotice("pausar el tiempo")
		}
		return m.togglePause(), nil
	case key.Matches(msg, m.KeyMap.SpeedUp):
		if !modeCanControlTime(m.Mode) {
			return m.modeBlockedNotice("cambiar velocidad")
		}
		return m.changeSpeed(2.0), nil
	case key.Matches(msg, m.KeyMap.SpeedDown):
		if !modeCanControlTime(m.Mode) {
			return m.modeBlockedNotice("cambiar velocidad")
		}
		return m.changeSpeed(0.5), nil
	case key.Matches(msg, m.KeyMap.Back):
		return m.goBack(), nil
	case key.Matches(msg, m.KeyMap.Debug):
		if !m.modeAllowsView(VIEW_DEBUG) {
			return m.modeBlockedNotice("abrir debug")
		}
		return m.openDebug(), nil
	default:
		return m.updateCurrentSubModel(msg)
	}
}

func (m AppModel) handleDebugInput(msg tea.KeyMsg) (AppModel, tea.Cmd, bool) {
	switch msg.Type {
	case tea.KeyEsc:
		return m.goBack(), nil, true
	case tea.KeyBackspace, tea.KeyCtrlH:
		if len(m.ConsoleInput) > 0 {
			m.ConsoleInput = m.ConsoleInput[:len(m.ConsoleInput)-1]
		}
		return m, nil, true
	case tea.KeyEnter:
		return m.runDebugCommand(), nil, true
	}
	if msg.String() == "`" {
		return m.goBack(), nil, true
	}
	if len(msg.Runes) == 0 {
		return m, nil, false
	}
	m.ConsoleInput += string(msg.Runes)
	return m, nil, true
}

func (m AppModel) runDebugCommand() AppModel {
	command := strings.TrimSpace(strings.ToLower(m.ConsoleInput))
	m.ConsoleInput = ""
	return m.executeConsoleCommand(command)
}

func (m AppModel) handleGameTick(msg GameTickMsg) (tea.Model, tea.Cmd) {
	if m.Orchestrator != nil {
		return m.handleOrchestratedGameTick(msg)
	}
	m.syncWorldClock(msg.Tick)
	m.lastIntents = m.processDecisions(msg.Tick.DeltaHours)
	processed := m.processEvents(msg.Tick.CurrentTime)
	m.detectEnding()

	cmds := make([]tea.Cmd, 0, len(processed)+1)
	for _, event := range processed {
		cmds = append(cmds, eventMsgCmd(event))
	}
	cmds = append(cmds, waitForGameTick(m.tickSub))
	if m.ActiveEnding != nil && m.CurrentView != VIEW_ENDINGS {
		m = m.changeView(VIEW_ENDINGS, "", "")
	}
	return m, tea.Batch(cmds...)
}

func (m AppModel) handleOrchestratedGameTick(msg GameTickMsg) (tea.Model, tea.Cmd) {
	processed := m.Orchestrator.OnTick(msg.Tick)
	snapshot := m.Orchestrator.GetWorldSnapshot()
	m.World = m.Orchestrator.World()
	m.Engine = m.Orchestrator.TimeEngine()
	m.templateEngine = m.Orchestrator.TemplateEngine()
	m.EventFeed = snapshot.EventFeed
	m.ActiveEnding = snapshot.ActiveEnding
	m.speed = snapshot.Speed
	m.isPaused = snapshot.IsPaused
	m.lastIntents, m.intentReasons = intentMapsFromSnapshot(snapshot)

	cmds := make([]tea.Cmd, 0, len(processed)+1)
	for _, event := range processed {
		if note, ok := NotificationFromEvent(event); ok {
			m.NotificationQueue = append(m.NotificationQueue, note)
			cmds = append(cmds, expireNotificationCmd(note.ID, 3*time.Second))
		}
	}
	cmds = append(cmds, waitForGameTick(m.tickSub))
	if m.ActiveEnding != nil && m.CurrentView != VIEW_ENDINGS {
		m = m.changeView(VIEW_ENDINGS, "", "")
	}
	return m, tea.Batch(cmds...)
}

func (m AppModel) handleGameEvent(msg GameEventMsg) (tea.Model, tea.Cmd) {
	m.EventFeed = append([]events.GameEvent{msg.Event}, m.EventFeed...)
	if len(m.EventFeed) > 50 {
		m.EventFeed = m.EventFeed[:50]
	}
	note, ok := NotificationFromEvent(msg.Event)
	if !ok {
		return m, nil
	}
	m.NotificationQueue = append(m.NotificationQueue, note)
	return m, expireNotificationCmd(note.ID, 3*time.Second)
}

func (m AppModel) updateCurrentSubModel(msg tea.Msg) (tea.Model, tea.Cmd) {
	sub, ok := m.SubModels[m.CurrentView]
	if !ok || sub == nil {
		return m, nil
	}
	sub = m.syncSubModelState(sub)
	updated, cmd := sub.Update(msg)
	updated = m.syncSubModelState(updated)
	m.SubModels = copySubModels(m.SubModels)
	m.SubModels[m.CurrentView] = updated
	return m, cmd
}

func (m AppModel) updateAllSubModels(msg tea.Msg) (tea.Model, tea.Cmd) {
	subModels := copySubModels(m.SubModels)
	cmds := make([]tea.Cmd, 0, len(subModels))
	for view, sub := range subModels {
		if sub == nil {
			continue
		}
		sub = m.syncSubModelState(sub)
		updated, cmd := sub.Update(msg)
		updated = m.syncSubModelState(updated)
		subModels[view] = updated
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	m.SubModels = subModels
	return m, tea.Batch(cmds...)
}

func (m AppModel) changeView(view ViewType, selectedPobleID, selectedBuildingID string) AppModel {
	if view == m.CurrentView {
		if selectedPobleID != "" {
			m.SelectedPobleID = selectedPobleID
		}
		if selectedBuildingID != "" {
			m.SelectedBuildingID = selectedBuildingID
		}
		return m
	}
	m.PreviousView = m.CurrentView
	m.CurrentView = view
	if selectedPobleID != "" {
		m.SelectedPobleID = selectedPobleID
	}
	if selectedBuildingID != "" {
		m.SelectedBuildingID = selectedBuildingID
	}
	return m
}

func (m AppModel) switchToView(view ViewType, selectedPobleID, selectedBuildingID string) (tea.Model, tea.Cmd) {
	m = m.changeView(view, selectedPobleID, selectedBuildingID)
	return m.activateCurrentView()
}

func (m AppModel) switchToAllowedView(view ViewType, selectedPobleID, selectedBuildingID string) (tea.Model, tea.Cmd) {
	if !m.modeAllowsView(view) {
		return m.modeBlockedNotice(view.String())
	}
	return m.switchToView(view, selectedPobleID, selectedBuildingID)
}

func (m AppModel) modeBlockedNotice(target string) (tea.Model, tea.Cmd) {
	note := NewNotification(NotificationDrama, fmt.Sprintf("Modo %s no permite %s.", m.effectiveMode(), target))
	m.NotificationQueue = append(m.NotificationQueue, note)
	return m, expireNotificationCmd(note.ID, 3*time.Second)
}

func (m AppModel) activateCurrentView() (tea.Model, tea.Cmd) {
	sub, ok := m.SubModels[m.CurrentView]
	if !ok || sub == nil {
		return m, nil
	}

	sub = m.syncSubModelState(sub)
	if aware, ok := sub.(viewEnterAwareModel); ok {
		updated, cmd := aware.OnEnter()
		updated = m.syncSubModelState(updated)
		m.SubModels = copySubModels(m.SubModels)
		m.SubModels[m.CurrentView] = updated
		return m, cmd
	}

	m.SubModels = copySubModels(m.SubModels)
	m.SubModels[m.CurrentView] = sub
	return m, nil
}

func (m AppModel) goBack() AppModel {
	if m.CurrentView == VIEW_DEBUG {
		m.IsDebugMode = false
	}
	if m.PreviousView == m.CurrentView {
		m.CurrentView = VIEW_MAIN_MAP
		return m
	}
	m.CurrentView, m.PreviousView = m.PreviousView, m.CurrentView
	return m
}

func (m AppModel) openDebug() AppModel {
	m.IsDebugMode = true
	m.Mode = config.GameModeGod
	return m.changeView(VIEW_DEBUG, "", "")
}

func (m AppModel) togglePause() AppModel {
	if m.Engine == nil {
		return m
	}
	if m.isPaused {
		m.Engine.Resume()
		m.isPaused = false
		return m
	}
	m.Engine.Pause()
	m.isPaused = true
	return m
}

func (m AppModel) changeSpeed(multiplier float64) AppModel {
	if multiplier <= 0 {
		return m
	}
	next := m.speed * multiplier
	if next < 0.25 {
		next = 0.25
	}
	if next > 8 {
		next = 8
	}
	m.speed = next
	if m.Engine != nil {
		m.Engine.SetSpeed(next)
	}
	m.isPaused = false
	return m
}

func (m AppModel) syncWorldClock(tick engine.GameTick) {
	if m.World != nil {
		m.World.Calendar = tick.CurrentTime
	}
}

func (m AppModel) processDecisions(deltaHours int) map[string]string {
	intents := map[string]string{}
	if m.World == nil {
		return intents
	}
	for _, poble := range m.World.GetAllPobles() {
		if poble == nil {
			continue
		}
		decision := ai.NewDecisionEngine(poble, m.World, m.rng)
		decision.Decide(deltaHours)
		intents[poble.ID] = decision.GetCurrentIntent()
	}
	return intents
}

func (m AppModel) processEvents(currentTime engine.GameTime) []events.GameEvent {
	if m.eventQueue == nil || m.World == nil {
		return nil
	}
	return events.ProcessTick(m.eventQueue, currentTime, m.World, m.rng, nil)
}

func (m *AppModel) detectEnding() {
	if m == nil || m.World == nil || m.ActiveEnding != nil {
		return
	}
	m.ActiveEnding = engine.CheckEndingConditions(m.World)
	if m.ActiveEnding != nil {
		_ = config.UnlockEnding(string(m.ActiveEnding.Type))
	}
}

func (m AppModel) restartCivilization() (tea.Model, tea.Cmd) {
	return m.beginNewCivilization()
}

func (m AppModel) expireNotification(id string) AppModel {
	filtered := m.NotificationQueue[:0]
	for _, note := range m.NotificationQueue {
		if note.ID != id {
			filtered = append(filtered, note)
		}
	}
	m.NotificationQueue = filtered
	return m
}

func (m AppModel) renderStatusBar(width int) string {
	current := m.currentWorldState()
	title := statusTitleStyle.Render("POBLATION " + strings.ToUpper(string(m.effectiveMode())))
	rest := fmt.Sprintf(" · Día %d · %02d:%02d · Población: %d · %s",
		current.Day.Day,
		current.Day.Hour,
		current.Day.Minute,
		current.Population,
		current.Era.String(),
	)
	line := title + rest
	return statusBarStyle.Width(width).Render(truncateRunes(line, width-2))
}

func (m AppModel) renderBody(width int) string {
	bodyHeight := m.Height - chromeHeight(m)
	if bodyHeight < 1 {
		bodyHeight = 1
	}
	if m.loading.Active {
		return m.loading.render(maxInt(40, width-2), maxInt(12, bodyHeight))
	}
	if width < 56 || m.Height < 12 {
		return m.renderSmallBody(width, bodyHeight)
	}
	return m.renderLargeBody(width, bodyHeight)
}

func (m AppModel) renderLargeBody(width, height int) string {
	body := m.currentSubView()
	if m.CurrentView == VIEW_DEBUG {
		body = m.debugView()
	}
	if m.CurrentView == VIEW_MAIN_MAP {
		return lipgloss.NewStyle().
			Width(maxInt(1, width-2)).
			Height(maxInt(1, height)).
			Render(body)
	}
	return viewFrameStyle.
		Width(maxInt(0, width-2)).
		Height(maxInt(1, height)).
		Render(body)
}

func (m AppModel) renderSmallBody(width, height int) string {
	if m.CurrentView == VIEW_MAIN_MAP {
		return lipgloss.NewStyle().
			Width(maxInt(1, width-2)).
			Height(maxInt(1, height)).
			Render(m.currentSubView())
	}
	label := fmt.Sprintf("%s · %d eventos", m.CurrentView.String(), len(m.EventFeed))
	return viewFrameStyle.
		Width(maxInt(0, width-2)).
		Height(maxInt(1, height)).
		Render(truncateRunes(label, width-6))
}

func (m AppModel) currentSubView() string {
	sub := m.SubModels[m.CurrentView]
	if sub == nil {
		return viewTitleStyle.Render(m.CurrentView.String())
	}
	sub = m.syncSubModelState(sub)
	m.SubModels = copySubModels(m.SubModels)
	m.SubModels[m.CurrentView] = sub
	return sub.View()
}

func (m AppModel) debugView() string {
	lines := []string{
		viewTitleStyle.Render("DEBUG"),
		fmt.Sprintf("Vista: %s", m.CurrentView.String()),
		fmt.Sprintf("Eventos en feed: %d", len(m.EventFeed)),
		fmt.Sprintf("Velocidad: %.2fx", m.speed),
		fmt.Sprintf("Pausado: %t", m.isPaused),
		consoleStyle.Render("> " + m.ConsoleInput),
	}
	if len(m.lastIntents) > 0 {
		lines = append(lines, mutedTextStyle.Render("Intenciones recientes:"))
		for id, intent := range m.lastIntents {
			lines = append(lines, fmt.Sprintf("%s: %s", id, intent))
		}
	}
	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func (m AppModel) renderNavigationBar(width int) string {
	if m.CurrentView == VIEW_MENU {
		text := "ENTER elegir  ESC volver  Q salir"
		return navBarStyle.Width(width).Render(truncateRunes(text, width-2))
	}
	if m.CurrentView == VIEW_SETTINGS {
		text := "TAB categoria  LEFT/RIGHT cambia  UP/DOWN mover  ESC volver"
		return navBarStyle.Width(width).Render(truncateRunes(text, width-2))
	}
	if m.CurrentView == VIEW_CREATE_POBLE {
		text := "ENTER seguir  ESC cancelar"
		return navBarStyle.Width(width).Render(truncateRunes(text, width-2))
	}
	text := navigationHelpForMode(m.effectiveMode())
	if width < 80 {
		text = compactNavigationHelpForMode(m.effectiveMode())
	}
	return navBarStyle.Width(width).Render(truncateRunes(text, width-2))
}

func navigationHelpForMode(mode config.GameMode) string {
	switch mode {
	case config.GameModeObserver:
		return "Observer  W mundo  P pobles  E eventos  S settlement  ESC atras  modo: `mode director`"
	case config.GameModeJournalist:
		return "Periodista  E eventos  S settlement  W mundo  periodico por consola  ESC atras"
	case config.GameModeKnown:
		return "Conocidos  M mente  P pobles  E eventos  ENTER seleccionar  ESC atras"
	case config.GameModeGod:
		return "Dios  M mente  W mundo  P pobles  E eventos  S settlement  SPACE pausa  +/- velocidad  ` debug"
	default:
		return "Director  M mente  W mundo  P pobles  E eventos  S settlement  SPACE pausa  +/- velocidad  ESC atras"
	}
}

func compactNavigationHelpForMode(mode config.GameMode) string {
	switch mode {
	case config.GameModeJournalist:
		return "Periodista  E eventos  S settlement  ESC atras"
	case config.GameModeKnown:
		return "Conocidos  M mente  P pobles  ESC atras"
	case config.GameModeObserver:
		return "Observer  W mundo  E eventos  ESC atras"
	case config.GameModeGod:
		return "Dios  M/W/P/E  SPACE pausa  ` debug"
	default:
		return "Director  M/W/P/E/S  SPACE pausa  ESC"
	}
}

func (m AppModel) renderNotificationOverlay(width int) string {
	if len(m.NotificationQueue) == 0 {
		return ""
	}
	latest := m.NotificationQueue[len(m.NotificationQueue)-1]
	text := strings.TrimSpace(latest.Message)
	if text == "" {
		text = string(latest.Type)
	}
	if hidden := len(m.NotificationQueue) - 1; hidden > 0 {
		text = fmt.Sprintf("%s  +%d mas", text, hidden)
	}
	text = truncateRunes("ALERTA  "+text, maxInt(12, width-2))
	return notificationStyle(latest.Type).Width(width).Render(text)
}

func (m AppModel) currentWorldState() world.WorldState {
	if m.World != nil {
		return m.World.GetWorldState()
	}
	if m.Engine != nil {
		state := world.WorldState{}
		state.Day = m.Engine.GetCurrentTime()
		return state
	}
	return world.WorldState{}
}

func (m AppModel) handleConsoleCommand(msg uiviews.ConsoleCommandMsg) (tea.Model, tea.Cmd) {
	before := len(m.NotificationQueue)
	m = m.executeConsoleCommand(msg.Command)
	if len(m.NotificationQueue) > before {
		note := m.NotificationQueue[len(m.NotificationQueue)-1]
		return m, expireNotificationCmd(note.ID, 3*time.Second)
	}
	return m, nil
}

func (m AppModel) executeConsoleCommand(raw string) AppModel {
	command := strings.TrimSpace(raw)
	if command == "" {
		return m
	}

	if m.Console == nil {
		m.Console = engine.NewConsoleSystem(m.World, m.Engine, m.rng)
	}
	result := m.Console.Execute(command)
	if result.ModeHint != "" {
		m = m.setMode(result.ModeHint)
	}
	if result.ClearFeed {
		m.EventFeed = nil
		m.NotificationQueue = nil
	}
	if result.Event != nil {
		m.EventFeed = append([]events.GameEvent{*result.Event}, m.EventFeed...)
		if len(m.EventFeed) > 50 {
			m.EventFeed = m.EventFeed[:50]
		}
	}
	switch result.ViewHint {
	case engine.ConsoleViewNewspaper:
		m = m.changeView(VIEW_NEWSPAPER, "", "")
	case engine.ConsoleViewEndings:
		if m.ActiveEnding == nil {
			m.ActiveEnding = engine.CheckEndingConditions(m.World)
		}
		m = m.changeView(VIEW_ENDINGS, "", "")
	}
	if m.Engine != nil {
		m.speed = m.Engine.Speed
		m.isPaused = m.Engine.IsPaused
	}
	m.NotificationQueue = append(m.NotificationQueue, NewNotification(NotificationDrama, result.Feedback))
	return m
}

// NewNotification creates a three-second notification.
func NewNotification(kind NotificationType, message string) Notification {
	now := time.Now()
	return Notification{
		ID:        fmt.Sprintf("%d", now.UnixNano()),
		Type:      kind,
		Message:   message,
		CreatedAt: now,
		ExpiresAt: now.Add(3 * time.Second),
	}
}

// NotificationFromEvent converts important game events into UI notifications.
func NotificationFromEvent(event events.GameEvent) (Notification, bool) {
	kind, ok := notificationKindForEvent(event.Type)
	if !ok {
		return Notification{}, false
	}
	message := strings.TrimSpace(event.Description)
	if strings.TrimSpace(message) == "" {
		message = notificationEventLabel(event.Type)
	}
	return NewNotification(kind, message), true
}

func notificationEventLabel(eventType events.EventType) string {
	switch eventType {
	case events.EventFightVerbal:
		return "Discusion fuerte"
	case events.EventFightPhysical:
		return "Pelea fisica"
	case events.EventBetrayalRevealed:
		return "Traicion revelada"
	case events.EventDivorce:
		return "Ruptura publica"
	case events.EventCoup:
		return "Golpe de poder"
	case events.EventRevolution:
		return "Revolucion"
	case events.EventEraChange:
		return "Cambio de era"
	default:
		return strings.ToLower(strings.ReplaceAll(string(eventType), "_", " "))
	}
}

func notificationKindForEvent(eventType events.EventType) (NotificationType, bool) {
	switch eventType {
	case events.EventDeathNatural, events.EventDeathAccident, events.EventDeathMurder, events.EventSuicide:
		return NotificationDeath, true
	case events.EventBirth, events.EventAdoption, events.EventPregnancy:
		return NotificationBirth, true
	case events.EventFightVerbal, events.EventFightPhysical, events.EventBetrayalRevealed,
		events.EventDivorce, events.EventCoup, events.EventRevolution, events.EventEraChange:
		return NotificationDrama, true
	default:
		return NotificationInfo, false
	}
}

func defaultSubModels() map[ViewType]tea.Model {
	models := map[ViewType]tea.Model{}
	for _, view := range allViewTypes() {
		switch view {
		case VIEW_MENU:
			models[view] = uiviews.NewMenuModel()
		case VIEW_MAIN_MAP:
			models[view] = uiviews.NewMapModel()
		case VIEW_MIND:
			models[view] = uiviews.NewMindModel()
		case VIEW_DIALOGUE:
			models[view] = uiviews.NewDialogueModel()
		case VIEW_WORLD_EXPLORE:
			models[view] = uiviews.NewExploreModel()
		case VIEW_HOUSE_EXPLORE:
			models[view] = uiviews.NewHouseModel()
		case VIEW_POBLES_LIST:
			models[view] = uiviews.NewPoblesModel()
		case VIEW_EVENTS_FEED:
			models[view] = uiviews.NewEventsModel()
		case VIEW_SETTLEMENT:
			models[view] = uiviews.NewSettlementModel()
		case VIEW_MINIGAME_SEX:
			models[view] = uiviews.NewEncounterModel()
		case VIEW_MINIGAME_FIGHT:
			models[view] = uiviews.NewFightModel()
		case VIEW_NEWSPAPER:
			models[view] = uiviews.NewNewspaperModel()
		case VIEW_ENDINGS:
			models[view] = uiviews.NewEndingModel()
		case VIEW_CREATE_POBLE:
			models[view] = uiviews.NewCreatePobleModel()
		case VIEW_SETTINGS:
			models[view] = uiviews.NewSettingsModel()
		default:
			models[view] = newStaticViewModel(view)
		}
	}
	return models
}

func allViewTypes() []ViewType {
	return []ViewType{
		VIEW_MENU, VIEW_MAIN_MAP, VIEW_MIND, VIEW_DIALOGUE, VIEW_WORLD_EXPLORE,
		VIEW_HOUSE_EXPLORE, VIEW_POBLES_LIST, VIEW_POBLE_DETAIL,
		VIEW_EVENTS_FEED, VIEW_SETTLEMENT, VIEW_MINIGAME_SEX,
		VIEW_MINIGAME_FIGHT, VIEW_NEWSPAPER, VIEW_ENDINGS,
		VIEW_CREATE_POBLE, VIEW_SETTINGS, VIEW_DEBUG,
	}
}

func copySubModels(source map[ViewType]tea.Model) map[ViewType]tea.Model {
	copied := make(map[ViewType]tea.Model, len(source))
	for view, model := range source {
		copied[view] = model
	}
	return copied
}

func (m AppModel) syncSubModelState(model tea.Model) tea.Model {
	aware, ok := model.(appStateAwareModel)
	if !ok {
		if resizable, ok := model.(resizableModel); ok {
			snapshot := m.snapshotForViews()
			return resizable.Resize(snapshot.Width, snapshot.Height)
		}
		return model
	}
	snapshot := m.snapshotForViews()
	model = aware.SyncAppState(snapshot)
	if resizable, ok := model.(resizableModel); ok {
		model = resizable.Resize(snapshot.Width, snapshot.Height)
	}
	return model
}

func (m AppModel) currentViewBlocksGlobalNavigation() bool {
	sub := m.SubModels[m.CurrentView]
	blocker, ok := sub.(globalNavigationBlockingModel)
	return ok && blocker.BlocksGlobalNavigation()
}

func (m AppModel) effectiveMode() config.GameMode {
	if m.Mode == "" {
		return config.GameModeDirector
	}
	return m.Mode
}

func (m AppModel) setMode(mode config.GameMode) AppModel {
	if _, ok := config.ParseGameMode(string(mode)); !ok {
		return m
	}
	m.Mode = mode
	m.IsDebugMode = mode == config.GameModeGod
	if m.Console != nil {
		m.Console.CurrentMode = mode
		m.Console.GodMode = mode == config.GameModeGod
		m.Console.NewspaperMode = mode == config.GameModeJournalist
	}
	_, _ = config.UpdateSettings(func(settings *config.Settings) {
		settings.Gameplay.ActiveMode = mode
	})
	if !m.modeAllowsView(m.CurrentView) {
		m = m.changeView(defaultViewForMode(mode), "", "")
	}
	return m
}

func (m AppModel) modeAllowsView(view ViewType) bool {
	if view == VIEW_MENU || view == VIEW_SETTINGS {
		return true
	}
	if view == VIEW_ENDINGS && m.ActiveEnding != nil {
		return true
	}
	switch m.effectiveMode() {
	case config.GameModeGod:
		return true
	case config.GameModeDirector:
		return view != VIEW_DEBUG
	case config.GameModeObserver:
		return view == VIEW_MAIN_MAP || view == VIEW_WORLD_EXPLORE || view == VIEW_POBLES_LIST ||
			view == VIEW_POBLE_DETAIL || view == VIEW_EVENTS_FEED || view == VIEW_SETTLEMENT ||
			view == VIEW_NEWSPAPER || view == VIEW_ENDINGS
	case config.GameModeJournalist:
		return view == VIEW_NEWSPAPER || view == VIEW_EVENTS_FEED || view == VIEW_SETTLEMENT ||
			view == VIEW_WORLD_EXPLORE || view == VIEW_MAIN_MAP || view == VIEW_ENDINGS
	case config.GameModeKnown:
		return view == VIEW_MAIN_MAP || view == VIEW_MIND || view == VIEW_DIALOGUE ||
			view == VIEW_HOUSE_EXPLORE || view == VIEW_POBLES_LIST || view == VIEW_POBLE_DETAIL ||
			view == VIEW_EVENTS_FEED
	default:
		return true
	}
}

func modeCanControlTime(mode config.GameMode) bool {
	return mode == "" || mode == config.GameModeDirector || mode == config.GameModeGod
}

func defaultViewForMode(mode config.GameMode) ViewType {
	switch mode {
	case config.GameModeJournalist:
		return VIEW_NEWSPAPER
	case config.GameModeKnown:
		return VIEW_POBLES_LIST
	default:
		return VIEW_MAIN_MAP
	}
}

func (m AppModel) snapshotForViews() uiviews.AppStateSnapshot {
	return uiviews.AppStateSnapshot{
		World:              m.World,
		EventFeed:          append([]events.GameEvent(nil), m.EventFeed...),
		SelectedPobleID:    m.SelectedPobleID,
		SelectedBuildingID: m.SelectedBuildingID,
		Width:              maxInt(1, m.Width-2),
		Height:             maxInt(1, m.Height-chromeHeight(m)),
		Speed:              m.speed,
		IsPaused:           m.isPaused,
		TemplateEngine:     m.templateEngine,
		IsDirectorMode:     m.IsDebugMode,
		Ending:             m.ActiveEnding,
		LastIntents:        copyStringMap(m.lastIntents),
		IntentReasons:      copyStringMap(m.intentReasons),
	}
}

func intentMapsFromSnapshot(snapshot engine.UISnapshot) (map[string]string, map[string]string) {
	intents := make(map[string]string, len(snapshot.Pobles))
	reasons := make(map[string]string, len(snapshot.Pobles))
	for _, poble := range snapshot.Pobles {
		if strings.TrimSpace(poble.ID) == "" {
			continue
		}
		intents[poble.ID] = poble.CurrentIntent
		reasons[poble.ID] = poble.IntentReason
	}
	return intents, reasons
}

func copyStringMap(input map[string]string) map[string]string {
	if len(input) == 0 {
		return map[string]string{}
	}
	output := make(map[string]string, len(input))
	for key, value := range input {
		output[key] = value
	}
	return output
}

func templateRoot() string {
	if root := strings.TrimSpace(os.Getenv("POBLATION_TEMPLATES_DIR")); root != "" {
		if _, err := os.Stat(root); err == nil {
			return root
		}
	}
	if _, err := os.Stat("templates"); err == nil {
		return "templates"
	}
	wd, err := os.Getwd()
	if err != nil {
		return "templates"
	}
	return filepath.Join(wd, "templates")
}

func chromeHeight(m AppModel) int {
	height := 2
	if len(m.NotificationQueue) > 0 {
		height++
	}
	return height
}

func safeWidth(width int) int {
	if width <= 0 {
		return 80
	}
	return width
}

func truncateRunes(text string, limit int) string {
	if limit <= 0 {
		return ""
	}
	runes := []rune(text)
	if len(runes) <= limit {
		return text
	}
	if limit <= 1 {
		return string(runes[:limit])
	}
	return string(runes[:limit-1]) + "."
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// String returns a compact display label for each view.
func (v ViewType) String() string {
	switch v {
	case VIEW_MENU:
		return "Menu"
	case VIEW_MAIN_MAP:
		return "Mapa"
	case VIEW_MIND:
		return "Mente"
	case VIEW_DIALOGUE:
		return "Dialogo"
	case VIEW_WORLD_EXPLORE:
		return "Mundo"
	case VIEW_HOUSE_EXPLORE:
		return "Casa"
	case VIEW_POBLES_LIST:
		return "Pobles"
	case VIEW_POBLE_DETAIL:
		return "Poble"
	case VIEW_EVENTS_FEED:
		return "Eventos"
	case VIEW_SETTLEMENT:
		return "Settlement"
	case VIEW_MINIGAME_SEX:
		return "Minijuego sexo"
	case VIEW_MINIGAME_FIGHT:
		return "Minijuego pelea"
	case VIEW_NEWSPAPER:
		return "Periodico"
	case VIEW_ENDINGS:
		return "Finales"
	case VIEW_CREATE_POBLE:
		return "Crear Poble"
	case VIEW_SETTINGS:
		return "Ajustes"
	case VIEW_DEBUG:
		return "Debug"
	default:
		return "Vista"
	}
}

func (m AppModel) beginNewCivilization() (tea.Model, tea.Cmd) {
	m = m.resetRuntime(time.Now().UnixNano())
	m = m.changeView(VIEW_CREATE_POBLE, "", "")
	model, viewCmd := m.activateCurrentView()
	if updated, ok := model.(AppModel); ok {
		m = updated
	}
	return m, tea.Batch(
		viewCmd,
		initializeRuntimeCmd(m.Engine, m.ctx),
		startTemplateLoadingCmd(m.templateEngine, templateRoot()),
		loadingPulseCmd(),
	)
}

func (m AppModel) loadSaveData(data *save.SaveData) (tea.Model, tea.Cmd) {
	if data == nil || data.WorldState == nil {
		note := NewNotification(NotificationDrama, "No habia nada valido que cargar.")
		m.NotificationQueue = append(m.NotificationQueue, note)
		return m, expireNotificationCmd(note.ID, 3*time.Second)
	}

	m = m.resetRuntime(data.Seed)
	m.World = data.WorldState
	m.World.Calendar = world.GameTime(data.CurrentTime)
	m.Engine.SetTime(data.CurrentTime)
	m.EventFeed = append([]events.GameEvent(nil), data.EventHistory...)
	m.Console = engine.NewConsoleSystem(m.World, m.Engine, m.rng)
	m = m.changeView(VIEW_MAIN_MAP, "", "")
	model, viewCmd := m.activateCurrentView()
	if updated, ok := model.(AppModel); ok {
		m = updated
	}
	return m, tea.Batch(
		viewCmd,
		initializeRuntimeCmd(m.Engine, m.ctx),
		startTemplateLoadingCmd(m.templateEngine, templateRoot()),
		loadingPulseCmd(),
	)
}

func (m AppModel) finishPobleCreation(msg uiviews.CreatePobleCompleteMsg) (tea.Model, tea.Cmd) {
	if msg.Poble == nil {
		note := NewNotification(NotificationDrama, "La creacion no devolvio ningun personaje.")
		m.NotificationQueue = append(m.NotificationQueue, note)
		return m, expireNotificationCmd(note.ID, 3*time.Second)
	}
	if m.World == nil {
		m = m.resetRuntime(time.Now().UnixNano())
	}

	existing := append([]*entities.Poble(nil), m.World.GetAllPobles()...)
	origin := foundingSpawnLocation(len(existing))
	if !m.World.AddPoble(msg.Poble, origin) {
		note := NewNotification(NotificationDrama, "No pude colocar ese Poble en la isla.")
		m.NotificationQueue = append(m.NotificationQueue, note)
		return m, expireNotificationCmd(note.ID, 3*time.Second)
	}
	created := m.World.GetPoble(msg.Poble.ID)
	if created == nil {
		created = msg.Poble
	}
	for _, other := range existing {
		linkFounders(created, other)
	}

	m.SelectedPobleID = created.ID
	m.World.Calendar = m.Engine.GetCurrentTime()

	population := m.World.GetPopulation()
	m = m.changeView(VIEW_CREATE_POBLE, created.ID, "")
	model, viewCmd := m.activateCurrentView()
	if updated, ok := model.(AppModel); ok {
		m = updated
	}

	if population < 2 {
		note := NewNotification(NotificationBirth, fmt.Sprintf("%s ya existe. Falta crear al menos 1 Poble mas.", created.Name))
		m.NotificationQueue = append(m.NotificationQueue, note)
		return m, tea.Batch(viewCmd, expireNotificationCmd(note.ID, 3*time.Second))
	}

	note := NewNotification(NotificationBirth, fmt.Sprintf("%s ya existe. Tienes %d Pobles iniciales.", created.Name, population))
	m.NotificationQueue = append(m.NotificationQueue, note)
	return m, tea.Batch(
		viewCmd,
		func() tea.Msg { return uiviews.CreatePobleAskMoreMsg{Count: population} },
		expireNotificationCmd(note.ID, 3*time.Second),
	)
}

func (m AppModel) openNextFounderCreation() (tea.Model, tea.Cmd) {
	m = m.changeView(VIEW_CREATE_POBLE, "", "")
	model, viewCmd := m.activateCurrentView()
	if updated, ok := model.(AppModel); ok {
		m = updated
	}
	return m, viewCmd
}

func (m AppModel) startCreatedCivilization() (tea.Model, tea.Cmd) {
	if m.World == nil || m.World.GetPopulation() < 2 {
		count := 0
		if m.World != nil {
			count = m.World.GetPopulation()
		}
		note := NewNotification(NotificationDrama, fmt.Sprintf("Necesitas 2 Pobles iniciales. Ahora tienes %d.", count))
		m.NotificationQueue = append(m.NotificationQueue, note)
		return m.openNextFounderCreation()
	}
	if m.SelectedPobleID == "" {
		if pobles := m.World.GetAllPobles(); len(pobles) > 0 && pobles[0] != nil {
			m.SelectedPobleID = pobles[0].ID
		}
	}
	m = m.changeView(VIEW_MAIN_MAP, m.SelectedPobleID, "")
	model, viewCmd := m.activateCurrentView()
	if updated, ok := model.(AppModel); ok {
		m = updated
	}
	note := NewNotification(NotificationInfo, fmt.Sprintf("La isla empieza con %d Pobles iniciales.", m.World.GetPopulation()))
	m.NotificationQueue = append(m.NotificationQueue, note)
	return m, tea.Batch(viewCmd, expireNotificationCmd(note.ID, 3*time.Second))
}

func foundingSpawnLocation(index int) world.Location {
	positions := []world.Location{
		{IslandID: "island_0", X: 12, Y: 10},
		{IslandID: "island_0", X: 18, Y: 10},
		{IslandID: "island_0", X: 14, Y: 13},
		{IslandID: "island_0", X: 20, Y: 13},
		{IslandID: "island_0", X: 16, Y: 16},
		{IslandID: "island_0", X: 22, Y: 16},
	}
	if index >= 0 && index < len(positions) {
		return positions[index]
	}
	return world.Location{IslandID: "island_0", X: 12 + (index%8)*2, Y: 10 + (index/8)*2}
}

func (m AppModel) resetRuntime(seed int64) AppModel {
	if m.cancel != nil {
		m.cancel()
	}
	if m.Engine != nil {
		m.Engine.Stop()
	}
	if seed == 0 {
		seed = time.Now().UnixNano()
	}

	ctx, cancel := context.WithCancel(context.Background())
	rng := rand.New(rand.NewSource(seed))
	m.World = world.NewWorld(seed)
	m.Engine = engine.NewTimeEngine()
	m.CurrentView = VIEW_MENU
	m.PreviousView = VIEW_MENU
	m.SelectedPobleID = ""
	m.SelectedBuildingID = ""
	m.EventFeed = []events.GameEvent{}
	m.NotificationQueue = []Notification{}
	m.ConsoleInput = ""
	m.SubModels = defaultSubModels()
	m.ctx = ctx
	m.cancel = cancel
	m.eventQueue = events.NewEventQueue(rng)
	m.templateEngine = templates.NewTemplateEngine(rng)
	m.rng = rng
	m.tickSub = nil
	m.lastIntents = map[string]string{}
	m.speed = 1.0
	m.isPaused = false
	m.Console = engine.NewConsoleSystem(m.World, m.Engine, rng)
	m.ActiveEnding = nil
	m.templateLoadSub = nil
	m.loading = newTemplateLoadingState()

	profile := config.LoadOrDefault()
	m.Engine.SetSpeed(profile.Settings.Gameplay.DefaultSimulationSpeed)
	m.speed = m.Engine.Speed
	return m
}

func spawnFoundingCompanion(founder *entities.Poble) *entities.Poble {
	if founder == nil {
		return nil
	}

	config := entities.PoblConfig{
		AgeRange: [2]int{
			maxInt(18, founder.Age-6),
			maxInt(18, founder.Age+4),
		},
	}

	switch founder.Sex {
	case entities.Male:
		sex := entities.Female
		config.Sex = &sex
	case entities.Female:
		sex := entities.Male
		config.Sex = &sex
	}

	companion, err := entities.GeneratePople(config, rand.New(rand.NewSource(time.Now().UnixNano()+77)))
	if err != nil {
		return nil
	}
	return companion
}

func linkFounders(a, b *entities.Poble) {
	if a == nil || b == nil {
		return
	}

	ab := entities.NewRelationship(b.ID, entities.RelationshipComplicated)
	ab.Familiarity = 68
	ab.Affection = 42
	ab.Trust = 48
	ab.Attraction = 37
	ab.Tags = []string{"founder", "day_zero"}

	ba := entities.NewRelationship(a.ID, entities.RelationshipComplicated)
	ba.Familiarity = 68
	ba.Affection = 39
	ba.Trust = 45
	ba.Attraction = 34
	ba.Tags = []string{"founder", "day_zero"}

	if a.Relationships == nil {
		a.Relationships = map[string]entities.Relationship{}
	}
	if b.Relationships == nil {
		b.Relationships = map[string]entities.Relationship{}
	}
	a.Relationships[b.ID] = ab
	b.Relationships[a.ID] = ba
}

type staticViewModel struct {
	view   ViewType
	width  int
	height int
}

func newStaticViewModel(view ViewType) staticViewModel {
	return staticViewModel{view: view, width: 80, height: 24}
}

func (m staticViewModel) Init() tea.Cmd {
	return nil
}

func (m staticViewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch typed := msg.(type) {
	case tea.WindowSizeMsg:
		return m.Resize(typed.Width, typed.Height), nil
	}
	return m, nil
}

func (m staticViewModel) Resize(width, height int) tea.Model {
	m.width = width
	m.height = height
	return m
}

func (m staticViewModel) View() string {
	lines := []string{
		viewTitleStyle.Render(m.view.String()),
		mutedTextStyle.Render("Vista lista para conectar su modelo propio."),
	}
	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func loadingPulseCmd() tea.Cmd {
	return tea.Tick(140*time.Millisecond, func(at time.Time) tea.Msg {
		return loadingPulseMsg{at: at}
	})
}

func loadTemplateCategories(engine *templates.TemplateEngine, root string, updates chan<- templateLoadUpdateMsg) {
	defer close(updates)

	categories, totals, totalTemplates, err := discoverTemplateCategories(root)
	if err != nil {
		updates <- templateLoadUpdateMsg{Done: true, Err: err}
		return
	}

	loadedByCategory := map[string]int{}
	loadedTemplates := 0
	for index, category := range categories {
		if loadErr := engine.LoadTemplates(filepath.Join(root, category)); loadErr != nil {
			updates <- templateLoadUpdateMsg{
				Category:         category,
				LoadedTemplates:  loadedTemplates,
				TotalTemplates:   totalTemplates,
				LoadedCategories: index,
				TotalCategories:  len(categories),
				CategoryLoaded:   cloneIntMap(loadedByCategory),
				CategoryTotals:   cloneIntMap(totals),
				Done:             true,
				Err:              loadErr,
			}
			return
		}

		loadedByCategory[category] = totals[category]
		loadedTemplates += totals[category]
		updates <- templateLoadUpdateMsg{
			Category:         category,
			LoadedTemplates:  loadedTemplates,
			TotalTemplates:   totalTemplates,
			LoadedCategories: index + 1,
			TotalCategories:  len(categories),
			CategoryLoaded:   cloneIntMap(loadedByCategory),
			CategoryTotals:   cloneIntMap(totals),
		}
	}

	updates <- templateLoadUpdateMsg{
		LoadedTemplates:  totalTemplates,
		TotalTemplates:   totalTemplates,
		LoadedCategories: len(categories),
		TotalCategories:  len(categories),
		CategoryLoaded:   cloneIntMap(loadedByCategory),
		CategoryTotals:   cloneIntMap(totals),
		Done:             true,
	}
}

func discoverTemplateCategories(root string) ([]string, map[string]int, int, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, nil, 0, fmt.Errorf("template categories: %w", err)
	}

	categories := []string{}
	totals := map[string]int{}
	totalTemplates := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		count, countErr := countTemplates(filepath.Join(root, name))
		if countErr != nil {
			return nil, nil, 0, countErr
		}
		categories = append(categories, name)
		totals[name] = count
		totalTemplates += count
	}

	sort.Strings(categories)
	return categories, totals, totalTemplates, nil
}

func countTemplates(root string) (int, error) {
	total := 0
	err := filepath.Walk(root, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() || strings.ToLower(filepath.Ext(path)) != ".txt" {
			return nil
		}
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		total += strings.Count(string(content), "[TEMPLATE:")
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("count templates in %s: %w", root, err)
	}
	return total, nil
}

func cloneIntMap(source map[string]int) map[string]int {
	if source == nil {
		return nil
	}
	cloned := make(map[string]int, len(source))
	for key, value := range source {
		cloned[key] = value
	}
	return cloned
}
