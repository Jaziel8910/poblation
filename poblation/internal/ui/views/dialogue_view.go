package views

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/user/poblation/internal/entities"
	"github.com/user/poblation/internal/events"
	"github.com/user/poblation/internal/templates"
)

// DialoguePhase marks the current dramatic beat of the conversation.
type DialoguePhase string

const (
	DialoguePhaseGreeting      DialoguePhase = "GREETING"
	DialoguePhaseMain          DialoguePhase = "MAIN"
	DialoguePhaseEmotionalPeak DialoguePhase = "EMOTIONAL_PEAK"
	DialoguePhaseResolution    DialoguePhase = "RESOLUTION"
	DialoguePhaseUnresolved    DialoguePhase = "UNRESOLVED"
)

// DialogueOutcomeType stores the final state of the exchange.
type DialogueOutcomeType string

const (
	DialogueOutcomeResolved      DialogueOutcomeType = "RESOLVED"
	DialogueOutcomeUnresolved    DialogueOutcomeType = "UNRESOLVED"
	DialogueOutcomeEscalated     DialogueOutcomeType = "ESCALATED"
	DialogueOutcomeEndedAbruptly DialogueOutcomeType = "ENDED_ABRUPTLY"
	DialogueOutcomeKiss          DialogueOutcomeType = "KISS"
	DialogueOutcomeFight         DialogueOutcomeType = "FIGHT"
	DialogueOutcomeConfessed     DialogueOutcomeType = "CONFESSED"
)

// DialogueLine is one visible line in the real-time conversation.
type DialogueLine struct {
	SpeakerID  string
	Text       string
	EmotionTag entities.EmotionType
	IsPause    bool
	Delay      int
}

// DialogueOutcome stores the end result of a conversation.
type DialogueOutcome struct {
	Type                DialogueOutcomeType
	RelationshipChanges map[string]int
	EventTriggered      *events.GameEvent
}

// DialogueModel renders a conversation that reveals itself line by line.
type DialogueModel struct {
	Participants      []*entities.Poble
	Lines             []DialogueLine
	CurrentLine       int
	Phase             DialoguePhase
	IsActive          bool
	Outcome           *DialogueOutcome
	state             AppStateSnapshot
	viewport          viewport.Model
	conversationKey   string
	locationLabel     string
	relationshipLabel string
	metaLine          string
	category          string
}

type dialogueAdvanceMsg struct{}

var (
	dialogueSurfaceStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(borderColor).
				Background(backgroundColor).
				Foreground(primaryColor).
				Padding(0, 1)

	dialogueHeaderStyle = lipgloss.NewStyle().
				Foreground(primaryColor).
				Bold(true)

	dialogueSubtleStyle = lipgloss.NewStyle().
				Foreground(mutedColor)

	dialogueMetaStyle = lipgloss.NewStyle().
				Foreground(mutedColor).
				Italic(true)

	dialogueOutcomeStyle = lipgloss.NewStyle().
				Foreground(secondaryColor).
				Bold(true)

	dialoguePauseStyle = lipgloss.NewStyle().
				Foreground(mutedColor).
				PaddingLeft(3)
)

// NewDialogueModel builds the dialogue view with a ready viewport.
func NewDialogueModel() DialogueModel {
	return DialogueModel{
		viewport: viewport.New(32, 12),
		Phase:    DialoguePhaseGreeting,
	}
}

// Init satisfies tea.Model.
func (m DialogueModel) Init() tea.Cmd {
	return nil
}

// SyncAppState refreshes shared state and resizes the viewport.
func (m DialogueModel) SyncAppState(snapshot AppStateSnapshot) tea.Model {
	m.state = snapshot
	m.resizeViewport()
	return m
}

func (m DialogueModel) Resize(width, height int) tea.Model {
	m.state.Width = width
	m.state.Height = height
	m.resizeViewport()
	return m
}

// OnEnter regenerates the conversation when the dialogue view opens.
func (m DialogueModel) OnEnter() (tea.Model, tea.Cmd) {
	m = m.generateConversation()
	if !m.IsActive || len(m.Lines) == 0 {
		return m, nil
	}
	return m, m.nextRevealCmd()
}

// Update handles timed line reveal, scrolling, and director interventions.
func (m DialogueModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch typed := msg.(type) {
	case tea.WindowSizeMsg:
		return m.Resize(typed.Width, typed.Height), nil
	case dialogueAdvanceMsg:
		return m.advanceConversation()
	case tea.KeyMsg:
		return m.handleKey(typed)
	default:
		return m, nil
	}
}

// View renders the conversation header, viewport, meta hint, and ending.
func (m DialogueModel) View() string {
	header := m.renderHeader()
	body := dialogueSurfaceStyle.Width(maxInt(24, m.state.Width-4)).Render(m.viewport.View())

	parts := []string{header, body}
	if meta := m.renderMeta(); meta != "" {
		parts = append(parts, meta)
	}
	if director := m.renderDirectorControls(); director != "" {
		parts = append(parts, director)
	}
	if outcome := m.renderOutcome(); outcome != "" {
		parts = append(parts, outcome)
	}
	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

func (m *DialogueModel) resizeViewport() {
	width := maxInt(24, m.state.Width-6)
	height := maxInt(8, m.state.Height-10)
	m.viewport.Width = width
	m.viewport.Height = height
	m.syncViewportContent()
}

func (m DialogueModel) renderHeader() string {
	names := participantNames(m.Participants)
	if names == "" {
		names = "Sin dialogo"
	}
	location := m.locationLabel
	if location == "" {
		location = "Lugar incierto"
	}
	timeLabel := "00:00"
	if m.state.World != nil {
		timeLabel = fmt.Sprintf("%02d:%02d", m.state.World.Calendar.Hour, m.state.World.Calendar.Minute)
	}
	main := fmt.Sprintf("%s · %s · %s", names, location, timeLabel)
	parts := []string{dialogueHeaderStyle.Render(main)}
	if strings.TrimSpace(m.relationshipLabel) != "" {
		parts = append(parts, dialogueSubtleStyle.Render(m.relationshipLabel))
	}
	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

func (m DialogueModel) renderMeta() string {
	if strings.TrimSpace(m.metaLine) == "" {
		return ""
	}
	return dialogueMetaStyle.Render("[" + m.metaLine + "]")
}

func (m DialogueModel) renderDirectorControls() string {
	if !m.state.IsDirectorMode || !m.IsActive {
		return ""
	}
	return dialogueSubtleStyle.Render("Intervenir: [A] Cambiar tema  [B] Intensificar  [C] Interrumpir")
}

func (m DialogueModel) renderOutcome() string {
	if m.IsActive || m.Outcome == nil {
		return ""
	}
	line := fmt.Sprintf("Outcome: %s", m.Outcome.Type)
	if m.Outcome.EventTriggered != nil && strings.TrimSpace(m.Outcome.EventTriggered.Description) != "" {
		line += " · " + m.Outcome.EventTriggered.Description
	}
	return dialogueOutcomeStyle.Render(line)
}

func (m DialogueModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
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
	case "a":
		if m.state.IsDirectorMode && m.IsActive {
			m.applyTopicShift()
		}
	case "b":
		if m.state.IsDirectorMode && m.IsActive {
			m.applyIntensify()
		}
	case "c":
		if m.state.IsDirectorMode && m.IsActive {
			m.applyInterrupt()
		}
	}
	m.syncViewportContent()
	if m.IsActive && len(m.Lines) > m.CurrentLine {
		return m, m.nextRevealCmd()
	}
	return m, nil
}

func (m DialogueModel) advanceConversation() (tea.Model, tea.Cmd) {
	if !m.IsActive || m.CurrentLine >= len(m.Lines) {
		return m.finishConversation(), nil
	}
	m.CurrentLine++
	m.Phase = dialoguePhaseForProgress(m.CurrentLine, len(m.Lines))
	m.syncViewportContent()
	if m.CurrentLine >= len(m.Lines) {
		return m.finishConversation(), nil
	}
	return m, m.nextRevealCmd()
}

func (m DialogueModel) nextRevealCmd() tea.Cmd {
	if !m.IsActive || m.CurrentLine >= len(m.Lines) {
		return nil
	}
	delay := m.Lines[m.CurrentLine].Delay
	if delay <= 0 {
		delay = 700
	}
	return tea.Tick(time.Duration(delay)*time.Millisecond, func(time.Time) tea.Msg {
		return dialogueAdvanceMsg{}
	})
}

func (m DialogueModel) finishConversation() DialogueModel {
	m.IsActive = false
	if m.CurrentLine < len(m.Lines) {
		m.CurrentLine = len(m.Lines)
	}
	m.Phase = finalPhaseForOutcome(m.category, m.Outcome)
	if m.Outcome == nil {
		m.Outcome = buildDialogueOutcome(m.category, m.Participants)
	}
	m.syncViewportContent()
	return m
}

func (m DialogueModel) generateConversation() DialogueModel {
	m.Participants = selectDialogueParticipants(m.state)
	m.locationLabel = dialogueLocationLabel(m.state, m.Participants)
	m.relationshipLabel = dialogueRelationshipLabel(m.Participants)
	m.metaLine = dialogueMetaLine(m.Participants)
	m.category = selectDialogueCategory(m.Participants)
	m.conversationKey = dialogueConversationKey(m.state, m.Participants, m.category)
	m.Lines = buildDialogueLines(m.state, m.Participants, m.category)
	m.CurrentLine = 0
	m.Phase = DialoguePhaseGreeting
	m.IsActive = len(m.Lines) > 0
	m.Outcome = nil
	m.syncViewportContent()
	return m
}

func (m *DialogueModel) syncViewportContent() {
	lines := make([]string, 0, m.CurrentLine+1)
	visible := m.CurrentLine
	if visible > len(m.Lines) {
		visible = len(m.Lines)
	}
	for _, line := range m.Lines[:visible] {
		lines = append(lines, m.renderLine(line))
	}
	content := strings.Join(lines, "\n")
	if strings.TrimSpace(content) == "" {
		content = dialogueSubtleStyle.Render("Todavia no hay nadie hablando.")
	}
	m.viewport.SetContent(content)
	m.viewport.GotoBottom()
}

func (m DialogueModel) renderLine(line DialogueLine) string {
	if line.IsPause {
		return dialoguePauseStyle.Render("...")
	}
	speaker := m.participantByID(line.SpeakerID)
	name := "ALGUIEN"
	if speaker != nil && strings.TrimSpace(speaker.Name) != "" {
		name = strings.ToUpper(speaker.Name)
	}
	label := m.speakerStyle(line.SpeakerID).Render("[" + name + "]")
	return label + ": " + line.Text
}

func (m DialogueModel) speakerStyle(speakerID string) lipgloss.Style {
	colors := []lipgloss.Color{secondaryColor, accentColor, successColor, warningColor}
	index := 0
	for i, participant := range m.Participants {
		if participant != nil && participant.ID == speakerID {
			index = i
			break
		}
	}
	return lipgloss.NewStyle().Foreground(colors[index%len(colors)]).Bold(true)
}

func (m DialogueModel) participantByID(id string) *entities.Poble {
	for _, participant := range m.Participants {
		if participant != nil && participant.ID == id {
			return participant
		}
	}
	return nil
}

func (m *DialogueModel) applyTopicShift() {
	if len(m.Participants) < 2 {
		return
	}
	a := m.Participants[0]
	b := m.Participants[1]
	m.Lines = append(m.Lines,
		DialogueLine{SpeakerID: a.ID, Text: "No. Cambiemos de tema antes de romper algo que no podemos reparar hoy.", EmotionTag: entities.EmotionAnxiety, Delay: 750},
		DialogueLine{SpeakerID: b.ID, Text: "Bien. Pero no finjas que esto desaparecio.", EmotionTag: entities.EmotionResentment, Delay: 780},
	)
}

func (m *DialogueModel) applyIntensify() {
	if len(m.Participants) < 2 {
		return
	}
	a := m.Participants[0]
	b := m.Participants[1]
	m.Phase = DialoguePhaseEmotionalPeak
	m.Lines = append(m.Lines,
		DialogueLine{IsPause: true, Delay: 900},
		DialogueLine{SpeakerID: a.ID, Text: "Entonces dilo entero. Ya estamos demasiado adentro para medias frases.", EmotionTag: entities.EmotionAnger, Delay: 820},
		DialogueLine{SpeakerID: b.ID, Text: "Eso es justo lo que te asusta escuchar.", EmotionTag: entities.EmotionContempt, Delay: 760},
	)
}

func (m *DialogueModel) applyInterrupt() {
	if len(m.Participants) == 0 {
		return
	}
	lastSpeaker := m.Participants[0]
	if len(m.Participants) > 1 {
		lastSpeaker = m.Participants[1]
	}
	if m.CurrentLine < len(m.Lines) {
		m.Lines = append(append([]DialogueLine{}, m.Lines[:m.CurrentLine]...),
			DialogueLine{IsPause: true, Delay: 900},
			DialogueLine{SpeakerID: lastSpeaker.ID, Text: "Ya. Hasta aqui.", EmotionTag: entities.EmotionAnger, Delay: 650},
		)
	}
	m.CurrentLine = len(m.Lines)
	m.Outcome = &DialogueOutcome{
		Type:                DialogueOutcomeEndedAbruptly,
		RelationshipChanges: map[string]int{},
		EventTriggered: &events.GameEvent{
			Type:        events.EventDecisionPoint,
			Description: "La conversacion se corto de golpe.",
		},
	}
	m.IsActive = false
	m.Phase = DialoguePhaseUnresolved
	m.syncViewportContent()
}

func selectDialogueParticipants(state AppStateSnapshot) []*entities.Poble {
	if state.World == nil {
		return nil
	}
	all := append([]*entities.Poble(nil), state.World.GetAllPobles()...)
	sort.SliceStable(all, func(i, j int) bool {
		return all[i].Name < all[j].Name
	})
	if len(all) == 0 {
		return nil
	}

	anchor := all[0]
	if state.SelectedPobleID != "" {
		if selected := state.World.GetPoble(state.SelectedPobleID); selected != nil {
			anchor = selected
		}
	}
	if len(all) == 1 {
		return []*entities.Poble{anchor}
	}

	type scoredTarget struct {
		poble *entities.Poble
		score float32
	}

	scored := make([]scoredTarget, 0, len(all)-1)
	for _, candidate := range all {
		if candidate == nil || candidate.ID == anchor.ID {
			continue
		}
		scored = append(scored, scoredTarget{
			poble: candidate,
			score: dialoguePartnerScore(anchor, candidate),
		})
	}
	sort.SliceStable(scored, func(i, j int) bool {
		if scored[i].score == scored[j].score {
			return scored[i].poble.Name < scored[j].poble.Name
		}
		return scored[i].score > scored[j].score
	})

	participants := []*entities.Poble{anchor}
	for index, candidate := range scored {
		if len(participants) >= 4 {
			break
		}
		if index == 0 || candidate.score >= 55 {
			participants = append(participants, candidate.poble)
		}
	}
	if len(participants) == 1 {
		participants = append(participants, scored[0].poble)
	}
	return participants
}

func dialoguePartnerScore(anchor, candidate *entities.Poble) float32 {
	relationship := relationshipBetween(anchor, candidate.ID)
	score := relationship.Trust + relationship.Attraction + relationship.Familiarity - relationship.Resentment
	score += float32(sharedMemoryCount(anchor, candidate.ID) * 12)
	if hasHiddenSecret(anchor) {
		score += 10
	}
	if candidate.CurrentMood == entities.MoodAngry || candidate.CurrentMood == entities.MoodObsessive {
		score += 6
	}
	return score
}

func buildDialogueLines(state AppStateSnapshot, participants []*entities.Poble, category string) []DialogueLine {
	if len(participants) == 0 {
		return nil
	}
	primary := participants[0]
	target := participants[minInt(1, len(participants)-1)]
	relationship := relationshipBetween(primary, target.ID)
	plan := dialoguePlan(category, relationship)
	lines := make([]DialogueLine, 0, 14)

	for index, step := range plan {
		lines = append(lines, loadTemplateDialogue(state, primary, target, relationship, step)...)
		if index == 0 {
			lines = append(lines, buildMemoryLines(primary, target)...)
		}
		if index < len(plan)-1 {
			lines = append(lines, DialogueLine{IsPause: true, Delay: 1000})
		}
	}

	lines = append(lines, buildSideParticipantLines(participants)...)
	if len(lines) == 0 {
		return fallbackDialogueLines(primary, target, relationship)
	}
	return lines
}

func dialoguePlan(category string, relationship entities.Relationship) []string {
	switch category {
	case "dialogues/argument/escalating":
		return []string{"dialogues/greeting/post_fight", "dialogues/argument/petty", "dialogues/argument/escalating"}
	case "dialogues/reconciliation/after_betrayal":
		return []string{"dialogues/greeting/post_fight", "dialogues/reconciliation/old_wounds/general", "dialogues/reconciliation/after_betrayal"}
	case "dialogues/confession/love":
		if relationship.Attraction >= 65 {
			return []string{"dialogues/flirt/subtle", "dialogues/confession/love"}
		}
		return []string{"dialogues/confession/love"}
	case "dialogues/confession/secret":
		return []string{"dialogues/gossip/general", "dialogues/confession/secret"}
	default:
		return []string{category}
	}
}

func loadTemplateDialogue(state AppStateSnapshot, primary, target *entities.Poble, relationship entities.Relationship, category string) []DialogueLine {
	engine := state.TemplateEngine
	if engine == nil || primary == nil || target == nil || state.World == nil {
		return nil
	}
	ctx := templateContextForDialogue(state, primary, target, relationship)
	template, err := engine.Select(category, ctx)
	if err != nil {
		return nil
	}
	rendered, err := engine.Render(template, ctx)
	if err != nil {
		return nil
	}
	return parseTemplateDialogue(rendered, primary, target)
}

func templateContextForDialogue(state AppStateSnapshot, primary, target *entities.Poble, relationship entities.Relationship) templates.TemplateContext {
	recent, old := relevantMemories(primary, target.ID)
	return templates.TemplateContext{
		Speaker:                primary,
		Target:                 target,
		Location:               dialogueLocationLabel(state, []*entities.Poble{primary, target}),
		GameTime:               state.World.Calendar,
		RecentMemory:           recent,
		OldMemory:              old,
		RelationshipWithTarget: &relationship,
		WorldState:             state.World.GetWorldState(),
		ExtraVars:              map[string]string{},
	}
}

func parseTemplateDialogue(rendered string, primary, target *entities.Poble) []DialogueLine {
	rawLines := strings.Split(strings.ReplaceAll(rendered, "\r\n", "\n"), "\n")
	lines := make([]DialogueLine, 0, len(rawLines))
	for _, raw := range rawLines {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		if raw == "..." {
			lines = append(lines, DialogueLine{IsPause: true, Delay: 950})
			continue
		}

		switch {
		case strings.HasPrefix(raw, "[") && strings.Contains(raw, "]:"):
			parts := strings.SplitN(raw, "]:", 2)
			speakerName := strings.Trim(strings.TrimPrefix(parts[0], "["), " ")
			text := strings.TrimSpace(parts[1])
			lines = append(lines, DialogueLine{
				SpeakerID:  dialogueSpeakerID(speakerName, primary, target),
				Text:       text,
				EmotionTag: inferLineEmotion(text),
				Delay:      dialogueLineDelay(text),
			})
		default:
			lines = append(lines, DialogueLine{
				SpeakerID:  primary.ID,
				Text:       raw,
				EmotionTag: inferLineEmotion(raw),
				Delay:      dialogueLineDelay(raw),
			})
		}
	}
	return lines
}

func dialogueSpeakerID(name string, primary, target *entities.Poble) string {
	if target != nil && strings.EqualFold(strings.TrimSpace(name), strings.TrimSpace(target.Name)) {
		return target.ID
	}
	if primary != nil && strings.EqualFold(strings.TrimSpace(name), strings.TrimSpace(primary.Name)) {
		return primary.ID
	}
	if strings.Contains(strings.ToLower(name), "target") && target != nil {
		return target.ID
	}
	if primary != nil {
		return primary.ID
	}
	if target != nil {
		return target.ID
	}
	return ""
}

func inferLineEmotion(text string) entities.EmotionType {
	lower := strings.ToLower(text)
	switch {
	case strings.Contains(lower, "gustas"), strings.Contains(lower, "quiero"), strings.Contains(lower, "bes"):
		return entities.EmotionLove
	case strings.Contains(lower, "ment"), strings.Contains(lower, "ocult"), strings.Contains(lower, "miedo"):
		return entities.EmotionAnxiety
	case strings.Contains(lower, "cans"), strings.Contains(lower, "basta"), strings.Contains(lower, "atrevas"):
		return entities.EmotionAnger
	case strings.Contains(lower, "perdon"), strings.Contains(lower, "culpa"), strings.Contains(lower, "deb"):
		return entities.EmotionGuilt
	case strings.Contains(lower, "recuerdo"), strings.Contains(lower, "acuerdo"):
		return entities.EmotionGrief
	default:
		return entities.EmotionCuriosity
	}
}

func dialogueLineDelay(text string) int {
	length := len([]rune(text))
	switch {
	case length >= 90:
		return 1100
	case length >= 45:
		return 850
	default:
		return 620
	}
}

func buildMemoryLines(primary, target *entities.Poble) []DialogueLine {
	recent, _ := relevantMemories(primary, target.ID)
	if recent == nil || strings.TrimSpace(recent.Summary) == "" {
		return nil
	}
	summary := trimSentence(recent.Summary)
	return []DialogueLine{
		{IsPause: true, Delay: 980},
		{SpeakerID: primary.ID, Text: "Todavia me acuerdo de " + summary + ".", EmotionTag: entities.EmotionGrief, Delay: 920},
		{SpeakerID: target.ID, Text: "Ya, y yo tambien. Ese es justo el problema.", EmotionTag: entities.EmotionResentment, Delay: 760},
	}
}

func buildSideParticipantLines(participants []*entities.Poble) []DialogueLine {
	if len(participants) < 3 {
		return nil
	}
	lines := make([]DialogueLine, 0, len(participants)-2)
	for _, participant := range participants[2:] {
		if participant == nil {
			continue
		}
		lines = append(lines, DialogueLine{
			SpeakerID:  participant.ID,
			Text:       "Yo solo digo que esto ya estaba raro antes de que yo llegara.",
			EmotionTag: entities.EmotionBoredom,
			Delay:      700,
		})
	}
	return lines
}

func fallbackDialogueLines(primary, target *entities.Poble, relationship entities.Relationship) []DialogueLine {
	if primary == nil || target == nil {
		return nil
	}
	lines := []DialogueLine{
		{SpeakerID: primary.ID, Text: "No se como empezar esta conversacion sin empeorarla.", EmotionTag: entities.EmotionAnxiety, Delay: 850},
		{SpeakerID: target.ID, Text: "Entonces no empieces suave. Ya estamos aqui.", EmotionTag: entities.EmotionCuriosity, Delay: 760},
	}
	if relationship.Resentment >= 60 {
		lines = append(lines,
			DialogueLine{SpeakerID: primary.ID, Text: "Sigo molesto por lo que paso.", EmotionTag: entities.EmotionAnger, Delay: 760},
			DialogueLine{SpeakerID: target.ID, Text: "Bien. Yo sigo molesto por lo que no dices.", EmotionTag: entities.EmotionResentment, Delay: 780},
		)
	} else {
		lines = append(lines,
			DialogueLine{SpeakerID: primary.ID, Text: "Solo queria que lo escucharas de mi boca.", EmotionTag: entities.EmotionGuilt, Delay: 790},
			DialogueLine{SpeakerID: target.ID, Text: "Entonces habla.", EmotionTag: entities.EmotionTrust, Delay: 620},
		)
	}
	return lines
}

func selectDialogueCategory(participants []*entities.Poble) string {
	if len(participants) < 2 {
		return "dialogues/gossip/general"
	}
	primary := participants[0]
	target := participants[1]
	relationship := relationshipBetween(primary, target.ID)
	memory := strongestDialogueMemory(primary, target.ID)
	switch {
	case memorySignalsOpenConflict(memory, relationship):
		return "dialogues/argument/escalating"
	case memorySignalsBetrayalRepair(memory, relationship):
		return "dialogues/reconciliation/after_betrayal"
	case memorySignalsIntimacy(memory, relationship):
		return "dialogues/confession/love"
	case relationship.Resentment >= 80:
		return "dialogues/argument/escalating"
	case relationship.Resentment >= 58:
		return "dialogues/argument/petty"
	case relationship.Trust >= 68 && hasHiddenSecret(primary):
		return "dialogues/confession/secret"
	case relationship.Attraction >= 75 && relationship.Trust >= 52:
		return "dialogues/confession/love"
	case relationship.Attraction >= 50:
		return "dialogues/flirt/subtle"
	case relationship.Trust >= 45 && relationship.Resentment >= 35:
		return "dialogues/reconciliation/after_betrayal"
	default:
		return "dialogues/gossip/general"
	}
}

func strongestDialogueMemory(source *entities.Poble, targetID string) *entities.Memory {
	recent, old := relevantMemories(source, targetID)
	switch {
	case recent == nil:
		return old
	case old == nil:
		return recent
	case old.EmotionIntensity > recent.EmotionIntensity:
		return old
	default:
		return recent
	}
}

func memorySignalsOpenConflict(memory *entities.Memory, relationship entities.Relationship) bool {
	if memory == nil {
		return false
	}
	if memory.Type == entities.MemoryTraumatic || memory.Type == entities.MemoryBetrayal || memory.Type == entities.MemoryViolent {
		return memory.EmotionIntensity >= 60 || relationship.Resentment >= 45
	}
	return memory.Type == entities.MemoryNegative && memory.EmotionIntensity >= 75 && relationship.Resentment >= 45
}

func memorySignalsBetrayalRepair(memory *entities.Memory, relationship entities.Relationship) bool {
	if memory == nil {
		return false
	}
	return memory.Type == entities.MemoryBetrayal && relationship.Trust >= 45 && relationship.Resentment < 75
}

func memorySignalsIntimacy(memory *entities.Memory, relationship entities.Relationship) bool {
	if memory == nil {
		return false
	}
	if memory.Type != entities.MemoryRomantic && memory.Type != entities.MemoryErotic {
		return false
	}
	return memory.EmotionIntensity >= 55 && relationship.Attraction >= 55 && relationship.Resentment < 55
}

func buildDialogueOutcome(category string, participants []*entities.Poble) *DialogueOutcome {
	if len(participants) < 2 {
		return &DialogueOutcome{Type: DialogueOutcomeUnresolved, RelationshipChanges: map[string]int{}}
	}
	primary := participants[0]
	target := participants[1]
	relationship := relationshipBetween(primary, target.ID)
	changeKey := primary.ID + ":" + target.ID
	outcome := &DialogueOutcome{
		Type:                DialogueOutcomeUnresolved,
		RelationshipChanges: map[string]int{changeKey: 0},
	}

	switch category {
	case "dialogues/confession/love":
		if relationship.Attraction >= 85 && relationship.Trust >= 70 {
			outcome.Type = DialogueOutcomeKiss
			outcome.RelationshipChanges[changeKey] = 18
			outcome.EventTriggered = &events.GameEvent{Type: events.EventSexualEncounter, Description: "El aire cambia y nadie retrocede."}
			return outcome
		}
		outcome.Type = DialogueOutcomeConfessed
		outcome.RelationshipChanges[changeKey] = 9
		outcome.EventTriggered = &events.GameEvent{Type: events.EventRevelation, Description: "Alguien dice lo que ya no podia ocultar."}
	case "dialogues/confession/secret":
		if relationship.Resentment >= 60 {
			outcome.Type = DialogueOutcomeEscalated
			outcome.RelationshipChanges[changeKey] = -12
			outcome.EventTriggered = &events.GameEvent{Type: events.EventBetrayalRevealed, Description: "La verdad abre mas de lo que cierra."}
			return outcome
		}
		outcome.Type = DialogueOutcomeConfessed
		outcome.RelationshipChanges[changeKey] = 6
		outcome.EventTriggered = &events.GameEvent{Type: events.EventRevelation, Description: "Una confesión cambia el peso de la habitacion."}
	case "dialogues/argument/escalating":
		outcome.Type = DialogueOutcomeFight
		outcome.RelationshipChanges[changeKey] = -20
		outcome.EventTriggered = &events.GameEvent{Type: events.EventFightVerbal, Description: "La conversacion termina hecha pelea."}
	case "dialogues/argument/petty":
		outcome.Type = DialogueOutcomeEscalated
		outcome.RelationshipChanges[changeKey] = -9
		outcome.EventTriggered = &events.GameEvent{Type: events.EventFightVerbal, Description: "El roce pequeno termina dejando marca."}
	case "dialogues/reconciliation/after_betrayal":
		if relationship.Trust >= 60 {
			outcome.Type = DialogueOutcomeResolved
			outcome.RelationshipChanges[changeKey] = 11
			outcome.EventTriggered = &events.GameEvent{Type: events.EventForgiveness, Description: "No se arregla todo, pero algo afloja."}
			return outcome
		}
		outcome.Type = DialogueOutcomeUnresolved
		outcome.RelationshipChanges[changeKey] = 2
	default:
		outcome.Type = DialogueOutcomeUnresolved
		outcome.RelationshipChanges[changeKey] = 1
		outcome.EventTriggered = &events.GameEvent{Type: events.EventGossipChain, Description: "Nadie cierra el tema del todo."}
	}
	return outcome
}

func dialoguePhaseForProgress(current, total int) DialoguePhase {
	if total <= 0 {
		return DialoguePhaseUnresolved
	}
	progress := float64(current) / float64(total)
	switch {
	case progress < 0.25:
		return DialoguePhaseGreeting
	case progress < 0.7:
		return DialoguePhaseMain
	case progress < 1:
		return DialoguePhaseEmotionalPeak
	default:
		return DialoguePhaseResolution
	}
}

func finalPhaseForOutcome(category string, outcome *DialogueOutcome) DialoguePhase {
	if outcome == nil {
		if strings.Contains(category, "argument") {
			return DialoguePhaseUnresolved
		}
		return DialoguePhaseResolution
	}
	switch outcome.Type {
	case DialogueOutcomeUnresolved, DialogueOutcomeEscalated, DialogueOutcomeEndedAbruptly, DialogueOutcomeFight:
		return DialoguePhaseUnresolved
	default:
		return DialoguePhaseResolution
	}
}

func dialogueLocationLabel(state AppStateSnapshot, participants []*entities.Poble) string {
	if state.World == nil || len(participants) == 0 || participants[0] == nil {
		return "Sin lugar"
	}
	location, ok := state.World.GetLocation(participants[0].ID)
	if !ok {
		return "Sin lugar"
	}
	island := state.World.GetIsland(location.IslandID)
	if island == nil {
		return fmt.Sprintf("%s (%d,%d)", location.IslandID, location.X, location.Y)
	}
	return fmt.Sprintf("%s (%d,%d)", island.Name, location.X, location.Y)
}

func dialogueRelationshipLabel(participants []*entities.Poble) string {
	if len(participants) < 2 || participants[0] == nil || participants[1] == nil {
		return ""
	}
	relationship := relationshipBetween(participants[0], participants[1].ID)
	label := strings.ToLower(strings.ReplaceAll(relationship.Type.String(), "_", " "))
	switch {
	case relationship.Resentment >= 70 && relationship.Attraction >= 50:
		return "toxic attraction · " + label
	case relationship.Trust >= 70 && relationship.Attraction >= 65:
		return "intimacy with teeth · " + label
	case relationship.Resentment >= 60:
		return "open resentment · " + label
	case relationship.Trust >= 60:
		return "fragile trust · " + label
	default:
		return label
	}
}

func dialogueMetaLine(participants []*entities.Poble) string {
	if len(participants) < 2 || participants[0] == nil {
		return ""
	}
	primary := participants[0]
	target := participants[1]
	relationship := relationshipBetween(primary, target.ID)
	if hasHiddenSecret(primary) {
		chance := clampInt(int(relationship.Trust+(primary.EmotionalState.Arousal*20)-(relationship.Resentment*0.2)), 12, 94)
		return fmt.Sprintf("%s esta considerando confesar - %d%% de probabilidad", primary.Name, chance)
	}
	if relationship.Attraction >= 60 {
		chance := clampInt(int(relationship.Attraction+(relationship.Trust*0.2)), 15, 91)
		return fmt.Sprintf("%s esta pensando en acercarse mas - %d%% de probabilidad", primary.Name, chance)
	}
	chance := clampInt(int(relationship.Resentment+(primary.Needs.Belonging*0.15)), 8, 88)
	return fmt.Sprintf("%s esta decidiendo si empujar la conversacion - %d%% de probabilidad", primary.Name, chance)
}

func dialogueConversationKey(state AppStateSnapshot, participants []*entities.Poble, category string) string {
	ids := make([]string, 0, len(participants))
	for _, participant := range participants {
		if participant != nil {
			ids = append(ids, participant.ID)
		}
	}
	clock := "0-0"
	if state.World != nil {
		clock = fmt.Sprintf("%d-%d", state.World.Calendar.Day, state.World.Calendar.Hour)
	}
	return strings.Join(ids, "|") + "|" + category + "|" + clock
}

func participantNames(participants []*entities.Poble) string {
	names := make([]string, 0, len(participants))
	for _, participant := range participants {
		if participant != nil && strings.TrimSpace(participant.Name) != "" {
			names = append(names, participant.Name)
		}
	}
	return strings.Join(names, " + ")
}

func relationshipBetween(source *entities.Poble, targetID string) entities.Relationship {
	if source == nil || source.Relationships == nil {
		return entities.NewRelationship(targetID, entities.RelationshipStranger)
	}
	if relationship, ok := source.Relationships[targetID]; ok {
		return relationship
	}
	return entities.NewRelationship(targetID, entities.RelationshipStranger)
}

func relevantMemories(source *entities.Poble, targetID string) (*entities.Memory, *entities.Memory) {
	if source == nil || targetID == "" {
		return nil, nil
	}
	var recent *entities.Memory
	var old *entities.Memory
	for index := range source.Memories {
		memory := source.Memories[index]
		if !memoryIncludes(memory, targetID) {
			continue
		}
		candidate := memory
		if recent == nil || candidate.Timestamp.ToMinutes() > recent.Timestamp.ToMinutes() {
			old = recent
			recent = &candidate
			continue
		}
		if old == nil || candidate.Timestamp.ToMinutes() < old.Timestamp.ToMinutes() {
			old = &candidate
		}
	}
	return recent, old
}

func memoryIncludes(memory entities.Memory, targetID string) bool {
	for _, participantID := range memory.Participants {
		if participantID == targetID {
			return true
		}
	}
	return false
}

func sharedMemoryCount(source *entities.Poble, targetID string) int {
	recent, old := relevantMemories(source, targetID)
	count := 0
	if recent != nil {
		count++
	}
	if old != nil {
		count++
	}
	return count
}

func hasHiddenSecret(poble *entities.Poble) bool {
	if poble == nil {
		return false
	}
	for _, secret := range poble.Secrets {
		if !secret.IsRevealed {
			return true
		}
	}
	return false
}

func trimSentence(text string) string {
	text = strings.TrimSpace(text)
	text = strings.TrimRight(text, ".!? ")
	if text == "" {
		return "algo que ninguno logro enterrar"
	}
	return text
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
