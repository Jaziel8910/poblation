package views

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	gameengine "github.com/user/poblation/internal/engine"
)

// RestartCivilizationMsg asks the app shell to start a fresh world.
type RestartCivilizationMsg struct{}
type endingRevealMsg struct{}

// EndingModel renders the final chapters and statistics.
type EndingModel struct {
	state     AppStateSnapshot
	ending    *gameengine.Ending
	viewport  viewport.Model
	step      int
	charIndex int
}

var (
	endingBackgroundStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("#080808")).
				Foreground(lipgloss.Color("#EDE9DC")).
				Padding(1, 2)

	endingTitleStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#F2C078")).
				Bold(true).
				MarginBottom(1)

	endingChapterStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#EDE9DC")).
				Width(78)

	endingStatsStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#A7D8DE")).
				Border(lipgloss.NormalBorder(), true).
				BorderForeground(lipgloss.Color("#3A6368")).
				Padding(1, 2)

	endingLastWordsStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#F28F8F")).
				Bold(true).
				MarginTop(1)
)

// NewEndingModel creates the special ending screen.
func NewEndingModel() EndingModel {
	return EndingModel{
		viewport: viewport.New(72, 18),
		step:     0,
	}
}

// Init satisfies tea.Model.
func (m EndingModel) Init() tea.Cmd {
	return nil
}

// SyncAppState refreshes world data and detected ending.
func (m EndingModel) SyncAppState(snapshot AppStateSnapshot) tea.Model {
	m.state = snapshot
	if snapshot.Ending != nil {
		m.ending = snapshot.Ending
	} else if snapshot.World != nil {
		m.ending = gameengine.CheckEndingConditions(snapshot.World)
	}
	m.resize()
	m.viewport.SetContent(m.content())
	return m
}

func (m EndingModel) Resize(width, height int) tea.Model {
	m.state.Width = width
	m.state.Height = height
	m.resize()
	m.viewport.SetContent(m.content())
	return m
}

// OnEnter starts chapter reveal from the beginning.
func (m EndingModel) OnEnter() (tea.Model, tea.Cmd) {
	m.step = 0
	m.charIndex = 0
	m.viewport.GotoTop()
	m.viewport.SetContent(m.content())
	if m.ending == nil || len(m.ending.NarrativeChapters) == 0 {
		return m, nil
	}
	return m, endingRevealCmd()
}

// Update reveals chapters one by one, then restarts after final stats.
func (m EndingModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch typed := msg.(type) {
	case tea.WindowSizeMsg:
		return m.Resize(typed.Width, typed.Height), nil
	case endingRevealMsg:
		return m.revealNextChar()
	case tea.KeyMsg:
		if m.ending != nil && m.step >= len(m.ending.NarrativeChapters) {
			return m, func() tea.Msg { return RestartCivilizationMsg{} }
		}
		chapterRunes := []rune(m.currentChapter())
		if len(chapterRunes) > 0 && m.charIndex < len(chapterRunes) {
			m.charIndex = len(chapterRunes)
			m.viewport.SetContent(m.content())
			return m, nil
		}
		m.step++
		m.charIndex = 0
		if m.step < len(m.ending.NarrativeChapters) {
			m.viewport.SetContent(m.content())
			return m, endingRevealCmd()
		}
	}
	m.viewport.SetContent(m.content())
	return m, nil
}

// View renders the ending as a full-screen final page.
func (m EndingModel) View() string {
	width := maxInt(32, m.state.Width-4)
	height := maxInt(12, m.state.Height-2)
	body := endingBackgroundStyle.Width(width).Height(height).Render(m.viewport.View())
	return body
}

func (m *EndingModel) resize() {
	m.viewport.Width = maxInt(28, m.state.Width-8)
	m.viewport.Height = maxInt(10, m.state.Height-6)
}

func (m EndingModel) content() string {
	if m.ending == nil {
		return endingTitleStyle.Render("Finales") + "\n" +
			"Todavia no hay un final ganado.\n\n" +
			"Presiona cualquier tecla para volver cuando la civilizacion ya haya hecho su desastre."
	}

	lines := []string{endingTitleStyle.Render(m.ending.Title)}
	visible := minInt(m.step, len(m.ending.NarrativeChapters))
	for i := 0; i < visible; i++ {
		chapter := fmt.Sprintf("CAPITULO %d\n%s", i+1, m.ending.NarrativeChapters[i])
		lines = append(lines, endingChapterStyle.Render(chapter))
	}
	if m.step < len(m.ending.NarrativeChapters) {
		current := m.partialChapter()
		if strings.TrimSpace(current) != "" {
			lines = append(lines, endingChapterStyle.Render(fmt.Sprintf("CAPITULO %d\n%s", m.step+1, current)))
		}
		lines = append(lines, MutedStyle.Render("ENTER acelera · cualquier tecla continua"))
		return strings.Join(lines, "\n\n")
	}
	if m.step >= len(m.ending.NarrativeChapters) {
		lines = append(lines, m.renderStats())
		lines = append(lines, endingLastWordsStyle.Render(m.ending.LastWords))
		lines = append(lines, "Presiona cualquier tecla para iniciar una nueva civilizacion")
	}
	return strings.Join(lines, "\n\n")
}

func (m EndingModel) revealNextChar() (tea.Model, tea.Cmd) {
	chapter := []rune(m.currentChapter())
	if len(chapter) == 0 {
		return m, nil
	}
	if m.charIndex < len(chapter) {
		m.charIndex++
		m.viewport.SetContent(m.content())
		return m, endingRevealCmd()
	}
	return m, nil
}

func (m EndingModel) currentChapter() string {
	if m.ending == nil || m.step < 0 || m.step >= len(m.ending.NarrativeChapters) {
		return ""
	}
	return m.ending.NarrativeChapters[m.step]
}

func (m EndingModel) partialChapter() string {
	chapter := []rune(m.currentChapter())
	if len(chapter) == 0 {
		return ""
	}
	limit := minInt(m.charIndex, len(chapter))
	return string(chapter[:limit])
}

func endingRevealCmd() tea.Cmd {
	return tea.Tick(20*time.Millisecond, func(time.Time) tea.Msg {
		return endingRevealMsg{}
	})
}

func (m EndingModel) renderStats() string {
	stats := m.ending.Statistics
	rows := []string{
		fmt.Sprintf("Dias totales: %d", stats.TotalDays),
		fmt.Sprintf("Pobles historicos: %d", stats.TotalPobles),
		fmt.Sprintf("Muertes: %d | Nacimientos: %d | Guerras: %d", stats.TotalDeaths, stats.TotalBirths, stats.TotalWars),
		fmt.Sprintf("Affairs: %d | Secretos revelados: %d | Rumores: %d", stats.TotalAffairs, stats.TotalSecretsRevealed, stats.TotalRumours),
		fmt.Sprintf("Evento mas dramatico: %s", fallbackViewString(stats.MostDramaticEvent, "ninguno")),
		fmt.Sprintf("Mas longevo: %s (%d)", fallbackViewString(stats.LongestLivedPople, "nadie"), stats.LongestLivedAge),
		fmt.Sprintf("Mas amado: %s | Mas odiado: %s", fallbackViewString(stats.MostLovedPople, "nadie"), fallbackViewString(stats.MostHatedPople, "nadie")),
		fmt.Sprintf("Mas rico: %s", fallbackViewString(stats.RicherPople, "nadie")),
		fmt.Sprintf("Primera pareja: %s", endingPairLabel(stats.FirstCoupleNames)),
		fmt.Sprintf("Rumor mas mutado: %s", fallbackViewString(stats.RumourMostMutated, "ninguno")),
	}
	return endingStatsStyle.Render(strings.Join(rows, "\n"))
}

func endingPairLabel(pair [2]string) string {
	if pair[0] == "" && pair[1] == "" {
		return "nadie"
	}
	if pair[1] == "" {
		return pair[0]
	}
	return pair[0] + " y " + pair[1]
}

func fallbackViewString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
