package views

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/user/poblation/internal/world"
)

// SettlementModel renders the civilization layer: government, laws, tech and pressure.
type SettlementModel struct {
	state    AppStateSnapshot
	viewport viewport.Model
}

var (
	settlementTitleStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#F2C078")).
				Bold(true)
	settlementSectionStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#A7D8DE")).
				Bold(true).
				MarginTop(1)
	settlementMutedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#8A8A8A"))
	settlementGoodStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#9BE28F"))
	settlementWarnStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#F2C078"))
	settlementBadStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#F28F8F"))
)

// NewSettlementModel creates the settlement screen.
func NewSettlementModel() SettlementModel {
	return SettlementModel{viewport: viewport.New(72, 20)}
}

// Init satisfies tea.Model.
func (m SettlementModel) Init() tea.Cmd {
	return nil
}

// SyncAppState refreshes civilization data from the root app.
func (m SettlementModel) SyncAppState(snapshot AppStateSnapshot) tea.Model {
	m.state = snapshot
	m.resize()
	m.viewport.SetContent(m.content())
	return m
}

func (m SettlementModel) Resize(width, height int) tea.Model {
	m.state.Width = width
	m.state.Height = height
	m.resize()
	m.viewport.SetContent(m.content())
	return m
}

func (m SettlementModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch typed := msg.(type) {
	case tea.WindowSizeMsg:
		return m.Resize(typed.Width, typed.Height), nil
	case tea.KeyMsg:
		switch typed.String() {
		case "up", "k":
			m.viewport.LineUp(1)
		case "down", "j":
			m.viewport.LineDown(1)
		case "pgup":
			m.viewport.HalfViewUp()
		case "pgdown":
			m.viewport.HalfViewDown()
		}
	}
	return m, nil
}

func (m SettlementModel) View() string {
	m.viewport.SetContent(m.content())
	return m.viewport.View()
}

func (m *SettlementModel) resize() {
	width := maxInt(40, m.state.Width-4)
	height := maxInt(12, m.state.Height-2)
	m.viewport.Width = width
	m.viewport.Height = height
}

func (m SettlementModel) content() string {
	if m.state.World == nil {
		return settlementMutedStyle.Render("No hay mundo cargado.")
	}
	w := m.state.World
	lines := []string{
		settlementTitleStyle.Render("SETTLEMENT"),
		fmt.Sprintf("Era: %s  Dia: %d  Poblacion: %d", w.Era, w.Calendar.Day, w.GetPopulation()),
	}
	lines = append(lines, m.governmentLines(w)...)
	lines = append(lines, m.economyLines(w)...)
	lines = append(lines, m.familyLines(w)...)
	lines = append(lines, m.techLines(w)...)
	lines = append(lines, m.recentInstitutionLines()...)
	return strings.Join(lines, "\n")
}

func (m SettlementModel) governmentLines(w *world.World) []string {
	lines := []string{settlementSectionStyle.Render("Gobierno")}
	if w.Government == nil {
		lines = append(lines, settlementMutedStyle.Render("Todavia no hay gobierno formal. Se forma cuando la poblacion crece."))
		return lines
	}
	leader := "nadie"
	if w.Government.Leader != nil && *w.Government.Leader != "" {
		leader = *w.Government.Leader
		if poble := w.GetPoble(leader); poble != nil {
			leader = fmt.Sprintf("%s (%s)", poble.Name, poble.ID)
		}
	}
	lines = append(lines,
		fmt.Sprintf("Tipo: %s", w.Government.Type),
		fmt.Sprintf("Lider: %s", leader),
		fmt.Sprintf("Legitimidad: %s", scoreLabel(w.Government.Legitimacy)),
		fmt.Sprintf("Estabilidad: %s", scoreLabel(w.Government.Stability)),
	)
	lines = append(lines, "Leyes:")
	for _, law := range w.Government.Laws {
		flag := "latente"
		if law.IsEnforced {
			flag = "activa"
		}
		lines = append(lines, fmt.Sprintf("  - %s [%s]: %s", law.ID, flag, law.Description))
	}
	return lines
}

func (m SettlementModel) economyLines(w *world.World) []string {
	resources := map[string]int{}
	for _, island := range w.Islands {
		if island == nil || !island.IsDiscovered {
			continue
		}
		for resource, amount := range island.Resources {
			resources[string(resource)] += amount
		}
	}
	keys := make([]string, 0, len(resources))
	for key := range resources {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	lines := []string{settlementSectionStyle.Render("Economia y recursos")}
	if len(keys) == 0 {
		return append(lines, settlementMutedStyle.Render("Sin recursos visibles."))
	}
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s:%d", key, resources[key]))
	}
	lines = append(lines, strings.Join(parts, "  "))
	return lines
}

func (m SettlementModel) familyLines(w *world.World) []string {
	lines := []string{settlementSectionStyle.Render("Familias y generaciones")}
	living := w.GetAllPobles()
	if len(living) == 0 {
		return append(lines, settlementMutedStyle.Render("No queda nadie para heredar nada."))
	}

	children := 0
	orphans := 0
	pregnancies := 0
	founders := 0
	largestLineage := ""
	largestLineageCount := 0
	lineages := map[string]int{}

	for _, poble := range living {
		if poble == nil {
			continue
		}
		if len(poble.Parents) == 0 || (poble.Parents[0] == "" && poble.Parents[1] == "") {
			founders++
		}
		if len(poble.Children) > 0 {
			lineages[poble.ID] = len(poble.Children)
			if len(poble.Children) > largestLineageCount {
				largestLineage = poble.Name
				largestLineageCount = len(poble.Children)
			}
		}
		if poble.Age < 18 {
			children++
			if parentMissing(w, poble.Parents[0]) && parentMissing(w, poble.Parents[1]) {
				orphans++
			}
		}
		for _, condition := range poble.Health.Conditions {
			if condition.String() == "PREGNANT" {
				pregnancies++
				break
			}
		}
	}

	lines = append(lines,
		fmt.Sprintf("Fundadores vivos: %d", founders),
		fmt.Sprintf("Menores: %d  Orfandad: %s", children, pressureLabel(orphans, children)),
		fmt.Sprintf("Embarazos activos: %d", pregnancies),
		fmt.Sprintf("Linajes con hijos: %d", len(lineages)),
	)
	if largestLineage != "" {
		lines = append(lines, fmt.Sprintf("Linaje mas grande: %s (%d hijos directos)", largestLineage, largestLineageCount))
	}
	return lines
}

func (m SettlementModel) techLines(w *world.World) []string {
	unlocked := w.TechTree.GetUnlockedFeatures()
	lines := []string{settlementSectionStyle.Render("Tecnologia")}
	if len(unlocked) == 0 {
		return append(lines, settlementMutedStyle.Render("Aun no hay tecnologia desbloqueada."))
	}
	limit := minInt(len(unlocked), 10)
	lines = append(lines, strings.Join(unlocked[:limit], ", "))
	if len(unlocked) > limit {
		lines = append(lines, settlementMutedStyle.Render(fmt.Sprintf("+%d mas", len(unlocked)-limit)))
	}
	return lines
}

func parentMissing(w *world.World, parentID string) bool {
	if strings.TrimSpace(parentID) == "" {
		return true
	}
	parent := w.GetPoble(parentID)
	return parent == nil || !parent.IsAlive
}

func pressureLabel(value int, total int) string {
	if total == 0 || value == 0 {
		return settlementGoodStyle.Render("0")
	}
	text := fmt.Sprintf("%d/%d", value, total)
	if value*2 >= total {
		return settlementBadStyle.Render(text)
	}
	return settlementWarnStyle.Render(text)
}

func (m SettlementModel) recentInstitutionLines() []string {
	lines := []string{settlementSectionStyle.Render("Consecuencias recientes")}
	count := 0
	for i := len(m.state.EventFeed) - 1; i >= 0 && count < 8; i-- {
		event := m.state.EventFeed[i]
		if !strings.Contains(string(event.Type), "ELECTION") &&
			!strings.Contains(string(event.Type), "COUP") &&
			!strings.Contains(string(event.Type), "REVOLUTION") &&
			!strings.Contains(strings.ToLower(event.Description), "law") &&
			!strings.Contains(strings.ToLower(event.Description), "government") {
			continue
		}
		text := event.Description
		if strings.TrimSpace(text) == "" {
			text = string(event.Type)
		}
		lines = append(lines, fmt.Sprintf("- Dia %d: %s", event.Timestamp.Day, text))
		count++
	}
	if count == 0 {
		lines = append(lines, settlementMutedStyle.Render("Todavia no hay elecciones, crisis o leyes aplicadas en el feed."))
	}
	return lines
}

func scoreLabel(score int) string {
	text := fmt.Sprintf("%d/100", score)
	switch {
	case score >= 70:
		return settlementGoodStyle.Render(text)
	case score >= 40:
		return settlementWarnStyle.Render(text)
	default:
		return settlementBadStyle.Render(text)
	}
}
