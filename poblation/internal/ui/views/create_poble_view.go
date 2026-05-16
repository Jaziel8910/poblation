package views

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/user/poblation/internal/entities"
)

// CreatePobleCompleteMsg returns one generated character to the app shell.
type CreatePobleCompleteMsg struct {
	Poble  *entities.Poble
	Config entities.PoblConfig
}

// CreatePobleCancelMsg asks the app shell to leave the creation flow.
type CreatePobleCancelMsg struct{}

type createStep int

const (
	createStepBasic createStep = iota + 1
	createStepOrientation
	createStepArchetype
	createStepHistory
	createStepConfirm
	createStepRevision
)

type archetypeOption struct {
	ID          string
	Label       string
	Short       string
	Description string
}

// CreatePobleModel renders the multi-step founder creation flow.
type CreatePobleModel struct {
	state AppStateSnapshot

	step   createStep
	form   *huh.Form
	status string

	name                  string
	sexChoice             string
	ageInput              string
	romanticChoice        string
	sexualIntensityChoice string
	archetypeChoice       string
	secretSeed            string
	basedOn               string
	revisionChoice        string
	confirmed             bool

	preview       *entities.Poble
	previewConfig entities.PoblConfig
	rng           *rand.Rand
}

var (
	createFrameStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(borderColor).
				Background(backgroundColor).
				Foreground(primaryColor).
				Padding(1, 2)

	createConfirmFrameStyle = lipgloss.NewStyle().
				Border(lipgloss.DoubleBorder()).
				BorderForeground(accentColor).
				Background(surfaceColor).
				Foreground(primaryColor).
				Padding(1, 2)

	createTitleStyle = lipgloss.NewStyle().
				Foreground(accentColor).
				Bold(true)

	createStepStyle = lipgloss.NewStyle().
			Foreground(secondaryColor).
			Bold(true)

	createHintStyle = lipgloss.NewStyle().
			Foreground(mutedColor)
)

// NewCreatePobleModel returns the founder creation flow.
func NewCreatePobleModel() CreatePobleModel {
	return CreatePobleModel{
		rng: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// Init satisfies tea.Model.
func (m CreatePobleModel) Init() tea.Cmd {
	return nil
}

// BlocksGlobalNavigation lets the form own all local keys.
func (m CreatePobleModel) BlocksGlobalNavigation() bool {
	return true
}

// SyncAppState keeps size context fresh.
func (m CreatePobleModel) SyncAppState(snapshot AppStateSnapshot) tea.Model {
	m.state = snapshot
	return m
}

func (m CreatePobleModel) Resize(width, height int) tea.Model {
	m.state.Width = width
	m.state.Height = height
	return m
}

// OnEnter resets the full creation flow.
func (m CreatePobleModel) OnEnter() (tea.Model, tea.Cmd) {
	m.name = ""
	m.sexChoice = "sorprendeme"
	m.ageInput = "26"
	m.romanticChoice = "procedural"
	m.sexualIntensityChoice = "procedural"
	m.archetypeChoice = "sorprendeme"
	m.secretSeed = ""
	m.basedOn = ""
	m.revisionChoice = "step3"
	m.confirmed = false
	m.preview = nil
	if m.rng == nil {
		m.rng = rand.New(rand.NewSource(time.Now().UnixNano()))
	}
	return m.setStep(createStepBasic)
}

// Update moves through the Huh steps.
func (m CreatePobleModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if size, ok := msg.(tea.WindowSizeMsg); ok {
		return m.Resize(size.Width, size.Height), nil
	}

	if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.String() == "esc" {
		return m, func() tea.Msg { return CreatePobleCancelMsg{} }
	}

	if m.form == nil {
		return m.setStep(createStepBasic)
	}

	updated, cmd := m.form.Update(msg)
	m.form = updated.(*huh.Form)
	if m.form.State == huh.StateCompleted {
		return m.finishCurrentStep()
	}
	return m, cmd
}

// View renders the active step inside a styled character-sheet shell.
func (m CreatePobleModel) View() string {
	if m.form == nil {
		return createFrameStyle.Render("Preparando el creador...")
	}

	header := lipgloss.JoinVertical(
		lipgloss.Left,
		createTitleStyle.Render("Ficha de personaje"),
		createStepStyle.Render(m.stepLabel()),
		createHintStyle.Render("ESC cancela la nueva civilizacion."),
	)

	body := lipgloss.JoinVertical(lipgloss.Left, header, "", m.form.View())
	if strings.TrimSpace(m.status) != "" {
		body = lipgloss.JoinVertical(lipgloss.Left, body, "", menuStatusStyle.Render(m.status))
	}

	frame := createFrameStyle
	if m.step == createStepConfirm || m.step == createStepRevision {
		frame = createConfirmFrameStyle
	}
	return frame.Width(maxInt(56, m.state.Width-8)).Render(body)
}

func (m CreatePobleModel) setStep(step createStep) (tea.Model, tea.Cmd) {
	m.step = step
	m.status = ""

	switch step {
	case createStepBasic:
		m.form = m.basicInfoForm()
	case createStepOrientation:
		m.form = m.orientationForm()
	case createStepArchetype:
		m.form = m.archetypeForm()
	case createStepHistory:
		m.form = m.historyForm()
	case createStepConfirm:
		if err := m.generatePreview(); err != nil {
			m.status = err.Error()
			m.form = m.historyForm()
			m.step = createStepHistory
			return m, m.form.Init()
		}
		m.form = m.confirmForm()
	case createStepRevision:
		m.form = m.revisionForm()
	default:
		m.form = m.basicInfoForm()
		m.step = createStepBasic
	}
	return m, m.form.Init()
}

func (m CreatePobleModel) finishCurrentStep() (tea.Model, tea.Cmd) {
	switch m.step {
	case createStepBasic:
		return m.setStep(createStepOrientation)
	case createStepOrientation:
		return m.setStep(createStepArchetype)
	case createStepArchetype:
		return m.setStep(createStepHistory)
	case createStepHistory:
		return m.setStep(createStepConfirm)
	case createStepConfirm:
		if m.confirmed {
			created := m.preview
			config := m.previewConfig
			return m, func() tea.Msg {
				return CreatePobleCompleteMsg{Poble: created, Config: config}
			}
		}
		return m.setStep(createStepRevision)
	case createStepRevision:
		if m.revisionChoice == "step1" {
			return m.setStep(createStepBasic)
		}
		return m.setStep(createStepArchetype)
	default:
		return m, nil
	}
}

func (m CreatePobleModel) basicInfoForm() *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Nombre").
				Placeholder("Como se llama?").
				Value(&m.name).
				CharLimit(30),
			huh.NewSelect[string]().
				Title("Sexo").
				Value(&m.sexChoice).
				Options(
					huh.NewOption("Hombre", "hombre"),
					huh.NewOption("Mujer", "mujer"),
					huh.NewOption("Intersex", "intersex"),
					huh.NewOption("Sorprendeme", "sorprendeme"),
				),
			huh.NewInput().
				Title("Edad").
				Value(&m.ageInput).
				Validate(validateFounderAge),
		).Title("Paso 1 - Info basica"),
	).WithTheme(menuModalTheme())
}

func (m CreatePobleModel) orientationForm() *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewNote().
				Title("Como lee el sistema esto").
				Description("La orientacion base marca hacia donde suele inclinarse el deseo y la intensidad sexual marca la fuerza del impulso. Si dejas algo procedural, el generador decide."),
			huh.NewSelect[string]().
				Title("Orientacion romantica").
				Value(&m.romanticChoice).
				Options(
					huh.NewOption("Completamente hetero", "hetero"),
					huh.NewOption("Mayormente hetero", "mostly_hetero"),
					huh.NewOption("Bisexual", "bi"),
					huh.NewOption("Mayormente homo", "mostly_homo"),
					huh.NewOption("Completamente homo", "homo"),
					huh.NewOption("Asexual", "asexual"),
					huh.NewOption("Prefiero no definir", "procedural"),
				),
			huh.NewSelect[string]().
				Title("Intensidad sexual").
				Value(&m.sexualIntensityChoice).
				Options(
					huh.NewOption("Muy baja (asexual)", "very_low"),
					huh.NewOption("Normal", "normal"),
					huh.NewOption("Alta", "high"),
					huh.NewOption("Muy alta", "very_high"),
					huh.NewOption("Procedural", "procedural"),
				),
		).Title("Paso 2 - Orientacion"),
	).WithTheme(menuModalTheme())
}

func (m CreatePobleModel) archetypeForm() *huh.Form {
	options := make([]huh.Option[string], 0, len(allArchetypeOptions()))
	for _, item := range allArchetypeOptions() {
		label := fmt.Sprintf("%s - %s", item.Label, item.Short)
		options = append(options, huh.NewOption(label, item.ID))
	}

	return huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Arquetipo").
				Value(&m.archetypeChoice).
				Options(options...),
			huh.NewNote().
				Title("Lectura larga del arquetipo").
				DescriptionFunc(func() string {
					return archetypeLongDescription(m.archetypeChoice)
				}, &m.archetypeChoice),
		).Title("Paso 3 - Arquetipo"),
	).WithTheme(menuModalTheme())
}

func (m CreatePobleModel) historyForm() *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewText().
				Title("Hay algo que este personaje sepa que nadie mas sabe?").
				Description("Opcional. Esto siembra el secreto inicial.").
				Value(&m.secretSeed).
				Lines(4).
				ExternalEditor(false),
			huh.NewText().
				Title("A quien esta basado este personaje?").
				Description("Opcional. Solo para notas internas del jugador.").
				Value(&m.basedOn).
				Lines(3).
				ExternalEditor(false),
		).Title("Paso 4 - Historia"),
	).WithTheme(menuModalTheme())
}

func (m CreatePobleModel) confirmForm() *huh.Form {
	m.confirmed = false
	return huh.NewForm(
		huh.NewGroup(
			huh.NewNote().
				Title("Resumen final").
				Description(m.renderPreviewSheet()),
			huh.NewConfirm().
				Title("Este personaje te parece bien?").
				Affirmative("Si, al mundo").
				Negative("No, revisar").
				Value(&m.confirmed),
		).Title("Paso 5 - Confirmacion"),
	).WithTheme(menuModalTheme())
}

func (m CreatePobleModel) revisionForm() *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Que quieres revisar?").
				Value(&m.revisionChoice).
				Options(
					huh.NewOption("Volver al Paso 3", "step3"),
					huh.NewOption("Volver al Paso 1", "step1"),
				),
		).Title("Revisar ficha"),
	).WithTheme(menuModalTheme())
}

func (m *CreatePobleModel) generatePreview() error {
	age, err := strconv.Atoi(strings.TrimSpace(m.ageInput))
	if err != nil || age < 18 || age > 80 {
		return fmt.Errorf("la edad debe estar entre 18 y 80")
	}

	config := entities.PoblConfig{
		Name:     strings.TrimSpace(m.name),
		AgeRange: [2]int{age, age},
	}

	if sex := sexFromChoice(m.sexChoice); sex != nil {
		config.Sex = sex
	}
	if romantic, sexual := orientationFromChoice(m.romanticChoice); romantic != nil || sexual != nil {
		config.RomanticOrientation = romantic
		config.SexualOrientation = sexual
	}
	config.InspirationNotes = joinInspirationNotes(m.secretSeed, m.basedOn)

	customArchetype := false
	if archetype, custom := archetypeFromChoice(m.archetypeChoice); archetype != nil {
		config.Archetype = archetype
		customArchetype = custom
	} else {
		customArchetype = custom
	}

	preview, err := entities.GeneratePople(config, rand.New(rand.NewSource(time.Now().UnixNano())))
	if err != nil {
		return fmt.Errorf("no pude generar el personaje: %w", err)
	}

	if customArchetype {
		preview.Archetype = entities.ArchetypeCustom
	}
	applyIntensityChoice(preview, m.sexualIntensityChoice, m.romanticChoice)
	seedInitialSecret(preview, m.secretSeed)

	m.preview = preview
	m.previewConfig = config
	return nil
}

func (m CreatePobleModel) renderPreviewSheet() string {
	if m.preview == nil {
		return "Todavia no hay previsualizacion."
	}

	p := m.preview
	lines := []string{
		renderSheetLine("Nombre", p.Name),
		renderSheetLine("Sexo", p.Sex.String()),
		renderSheetLine("Edad", fmt.Sprintf("%d", p.Age)),
		renderSheetLine("Arquetipo", strings.Title(strings.ToLower(p.Archetype.String()))),
		renderSheetLine("Orientacion", summarizeOrientation(p)),
		renderSheetLine("Mood inicial", strings.Title(strings.ToLower(p.CurrentMood.String()))),
		"",
		"Personalidad",
		renderTraitBar("Openness", p.Personality.Openness),
		renderTraitBar("Extra", p.Personality.Extraversion),
		renderTraitBar("Agree", p.Personality.Agreeableness),
		renderTraitBar("Neuro", p.Personality.Neuroticism),
		renderTraitBar("Ambition", p.Personality.Ambition),
		renderTraitBar("Jealousy", p.Personality.Jealousy),
		renderTraitBar("Horniness", p.Personality.Horniness),
		"",
		"Secreto inicial",
		firstSecretPreview(p),
	}

	if strings.TrimSpace(p.Appearance) != "" {
		lines = append(lines, "", "Apariencia", trimMenuText(p.Appearance, 220))
	}
	if strings.TrimSpace(m.basedOn) != "" {
		lines = append(lines, "", "Nota interna", trimMenuText(strings.TrimSpace(m.basedOn), 140))
	}
	return strings.Join(lines, "\n")
}

func (m CreatePobleModel) stepLabel() string {
	switch m.step {
	case createStepBasic:
		return "Paso 1 de 5 - Info basica"
	case createStepOrientation:
		return "Paso 2 de 5 - Orientacion"
	case createStepArchetype:
		return "Paso 3 de 5 - Arquetipo"
	case createStepHistory:
		return "Paso 4 de 5 - Historia"
	case createStepConfirm:
		return "Paso 5 de 5 - Confirmacion"
	case createStepRevision:
		return "Revision"
	default:
		return "Creador"
	}
}

func validateFounderAge(value string) error {
	age, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return fmt.Errorf("escribe un numero")
	}
	if age < 18 || age > 80 {
		return fmt.Errorf("usa una edad entre 18 y 80")
	}
	return nil
}

func sexFromChoice(choice string) *entities.Sex {
	switch choice {
	case "hombre":
		sex := entities.Male
		return &sex
	case "mujer":
		sex := entities.Female
		return &sex
	case "intersex":
		sex := entities.Intersex
		return &sex
	default:
		return nil
	}
}

func orientationFromChoice(choice string) (*float32, *float32) {
	var value float32
	switch choice {
	case "hetero":
		value = 0.0
	case "mostly_hetero":
		value = 0.25
	case "bi":
		value = 0.5
	case "mostly_homo":
		value = 0.75
	case "homo":
		value = 1.0
	case "asexual":
		value = 0.5
	default:
		return nil, nil
	}
	return &value, &value
}

func archetypeFromChoice(choice string) (*entities.ArchetypeID, bool) {
	switch choice {
	case "ruler":
		value := entities.ArchetypeRuler
		return &value, false
	case "lover":
		value := entities.ArchetypeLover
		return &value, false
	case "jester":
		value := entities.ArchetypeJester
		return &value, false
	case "sage":
		value := entities.ArchetypeSage
		return &value, false
	case "rebel":
		value := entities.ArchetypeRebel
		return &value, false
	case "caretaker":
		value := entities.ArchetypeCaretaker
		return &value, false
	case "villain":
		value := entities.ArchetypeVillain
		return &value, false
	case "ghost":
		value := entities.ArchetypeGhost
		return &value, false
	case "addict":
		value := entities.ArchetypeAddict
		return &value, false
	case "prophet":
		value := entities.ArchetypeProphet
		return &value, false
	case "schemer":
		value := entities.ArchetypeSchemer
		return &value, false
	case "innocent":
		value := entities.ArchetypeInnocent
		return &value, false
	case "warrior":
		value := entities.ArchetypeWarrior
		return &value, false
	case "drifter":
		value := entities.ArchetypeDrifter
		return &value, false
	case "mirror":
		value := entities.ArchetypeMirror
		return &value, false
	case "custom":
		return nil, true
	default:
		return nil, false
	}
}

func applyIntensityChoice(poble *entities.Poble, intensityChoice, romanticChoice string) {
	if poble == nil {
		return
	}

	switch intensityChoice {
	case "very_low":
		poble.Orientation.Intensity = 0.05
		poble.Personality.Horniness = 8
	case "normal":
		poble.Orientation.Intensity = 0.50
		poble.Personality.Horniness = 50
	case "high":
		poble.Orientation.Intensity = 0.78
		poble.Personality.Horniness = 74
	case "very_high":
		poble.Orientation.Intensity = 0.95
		poble.Personality.Horniness = 92
	}

	if romanticChoice == "asexual" {
		poble.Orientation.Intensity = 0.03
		poble.Personality.Horniness = 5
	}
}

func seedInitialSecret(poble *entities.Poble, secretSeed string) {
	secretSeed = strings.TrimSpace(secretSeed)
	if poble == nil || secretSeed == "" {
		return
	}

	if len(poble.Secrets) == 0 {
		poble.Secrets = append(poble.Secrets, entities.NewSecret(
			fmt.Sprintf("secret_%d", time.Now().UnixNano()),
			entities.SecretTraumaEvent,
			secretSeed,
		))
		return
	}
	poble.Secrets[0].Content = secretSeed
}

func joinInspirationNotes(secretSeed, basedOn string) string {
	parts := []string{}
	if trimmed := strings.TrimSpace(secretSeed); trimmed != "" {
		parts = append(parts, "Secreto base: "+trimmed)
	}
	if trimmed := strings.TrimSpace(basedOn); trimmed != "" {
		parts = append(parts, "Inspiracion: "+trimmed)
	}
	return strings.Join(parts, " | ")
}

func renderSheetLine(label, value string) string {
	return lipgloss.NewStyle().Foreground(secondaryColor).Render(label+": ") + value
}

func renderTraitBar(label string, value float32) string {
	filled := int((value / 100) * 12)
	if filled < 0 {
		filled = 0
	}
	if filled > 12 {
		filled = 12
	}
	bar := strings.Repeat("█", filled) + strings.Repeat("░", 12-filled)
	return fmt.Sprintf("%-10s %s %3.0f", label, bar, value)
}

func summarizeOrientation(poble *entities.Poble) string {
	if poble == nil {
		return "procedural"
	}

	lean := "bi"
	switch {
	case poble.Orientation.Romantic <= 0.10:
		lean = "hetero"
	case poble.Orientation.Romantic <= 0.35:
		lean = "mostly hetero"
	case poble.Orientation.Romantic >= 0.90:
		lean = "homo"
	case poble.Orientation.Romantic >= 0.65:
		lean = "mostly homo"
	}

	intensity := "normal"
	switch {
	case poble.Orientation.Intensity <= 0.10:
		intensity = "muy baja"
	case poble.Orientation.Intensity >= 0.85:
		intensity = "muy alta"
	case poble.Orientation.Intensity >= 0.65:
		intensity = "alta"
	}
	return lean + " / intensidad " + intensity
}

func firstSecretPreview(poble *entities.Poble) string {
	if poble == nil || len(poble.Secrets) == 0 {
		return "No se genero un secreto claro todavia."
	}
	return trimMenuText(poble.Secrets[0].Content, 220)
}

func allArchetypeOptions() []archetypeOption {
	return []archetypeOption{
		{ID: "ruler", Label: "Ruler", Short: "controla el aire de la habitacion", Description: "Necesita ordenar el caos y enseguida convierte cualquier grupo en una estructura con jerarquia, reglas y territorio."},
		{ID: "lover", Label: "Lover", Short: "vive pegado al deseo", Description: "Lee el mundo a traves del vinculo, la intimidad y el hambre de sentirse elegido aunque eso le meta en problemas."},
		{ID: "jester", Label: "Jester", Short: "tapa el vacio con ruido", Description: "Hace chistes, revuelve dinamicas y usa el humor para esquivar el dolor, la verguenza o el silencio pesado."},
		{ID: "sage", Label: "Sage", Short: "piensa antes de tocar", Description: "Observa, analiza y suele querer entenderlo todo antes de moverse, incluso cuando la vida pide menos teoria y mas cuerpo."},
		{ID: "rebel", Label: "Rebel", Short: "se pica con cualquier orden", Description: "Lleva la contraria por instinto, a veces por justicia y a veces solo porque obedecer le sabe a derrota."},
		{ID: "caretaker", Label: "Caretaker", Short: "carga el peso de los demas", Description: "Protege, cuida y remienda, pero esa entrega puede volverse control, culpa o agotamiento si nadie le devuelve nada."},
		{ID: "villain", Label: "Villain", Short: "se permite cruzar lineas", Description: "Tolera la crueldad mejor que otros y puede convertir el resentimiento o la ambicion en una forma real de poder."},
		{ID: "ghost", Label: "Ghost", Short: "vive medio ido", Description: "Se retira, observa desde el borde y a veces parece no estar, pero justo ahi suele guardar lo que mas pesa."},
		{ID: "addict", Label: "Addict", Short: "persigue compulsiones", Description: "Tiene patrones de alivio inmediato y repeticion peligrosa; el deseo le manda mas de lo que le gustaria admitir."},
		{ID: "prophet", Label: "Prophet", Short: "convierte intuicion en destino", Description: "Siente que ve algo mas grande que el presente y puede arrastrar a otros a creer, seguir o temer esa vision."},
		{ID: "schemer", Label: "Schemer", Short: "siempre va dos jugadas delante", Description: "Piensa en capas, mide costos y suele preferir la maniobra elegante a la confrontacion directa."},
		{ID: "innocent", Label: "Innocent", Short: "quiere creer en algo limpio", Description: "Sostiene bondad o ingenuidad incluso donde no conviene, lo que puede volverlo tierno, ciego o peligrosamente facil de herir."},
		{ID: "warrior", Label: "Warrior", Short: "resuelve de frente", Description: "Entra al conflicto sin temblar mucho y convierte la friccion en energia, proteccion o dominacion."},
		{ID: "drifter", Label: "Drifter", Short: "nunca termina de quedarse", Description: "Se mueve liviano, cambia de centro facil y puede parecer libre o incapaz de echar raices."},
		{ID: "mirror", Label: "Mirror", Short: "absorbe a quien tiene delante", Description: "Se adapta, refleja y aprende rapido las formas ajenas, a veces tanto que le cuesta saber donde termina la otra persona y donde empieza el."},
		{ID: "custom", Label: "Custom", Short: "mezcla libre con base procedural", Description: "Sirve cuando quieres salirte del molde. El motor usa una base procedural, pero te deja empujar el personaje con tus notas."},
		{ID: "sorprendeme", Label: "Sorprendeme", Short: "random weighted", Description: "El generador elige con pesos narrativos normales del juego para darte algo coherente y con drama util."},
	}
}

func archetypeLongDescription(choice string) string {
	for _, option := range allArchetypeOptions() {
		if option.ID == choice {
			return option.Description
		}
	}
	return "Dejalo procedural si quieres que el juego decida el tono base del personaje."
}
