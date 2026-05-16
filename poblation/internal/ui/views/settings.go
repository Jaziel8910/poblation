package views

import (
	"fmt"
	"slices"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/user/poblation/internal/config"
)

type settingsCategory string

const (
	settingsGraphics      settingsCategory = "graphics"
	settingsAudio         settingsCategory = "audio"
	settingsGameplay      settingsCategory = "gameplay"
	settingsContent       settingsCategory = "content"
	settingsInterface     settingsCategory = "interface"
	settingsAccessibility settingsCategory = "accessibility"
	settingsControls      settingsCategory = "controls"
)

type settingsCategoryItem struct {
	label    string
	category settingsCategory
}

func (s settingsCategoryItem) Title() string       { return s.label }
func (s settingsCategoryItem) Description() string { return "" }
func (s settingsCategoryItem) FilterValue() string { return s.label }

type settingsField struct {
	Label       string
	Description string
	Value       string
}

// CloseSettingsMsg asks the app shell to leave the settings screen.
type CloseSettingsMsg struct{}

// SettingsModel renders the full settings screen with multiple categories.
type SettingsModel struct {
	state          AppStateSnapshot
	profile        config.Profile
	categoryList   list.Model
	selectedField  int
	status         string
	adultPrompt    *huh.Form
	adultConfirmed bool
}

var (
	settingsFrameStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(borderColor).
				Background(backgroundColor).
				Foreground(primaryColor).
				Padding(1, 2)

	settingsFieldStyle = lipgloss.NewStyle().
				Foreground(primaryColor)

	settingsFieldSelectedStyle = lipgloss.NewStyle().
					Foreground(accentColor).
					Bold(true)

	settingsValueStyle = lipgloss.NewStyle().
				Foreground(secondaryColor).
				Bold(true)

	settingsDescStyle = lipgloss.NewStyle().
				Foreground(mutedColor)
)

// NewSettingsModel returns the functional settings screen.
func NewSettingsModel() SettingsModel {
	categories := list.New(settingsCategoryItems(), list.NewDefaultDelegate(), 24, 18)
	categories.Title = "Categorias"
	categories.SetShowFilter(false)
	categories.SetShowStatusBar(false)
	categories.SetFilteringEnabled(false)
	categories.SetShowHelp(false)

	return SettingsModel{
		categoryList: categories,
	}
}

// Init satisfies tea.Model.
func (m SettingsModel) Init() tea.Cmd {
	return nil
}

// BlocksGlobalNavigation keeps global shortcuts from hijacking local edits.
func (m SettingsModel) BlocksGlobalNavigation() bool {
	return true
}

// SyncAppState stores sizing data.
func (m SettingsModel) SyncAppState(snapshot AppStateSnapshot) tea.Model {
	m.state = snapshot
	m.resize()
	return m
}

func (m SettingsModel) Resize(width, height int) tea.Model {
	m.state.Width = width
	m.state.Height = height
	m.resize()
	return m
}

// OnEnter refreshes profile data from disk.
func (m SettingsModel) OnEnter() (tea.Model, tea.Cmd) {
	m.profile = config.LoadOrDefault()
	m.selectedField = 0
	m.status = ""
	m.adultPrompt = nil
	m.resize()
	return m, nil
}

// Update handles category nav, field edits and adult-content confirmation.
func (m SettingsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if size, ok := msg.(tea.WindowSizeMsg); ok {
		return m.Resize(size.Width, size.Height), nil
	}

	if m.adultPrompt != nil {
		updated, cmd := m.adultPrompt.Update(msg)
		m.adultPrompt = updated.(*huh.Form)
		if m.adultPrompt.State == huh.StateCompleted {
			return m.finishAdultPrompt()
		}
		return m, cmd
	}

	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	switch keyMsg.String() {
	case "tab":
		updated, cmd := m.categoryList.Update(keyMsg)
		m.categoryList = updated
		m.selectedField = 0
		return m, cmd
	case "up":
		if m.selectedField > 0 {
			m.selectedField--
			return m, nil
		}
		updated, cmd := m.categoryList.Update(keyMsg)
		m.categoryList = updated
		m.selectedField = 0
		return m, cmd
	case "down":
		fields := m.currentFields()
		if m.selectedField < len(fields)-1 {
			m.selectedField++
			return m, nil
		}
		return m, nil
	case "left":
		return m.adjustCurrentField(-1)
	case "right":
		return m.adjustCurrentField(1)
	case "enter":
		return m.adjustCurrentField(0)
	case "esc", "backspace":
		return m, func() tea.Msg { return CloseSettingsMsg{} }
	default:
		updated, cmd := m.categoryList.Update(keyMsg)
		m.categoryList = updated
		return m, cmd
	}
}

// View renders categories on the left and current category options on the right.
func (m SettingsModel) View() string {
	left := settingsFrameStyle.Width(maxInt(22, m.state.Width/3-4)).Render(m.categoryList.View())
	right := settingsFrameStyle.Width(maxInt(36, (m.state.Width*2)/3-8)).Render(m.renderFieldsPanel())

	content := lipgloss.JoinHorizontal(lipgloss.Top, left, right)
	if m.adultPrompt != nil {
		prompt := menuDeleteFrameStyle.Width(maxInt(30, m.state.Width-16)).Render(m.adultPrompt.View())
		return lipgloss.JoinVertical(lipgloss.Left, content, "", prompt)
	}
	return lipgloss.JoinVertical(
		lipgloss.Left,
		content,
		"",
		menuStatusStyle.Render(fallbackMenuStatus(m.status, "LEFT/RIGHT cambia · TAB cambia categoria · ESC vuelve")),
	)
}

func (m *SettingsModel) resize() {
	m.categoryList.SetSize(maxInt(18, m.state.Width/3-10), maxInt(10, m.state.Height-10))
}

func (m SettingsModel) currentCategory() settingsCategory {
	selected, ok := m.categoryList.SelectedItem().(settingsCategoryItem)
	if !ok {
		return settingsGraphics
	}
	return selected.category
}

func (m SettingsModel) currentFields() []settingsField {
	s := m.profile.Settings
	switch m.currentCategory() {
	case settingsGraphics:
		return []settingsField{
			{"Color mode", "Tema base del UI. De momento manda el estilo general del terminal.", s.Graphics.ColorMode},
			{"UI scale", "Escala pensada para paneles y densidad visual.", fmt.Sprintf("%d%%", s.Graphics.UIScale)},
			{"Reduce motion", "Baja animacion y scroll agresivo.", onOff(s.Graphics.ReduceMotion)},
			{"Weather FX", "Permite adornos visuales de clima y ambiente.", onOff(s.Graphics.ShowWeatherFX)},
			{"VSync", "Mantiene redibujos mas estables cuando el terminal coopera.", onOff(s.Graphics.VSync)},
			{"Target FPS", "Tope deseado para refresco visual.", fmt.Sprintf("%d", s.Graphics.TargetFPS)},
		}
	case settingsAudio:
		return []settingsField{
			{"Mute all", "Silencia cualquier capa de audio futura.", onOff(s.Audio.MuteAll)},
			{"Master volume", "Volumen global.", fmt.Sprintf("%d%%", s.Audio.MasterVolume)},
			{"Music volume", "Musica y capas largas.", fmt.Sprintf("%d%%", s.Audio.MusicVolume)},
			{"Ambience volume", "Ambiente, clima y cama sonora.", fmt.Sprintf("%d%%", s.Audio.AmbienceVolume)},
			{"UI volume", "Clicks, confirmaciones y sonido de interfaz.", fmt.Sprintf("%d%%", s.Audio.UIVolume)},
		}
	case settingsGameplay:
		return []settingsField{
			{"Autosave", "Guarda solo cada cierto tiempo si esta activo.", onOff(s.Gameplay.AutoSaveEnabled)},
			{"Autosave interval", "Minutos entre autosaves.", fmt.Sprintf("%d min", s.Gameplay.AutoSaveMinutes)},
			{"Pause on focus loss", "Pausa cuando la ventana pierde foco.", onOff(s.Gameplay.PauseOnFocusLoss)},
			{"Default speed", "Velocidad base del tiempo de simulacion.", fmt.Sprintf("%.1fx", s.Gameplay.DefaultSimulationSpeed)},
			{"Tutorial hints", "Ayudas suaves para sistemas nuevos.", onOff(s.Gameplay.TutorialHints)},
			{"Confirm risky acts", "Pide doble chequeo antes de acciones brutas.", onOff(s.Gameplay.ConfirmDangerousActs)},
			{"Drama density", "Cuanto ruido emocional quieres por tick.", s.Gameplay.DramaDensity},
		}
	case settingsContent:
		return []settingsField{
			{"Adult content", "Por defecto va apagado. Al encenderlo confirmas que eres mayor de edad en tu region.", onOff(s.Content.AdultContentEnabled)},
			{"Explicit language", "Permite lenguaje mas directo y sucio.", onOff(s.Content.ExplicitLanguage)},
			{"Graphic violence", "Sube el detalle de peleas y heridas.", onOff(s.Content.GraphicViolence)},
			{"Incest content", "Mantiene activo ese eje duro del lore cuando toque.", onOff(s.Content.IncestContent)},
			{"Body horror", "Permite imagineria corporal mas fea.", onOff(s.Content.BodyHorror)},
			{"Pregnancy detail", "Describe embarazo y complicaciones con mas detalle.", onOff(s.Content.PregnancyDetail)},
		}
	case settingsInterface:
		return []settingsField{
			{"Timestamps", "Muestra marcas de tiempo en paneles y registros.", onOff(s.Interface.ShowTimestamps)},
			{"Compact feed", "Aprieta mas el feed para ver mas linea por pantalla.", onOff(s.Interface.CompactFeed)},
			{"Poble badges", "Enseña marcadores cortos sobre estado y rol.", onOff(s.Interface.ShowPobleBadges)},
			{"Persistent notices", "Las notificaciones no se evaporan tan rapido.", onOff(s.Interface.PersistentNotices)},
			{"Map labels", "Intenta mantener nombres y marcas visibles.", onOff(s.Interface.MapLabels)},
			{"Save thumbnails", "Permite miniaturas de texto en la lista de saves.", onOff(s.Interface.ShowSaveThumbnails)},
		}
	case settingsAccessibility:
		return []settingsField{
			{"High contrast", "Empuja mas la diferencia entre fondo y texto.", onOff(s.Accessibility.HighContrast)},
			{"Screen reader mode", "Pensado para terminales con lectores de pantalla.", onOff(s.Accessibility.ScreenReaderMode)},
			{"Reduce flashing", "Baja parpadeos y destellos.", onOff(s.Accessibility.ReduceFlashing)},
			{"Dyslexia friendly", "Deja el texto un poco mas limpio y espacioso.", onOff(s.Accessibility.DyslexiaFriendly)},
			{"Bigger spacing", "Mete aire extra entre bloques.", onOff(s.Accessibility.BiggerLineSpacing)},
			{"Hold to confirm", "Pide una confirmacion mas deliberada.", onOff(s.Accessibility.HoldToConfirm)},
		}
	case settingsControls:
		return []settingsField{
			{"Vim navigation", "Activa j/k/h/l como atajos extra.", onOff(s.Controls.VimNavigation)},
			{"WASD map pan", "Mueve camara del mapa con WASD.", onOff(s.Controls.WASDMapPan)},
			{"Invert scroll", "Invierte direccion del scroll vertical.", onOff(s.Controls.InvertScroll)},
			{"Confirm on quit", "Pide un si final antes de cerrar.", onOff(s.Controls.ConfirmOnQuit)},
			{"Quick pause", "Mantiene la pausa rapida en la barra espaciadora.", onOff(s.Controls.QuickPause)},
		}
	default:
		return nil
	}
}

func (m SettingsModel) renderFieldsPanel() string {
	fields := m.currentFields()
	categoryTitle := strings.ToUpper(string(m.currentCategory()))
	lines := []string{menuDetailTitleStyle.Render(categoryTitle), ""}

	for i, field := range fields {
		labelStyle := settingsFieldStyle
		if i == m.selectedField {
			labelStyle = settingsFieldSelectedStyle
		}
		lines = append(lines,
			labelStyle.Render(field.Label)+": "+settingsValueStyle.Render(field.Value),
			settingsDescStyle.Render(field.Description),
			"",
		)
	}

	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func (m SettingsModel) adjustCurrentField(direction int) (tea.Model, tea.Cmd) {
	category := m.currentCategory()
	fields := m.currentFields()
	if len(fields) == 0 {
		return m, nil
	}
	index := clampInt(m.selectedField, 0, len(fields)-1)

	if category == settingsContent && index == 0 && !m.profile.Settings.Content.AdultContentEnabled {
		if direction >= 0 {
			return m.beginAdultPrompt()
		}
	}

	next := m.profile.Settings
	switch category {
	case settingsGraphics:
		adjustGraphicsField(&next, index, direction)
	case settingsAudio:
		adjustAudioField(&next, index, direction)
	case settingsGameplay:
		adjustGameplayField(&next, index, direction)
	case settingsContent:
		adjustContentField(&next, index, direction)
	case settingsInterface:
		adjustInterfaceField(&next, index, direction)
	case settingsAccessibility:
		adjustAccessibilityField(&next, index, direction)
	case settingsControls:
		adjustControlsField(&next, index, direction)
	}

	m.profile.Settings = next
	if err := config.Save(m.profile); err != nil {
		m.status = "No pude guardar el ajuste."
		return m, nil
	}
	m.status = "Ajuste guardado."
	return m, nil
}

func (m SettingsModel) beginAdultPrompt() (tea.Model, tea.Cmd) {
	m.adultConfirmed = false
	m.adultPrompt = huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("¿Activar contenido adulto?").
				Description("Esto deja activo contenido sexual y mas explicitud. Al hacerlo confirmas que eres mayor de edad en tu region.").
				Affirmative("Si, activarlo").
				Negative("No, dejarlo apagado").
				Value(&m.adultConfirmed),
		).Title("Contenido"),
	).WithTheme(menuModalTheme())
	return m, m.adultPrompt.Init()
}

func (m SettingsModel) finishAdultPrompt() (tea.Model, tea.Cmd) {
	if !m.adultConfirmed {
		m.adultPrompt = nil
		m.status = "Contenido adulto sigue apagado."
		return m, nil
	}

	m.adultPrompt = nil
	if err := config.EnableAdultContent(); err != nil {
		m.status = "No pude guardar ese ajuste."
		return m, nil
	}
	m.profile = config.LoadOrDefault()
	m.status = "Contenido adulto activado."
	return m, nil
}

func settingsCategoryItems() []list.Item {
	return []list.Item{
		settingsCategoryItem{label: "Graficos", category: settingsGraphics},
		settingsCategoryItem{label: "Audio", category: settingsAudio},
		settingsCategoryItem{label: "Gameplay", category: settingsGameplay},
		settingsCategoryItem{label: "Contenido", category: settingsContent},
		settingsCategoryItem{label: "Interfaz", category: settingsInterface},
		settingsCategoryItem{label: "Accesibilidad", category: settingsAccessibility},
		settingsCategoryItem{label: "Controles", category: settingsControls},
	}
}

func onOff(value bool) string {
	if value {
		return "On"
	}
	return "Off"
}

func adjustGraphicsField(settings *config.Settings, index, direction int) {
	switch index {
	case 0:
		settings.Graphics.ColorMode = cycleString(settings.Graphics.ColorMode, []string{"theme", "muted", "high-contrast"}, direction)
	case 1:
		settings.Graphics.UIScale = stepInt(settings.Graphics.UIScale, direction, 10, 80, 140)
	case 2:
		settings.Graphics.ReduceMotion = toggleBool(settings.Graphics.ReduceMotion, direction)
	case 3:
		settings.Graphics.ShowWeatherFX = toggleBool(settings.Graphics.ShowWeatherFX, direction)
	case 4:
		settings.Graphics.VSync = toggleBool(settings.Graphics.VSync, direction)
	case 5:
		settings.Graphics.TargetFPS = stepFromList(settings.Graphics.TargetFPS, []int{24, 30, 45, 60, 90, 120}, direction)
	}
}

func adjustAudioField(settings *config.Settings, index, direction int) {
	switch index {
	case 0:
		settings.Audio.MuteAll = toggleBool(settings.Audio.MuteAll, direction)
	case 1:
		settings.Audio.MasterVolume = stepInt(settings.Audio.MasterVolume, direction, 5, 0, 100)
	case 2:
		settings.Audio.MusicVolume = stepInt(settings.Audio.MusicVolume, direction, 5, 0, 100)
	case 3:
		settings.Audio.AmbienceVolume = stepInt(settings.Audio.AmbienceVolume, direction, 5, 0, 100)
	case 4:
		settings.Audio.UIVolume = stepInt(settings.Audio.UIVolume, direction, 5, 0, 100)
	}
}

func adjustGameplayField(settings *config.Settings, index, direction int) {
	switch index {
	case 0:
		settings.Gameplay.AutoSaveEnabled = toggleBool(settings.Gameplay.AutoSaveEnabled, direction)
	case 1:
		settings.Gameplay.AutoSaveMinutes = stepFromList(settings.Gameplay.AutoSaveMinutes, []int{5, 10, 15, 20, 30, 60}, direction)
	case 2:
		settings.Gameplay.PauseOnFocusLoss = toggleBool(settings.Gameplay.PauseOnFocusLoss, direction)
	case 3:
		settings.Gameplay.DefaultSimulationSpeed = stepFromListFloat(settings.Gameplay.DefaultSimulationSpeed, []float64{0.5, 1.0, 2.0, 4.0}, direction)
	case 4:
		settings.Gameplay.TutorialHints = toggleBool(settings.Gameplay.TutorialHints, direction)
	case 5:
		settings.Gameplay.ConfirmDangerousActs = toggleBool(settings.Gameplay.ConfirmDangerousActs, direction)
	case 6:
		settings.Gameplay.DramaDensity = cycleString(settings.Gameplay.DramaDensity, []string{"low", "normal", "high", "chaotic"}, direction)
	}
}

func adjustContentField(settings *config.Settings, index, direction int) {
	switch index {
	case 0:
		settings.Content.AdultContentEnabled = toggleBool(settings.Content.AdultContentEnabled, direction)
	case 1:
		settings.Content.ExplicitLanguage = toggleBool(settings.Content.ExplicitLanguage, direction)
	case 2:
		settings.Content.GraphicViolence = toggleBool(settings.Content.GraphicViolence, direction)
	case 3:
		settings.Content.IncestContent = toggleBool(settings.Content.IncestContent, direction)
	case 4:
		settings.Content.BodyHorror = toggleBool(settings.Content.BodyHorror, direction)
	case 5:
		settings.Content.PregnancyDetail = toggleBool(settings.Content.PregnancyDetail, direction)
	}
}

func adjustInterfaceField(settings *config.Settings, index, direction int) {
	switch index {
	case 0:
		settings.Interface.ShowTimestamps = toggleBool(settings.Interface.ShowTimestamps, direction)
	case 1:
		settings.Interface.CompactFeed = toggleBool(settings.Interface.CompactFeed, direction)
	case 2:
		settings.Interface.ShowPobleBadges = toggleBool(settings.Interface.ShowPobleBadges, direction)
	case 3:
		settings.Interface.PersistentNotices = toggleBool(settings.Interface.PersistentNotices, direction)
	case 4:
		settings.Interface.MapLabels = toggleBool(settings.Interface.MapLabels, direction)
	case 5:
		settings.Interface.ShowSaveThumbnails = toggleBool(settings.Interface.ShowSaveThumbnails, direction)
	}
}

func adjustAccessibilityField(settings *config.Settings, index, direction int) {
	switch index {
	case 0:
		settings.Accessibility.HighContrast = toggleBool(settings.Accessibility.HighContrast, direction)
	case 1:
		settings.Accessibility.ScreenReaderMode = toggleBool(settings.Accessibility.ScreenReaderMode, direction)
	case 2:
		settings.Accessibility.ReduceFlashing = toggleBool(settings.Accessibility.ReduceFlashing, direction)
	case 3:
		settings.Accessibility.DyslexiaFriendly = toggleBool(settings.Accessibility.DyslexiaFriendly, direction)
	case 4:
		settings.Accessibility.BiggerLineSpacing = toggleBool(settings.Accessibility.BiggerLineSpacing, direction)
	case 5:
		settings.Accessibility.HoldToConfirm = toggleBool(settings.Accessibility.HoldToConfirm, direction)
	}
}

func adjustControlsField(settings *config.Settings, index, direction int) {
	switch index {
	case 0:
		settings.Controls.VimNavigation = toggleBool(settings.Controls.VimNavigation, direction)
	case 1:
		settings.Controls.WASDMapPan = toggleBool(settings.Controls.WASDMapPan, direction)
	case 2:
		settings.Controls.InvertScroll = toggleBool(settings.Controls.InvertScroll, direction)
	case 3:
		settings.Controls.ConfirmOnQuit = toggleBool(settings.Controls.ConfirmOnQuit, direction)
	case 4:
		settings.Controls.QuickPause = toggleBool(settings.Controls.QuickPause, direction)
	}
}

func toggleBool(value bool, direction int) bool {
	if direction == 0 {
		return !value
	}
	if direction > 0 {
		return true
	}
	return false
}

func stepInt(current, direction, step, minValue, maxValue int) int {
	if direction == 0 {
		direction = 1
	}
	next := current + step*direction
	if next < minValue {
		next = minValue
	}
	if next > maxValue {
		next = maxValue
	}
	return next
}

func stepFromList(current int, values []int, direction int) int {
	if len(values) == 0 {
		return current
	}
	index := slices.Index(values, current)
	if index == -1 {
		index = 0
	}
	if direction == 0 {
		direction = 1
	}
	index += direction
	if index < 0 {
		index = 0
	}
	if index >= len(values) {
		index = len(values) - 1
	}
	return values[index]
}

func stepFromListFloat(current float64, values []float64, direction int) float64 {
	if len(values) == 0 {
		return current
	}
	index := slices.Index(values, current)
	if index == -1 {
		index = 0
	}
	if direction == 0 {
		direction = 1
	}
	index += direction
	if index < 0 {
		index = 0
	}
	if index >= len(values) {
		index = len(values) - 1
	}
	return values[index]
}

func cycleString(current string, values []string, direction int) string {
	if len(values) == 0 {
		return current
	}
	index := slices.Index(values, current)
	if index == -1 {
		index = 0
	}
	if direction == 0 {
		direction = 1
	}
	index += direction
	if index < 0 {
		index = 0
	}
	if index >= len(values) {
		index = len(values) - 1
	}
	return values[index]
}
