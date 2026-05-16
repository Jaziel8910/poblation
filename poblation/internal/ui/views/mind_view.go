package views

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/user/poblation/internal/entities"
)

type mindPulseMsg struct{}

// MindModel renders the selected poble mental view.
type MindModel struct {
	state       AppStateSnapshot
	cursorFrame int
}

// NewMindModel creates the dedicated mental view.
func NewMindModel() MindModel {
	return MindModel{}
}

func (m MindModel) Init() tea.Cmd {
	return mindPulseCmd()
}

func (m MindModel) SyncAppState(snapshot AppStateSnapshot) tea.Model {
	m.state = snapshot
	return m
}

func (m MindModel) OnEnter() (tea.Model, tea.Cmd) {
	return m, mindPulseCmd()
}

func (m MindModel) Resize(width, height int) tea.Model {
	m.state.Width = width
	m.state.Height = height
	return m
}

func (m MindModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch typed := msg.(type) {
	case tea.WindowSizeMsg:
		return m.Resize(typed.Width, typed.Height), nil
	case mindPulseMsg:
		m.cursorFrame = (m.cursorFrame + 1) % 2
		return m, mindPulseCmd()
	default:
		return m, nil
	}
}

func (m MindModel) View() string {
	layout := LayoutManager{Width: m.state.Width, Height: m.state.Height}
	width := maxInt(28, m.state.Width-2)
	body := []string{
		HeaderStyle.Render("MENTE " + m.cursor()),
		MutedStyle.Render("Vista mental privada del poble seleccionado."),
		"",
	}

	if poble := selectedMindPoble(m.state); poble != nil {
		body = append(body, m.renderPobleMind(*poble, layout))
	} else {
		body = append(body, MutedStyle.Render("No hay un poble seleccionado todavia."))
		body = append(body, MutedStyle.Render("Entra al mapa, elige a alguien y vuelve con M."))
	}

	return panelStyle("accent").Width(width).Padding(1, 2).Render(strings.Join(body, "\n"))
}

func (m MindModel) cursor() string {
	if m.cursorFrame%2 == 0 {
		return "◉"
	}
	return "◎"
}

func (m MindModel) renderPobleMind(poble entities.Poble, layout LayoutManager) string {
	lines := []string{
		SubheaderStyle.Render(fmt.Sprintf("%s %s", moodEmoji(poble.CurrentMood), poble.Name)),
		BodyStyle.Render(fmt.Sprintf("Mood: %s", poble.CurrentMood.String())),
		BodyStyle.Render(fmt.Sprintf("Estabilidad mental: %d", poble.Mental.Stability)),
		BodyStyle.Render(fmt.Sprintf("Terapia: %d · Traumas: %d", poble.Mental.TherapyLevel, len(poble.Mental.Traumas))),
	}

	if len(poble.Mental.Conditions) > 0 {
		conditions := make([]string, 0, len(poble.Mental.Conditions))
		for _, condition := range poble.Mental.Conditions {
			conditions = append(conditions, string(condition))
		}
		lines = append(lines, WarningStyle.Render("Condiciones: "+strings.Join(conditions, ", ")))
	}

	if len(poble.Secrets) > 0 {
		lines = append(lines, AccentStyle.Render(fmt.Sprintf("Secretos cargados: %d", len(poble.Secrets))))
	}

	if !layout.IsCompactHeight() {
		lines = append(lines, "")
		lines = append(lines, m.renderMindRelations(poble))
	}

	return strings.Join(lines, "\n")
}

func (m MindModel) renderMindRelations(poble entities.Poble) string {
	if len(poble.Relationships) == 0 {
		return MutedStyle.Render("Sin relaciones relevantes visibles.")
	}

	lines := []string{SubheaderStyle.Render("Relaciones que tiran del pensamiento")}
	count := 0
	for _, rel := range poble.Relationships {
		if count >= 5 {
			break
		}
		style := lipgloss.NewStyle().Foreground(relationColor(rel.Type))
		lines = append(lines, style.Render(fmt.Sprintf("%s · afecto %.0f · confianza %.0f", rel.Type.String(), rel.Affection, rel.Trust)))
		count++
	}
	return strings.Join(lines, "\n")
}

func selectedMindPoble(state AppStateSnapshot) *entities.Poble {
	if state.World == nil {
		return nil
	}
	if strings.TrimSpace(state.SelectedPobleID) != "" {
		if poble := state.World.GetPoble(state.SelectedPobleID); poble != nil {
			return poble
		}
	}
	all := state.World.GetAllPobles()
	if len(all) == 0 {
		return nil
	}
	return all[0]
}

func mindPulseCmd() tea.Cmd {
	return tea.Tick(450*time.Millisecond, func(time.Time) tea.Msg {
		return mindPulseMsg{}
	})
}
