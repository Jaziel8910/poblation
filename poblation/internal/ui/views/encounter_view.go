package views

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/user/poblation/internal/ai"
	"github.com/user/poblation/internal/config"
	"github.com/user/poblation/internal/entities"
	"github.com/user/poblation/internal/minigames"
	"github.com/user/poblation/internal/templates"
	"github.com/user/poblation/internal/world"
)

type encounterRevealMsg struct{}

// ExitEncounterMsg asks the root app to return to the main map.
type ExitEncounterMsg struct{}

// EncounterModel renders the private sexual encounter minigame.
type EncounterModel struct {
	Encounter        minigames.EncounterState
	state            AppStateSnapshot
	viewport         viewport.Model
	script           []minigames.EncounterChoice
	renderedLines    []string
	currentIndex     int
	waitingChoice    bool
	showOutcome      bool
	resultApplied    bool
	locationLabel    string
	relationshipInfo string
}

var (
	encounterFrameStyle = lipgloss.NewStyle().
				Border(lipgloss.DoubleBorder()).
				BorderForeground(accentColor).
				Background(surfaceColor).
				Foreground(primaryColor).
				Padding(0, 1)

	encounterHeaderStyle = lipgloss.NewStyle().
				Foreground(accentColor).
				Bold(true)

	encounterPhaseStyle = lipgloss.NewStyle().
				Foreground(secondaryColor).
				Bold(true)

	encounterMutedStyle = lipgloss.NewStyle().
				Foreground(mutedColor)

	encounterOptionStyle = lipgloss.NewStyle().
				Foreground(warningColor)

	encounterOutcomeStyle = lipgloss.NewStyle().
				Foreground(successColor).
				Bold(true)
)

// NewEncounterModel creates the minigame view.
func NewEncounterModel() EncounterModel {
	return EncounterModel{
		viewport: viewport.New(48, 14),
	}
}

// Init satisfies tea.Model.
func (m EncounterModel) Init() tea.Cmd {
	return nil
}

// SyncAppState refreshes shared world/template data.
func (m EncounterModel) SyncAppState(snapshot AppStateSnapshot) tea.Model {
	m.state = snapshot
	m.resizeViewport()
	return m
}

func (m EncounterModel) Resize(width, height int) tea.Model {
	m.state.Width = width
	m.state.Height = height
	m.resizeViewport()
	return m
}

// OnEnter builds a fresh private encounter script.
func (m EncounterModel) OnEnter() (tea.Model, tea.Cmd) {
	m = m.buildEncounter()
	if len(m.script) == 0 {
		m.showOutcome = true
		return m, nil
	}
	return m, m.nextRevealCmd()
}

// Update handles reveal timing, choices, and exit flow.
func (m EncounterModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch typed := msg.(type) {
	case tea.WindowSizeMsg:
		return m.Resize(typed.Width, typed.Height), nil
	case encounterRevealMsg:
		return m.revealCurrentChoice()
	case tea.KeyMsg:
		return m.handleKey(typed)
	default:
		return m, nil
	}
}

// View renders the encounter log and current options/outcome.
func (m EncounterModel) View() string {
	width := maxInt(28, m.state.Width-2)
	header := m.renderHeader()
	body := encounterFrameStyle.Width(width).Height(maxInt(10, m.state.Height-2)).Render(m.viewport.View())
	footer := m.renderFooter()
	return lipgloss.JoinVertical(lipgloss.Left, header, body, footer)
}

func (m EncounterModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.showOutcome {
		switch msg.String() {
		case "enter", "x":
			return m, func() tea.Msg {
				return ExitEncounterMsg{}
			}
		case "up":
			m.viewport.ScrollUp(1)
		case "down":
			m.viewport.ScrollDown(1)
		}
		return m, nil
	}

	switch msg.String() {
	case "x":
		return m, func() tea.Msg {
			return ExitEncounterMsg{}
		}
	case "up":
		m.viewport.ScrollUp(1)
		m.syncViewport()
		return m, nil
	case "down":
		m.viewport.ScrollDown(1)
		m.syncViewport()
		return m, nil
	case "1", "2", "3":
		if !m.waitingChoice {
			return m, nil
		}
		index := int(msg.String()[0] - '1')
		return m.resolveChoice(index)
	case "enter":
		if !m.waitingChoice {
			return m, nil
		}
		return m.resolveChoice(-1)
	default:
		return m, nil
	}
}

func (m EncounterModel) revealCurrentChoice() (tea.Model, tea.Cmd) {
	if m.currentIndex >= len(m.script) {
		return m.finishEncounter(), nil
	}
	current := m.script[m.currentIndex]
	if m.phaseChanged(current.Phase) {
		m.renderedLines = append(m.renderedLines, encounterPhaseStyle.Render("["+string(current.Phase)+"]"))
	}
	m.renderedLines = append(m.renderedLines, wrapEncounterText(current.Text))
	m.waitingChoice = true
	m.syncViewport()
	return m, nil
}

func (m EncounterModel) resolveChoice(playerIndex int) (tea.Model, tea.Cmd) {
	if m.currentIndex >= len(m.script) {
		return m.finishEncounter(), nil
	}
	current := m.script[m.currentIndex]
	var chosen *minigames.ChoiceOption
	if playerIndex >= 0 && playerIndex < len(current.OptionsForPoble1) {
		selected := current.OptionsForPoble1[playerIndex]
		chosen = &selected
	}
	first, second := minigames.ResolveChoice(&m.Encounter, current, chosen)
	m.renderedLines = append(m.renderedLines, encounterMutedStyle.Render(m.choiceSummary(first, second)))
	m.currentIndex++
	m.waitingChoice = false
	m.syncViewport()
	if m.currentIndex >= len(m.script) {
		return m.finishEncounter(), nil
	}
	return m, m.nextRevealCmd()
}

func (m EncounterModel) finishEncounter() EncounterModel {
	m.showOutcome = true
	m.waitingChoice = false
	if m.Encounter.Context.A != nil && m.Encounter.Context.B != nil {
		m.Encounter.PreferenceMatch = minigames.ResolvePreferenceMatch(m.Encounter.Context.A, m.Encounter.Context.B)
		m.Encounter.Aftermath = minigames.GenerateAftermath(m.Encounter.Context, m.Encounter.PreferenceMatch)
	}
	if !m.resultApplied {
		minigames.ApplyEncounterResult(m.Encounter, m.state.World)
		m.resultApplied = true
	}
	m.syncViewport()
	return m
}

func (m EncounterModel) buildEncounter() EncounterModel {
	participants := selectEncounterParticipants(m.state)
	m.Encounter = minigames.NewEncounterState(participants, worldTimeOrZero(m.state))
	m.locationLabel = dialogueLocationLabel(m.state, []*entities.Poble{participants[0], participants[1]})
	m.relationshipInfo = dialogueRelationshipLabel([]*entities.Poble{participants[0], participants[1]})
	m.script = buildEncounterScript(m.state, m.Encounter)
	m.renderedLines = []string{}
	m.currentIndex = 0
	m.waitingChoice = false
	m.showOutcome = false
	m.resultApplied = false
	m.syncViewport()
	return m
}

func (m *EncounterModel) resizeViewport() {
	m.viewport.Width = maxInt(28, m.state.Width-8)
	m.viewport.Height = maxInt(10, m.state.Height-10)
	m.syncViewport()
}

func (m *EncounterModel) syncViewport() {
	var content string
	if m.showOutcome {
		content = m.outcomeContent()
	} else {
		lines := append([]string{}, m.renderedLines...)
		if m.waitingChoice && m.currentIndex < len(m.script) {
			lines = append(lines, "")
			lines = append(lines, m.currentOptionsText()...)
		}
		if len(lines) == 0 {
			lines = append(lines, encounterMutedStyle.Render("Nadie se atreve todavia."))
		}
		content = strings.Join(lines, "\n")
	}
	m.viewport.SetContent(content)
	m.viewport.GotoBottom()
}

func (m EncounterModel) renderHeader() string {
	names := participantNames([]*entities.Poble{m.Encounter.Participants[0], m.Encounter.Participants[1]})
	if names == "" {
		names = "Encuentro privado"
	}
	head := encounterHeaderStyle.Render(names + " · " + m.locationLabel)
	if strings.TrimSpace(m.relationshipInfo) == "" {
		return head
	}
	return lipgloss.JoinVertical(lipgloss.Left, head, encounterMutedStyle.Render(m.relationshipInfo))
}

func (m EncounterModel) renderFooter() string {
	if m.showOutcome {
		return encounterMutedStyle.Render("ENTER vuelve al mapa · sin detalles en historial publico")
	}
	lines := []string{
		fmt.Sprintf("Mood: %s", m.Encounter.Mood),
		"1/2/3 intervienes · ENTER los dejas decidir · X salir discretamente",
	}
	return encounterMutedStyle.Render(strings.Join(lines, "   "))
}

func (m EncounterModel) outcomeContent() string {
	delta := encounterDeltaForPair(m.Encounter)
	summary := m.Encounter.Aftermath.VisibleSummary
	if strings.TrimSpace(summary) == "" && config.GetContentLevel() == config.ContentRestricted {
		summary = fmt.Sprintf("[%s y %s pasaron tiempo juntos]", participantName(m.Encounter.Participants[0]), participantName(m.Encounter.Participants[1]))
	}
	lines := []string{
		encounterOutcomeStyle.Render("Consecuencias"),
		"",
		summary,
		"",
		fmt.Sprintf("Relacion: %+d", delta),
		fmt.Sprintf("Mood del encuentro: %s", m.Encounter.Mood),
		fmt.Sprintf("Posible embarazo: %t", m.Encounter.WillLeadToPregnancy),
		fmt.Sprintf("Posible STI: %t", m.Encounter.STITransmissionOccurred),
		"",
		encounterMutedStyle.Render("El encuentro no entra al historial publico. Solo quedan los cambios."),
	}
	if len(m.Encounter.FutureHints) > 0 {
		lines = append(lines, "", encounterMutedStyle.Render("Eco futuro: "+strings.Join(uniqueStrings(m.Encounter.FutureHints), ", ")))
	}
	return strings.Join(lines, "\n")
}

func (m EncounterModel) currentOptionsText() []string {
	current := m.script[m.currentIndex]
	lines := []string{encounterOptionStyle.Render("Intervenir o dejar que pase:")}
	for index, option := range current.OptionsForPoble1 {
		lines = append(lines, fmt.Sprintf("%d. %s", index+1, option.Text))
	}
	lines = append(lines, encounterMutedStyle.Render("ENTER deja que ambos decidan solos"))
	return lines
}

func (m EncounterModel) phaseChanged(current minigames.EncounterPhase) bool {
	for index := len(m.Encounter.Choices) - 1; index >= 0; index-- {
		return m.Encounter.Choices[index].Phase != current
	}
	return true
}

func (m EncounterModel) nextRevealCmd() tea.Cmd {
	if m.currentIndex >= len(m.script) {
		return nil
	}
	delay := m.script[m.currentIndex].Delay
	if delay <= 0 {
		delay = 700
	}
	return tea.Tick(time.Duration(delay)*time.Millisecond, func(time.Time) tea.Msg {
		return encounterRevealMsg{}
	})
}

func (m EncounterModel) choiceSummary(first, second minigames.ChoiceOption) string {
	names := []string{}
	if m.Encounter.Participants[0] != nil && strings.TrimSpace(first.Text) != "" {
		names = append(names, m.Encounter.Participants[0].Name+": "+first.Text)
	}
	if m.Encounter.Participants[1] != nil && strings.TrimSpace(second.Text) != "" {
		names = append(names, m.Encounter.Participants[1].Name+": "+second.Text)
	}
	return strings.Join(names, " · ")
}

func buildEncounterScript(snapshot AppStateSnapshot, state minigames.EncounterState) []minigames.EncounterChoice {
	participants := []*entities.Poble{state.Participants[0], state.Participants[1]}
	if participants[0] == nil || participants[1] == nil {
		return nil
	}
	category := encounterCategoryFor(state)
	script := []minigames.EncounterChoice{}
	script = append(script, renderEncounterChoices(snapshot, state, category, minigames.EncounterPhaseTension, 3)...)
	script = append(script, renderEncounterChoices(snapshot, state, category, minigames.EncounterPhaseInitiation, 2)...)
	script = append(script, renderEncounterChoices(snapshot, state, category, minigames.EncounterPhaseEncounter, 4)...)
	script = append(script, renderEncounterChoices(snapshot, state, "dialogues/encounter/aftermath", minigames.EncounterPhaseAftermath, 2)...)
	return script
}

func renderEncounterChoices(snapshot AppStateSnapshot, state minigames.EncounterState, category string, phase minigames.EncounterPhase, count int) []minigames.EncounterChoice {
	result := make([]minigames.EncounterChoice, 0, count)
	for index := 0; index < count; index++ {
		text := renderEncounterText(snapshot, state, category)
		if text == "" {
			text = fallbackEncounterText(state, phase, index)
		}
		result = append(result, minigames.EncounterChoice{
			Phase:               phase,
			Text:                text,
			OptionsForPoble1:    encounterOptionsFor(state, phase),
			AutoChoiceForPoble1: encounterAutoChoice(state, phase, 0),
			AutoChoiceForPoble2: encounterAutoChoice(state, phase, 1),
			Consequence:         encounterConsequenceFor(state, phase, index),
			Delay:               encounterDelay(text),
		})
	}
	return result
}

func renderEncounterText(snapshot AppStateSnapshot, state minigames.EncounterState, category string) string {
	if snapshot.TemplateEngine == nil || snapshot.World == nil || state.Participants[0] == nil || state.Participants[1] == nil {
		return ""
	}
	relationship := relationshipBetween(state.Participants[0], state.Participants[1].ID)
	recent, old := relevantMemories(state.Participants[0], state.Participants[1].ID)
	loc, _ := encounterWorldLocation(snapshot.World, state.Participants[0])
	extraVars := map[string]string{}
	if loc != nil {
		extraVars["location_name"] = loc.Name
		extraVars["location_atmosphere"] = encounterAtmosphereLabel(loc)
		extraVars["location_memory"] = encounterLocationMemory(loc)
		extraVars["template_modifiers"] = strings.Join(loc.TemplateModifiers, ",")
	}
	extraVars["power_dynamic"] = state.Context.Power.Description
	extraVars["encounter_mood"] = string(state.Mood)
	extraVars["encounter_type"] = strings.ToLower(state.Type.String())
	ctx := templates.TemplateContext{
		Speaker:                state.Participants[0],
		Target:                 state.Participants[1],
		Location:               dialogueLocationLabel(snapshot, []*entities.Poble{state.Participants[0], state.Participants[1]}),
		GameTime:               snapshot.World.Calendar,
		RecentMemory:           recent,
		OldMemory:              old,
		RelationshipWithTarget: &relationship,
		WorldState:             snapshot.World.GetWorldState(),
		ExtraVars:              extraVars,
	}
	template, err := snapshot.TemplateEngine.Select(category, ctx)
	if err != nil {
		return ""
	}
	rendered, err := snapshot.TemplateEngine.Render(template, ctx)
	if err != nil {
		return ""
	}
	return normalizeEncounterText(rendered)
}

func encounterOptionsFor(state minigames.EncounterState, phase minigames.EncounterPhase) []minigames.ChoiceOption {
	switch phase {
	case minigames.EncounterPhaseTension:
		return []minigames.ChoiceOption{
			{Text: "acercarse un poco mas", RelationshipDelta: 4, TrustDelta: 2, AttractionDelta: 5, MoodShift: entities.EmotionHope},
			{Text: "hacer un chiste para romper el pulso", RelationshipDelta: 2, TrustDelta: 3, AttractionDelta: 1, MoodShift: entities.EmotionJoy},
			{Text: "preguntar suave si sigue bien", RelationshipDelta: 3, TrustDelta: 5, AttractionDelta: 1, UsesProtection: true, MoodShift: entities.EmotionTrust},
		}
	case minigames.EncounterPhaseInitiation:
		return []minigames.ChoiceOption{
			{Text: "tomar la mano y no apurarlo", RelationshipDelta: 5, TrustDelta: 5, AttractionDelta: 2, MoodShift: entities.EmotionTrust},
			{Text: "dejar que la intensidad suba sola", RelationshipDelta: 4, TrustDelta: 1, AttractionDelta: 6, MoodShift: entities.EmotionLust},
			{Text: "pedir una pausa de un segundo", RelationshipDelta: 1, TrustDelta: 4, AttractionDelta: 0, UsesProtection: true, MoodShift: entities.EmotionRelief},
		}
	case minigames.EncounterPhaseEncounter:
		return []minigames.ChoiceOption{
			{Text: "seguir el ritmo y decir lo que siente", RelationshipDelta: 6, TrustDelta: 3, AttractionDelta: 5, MoodShift: entities.EmotionLove},
			{Text: "bajar el ritmo para quedarse presentes", RelationshipDelta: 4, TrustDelta: 6, AttractionDelta: 2, UsesProtection: true, MoodShift: entities.EmotionRelief},
			{Text: "reirse un poco del nervio", RelationshipDelta: 3, TrustDelta: 2, AttractionDelta: 2, MoodShift: entities.EmotionJoy},
		}
	default:
		return []minigames.ChoiceOption{
			{Text: "quedarse cerca sin hablar de mas", RelationshipDelta: 4, TrustDelta: 4, AttractionDelta: 1, MoodShift: entities.EmotionRelief},
			{Text: "decir algo honesto antes de irse", RelationshipDelta: 5, TrustDelta: 5, AttractionDelta: 2, MoodShift: entities.EmotionHope},
			{Text: "cubrir la incomodidad con humor", RelationshipDelta: 1, TrustDelta: 1, AttractionDelta: 0, MoodShift: entities.EmotionJoy},
		}
	}
}

func encounterAutoChoice(state minigames.EncounterState, phase minigames.EncounterPhase, participantIndex int) minigames.ChoiceOption {
	options := encounterOptionsFor(state, phase)
	if len(options) == 0 {
		return minigames.ChoiceOption{}
	}
	switch state.Mood {
	case minigames.EncounterMoodTender:
		return options[minInt(2, len(options)-1)]
	case minigames.EncounterMoodPassionate:
		return options[0]
	case minigames.EncounterMoodAwkward:
		return options[minInt(1, len(options)-1)]
	case minigames.EncounterMoodComplicated:
		if participantIndex == 0 {
			return options[minInt(1, len(options)-1)]
		}
		return options[minInt(2, len(options)-1)]
	default:
		return options[participantIndex%len(options)]
	}
}

func encounterConsequenceFor(state minigames.EncounterState, phase minigames.EncounterPhase, index int) minigames.ChoiceConsequence {
	hint := ""
	if phase == minigames.EncounterPhaseAftermath && state.Mood == minigames.EncounterMoodComplicated {
		hint = "complication"
	}
	if phase == minigames.EncounterPhaseEncounter && index == 0 && state.WillLeadToPregnancy {
		hint = "pregnancy_possible"
	}
	return minigames.ChoiceConsequence{
		RelationshipDelta: 2,
		TrustDelta:        1,
		AttractionDelta:   2,
		FutureHint:        hint,
		MoodShift:         entities.EmotionHope,
	}
}

func selectEncounterParticipants(snapshot AppStateSnapshot) [2]*entities.Poble {
	if pair := activeIntimacyPair(snapshot); pair[0] != nil && pair[1] != nil {
		return pair
	}
	all := selectDialogueParticipants(snapshot)
	if len(all) >= 2 {
		return [2]*entities.Poble{all[0], all[1]}
	}
	if len(all) == 1 {
		return [2]*entities.Poble{all[0], nil}
	}
	return [2]*entities.Poble{}
}

func activeIntimacyPair(snapshot AppStateSnapshot) [2]*entities.Poble {
	if snapshot.World == nil {
		return [2]*entities.Poble{}
	}
	for _, event := range snapshot.World.ActiveEvents {
		if event.Type != ai.GameEventIntimacy {
			continue
		}
		first := snapshot.World.GetPoble(event.PrimaryActor)
		second := snapshot.World.GetPoble(event.TargetID)
		if first != nil && second != nil {
			return [2]*entities.Poble{first, second}
		}
		if len(event.Participants) >= 2 {
			first = snapshot.World.GetPoble(event.Participants[0])
			second = snapshot.World.GetPoble(event.Participants[1])
			if first != nil && second != nil {
				return [2]*entities.Poble{first, second}
			}
		}
	}
	return [2]*entities.Poble{}
}

func fallbackEncounterText(state minigames.EncounterState, phase minigames.EncounterPhase, index int) string {
	first := state.Participants[0]
	second := state.Participants[1]
	if first == nil || second == nil {
		return "La intimidad no encuentra a nadie esta vez."
	}
	switch phase {
	case minigames.EncounterPhaseTension:
		return fmt.Sprintf("%s y %s se quedan quietos un segundo de mas. Ya no parece accidente.", first.Name, second.Name)
	case minigames.EncounterPhaseInitiation:
		return fmt.Sprintf("El primer paso no suena grande, pero cambia el aire entre %s y %s.", first.Name, second.Name)
	case minigames.EncounterPhaseEncounter:
		if index%2 == 0 {
			return "Nadie presume control. Solo siguen estando ahi, mas cerca, mas honestos de lo comodo."
		}
		return fmt.Sprintf("%s dice poco. %s entiende demasiado.", first.Name, second.Name)
	default:
		return fmt.Sprintf("Despues, lo que queda entre %s y %s no es silencio. Es algo mas dificil de ordenar.", first.Name, second.Name)
	}
}

func normalizeEncounterText(text string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\n", " ")
	text = strings.Join(strings.Fields(text), " ")
	return strings.TrimSpace(text)
}

func encounterDelay(text string) int {
	length := len([]rune(text))
	switch {
	case length > 130:
		return 1100
	case length > 75:
		return 900
	default:
		return 700
	}
}

func wrapEncounterText(text string) string {
	return text
}

func encounterDeltaForPair(state minigames.EncounterState) int {
	if state.Aftermath.RelationshipShift != 0 {
		return state.Aftermath.RelationshipShift
	}
	first := state.Participants[0]
	second := state.Participants[1]
	if first == nil || second == nil {
		return 0
	}
	return state.RelationshipDelta[first.ID+":"+second.ID+":relationship"]
}

func worldTimeOrZero(snapshot AppStateSnapshot) entities.GameTime {
	if snapshot.World == nil {
		return entities.NewGameTime(0, 0, 0)
	}
	return snapshot.World.Calendar
}

func uniqueStrings(values []string) []string {
	seen := map[string]struct{}{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func encounterCategoryFor(state minigames.EncounterState) string {
	switch state.Type {
	case minigames.EncounterTender:
		return "dialogues/encounter/tender"
	case minigames.EncounterPassionate:
		return "dialogues/encounter/passionate"
	case minigames.EncounterDesperate:
		return "dialogues/encounter/desperate"
	case minigames.EncounterAngry:
		return "dialogues/encounter/angry"
	case minigames.EncounterCurious:
		return "dialogues/encounter/curious"
	case minigames.EncounterSecret, minigames.EncounterTransactional:
		return "dialogues/encounter/secret"
	case minigames.EncounterLast:
		return "dialogues/encounter/last"
	case minigames.EncounterFirstEver:
		return "dialogues/encounter/first_ever"
	default:
		return "dialogues/encounter/complicated"
	}
}

func encounterWorldLocation(gameWorld *world.World, speaker *entities.Poble) (*world.Location, bool) {
	if gameWorld == nil || speaker == nil {
		return nil, false
	}
	loc, ok := gameWorld.GetLocation(speaker.ID)
	if !ok {
		return nil, false
	}
	return &loc, true
}

func encounterAtmosphereLabel(loc *world.Location) string {
	if loc == nil {
		return "quieto"
	}
	if value := firstAtmosphereKey(loc.Atmosphere.Occupancy); value != "" {
		switch value {
		case "empty":
			return "vacío y demasiado claro"
		case "quiet":
			return "quieto, casi cómplice"
		case "packed":
			return "lleno, con ojos en todas partes"
		default:
			return "activo, con ruido alrededor"
		}
	}
	return "cargado de expectativa"
}

func encounterLocationMemory(loc *world.Location) string {
	if loc == nil || len(loc.ActiveMemories) == 0 {
		return "nadie ha dejado una marca clara aquí"
	}
	return loc.ActiveMemories[len(loc.ActiveMemories)-1].Summary
}

func firstAtmosphereKey(values map[string]float32) string {
	bestKey := ""
	bestValue := float32(-1)
	for key, value := range values {
		if value > bestValue {
			bestKey = key
			bestValue = value
		}
	}
	return bestKey
}

func participantName(poble *entities.Poble) string {
	if poble == nil || strings.TrimSpace(poble.Name) == "" {
		return "Alguien"
	}
	return poble.Name
}
