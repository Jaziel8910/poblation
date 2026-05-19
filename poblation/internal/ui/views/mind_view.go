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

	if intent := m.state.LastIntents[poble.ID]; intent != "" {
		lines = append(lines, AccentStyle.Render("Ahora: "+humanIntent(intent)))
	}
	if reason := m.state.IntentReasons[poble.ID]; reason != "" {
		lines = append(lines, MutedStyle.Render("Por que: "+humanReason(reason, poble)))
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

	lines = append(lines, "")
	lines = append(lines, m.renderInnerLife(poble))

	if !layout.IsCompactHeight() {
		lines = append(lines, "")
		lines = append(lines, m.renderMindRelations(poble))
	}

	return strings.Join(lines, "\n")
}

func (m MindModel) renderInnerLife(poble entities.Poble) string {
	lines := []string{SubheaderStyle.Render("Vida interior")}
	if len(poble.Thoughts) > 0 {
		thought := poble.Thoughts[len(poble.Thoughts)-1]
		lines = append(lines, BodyStyle.Render("Pensamiento: "+shortMindText(thought.Text, 110)))
	} else {
		lines = append(lines, MutedStyle.Render("Pensamiento: todavia sin registro."))
	}
	if len(poble.Dreams) > 0 {
		dream := poble.Dreams[len(poble.Dreams)-1]
		lines = append(lines, BodyStyle.Render("Ultimo sueno: "+shortMindText(dream.Text, 110)))
	}
	if len(poble.DiaryEntries) > 0 {
		diary := poble.DiaryEntries[len(poble.DiaryEntries)-1]
		lines = append(lines, AccentStyle.Render("Diario: "+shortMindText(diary.Text, 110)))
	}
	if len(poble.Letters) > 0 {
		letter := poble.Letters[len(poble.Letters)-1]
		status := "guardada"
		if letter.IsSent {
			status = "enviada"
		}
		lines = append(lines, AccentStyle.Render(fmt.Sprintf("Carta %s: %s", status, shortMindText(letter.Text, 100))))
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

func shortMindText(text string, limit int) string {
	text = strings.Join(strings.Fields(text), " ")
	if limit <= 3 || len(text) <= limit {
		return text
	}
	return text[:limit-3] + "..."
}

func humanIntent(intent string) string {
	intent = strings.TrimPrefix(intent, "intent:")
	if strings.TrimSpace(intent) == "" || intent == "IDLE" {
		return "quieto, mirando demasiado"
	}
	parts := strings.Split(intent, " target:")
	action := strings.ReplaceAll(strings.ToLower(parts[0]), "_", " ")
	if len(parts) > 1 && strings.TrimSpace(parts[1]) != "" {
		return action + " con " + parts[1]
	}
	return action
}

func humanReason(reason string, poble entities.Poble) string {
	reason = strings.TrimSpace(reason)
	switch {
	case reason == "" || reason == "reason:idle":
		return "no hay impulso dominante"
	case reason == "reason:memory":
		return strongestMemoryReason(poble)
	case reason == "reason:need":
		return highestNeedReason(poble)
	case reason == "reason:reconciliation":
		return "un recuerdo viejo todavia pide arreglo"
	case reason == "reason:impulse":
		return "impulso raro del momento"
	case strings.HasPrefix(reason, "archetype:"):
		return "su arquetipo empuja: " + strings.TrimPrefix(reason, "archetype:")
	case strings.HasPrefix(reason, "emotion:"):
		return "emocion activa: " + strings.TrimPrefix(reason, "emotion:")
	case strings.HasPrefix(reason, "world:"):
		return "el mundo lo presiona: " + strings.TrimPrefix(reason, "world:")
	case strings.HasPrefix(reason, "relationship:"):
		return "la relacion pesa: " + strings.TrimPrefix(reason, "relationship:")
	default:
		return "decision interna"
	}
}

func strongestMemoryReason(poble entities.Poble) string {
	if len(poble.Memories) == 0 {
		return "un recuerdo sin nombre esta pesando"
	}
	best := poble.Memories[0]
	for _, memory := range poble.Memories[1:] {
		if memory.EmotionIntensity > best.EmotionIntensity {
			best = memory
		}
	}
	if strings.TrimSpace(best.Summary) == "" {
		return "recuerdo " + strings.ToLower(best.Type.String())
	}
	return shortMindText(best.Summary, 90)
}

func highestNeedReason(poble entities.Poble) string {
	type needScore struct {
		name  string
		value float32
	}
	needs := []needScore{
		{"hambre", poble.Needs.Hunger},
		{"sed", poble.Needs.Thirst},
		{"sueno", poble.Needs.Sleep},
		{"seguridad", poble.Needs.Safety},
		{"compania", poble.Needs.Belonging},
		{"estima", poble.Needs.Esteem},
		{"sexo", poble.Needs.Sex},
		{"poder", poble.Needs.Power},
		{"proposito", poble.Needs.Purpose},
	}
	best := needs[0]
	for _, need := range needs[1:] {
		if need.value > best.value {
			best = need
		}
	}
	return fmt.Sprintf("%s %.0f/100", best.name, best.value)
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
