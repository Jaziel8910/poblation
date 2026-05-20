package views

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/user/poblation/internal/entities"
	"github.com/user/poblation/internal/events"
)

// EventsModel renders the real event feed as a first-class screen.
type EventsModel struct {
	state  AppStateSnapshot
	cursor int
}

func NewEventsModel() EventsModel {
	return EventsModel{}
}

func (m EventsModel) Init() tea.Cmd {
	return nil
}

func (m EventsModel) SyncAppState(snapshot AppStateSnapshot) tea.Model {
	m.state = snapshot
	m.cursor = clampInt(m.cursor, 0, maxInt(0, len(snapshot.EventFeed)-1))
	return m
}

func (m EventsModel) Resize(width, height int) tea.Model {
	m.state.Width = width
	m.state.Height = height
	return m
}

func (m EventsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch typed := msg.(type) {
	case tea.WindowSizeMsg:
		return m.Resize(typed.Width, typed.Height), nil
	case tea.KeyMsg:
		switch typed.String() {
		case "up", "k":
			m.cursor = maxInt(0, m.cursor-1)
		case "down", "j":
			m.cursor = minInt(maxInt(0, len(m.state.EventFeed)-1), m.cursor+1)
		}
	}
	return m, nil
}

func (m EventsModel) View() string {
	width := maxInt(42, m.state.Width-2)
	height := maxInt(12, m.state.Height-4)
	lines := []string{
		HeaderStyle.Render("EVENTOS"),
		MutedStyle.Render("Historial jugable del mundo. ↑/↓ cambia seleccion."),
		"",
	}
	if len(m.state.EventFeed) == 0 {
		lines = append(lines, MutedStyle.Render("Todavia no hay eventos. Avanza tiempo desde el mapa o la consola."))
		return panelStyle("accent").Width(width).Height(height).Padding(1, 2).Render(strings.Join(lines, "\n"))
	}

	limit := maxInt(1, minInt(len(m.state.EventFeed), height-12))
	for i := 0; i < limit; i++ {
		event := m.state.EventFeed[i]
		prefix := "  "
		style := BodyStyle
		if i == m.cursor {
			prefix = "> "
			style = AccentStyle
		}
		public := "privado"
		if event.IsPublic {
			public = "publico"
		}
		line := fmt.Sprintf("%s[%s] %s · %s · %d pobles · %d consecuencias",
			prefix, compactTime(event.Timestamp), event.Type, public, len(event.Participants), len(event.Consequences))
		lines = append(lines, style.Render(truncateRunes(line, width-6)))
		description := strings.TrimSpace(event.Description)
		if description != "" {
			lines = append(lines, MutedStyle.Render(truncateRunes("    "+description, width-8)))
		}
	}

	lines = append(lines, "")
	lines = append(lines, m.eventDetails(m.state.EventFeed[m.cursor], width))
	return panelStyle("accent").Width(width).Height(height).Padding(1, 2).Render(strings.Join(lines, "\n"))
}

func (m EventsModel) eventDetails(event events.GameEvent, width int) string {
	lines := []string{SubheaderStyle.Render("Evento seleccionado")}
	lines = append(lines, BodyStyle.Render(fmt.Sprintf("ID: %s", event.ID)))
	lines = append(lines, BodyStyle.Render(fmt.Sprintf("Tipo: %s · Tiempo: %s", event.Type, compactTime(event.Timestamp))))
	if len(event.Participants) > 0 {
		lines = append(lines, BodyStyle.Render("Participantes: "+strings.Join(event.Participants, ", ")))
	}
	if len(event.Consequences) > 0 {
		lines = append(lines, AccentStyle.Render(fmt.Sprintf("Consecuencias reales: %d", len(event.Consequences))))
		for i, consequence := range event.Consequences {
			if i >= 3 {
				break
			}
			lines = append(lines, MutedStyle.Render(truncateRunes(fmt.Sprintf("- %s -> %s %.0f delay %d",
				consequence.Type, consequence.TargetID, consequence.Value, consequence.Delay), width-8)))
		}
	}
	if len(event.ChildEvents) > 0 {
		lines = append(lines, MutedStyle.Render("Eventos hijos: "+strings.Join(event.ChildEvents, ", ")))
	}
	return strings.Join(lines, "\n")
}

func compactTime(t entities.GameTime) string {
	return fmt.Sprintf("%03d %02d:%02d", t.Day, t.Hour, t.Minute)
}
