package views

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/user/poblation/internal/ai"
	"github.com/user/poblation/internal/entities"
	"github.com/user/poblation/internal/minigames"
)

type fightRevealMsg struct{}

// ExitFightMsg asks the root app to return to the map after the fight view closes.
type ExitFightMsg struct{}

// FightModel renders the escalating fight minigame.
type FightModel struct {
	Fight             minigames.FightState
	state             AppStateSnapshot
	viewport          viewport.Model
	renderedLines     []string
	currentIndex      int
	showOutcome       bool
	resultApplied     bool
	locationLabel     string
	relationshipLabel string
}

var (
	fightHeaderStyle = lipgloss.NewStyle().
				Foreground(accentColor).
				Bold(true)

	fightMutedStyle = lipgloss.NewStyle().
			Foreground(mutedColor)

	fightPhaseStyle = lipgloss.NewStyle().
			Foreground(warningColor).
			Bold(true)

	fightDangerStyle = lipgloss.NewStyle().
				Foreground(dangerColor)

	fightOutcomeStyle = lipgloss.NewStyle().
				Foreground(successColor).
				Bold(true)
)

// NewFightModel creates the fight view.
func NewFightModel() FightModel {
	return FightModel{
		viewport: viewport.New(48, 14),
	}
}

// Init satisfies tea.Model.
func (m FightModel) Init() tea.Cmd {
	return nil
}

// SyncAppState refreshes world data and viewport size.
func (m FightModel) SyncAppState(snapshot AppStateSnapshot) tea.Model {
	m.state = snapshot
	m.resizeViewport()
	return m
}

func (m FightModel) Resize(width, height int) tea.Model {
	m.state.Width = width
	m.state.Height = height
	m.resizeViewport()
	return m
}

// OnEnter builds a fresh fight scene from the current world snapshot.
func (m FightModel) OnEnter() (tea.Model, tea.Cmd) {
	m = m.buildFight()
	if len(m.Fight.Beats) == 0 {
		m.showOutcome = true
		return m, nil
	}
	return m, m.nextRevealCmd()
}

// Update handles line reveal, scrolling, and interventions.
func (m FightModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch typed := msg.(type) {
	case tea.WindowSizeMsg:
		return m.Resize(typed.Width, typed.Height), nil
	case fightRevealMsg:
		return m.revealNextBeat()
	case tea.KeyMsg:
		return m.handleKey(typed)
	default:
		return m, nil
	}
}

// View renders the fight scene.
func (m FightModel) View() string {
	width := maxInt(28, m.state.Width-2)
	frame := m.frameStyle().Width(width).Height(maxInt(10, m.state.Height-2)).Render(m.viewport.View())
	return lipgloss.JoinVertical(lipgloss.Left, m.renderHeader(), frame, m.renderFooter())
}

func (m FightModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up":
		m.viewport.ScrollUp(1)
		m.syncViewport()
		return m, nil
	case "down":
		m.viewport.ScrollDown(1)
		m.syncViewport()
		return m, nil
	case "x":
		m.applyFightResultOnce()
		return m, func() tea.Msg { return ExitFightMsg{} }
	case "d":
		if m.showOutcome || !m.Fight.CanDeescalate {
			return m, nil
		}
		minigames.DeescalateFight(&m.Fight)
		m.renderedLines = append(m.renderedLines, fightMutedStyle.Render("Intervencion: alguien consigue romper el impulso antes del peor momento."))
		m.showOutcome = true
		m.applyFightResultOnce()
		m.syncViewport()
		return m, nil
	case "enter":
		if !m.showOutcome {
			return m, nil
		}
		return m, func() tea.Msg { return ExitFightMsg{} }
	default:
		return m, nil
	}
}

func (m FightModel) revealNextBeat() (tea.Model, tea.Cmd) {
	if m.currentIndex >= len(m.Fight.Beats) {
		m.showOutcome = true
		m.applyFightResultOnce()
		m.syncViewport()
		return m, nil
	}

	beat := m.Fight.Beats[m.currentIndex]
	m.Fight.Phase = beat.Phase
	m.renderedLines = append(m.renderedLines, m.renderBeat(beat))
	m.currentIndex++
	if m.currentIndex >= len(m.Fight.Beats) {
		m.showOutcome = true
		m.applyFightResultOnce()
		m.syncViewport()
		return m, nil
	}
	m.syncViewport()
	return m, m.nextRevealCmd()
}

func (m FightModel) renderBeat(beat minigames.FightBeat) string {
	tag := fightPhaseStyle.Render("[" + string(beat.Phase) + "]")
	text := normalizeFightText(beat.Text)
	if beat.Danger {
		return tag + "\n" + fightDangerStyle.Render(text)
	}
	return tag + "\n" + text
}

func (m FightModel) buildFight() FightModel {
	attacker, defender, trigger := selectFightParticipants(m.state)
	m.Fight = minigames.NewFightState(attacker, defender, m.state.World, trigger)
	m.Fight.Beats = minigames.BuildFightBeats(m.Fight, m.state.TemplateEngine, m.state.World)
	m.locationLabel = dialogueLocationLabel(m.state, []*entities.Poble{attacker, defender})
	m.relationshipLabel = dialogueRelationshipLabel([]*entities.Poble{attacker, defender})
	m.renderedLines = nil
	m.currentIndex = 0
	m.showOutcome = false
	m.resultApplied = false
	m.syncViewport()
	return m
}

func (m *FightModel) resizeViewport() {
	m.viewport.Width = maxInt(28, m.state.Width-8)
	m.viewport.Height = maxInt(10, m.state.Height-10)
	m.syncViewport()
}

func (m *FightModel) syncViewport() {
	content := strings.Join(m.renderedLines, "\n\n")
	if m.showOutcome {
		content = strings.Join([]string{content, "", m.outcomeContent()}, "\n")
	}
	if strings.TrimSpace(content) == "" {
		content = fightMutedStyle.Render("Todavia no hay una pelea aqui.")
	}
	m.viewport.SetContent(strings.TrimSpace(content))
	m.viewport.GotoBottom()
}

func (m FightModel) renderHeader() string {
	names := participantNames([]*entities.Poble{m.Fight.Attacker, m.Fight.Defender})
	if names == "" {
		names = "Pelea"
	}
	timeLabel := "00:00"
	if m.state.World != nil {
		timeLabel = fmt.Sprintf("%02d:%02d", m.state.World.Calendar.Hour, m.state.World.Calendar.Minute)
	}
	header := fightHeaderStyle.Render(fmt.Sprintf("%s · %s · %s", names, m.locationLabel, timeLabel))
	if strings.TrimSpace(m.relationshipLabel) == "" {
		return header
	}
	return lipgloss.JoinVertical(lipgloss.Left, header, fightMutedStyle.Render(m.relationshipLabel))
}

func (m FightModel) renderFooter() string {
	if m.showOutcome {
		return fightMutedStyle.Render("ENTER vuelve al mapa · X salir")
	}
	if m.Fight.CanDeescalate {
		return fightMutedStyle.Render("D intentar deescalar · X salir · flechas scroll")
	}
	return fightMutedStyle.Render("X salir · flechas scroll")
}

func (m FightModel) outcomeContent() string {
	if m.Fight.Outcome == nil {
		return fightOutcomeStyle.Render("Consecuencias pendientes")
	}
	lines := []string{
		fightOutcomeStyle.Render("Consecuencias"),
		"",
		fmt.Sprintf("Resultado: %s", m.Fight.Outcome.Type),
		fmt.Sprintf("Intensidad final: %d/10", m.Fight.Intensity),
		fmt.Sprintf("Hubo arma: %t", m.Fight.Weapon != nil),
		fmt.Sprintf("Testigos: %d", len(m.Fight.WitnessIDs)),
	}
	if m.Fight.Outcome.Type == minigames.FightOutcomeFatal {
		lines = append(lines, "Muerte: si")
	}
	if delta := fightDeltaForPair(m.Fight); delta != 0 {
		lines = append(lines, fmt.Sprintf("Relacion: %+d", delta))
	}
	if strings.TrimSpace(m.Fight.Outcome.EventSummary) != "" {
		lines = append(lines, "", fightMutedStyle.Render(m.Fight.Outcome.EventSummary))
	}
	return strings.Join(lines, "\n")
}

func (m *FightModel) applyFightResultOnce() {
	if m.resultApplied {
		return
	}
	minigames.ApplyFightResult(m.Fight, m.state.World)
	m.resultApplied = true
}

func (m FightModel) nextRevealCmd() tea.Cmd {
	if m.currentIndex >= len(m.Fight.Beats) {
		return nil
	}
	delay := m.Fight.Beats[m.currentIndex].Delay
	if delay <= 0 {
		delay = 850
	}
	return tea.Tick(time.Duration(delay)*time.Millisecond, func(time.Time) tea.Msg {
		return fightRevealMsg{}
	})
}

func (m FightModel) frameStyle() lipgloss.Style {
	border := borderColor
	if m.Fight.Phase == minigames.FightPhasePhysical {
		border = dangerColor
	}
	return lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(border).
		Background(surfaceColor).
		Foreground(primaryColor).
		Padding(0, 1)
}

func selectFightParticipants(snapshot AppStateSnapshot) (*entities.Poble, *entities.Poble, string) {
	if first, second, trigger := activeFightPair(snapshot); first != nil && second != nil {
		return first, second, trigger
	}
	if snapshot.World == nil {
		return nil, nil, ""
	}
	anchor := snapshot.World.GetPoble(snapshot.SelectedPobleID)
	if anchor == nil {
		all := snapshot.World.GetAllPobles()
		if len(all) > 0 {
			anchor = all[0]
		}
	}
	if anchor == nil {
		return nil, nil, ""
	}

	var rival *entities.Poble
	var rivalScore float32 = -999
	for _, candidate := range snapshot.World.GetAllPobles() {
		if candidate == nil || candidate.ID == anchor.ID {
			continue
		}
		rel := relationshipBetween(anchor, candidate.ID)
		score := rel.Resentment + candidate.Personality.Cruelty*0.2 + candidate.Personality.Neuroticism*0.1
		if score > rivalScore {
			rivalScore = score
			rival = candidate
		}
	}
	return anchor, rival, fallbackFightTriggerText(anchor, rival)
}

func activeFightPair(snapshot AppStateSnapshot) (*entities.Poble, *entities.Poble, string) {
	if snapshot.World == nil {
		return nil, nil, ""
	}
	for _, event := range snapshot.World.ActiveEvents {
		if event.Type != ai.GameEventConflict {
			continue
		}
		first := snapshot.World.GetPoble(event.PrimaryActor)
		second := snapshot.World.GetPoble(event.TargetID)
		if first == nil && len(event.Participants) > 0 {
			first = snapshot.World.GetPoble(event.Participants[0])
		}
		if second == nil && len(event.Participants) > 1 {
			second = snapshot.World.GetPoble(event.Participants[1])
		}
		if first != nil && second != nil {
			return first, second, event.Description
		}
	}
	return nil, nil, ""
}

func fallbackFightTriggerText(attacker, defender *entities.Poble) string {
	if attacker == nil || defender == nil {
		return "una discusion rota"
	}
	relationship := relationshipBetween(attacker, defender.ID)
	switch {
	case relationship.Resentment >= 70:
		return "algo viejo que sigue sangrando"
	case relationship.Attraction >= 50 && relationship.Resentment >= 40:
		return "celos y orgullo mal mezclados"
	default:
		return "una discusion que venia pidiendo cuerpo"
	}
}

func normalizeFightText(text string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.TrimSpace(text)
	if text == "" {
		return "..."
	}
	return text
}

func fightDeltaForPair(state minigames.FightState) int {
	if state.Attacker == nil || state.Defender == nil {
		return 0
	}
	return state.RelationshipDelta[state.Attacker.ID+":"+state.Defender.ID]
}
