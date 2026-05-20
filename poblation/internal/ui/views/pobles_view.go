package views

import (
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/user/poblation/internal/entities"
)

// PoblesModel renders a playable roster instead of a placeholder screen.
type PoblesModel struct {
	state  AppStateSnapshot
	cursor int
}

func NewPoblesModel() PoblesModel {
	return PoblesModel{}
}

func (m PoblesModel) Init() tea.Cmd {
	return nil
}

func (m PoblesModel) SyncAppState(snapshot AppStateSnapshot) tea.Model {
	m.state = snapshot
	m.cursor = clampInt(m.cursor, 0, maxInt(0, len(m.pobles())-1))
	return m
}

func (m PoblesModel) Resize(width, height int) tea.Model {
	m.state.Width = width
	m.state.Height = height
	return m
}

func (m PoblesModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch typed := msg.(type) {
	case tea.WindowSizeMsg:
		return m.Resize(typed.Width, typed.Height), nil
	case tea.KeyMsg:
		switch typed.String() {
		case "up", "k":
			m.cursor = maxInt(0, m.cursor-1)
		case "down", "j":
			m.cursor = minInt(maxInt(0, len(m.pobles())-1), m.cursor+1)
		case "enter":
			pobles := m.pobles()
			if len(pobles) > 0 {
				return m, func() tea.Msg { return OpenPobleDetailMsg{PobleID: pobles[m.cursor].ID} }
			}
		}
	}
	return m, nil
}

func (m PoblesModel) View() string {
	width := maxInt(42, m.state.Width-2)
	lines := []string{
		HeaderStyle.Render("POBLES"),
		MutedStyle.Render("Gente viva del pueblo. Enter abre detalle, ↑/↓ cambia seleccion."),
		"",
	}
	pobles := m.pobles()
	if len(pobles) == 0 {
		lines = append(lines, MutedStyle.Render("No hay Pobles todavia."))
		return panelStyle("accent").Width(width).Padding(1, 2).Render(strings.Join(lines, "\n"))
	}

	for i, poble := range pobles {
		prefix := "  "
		style := BodyStyle
		if i == m.cursor {
			prefix = "> "
			style = AccentStyle
		}
		alive := "vivo"
		if !poble.IsAlive {
			alive = "muerto"
			style = MutedStyle
		}
		intent := humanIntent(m.state.LastIntents[poble.ID])
		if intent == "quieto, mirando demasiado" {
			intent = "sin accion fuerte"
		}
		relations := len(poble.Relationships)
		memories := len(poble.Memories)
		secrets := len(poble.Secrets)
		line := fmt.Sprintf("%s%s · %d · %s · %s · mood %s · rel %d · mem %d · secretos %d",
			prefix, poble.Name, poble.Age, strings.ToLower(poble.Archetype.String()), alive,
			strings.ToLower(poble.CurrentMood.String()), relations, memories, secrets)
		lines = append(lines, style.Render(truncateRunes(line, width-6)))
		lines = append(lines, MutedStyle.Render(truncateRunes("    ahora: "+intent, width-8)))
	}

	lines = append(lines, "")
	lines = append(lines, m.selectedDetails(pobles[m.cursor], width))
	return panelStyle("accent").Width(width).Padding(1, 2).Render(strings.Join(lines, "\n"))
}

func (m PoblesModel) pobles() []entities.Poble {
	if m.state.World == nil {
		return nil
	}
	raw := m.state.World.GetAllPobles()
	pobles := make([]entities.Poble, 0, len(raw))
	for _, poble := range raw {
		if poble != nil {
			pobles = append(pobles, *poble)
		}
	}
	sort.SliceStable(pobles, func(i, j int) bool {
		if pobles[i].IsAlive != pobles[j].IsAlive {
			return pobles[i].IsAlive
		}
		return strings.ToLower(pobles[i].Name) < strings.ToLower(pobles[j].Name)
	})
	return pobles
}

func (m PoblesModel) selectedDetails(poble entities.Poble, width int) string {
	lines := []string{SubheaderStyle.Render("Detalle rapido")}
	if reason := m.state.IntentReasons[poble.ID]; reason != "" {
		lines = append(lines, BodyStyle.Render(truncateRunes("Por que actua asi: "+humanReason(reason, poble), width-6)))
	}
	if len(poble.Thoughts) > 0 {
		thought := poble.Thoughts[len(poble.Thoughts)-1]
		lines = append(lines, BodyStyle.Render(truncateRunes("Pensamiento: "+thought.Text, width-6)))
	}
	if len(poble.Memories) > 0 {
		best := poble.Memories[0]
		for _, memory := range poble.Memories[1:] {
			if memory.EmotionIntensity > best.EmotionIntensity {
				best = memory
			}
		}
		lines = append(lines, AccentStyle.Render(truncateRunes("Recuerdo que pesa: "+best.Summary, width-6)))
	}
	if len(lines) == 1 {
		lines = append(lines, MutedStyle.Render("Todavia no hay pensamiento o recuerdo fuerte visible."))
	}
	return strings.Join(lines, "\n")
}
