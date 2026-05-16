package views

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/user/poblation/internal/assets"
	"github.com/user/poblation/internal/config"
	"github.com/user/poblation/internal/save"
)

// MenuNewCivilizationMsg asks the app shell to reset and start a new run.
type MenuNewCivilizationMsg struct{}

// MenuOpenSettingsMsg asks the app shell to open the settings screen.
type MenuOpenSettingsMsg struct{}

// MenuLoadSaveMsg carries one loaded save back to the app shell.
type MenuLoadSaveMsg struct {
	Data *save.SaveData
}

// MenuQuitMsg asks the app shell to exit the program.
type MenuQuitMsg struct{}

type menuScreen string

const (
	menuScreenMain    menuScreen = "main"
	menuScreenSaves   menuScreen = "saves"
	menuScreenEndings menuScreen = "endings"
	menuScreenCredits menuScreen = "credits"
)

type menuItem struct {
	title       string
	description string
	action      string
}

func (m menuItem) Title() string       { return m.title }
func (m menuItem) Description() string { return m.description }
func (m menuItem) FilterValue() string { return m.title + " " + m.description }

type saveItem struct {
	meta save.SaveMetadata
}

func (s saveItem) Title() string {
	if s.meta.IsAutoSave {
		return "Autosave"
	}
	return fmt.Sprintf("Slot %d", s.meta.Slot)
}

func (s saveItem) Description() string {
	if s.meta.IsEmpty {
		return "— Vacio —"
	}
	date := "sin fecha"
	if !s.meta.SavedAt.IsZero() {
		date = s.meta.SavedAt.Format("2006-01-02 15:04")
	}
	return fmt.Sprintf(
		"Dia %d · Poblacion %d · %s · %s · %s",
		s.meta.Day,
		s.meta.Population,
		s.meta.Era.String(),
		trimMenuText(s.meta.MostDramaticEvent, 36),
		date,
	)
}

func (s saveItem) FilterValue() string { return s.Title() + " " + s.Description() }

type endingItem struct {
	id    string
	title string
	blurb string
}

func (e endingItem) Title() string       { return e.title }
func (e endingItem) Description() string { return e.blurb }
func (e endingItem) FilterValue() string { return e.title + " " + e.blurb }

// MenuModel renders the splash/title screen and main menu flow.
type MenuModel struct {
	state        AppStateSnapshot
	screen       menuScreen
	menuList     list.Model
	saveList     list.Model
	endingList   list.Model
	viewport     viewport.Model
	tagline      string
	status       string
	profile      config.Profile
	deletePrompt *huh.Form
	deleteChoice bool
	deleteTarget save.SaveMetadata
}

var (
	menuTaglines = []string{
		"La humanidad se extinguio. Quedan tu y alguien mas.",
		"¿Puedes reconstruir la humanidad? ¿Deberias?",
		"6 mil millones de personas. Quedan 2. Tu decides que pasa.",
		"El fin del mundo fue solo el comienzo del drama.",
		"Dos vidas, una isla, demasiados secretos para tan poca gente.",
		"Sobrevivir era la parte facil. Lo dificil es convivir.",
		"El ultimo pueblo del mundo tambien necesita chisme.",
		"La civilizacion vuelve a empezar con equipaje emocional.",
		"Cada familia nueva hereda hambre, deseo y mala memoria.",
		"Cuando ya no queda nadie, cada decision pesa como historia.",
	}

	menuFrameStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(borderColor).
			Background(backgroundColor).
			Foreground(primaryColor).
			Padding(1, 2)

	menuSplashStyle = lipgloss.NewStyle().
			Foreground(accentColor).
			Bold(true)

	menuTaglineStyle = lipgloss.NewStyle().
				Foreground(secondaryColor).
				Bold(true)

	menuVersionStyle = lipgloss.NewStyle().
				Foreground(mutedColor)

	menuStatusStyle = lipgloss.NewStyle().
			Foreground(warningColor)

	menuDetailStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(borderColor).
			Foreground(primaryColor).
			Padding(1, 2)

	menuDetailTitleStyle = lipgloss.NewStyle().
				Foreground(secondaryColor).
				Bold(true)

	menuDeleteFrameStyle = lipgloss.NewStyle().
				Border(lipgloss.DoubleBorder()).
				BorderForeground(dangerColor).
				Background(surfaceColor).
				Foreground(primaryColor).
				Padding(1, 2)
)

// NewMenuModel returns the main title/menu screen.
func NewMenuModel() MenuModel {
	menu := list.New(menuItems(), list.NewDefaultDelegate(), 34, 12)
	menu.Title = "Menu principal"
	menu.SetShowFilter(false)
	menu.SetShowStatusBar(false)
	menu.SetFilteringEnabled(false)
	menu.SetShowHelp(false)

	saves := list.New([]list.Item{}, list.NewDefaultDelegate(), 42, 14)
	saves.Title = "Saves"
	saves.SetShowFilter(false)
	saves.SetShowStatusBar(false)
	saves.SetFilteringEnabled(false)
	saves.SetShowHelp(false)

	endings := list.New([]list.Item{}, list.NewDefaultDelegate(), 36, 14)
	endings.Title = "Galeria de finales"
	endings.SetShowFilter(false)
	endings.SetShowStatusBar(false)
	endings.SetFilteringEnabled(false)
	endings.SetShowHelp(false)

	return MenuModel{
		screen:     menuScreenMain,
		menuList:   menu,
		saveList:   saves,
		endingList: endings,
		viewport:   viewport.New(48, 16),
	}
}

// Init satisfies tea.Model.
func (m MenuModel) Init() tea.Cmd {
	return nil
}

// BlocksGlobalNavigation lets the menu own its own key handling.
func (m MenuModel) BlocksGlobalNavigation() bool {
	return true
}

// SyncAppState keeps viewport sizes fresh.
func (m MenuModel) SyncAppState(snapshot AppStateSnapshot) tea.Model {
	m.state = snapshot
	m.resize()
	return m
}

func (m MenuModel) Resize(width, height int) tea.Model {
	m.state.Width = width
	m.state.Height = height
	m.resize()
	return m
}

// OnEnter refreshes menu data and rotates the tagline.
func (m MenuModel) OnEnter() (tea.Model, tea.Cmd) {
	m.profile = config.LoadOrDefault()
	m.tagline = randomTagline()
	m.screen = menuScreenMain
	m.status = ""
	m.deletePrompt = nil
	m.rebuildSaveList()
	m.rebuildEndingList()
	m.viewport.SetContent(renderCreditsNarrative(m.profile))
	m.viewport.GotoTop()
	m.resize()
	return m, nil
}

// Update handles menu navigation and sub-screens.
func (m MenuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if size, ok := msg.(tea.WindowSizeMsg); ok {
		return m.Resize(size.Width, size.Height), nil
	}

	if m.deletePrompt != nil {
		updated, cmd := m.deletePrompt.Update(msg)
		m.deletePrompt = updated.(*huh.Form)
		if m.deletePrompt.State == huh.StateCompleted {
			return m.finishDeletePrompt()
		}
		return m, cmd
	}

	keyMsg, isKey := msg.(tea.KeyMsg)
	if isKey && m.handleBackKey(keyMsg) {
		return m, nil
	}

	switch m.screen {
	case menuScreenMain:
		return m.updateMainMenu(msg)
	case menuScreenSaves:
		return m.updateSaves(msg)
	case menuScreenEndings:
		return m.updateEndings(msg)
	case menuScreenCredits:
		return m.updateCredits(msg)
	default:
		return m, nil
	}
}

// View renders the title screen or one of the sub-pages.
func (m MenuModel) View() string {
	switch m.screen {
	case menuScreenSaves:
		return m.renderSavesScreen()
	case menuScreenEndings:
		return m.renderEndingsScreen()
	case menuScreenCredits:
		return m.renderCreditsScreen()
	default:
		return m.renderMainScreen()
	}
}

func (m *MenuModel) resize() {
	width := maxInt(34, m.state.Width-10)
	height := maxInt(10, m.state.Height-10)
	m.menuList.SetSize(maxInt(28, width/2), maxInt(8, height))
	m.saveList.SetSize(maxInt(32, width/2), maxInt(10, height))
	m.endingList.SetSize(maxInt(30, width/2), maxInt(10, height))
	m.viewport.Width = maxInt(30, width/2)
	m.viewport.Height = maxInt(10, height)
}

func (m MenuModel) updateMainMenu(msg tea.Msg) (tea.Model, tea.Cmd) {
	updated, cmd := m.menuList.Update(msg)
	m.menuList = updated

	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok || keyMsg.String() != "enter" {
		return m, cmd
	}

	selected, ok := m.menuList.SelectedItem().(menuItem)
	if !ok {
		return m, nil
	}

	switch selected.action {
	case "new":
		return m, func() tea.Msg { return MenuNewCivilizationMsg{} }
	case "continue":
		m.screen = menuScreenSaves
		m.status = "ENTER carga · D borra · ESC vuelve"
		return m, nil
	case "endings":
		m.screen = menuScreenEndings
		m.status = "ESC vuelve al menu"
		m.syncEndingViewport()
		return m, nil
	case "settings":
		return m, func() tea.Msg { return MenuOpenSettingsMsg{} }
	case "credits":
		m.screen = menuScreenCredits
		m.viewport.SetContent(renderCreditsNarrative(m.profile))
		m.viewport.GotoTop()
		m.status = "ESC vuelve al menu"
		return m, nil
	case "quit":
		return m, func() tea.Msg { return MenuQuitMsg{} }
	default:
		return m, nil
	}
}

func (m MenuModel) updateSaves(msg tea.Msg) (tea.Model, tea.Cmd) {
	updated, cmd := m.saveList.Update(msg)
	m.saveList = updated

	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, cmd
	}

	switch keyMsg.String() {
	case "enter":
		return m.loadSelectedSave()
	case "d":
		return m.beginDeletePrompt()
	case "r":
		m.rebuildSaveList()
		m.status = "Saves refrescados."
		return m, nil
	default:
		return m, cmd
	}
}

func (m MenuModel) updateEndings(msg tea.Msg) (tea.Model, tea.Cmd) {
	updated, cmd := m.endingList.Update(msg)
	m.endingList = updated

	if _, ok := msg.(tea.KeyMsg); ok {
		m.syncEndingViewport()
	}
	return m, cmd
}

func (m MenuModel) updateCredits(msg tea.Msg) (tea.Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	switch keyMsg.String() {
	case "up":
		m.viewport.ScrollUp(1)
	case "down":
		m.viewport.ScrollDown(1)
	case "pgup":
		m.viewport.ScrollUp(maxInt(1, m.viewport.Height/2))
	case "pgdown":
		m.viewport.ScrollDown(maxInt(1, m.viewport.Height/2))
	case "home":
		m.viewport.GotoTop()
	case "end":
		m.viewport.GotoBottom()
	}
	return m, nil
}

func (m *MenuModel) handleBackKey(msg tea.KeyMsg) bool {
	switch msg.String() {
	case "esc", "backspace":
		if m.deletePrompt != nil {
			m.deletePrompt = nil
			m.status = "Borrado cancelado."
			return true
		}
		if m.screen != menuScreenMain {
			m.screen = menuScreenMain
			m.status = ""
			return true
		}
	}
	return false
}

func (m MenuModel) renderMainScreen() string {
	width := maxInt(70, m.state.Width-2)
	logoBlock := renderMenuLogo()
	listBlock := menuFrameStyle.Width(maxInt(34, width/2)).Render(m.menuList.View())

	body := lipgloss.JoinVertical(
		lipgloss.Left,
		logoBlock,
		"",
		menuTaglineStyle.Render(m.tagline),
		"",
		listBlock,
	)

	if strings.TrimSpace(m.status) != "" {
		body = lipgloss.JoinVertical(lipgloss.Left, body, "", menuStatusStyle.Render(m.status))
	}

	versionLine := lipgloss.PlaceHorizontal(width, lipgloss.Right, menuVersionStyle.Render("v"+config.GameVersion))
	return lipgloss.JoinVertical(lipgloss.Left, body, "", versionLine)
}

func (m MenuModel) renderSavesScreen() string {
	left := menuFrameStyle.Width(maxInt(34, m.state.Width/2-4)).Render(m.saveList.View())
	right := menuDetailStyle.Width(maxInt(30, m.state.Width/2-6)).Render(m.renderSaveDetail())
	content := lipgloss.JoinHorizontal(lipgloss.Top, left, right)
	if m.deletePrompt == nil {
		return lipgloss.JoinVertical(lipgloss.Left, content, "", menuStatusStyle.Render(fallbackMenuStatus(m.status, "ENTER carga · D borra · R refresca · ESC vuelve")))
	}
	prompt := menuDeleteFrameStyle.Width(maxInt(28, m.state.Width-14)).Render(m.deletePrompt.View())
	return lipgloss.JoinVertical(
		lipgloss.Left,
		content,
		"",
		prompt,
	)
}

func (m MenuModel) renderEndingsScreen() string {
	left := menuFrameStyle.Width(maxInt(30, m.state.Width/2-4)).Render(m.endingList.View())
	right := menuDetailStyle.Width(maxInt(32, m.state.Width/2-6)).Render(m.viewport.View())
	return lipgloss.JoinVertical(
		lipgloss.Left,
		lipgloss.JoinHorizontal(lipgloss.Top, left, right),
		"",
		menuStatusStyle.Render(fallbackMenuStatus(m.status, "ESC vuelve al menu")),
	)
}

func (m MenuModel) renderCreditsScreen() string {
	block := menuFrameStyle.Width(maxInt(56, m.state.Width-10)).Height(maxInt(16, m.state.Height-8)).Render(m.viewport.View())
	return lipgloss.JoinVertical(
		lipgloss.Left,
		menuDetailTitleStyle.Render("Libro de historia de la civilizacion"),
		block,
		"",
		menuStatusStyle.Render(fallbackMenuStatus(m.status, "ESC vuelve al menu")),
	)
}

func (m MenuModel) renderSaveDetail() string {
	selected, ok := m.saveList.SelectedItem().(saveItem)
	if !ok {
		return "No hay save seleccionado."
	}
	if selected.meta.IsEmpty {
		title := "Autosave vacio"
		if !selected.meta.IsAutoSave {
			title = fmt.Sprintf("Slot %d vacio", selected.meta.Slot)
		}
		return lipgloss.JoinVertical(
			lipgloss.Left,
			menuDetailTitleStyle.Render(title),
			"",
			menuVersionStyle.Render("Todavia no hay nada guardado aqui."),
		)
	}

	date := selected.meta.SavedAt.Format("2006-01-02 15:04")
	lines := []string{
		menuDetailTitleStyle.Render(selected.Title()),
		"",
		fmt.Sprintf("Dia guardado: %d", selected.meta.Day),
		fmt.Sprintf("Poblacion: %d", selected.meta.Population),
		fmt.Sprintf("Era: %s", selected.meta.Era.String()),
		fmt.Sprintf("Fecha de guardado: %s", date),
		"",
		"Evento mas dramatico:",
		trimMenuText(selected.meta.MostDramaticEvent, 260),
	}
	if strings.TrimSpace(selected.meta.Screenshot) != "" {
		lines = append(lines, "", "Miniatura:", selected.meta.Screenshot)
	}
	return strings.Join(lines, "\n")
}

func (m *MenuModel) syncEndingViewport() {
	selected, ok := m.endingList.SelectedItem().(endingItem)
	if !ok {
		m.viewport.SetContent("Todavia no hay finales desbloqueados.")
		return
	}
	lines := []string{
		menuDetailTitleStyle.Render(selected.title),
		"",
		selected.blurb,
		"",
		renderEndingLore(selected.id),
	}
	m.viewport.SetContent(strings.Join(lines, "\n"))
	m.viewport.GotoTop()
}

func (m *MenuModel) rebuildSaveList() {
	items := make([]list.Item, 0, 6)
	for _, meta := range save.ListSaves() {
		items = append(items, saveItem{meta: meta})
	}
	m.saveList.SetItems(items)
}

func (m *MenuModel) rebuildEndingList() {
	items := make([]list.Item, 0, len(m.profile.UnlockedEndings))
	for _, id := range m.profile.UnlockedEndings {
		items = append(items, endingItem{
			id:    id,
			title: endingDisplayTitle(id),
			blurb: endingDisplayBlurb(id),
		})
	}
	if len(items) == 0 {
		items = append(items, endingItem{
			id:    "",
			title: "Sin finales todavia",
			blurb: "La civilizacion aun no ha dejado una leyenda digna de museo.",
		})
	}
	m.endingList.SetItems(items)
	m.syncEndingViewport()
}

func (m MenuModel) loadSelectedSave() (tea.Model, tea.Cmd) {
	selected, ok := m.saveList.SelectedItem().(saveItem)
	if !ok {
		return m, nil
	}
	if selected.meta.IsEmpty {
		m.status = "Ese slot esta vacio."
		return m, nil
	}

	var (
		data *save.SaveData
		err  error
	)
	if selected.meta.IsAutoSave {
		data, err = save.LoadAutosave()
	} else {
		data, err = save.Load(selected.meta.Slot)
	}
	if err != nil {
		m.status = "No pude cargar ese save."
		return m, nil
	}
	return m, func() tea.Msg {
		return MenuLoadSaveMsg{Data: data}
	}
}

func (m MenuModel) beginDeletePrompt() (tea.Model, tea.Cmd) {
	selected, ok := m.saveList.SelectedItem().(saveItem)
	if !ok || selected.meta.IsEmpty {
		m.status = "Nada que borrar aqui."
		return m, nil
	}

	m.deleteTarget = selected.meta
	m.deleteChoice = false
	m.deletePrompt = huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title(deletePromptTitle(selected.meta)).
				Description("Esto borra el archivo del disco.").
				Affirmative("Si, borrarlo").
				Negative("No, dejarlo").
				Value(&m.deleteChoice),
		).Title("Confirmar borrado"),
	).WithTheme(menuModalTheme())
	return m, m.deletePrompt.Init()
}

func (m MenuModel) finishDeletePrompt() (tea.Model, tea.Cmd) {
	target := m.deleteTarget
	confirmed := m.deleteChoice
	m.deletePrompt = nil
	m.deleteChoice = false

	if !confirmed {
		m.status = "Borrado cancelado."
		return m, nil
	}

	var err error
	if target.IsAutoSave {
		err = save.DeleteAutosave()
	} else {
		err = save.Delete(target.Slot)
	}
	if err != nil {
		m.status = "No pude borrar ese save."
		return m, nil
	}

	m.rebuildSaveList()
	m.status = "Save borrado."
	return m, nil
}

func menuItems() []list.Item {
	return []list.Item{
		menuItem{title: "Nueva Civilizacion", description: "Empieza una isla nueva y entra al flow de los pobles.", action: "new"},
		menuItem{title: "Continuar", description: "Revisa slots, autosave y carga una historia ya empezada.", action: "continue"},
		menuItem{title: "Galeria de Finales", description: "Mira las leyendas que ya has desbloqueado.", action: "endings"},
		menuItem{title: "Ajustes", description: "Graficos, audio, contenido, accesibilidad y mas.", action: "settings"},
		menuItem{title: "Creditos", description: "La version in-character del libro de historia.", action: "credits"},
		menuItem{title: "Salir", description: "Cierra POBLATION.", action: "quit"},
	}
}

func randomTagline() string {
	seed := time.Now().UnixNano()
	rng := rand.New(rand.NewSource(seed))
	return menuTaglines[rng.Intn(len(menuTaglines))]
}

func renderMenuLogo() string {
	if rendered := terminalPNG(assets.WordmarkPNG); rendered != "" {
		return rendered
	}
	return menuSplashStyle.Render("POBLATION")
}

func trimMenuText(value string, limit int) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "— Vacio —"
	}
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit-1]) + "…"
}

func fallbackMenuStatus(current, fallback string) string {
	if strings.TrimSpace(current) == "" {
		return fallback
	}
	return current
}

func deletePromptTitle(meta save.SaveMetadata) string {
	if meta.IsAutoSave {
		return "¿Borrar el autosave?"
	}
	return fmt.Sprintf("¿Borrar el slot %d?", meta.Slot)
}

func renderCreditsNarrative(profile config.Profile) string {
	endingCount := len(profile.UnlockedEndings)
	lines := []string{
		"En los anales de la civilizacion, hubo creadores que no aparecieron en las plazas pero dejaron leyes, colores y caprichos pegados a la madera de cada menu.",
		"",
		fmt.Sprintf("Esta era del archivo se conoce como la version %s.", config.GameVersion),
		fmt.Sprintf("Hasta ahora el libro recuerda %d finales dignos de conservar.", endingCount),
		"",
		"No levantaron templos con piedra. Lo hicieron con atajos, paletas, sliders y la insistencia de que incluso el desastre tuviera buena tipografia.",
		"",
		"Si alguien pregunta quienes hicieron esto, la historia responde asi:",
		"",
		"Los primeros dioses dieron cuerda al reloj.",
		"Otros nombraron a los pobles antes de que aprendieran a mentir solos.",
		"Alguien negocio con el caos para que el autosave llegara antes del arrepentimiento.",
		"Y otra mano escondio contenido adulto detras de un ajuste, porque hasta el deseo necesita consentimiento.",
		"",
		"Cuando la isla recuerde a sus autores, no dira creditos.",
		"Dira: aqui estuvieron quienes no le tuvieron miedo a empezar otra vez.",
	}
	return strings.Join(lines, "\n")
}

func endingDisplayTitle(id string) string {
	switch id {
	case "END_LOVE":
		return "End Love"
	case "END_DYNASTY":
		return "End Dynasty"
	case "END_WAR":
		return "End War"
	case "END_UTOPIA":
		return "End Utopia"
	case "END_CULT":
		return "End Cult"
	case "END_ALONE":
		return "End Alone"
	case "END_RESET":
		return "End Reset"
	case "END_MYTH":
		return "End Myth"
	default:
		return "Final sin nombre"
	}
}

func endingDisplayBlurb(id string) string {
	switch id {
	case "END_LOVE":
		return "La humanidad termino como una historia de amor y perdida."
	case "END_DYNASTY":
		return "Una sola sangre aprendio a llamarse pais."
	case "END_WAR":
		return "Todo lo que quedaba se quemo peleando."
	case "END_UTOPIA":
		return "La isla logro algo parecido a una paz improbable."
	case "END_CULT":
		return "La fe dejo de ser consuelo y se volvio sistema."
	case "END_ALONE":
		return "Solo quedo una voz para recordar a todas las demas."
	case "END_RESET":
		return "La civilizacion volvio a empezar despues del colapso."
	case "END_MYTH":
		return "La historia se volvio tan vieja que ya parecia leyenda."
	default:
		return "Un final desbloqueado por el simple hecho de insistir."
	}
}

func renderEndingLore(id string) string {
	switch id {
	case "END_LOVE":
		return "Las cronicas dicen que el ultimo recuerdo del mundo no fue una guerra, sino una promesa demasiado humana para sobrevivir intacta."
	case "END_DYNASTY":
		return "A fuerza de hijos, leyes y apellidos repetidos, una casa aprendio a confundirse con el destino de todos."
	case "END_WAR":
		return "Los restos de la historia no encontraron acuerdo. Solo nombres quebrados y una arena demasiado acostumbrada a la sangre."
	case "END_UTOPIA":
		return "No fue perfeccion. Fue trabajo lento, rutina y la rara terquedad de cuidar lo comun sin volverlo jaula."
	case "END_CULT":
		return "Cada verdad empezo a sonar como mandamiento y toda duda fue tratada como pecado administrativo."
	case "END_ALONE":
		return "Con una sola persona respirando, la especie descubrio que el silencio tambien puede ser herencia."
	case "END_RESET":
		return "La civilizacion cayo sobre si misma y dejo una nota brutal: aun se puede empezar otra vez, pero no gratis."
	case "END_MYTH":
		return "La memoria se adelgazo tanto que la historia ya no se sabia a si misma. Solo quedaban simbolos."
	default:
		return "Este final todavia no encontro una glosa decente."
	}
}

func menuModalTheme() *huh.Theme {
	theme := huh.ThemeBase()
	button := lipgloss.NewStyle().Padding(0, 2).MarginRight(1)

	theme.Form.Base = lipgloss.NewStyle().Background(surfaceColor).Foreground(primaryColor)
	theme.Group.Title = lipgloss.NewStyle().Foreground(secondaryColor).Bold(true)
	theme.Group.Description = lipgloss.NewStyle().Foreground(mutedColor)
	theme.Focused.Base = lipgloss.NewStyle().PaddingLeft(1).BorderStyle(lipgloss.NormalBorder()).BorderLeft(true).BorderForeground(borderColor)
	theme.Focused.Card = theme.Focused.Base
	theme.Focused.Title = lipgloss.NewStyle().Foreground(primaryColor).Bold(true)
	theme.Focused.NoteTitle = lipgloss.NewStyle().Foreground(secondaryColor).Bold(true)
	theme.Focused.Description = lipgloss.NewStyle().Foreground(mutedColor)
	theme.Focused.ErrorIndicator = lipgloss.NewStyle().Foreground(dangerColor).SetString(" !")
	theme.Focused.ErrorMessage = lipgloss.NewStyle().Foreground(dangerColor)
	theme.Focused.SelectSelector = lipgloss.NewStyle().Foreground(accentColor).SetString("> ")
	theme.Focused.Option = lipgloss.NewStyle().Foreground(primaryColor)
	theme.Focused.FocusedButton = button.Foreground(backgroundColor).Background(accentColor).Bold(true)
	theme.Focused.BlurredButton = button.Foreground(primaryColor).Background(borderColor)
	theme.Focused.Next = theme.Focused.FocusedButton
	theme.Focused.TextInput.Cursor = lipgloss.NewStyle().Foreground(successColor)
	theme.Focused.TextInput.CursorText = lipgloss.NewStyle().Foreground(primaryColor)
	theme.Focused.TextInput.Placeholder = lipgloss.NewStyle().Foreground(mutedColor)
	theme.Focused.TextInput.Prompt = lipgloss.NewStyle().Foreground(accentColor).Bold(true)
	theme.Focused.TextInput.Text = lipgloss.NewStyle().Foreground(primaryColor)

	theme.Blurred = theme.Focused
	theme.Blurred.Base = lipgloss.NewStyle().PaddingLeft(1)
	theme.Blurred.Card = theme.Blurred.Base
	theme.Blurred.NoteTitle = lipgloss.NewStyle().Foreground(mutedColor).Bold(true)
	theme.Blurred.Title = lipgloss.NewStyle().Foreground(mutedColor)
	theme.Blurred.Description = lipgloss.NewStyle().Foreground(mutedColor)
	return theme
}
