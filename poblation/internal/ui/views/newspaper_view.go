package views

import (
	"fmt"
	"math/rand"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/user/poblation/internal/entities"
	"github.com/user/poblation/internal/world"
)

// NewspaperEdition stores one printable daily edition.
type NewspaperEdition struct {
	Day      entities.GameTime
	Title    string
	Sections []NewsSection
}

// NewsSection stores one newspaper section and its articles.
type NewsSection struct {
	Name     string
	Articles []NewsArticle
}

// NewsArticle stores one short newspaper article.
type NewsArticle struct {
	Headline    string
	Body        string
	SourceEvent world.GameEvent
}

// NewspaperModel renders the journalist mode as a text-first newspaper.
type NewspaperModel struct {
	Editions       []NewspaperEdition
	CurrentEdition *NewspaperEdition
	state          AppStateSnapshot
	viewport       viewport.Model
}

var (
	newspaperFrameStyle = lipgloss.NewStyle().
				Border(lipgloss.DoubleBorder()).
				BorderForeground(lipgloss.Color("#B88A44")).
				Background(lipgloss.Color("#F3E7C9")).
				Foreground(lipgloss.Color("#2B2116")).
				Padding(1, 2)

	newspaperNameplateStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#5A2D0C")).
				Align(lipgloss.Center)

	newspaperRuleStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#8D6E3F"))

	newspaperSectionStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#6B1F14"))

	newspaperHeadlineStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#1E1A16"))

	newspaperBodyStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#332A22"))

	newspaperMutedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#766552"))
)

// NewNewspaperModel builds the newspaper viewport.
func NewNewspaperModel() NewspaperModel {
	return NewspaperModel{
		Editions: []NewspaperEdition{},
		viewport: viewport.New(72, 18),
	}
}

// Init satisfies tea.Model.
func (m NewspaperModel) Init() tea.Cmd {
	return nil
}

// SyncAppState stores the latest app snapshot and resizes the paper.
func (m NewspaperModel) SyncAppState(snapshot AppStateSnapshot) tea.Model {
	m.state = snapshot
	m.resize()
	return m
}

func (m NewspaperModel) Resize(width, height int) tea.Model {
	m.state.Width = width
	m.state.Height = height
	m.resize()
	return m
}

// OnEnter generates or refreshes the current daily edition.
func (m NewspaperModel) OnEnter() (tea.Model, tea.Cmd) {
	if m.state.World == nil {
		return m, nil
	}
	edition := GenerateDailyEdition(m.state.World, m.state.World.Calendar)
	m.CurrentEdition = &edition
	m.rememberEdition(edition)
	m.viewport.SetContent(m.renderEdition())
	m.viewport.GotoTop()
	return m, nil
}

// Update handles scrolling and regeneration.
func (m NewspaperModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch typed := msg.(type) {
	case tea.WindowSizeMsg:
		return m.Resize(typed.Width, typed.Height), nil
	case tea.KeyMsg:
		switch typed.String() {
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
		case "r":
			if m.state.World != nil {
				edition := GenerateDailyEdition(m.state.World, m.state.World.Calendar)
				m.CurrentEdition = &edition
				m.rememberEdition(edition)
			}
		}
	}
	m.viewport.SetContent(m.renderEdition())
	return m, nil
}

// View renders the full newspaper surface.
func (m NewspaperModel) View() string {
	width := maxInt(28, m.state.Width-4)
	height := maxInt(10, m.state.Height-2)
	return newspaperFrameStyle.Width(width).Height(height).Render(m.viewport.View())
}

// GenerateDailyEdition converts the day's world events into a newspaper edition.
func GenerateDailyEdition(gameWorld *world.World, day entities.GameTime) NewspaperEdition {
	edition := NewspaperEdition{
		Day:      day,
		Title:    newspaperTitle(gameWorld),
		Sections: []NewsSection{},
	}
	if gameWorld == nil {
		return edition
	}

	todayEvents := dailySignificantEvents(gameWorld, day)
	if len(todayEvents) == 0 {
		edition.Sections = append(edition.Sections,
			NewsSection{
				Name: "PRINCIPAL",
				Articles: []NewsArticle{{
					Headline: "La isla se tomo el dia para respirar",
					Body:     "No hubo un escandalo lo bastante grande para robarse la portada. Eso no significa paz; solo significa que el desastre fue discreto.",
				}},
			},
			buildClimateSection(gameWorld, day),
			buildEditorialSection(gameWorld, day, nil),
		)
		return edition
	}

	sort.SliceStable(todayEvents, func(i, j int) bool {
		return eventScore(todayEvents[i]) > eventScore(todayEvents[j])
	})

	leadArticles, socialArticles, healthArticles, economyArticles := categorizeArticles(gameWorld, todayEvents)
	obits := buildObituaryArticles(gameWorld, todayEvents)
	climate := buildClimateSection(gameWorld, day)
	editorial := buildEditorialSection(gameWorld, day, todayEvents)

	if len(leadArticles) > 0 {
		edition.Sections = append(edition.Sections, NewsSection{Name: "PRINCIPAL", Articles: leadArticles})
	}
	if len(socialArticles) > 0 {
		edition.Sections = append(edition.Sections, NewsSection{Name: "SOCIAL", Articles: socialArticles})
	}
	if len(healthArticles) > 0 {
		edition.Sections = append(edition.Sections, NewsSection{Name: "SALUD", Articles: healthArticles})
	}
	if len(economyArticles) > 0 {
		edition.Sections = append(edition.Sections, NewsSection{Name: "ECONOMIA", Articles: economyArticles})
	}
	if len(obits) > 0 {
		edition.Sections = append(edition.Sections, NewsSection{Name: "NECROLOGICA", Articles: obits})
	}
	edition.Sections = append(edition.Sections, climate, editorial)
	return edition
}

func (m *NewspaperModel) resize() {
	m.viewport.Width = maxInt(28, m.state.Width-10)
	m.viewport.Height = maxInt(8, m.state.Height-8)
}

func (m *NewspaperModel) rememberEdition(edition NewspaperEdition) {
	for i := range m.Editions {
		if m.Editions[i].Day.Day == edition.Day.Day {
			m.Editions[i] = edition
			return
		}
	}
	m.Editions = append(m.Editions, edition)
	sort.SliceStable(m.Editions, func(i, j int) bool {
		return m.Editions[i].Day.ToMinutes() < m.Editions[j].Day.ToMinutes()
	})
}

func (m NewspaperModel) renderEdition() string {
	if m.CurrentEdition == nil {
		return "No hay edicion impresa todavia."
	}

	width := maxInt(24, m.viewport.Width-2)
	header := []string{
		newspaperNameplateStyle.Width(width).Render(fmt.Sprintf("%s · Dia %d", m.CurrentEdition.Title, m.CurrentEdition.Day.Day)),
		newspaperRuleStyle.Width(width).Render(strings.Repeat("=", maxInt(12, width-2))),
	}

	body := make([]string, 0, len(m.CurrentEdition.Sections)+8)
	for _, section := range m.CurrentEdition.Sections {
		body = append(body, newspaperSectionStyle.Render(section.Name))
		for _, article := range section.Articles {
			body = append(body, newspaperHeadlineStyle.Render(article.Headline))
			body = append(body, newspaperBodyStyle.Width(width).Render(article.Body))
			body = append(body, "")
		}
	}
	body = append(body, newspaperMutedStyle.Render("Scroll: ↑ ↓ PgUp PgDn · R para reimprimir"))

	return lipgloss.JoinVertical(lipgloss.Left, append(header, body...)...)
}

func newspaperTitle(gameWorld *world.World) string {
	if gameWorld == nil {
		return "EL POBLE TIMES"
	}
	switch gameWorld.Era {
	case entities.EraZero:
		return "EL POBLE TIMES"
	case entities.EraOne:
		return "LA GACETA DEL ORIGEN"
	case entities.EraTwo:
		return "EL ECO DE LA ISLA"
	case entities.EraThree:
		return "EL POBLE HERALD"
	case entities.EraFour:
		return "LA VOZ CIVIL"
	default:
		return "EL POBLE TIMES"
	}
}

func dailySignificantEvents(gameWorld *world.World, day entities.GameTime) []world.GameEvent {
	today := make([]world.GameEvent, 0, 12)
	seen := map[string]bool{}
	for _, event := range append([]world.GameEvent{}, gameWorld.EventHistory...) {
		if event.Time.Day != day.Day || seen[event.ID] {
			continue
		}
		today = append(today, event)
		seen[event.ID] = true
	}
	for _, event := range append([]world.GameEvent{}, gameWorld.ActiveEvents...) {
		if event.Time.Day != day.Day || seen[event.ID] {
			continue
		}
		today = append(today, event)
		seen[event.ID] = true
	}
	sort.SliceStable(today, func(i, j int) bool {
		return eventScore(today[i]) > eventScore(today[j])
	})
	if len(today) > 12 {
		today = today[:12]
	}
	return today
}

func categorizeArticles(gameWorld *world.World, events []world.GameEvent) ([]NewsArticle, []NewsArticle, []NewsArticle, []NewsArticle) {
	lead := []NewsArticle{}
	social := []NewsArticle{}
	health := []NewsArticle{}
	economy := []NewsArticle{}

	for _, event := range events {
		article := articleFromEvent(gameWorld, event)
		switch sectionForEvent(event) {
		case "PRINCIPAL":
			lead = append(lead, article)
		case "SOCIAL":
			social = append(social, article)
		case "SALUD":
			health = append(health, article)
		case "ECONOMIA":
			economy = append(economy, article)
		}
	}

	lead = trimArticles(lead, 4)
	social = trimArticles(social, 3)
	health = trimArticles(health, 2)
	economy = trimArticles(economy, 2)
	return lead, social, health, economy
}

func buildObituaryArticles(gameWorld *world.World, events []world.GameEvent) []NewsArticle {
	obits := []NewsArticle{}
	for _, event := range events {
		if !isDeathEvent(event) {
			continue
		}
		name := eventPrimaryName(gameWorld, event)
		survivors := survivorLine(gameWorld, event)
		obits = append(obits, NewsArticle{
			Headline: fmt.Sprintf("Murio %s", name),
			Body: fmt.Sprintf("%s salio de la historia hoy. %s La isla ya encontro una forma incomoda de seguir sin esa persona.",
				name,
				survivors,
			),
			SourceEvent: event,
		})
	}
	return trimArticles(obits, 6)
}

func buildClimateSection(gameWorld *world.World, day entities.GameTime) NewsSection {
	todayClimate := []world.GameEvent{}
	yesterdayWarningIgnored := false

	for _, event := range append([]world.GameEvent{}, gameWorld.EventHistory...) {
		if event.Time.Day == day.Day && isClimateEvent(event) {
			todayClimate = append(todayClimate, event)
		}
		if event.Time.Day == day.Day-1 && isClimateEvent(event) {
			yesterdayWarningIgnored = true
		}
	}

	body := "El cielo no prometio nada raro hoy. Eso suele ser cuando mas confianza da para hacer tonterias."
	headline := "Clima sospechosamente utilizable"
	if len(todayClimate) > 0 {
		headline = "Clima con ganas de arruinar planes"
		body = climateBody(todayClimate)
	} else if yesterdayWarningIgnored {
		body = "Ayer hubo advertencias y, como era costumbre, alguien decidio usarlas como sugerencia decorativa. Hoy el clima parece tranquilo, lo cual no borra la humillacion anterior."
	}

	return NewsSection{
		Name: "CLIMA",
		Articles: []NewsArticle{{
			Headline: headline,
			Body:     body,
		}},
	}
}

func buildEditorialSection(gameWorld *world.World, day entities.GameTime, todayEvents []world.GameEvent) NewsSection {
	writer := randomLivingPoble(gameWorld, day)
	if writer == nil {
		return NewsSection{
			Name: "EDITORIAL",
			Articles: []NewsArticle{{
				Headline: "La opinion de nadie",
				Body:     "Nadie quiso opinar hoy, lo cual ya es una opinion politica bastante fuerte.",
			}},
		}
	}

	topic := "que las sillas de la plaza crujen demasiado"
	if len(todayEvents) > 0 {
		topic = editorialTopic(todayEvents[0])
	}
	body := fmt.Sprintf("%s escribio que lo verdaderamente importante del dia fue %s. Puede que tenga razon. Puede que solo necesitara una excusa para hablar de otra cosa.", writer.Name, topic)
	headline := fmt.Sprintf("Editorial: %s opina con mucha seguridad", writer.Name)
	return NewsSection{
		Name: "EDITORIAL",
		Articles: []NewsArticle{{
			Headline: headline,
			Body:     body,
		}},
	}
}

func articleFromEvent(gameWorld *world.World, event world.GameEvent) NewsArticle {
	return NewsArticle{
		Headline:    headlineForEvent(gameWorld, event),
		Body:        bodyForEvent(gameWorld, event),
		SourceEvent: event,
	}
}

func headlineForEvent(gameWorld *world.World, event world.GameEvent) string {
	name := eventPrimaryName(gameWorld, event)
	switch {
	case isDeathEvent(event):
		return fmt.Sprintf("Luto en la isla: cae %s", name)
	case hasAnyTag(event.Tags, "war", "violence") || event.Type == "CONFLICT" || event.Type == "THREAT":
		return fmt.Sprintf("Tension abierta: %s deja de fingir calma", name)
	case hasAnyTag(event.Tags, "technology"):
		return fmt.Sprintf("Progreso con ego: %s altera el futuro", cleanSentence(event.Description))
	case hasAnyTag(event.Tags, "trade", "currency", "inheritance", "gambling"):
		return fmt.Sprintf("Dinero con drama: %s", punchyEconomyTitle(event.Description))
	case hasAnyTag(event.Tags, "theft", "law"):
		return fmt.Sprintf("Economia creativa: %s", cleanSentence(event.Description))
	case hasAnyTag(event.Tags, "plague", "sick", "illness", "medicine"):
		return fmt.Sprintf("Salud publica en modo improvisacion: %s", name)
	case hasAnyTag(event.Tags, "birth"):
		return fmt.Sprintf("Llega alguien nuevo y ya hereda problemas: %s", name)
	default:
		return cleanHeadline(event.Description, string(event.Type))
	}
}

func bodyForEvent(gameWorld *world.World, event world.GameEvent) string {
	base := cleanSentence(event.Description)
	if base == "" {
		base = "El suceso ocurrio y todavia nadie logro contarlo sin exagerar."
	}
	name := eventPrimaryName(gameWorld, event)
	second := eventSecondaryName(gameWorld, event)

	switch {
	case isDeathEvent(event):
		return fmt.Sprintf("%s marco el dia con peso propio. %s %s", base, survivorLine(gameWorld, event), closingTone(event))
	case hasAnyTag(event.Tags, "war", "violence") || event.Type == "CONFLICT" || event.Type == "THREAT":
		return fmt.Sprintf("%s %s y %s empujaron la jornada hacia el tipo de titular que nadie quiere ver repetido demasiado seguido.", base, name, fallbackPerson(second, "alguien mas"))
	case hasAnyTag(event.Tags, "technology"):
		return fmt.Sprintf("%s La noticia entro con olor a avance y a consecuencias futuras que todavia no caben completas en una sola edicion.", base)
	case hasAnyTag(event.Tags, "trade", "currency", "inheritance", "gambling", "theft", "law"):
		return fmt.Sprintf("%s El bolsillo colectivo reacciono peor que la dignidad publica, que ya venia frágil de antes.", base)
	case hasAnyTag(event.Tags, "plague", "sick", "illness", "medicine"):
		return fmt.Sprintf("%s La salud de la comunidad vuelve a depender de si alguien decidio tomarse en serio lo obvio a tiempo.", base)
	default:
		return fmt.Sprintf("%s Nadie admite que esto les importo tanto, pero todo el pueblo ya lo esta repitiendo con detalles nuevos.", base)
	}
}

func sectionForEvent(event world.GameEvent) string {
	switch {
	case isDeathEvent(event), hasAnyTag(event.Tags, "war", "violence"), event.Type == "CONFLICT", event.Type == "THREAT":
		return "PRINCIPAL"
	case hasAnyTag(event.Tags, "illness", "sick", "medicine", "plague", "birth"):
		return "SALUD"
	case hasAnyTag(event.Tags, "trade", "currency", "inheritance", "gambling", "theft", "law"):
		return "ECONOMIA"
	default:
		return "SOCIAL"
	}
}

func eventScore(event world.GameEvent) float32 {
	score := event.Severity * 100
	switch {
	case isDeathEvent(event):
		score += 160
	case hasAnyTag(event.Tags, "war", "violence"):
		score += 130
	case event.Type == "CONFLICT" || event.Type == "THREAT":
		score += 110
	case hasAnyTag(event.Tags, "illness", "plague"):
		score += 80
	case hasAnyTag(event.Tags, "technology"):
		score += 70
	case hasAnyTag(event.Tags, "trade", "currency", "inheritance", "gambling", "theft"):
		score += 60
	}
	return score + float32(len(event.Participants)*4)
}

func climateBody(events []world.GameEvent) string {
	chunks := make([]string, 0, len(events))
	for _, event := range events {
		switch {
		case strings.Contains(strings.ToLower(event.ID), "storm"):
			chunks = append(chunks, "Se reportan tormentas con talento para convertir advertencias en anecdotas humillantes.")
		case strings.Contains(strings.ToLower(event.ID), "drought"):
			chunks = append(chunks, "La sequia sigue recordando que el agua no obedece discursos.")
		default:
			chunks = append(chunks, cleanSentence(event.Description))
		}
	}
	return strings.Join(chunks, " ")
}

func survivorLine(gameWorld *world.World, event world.GameEvent) string {
	deceasedID := ""
	if len(event.Participants) > 0 {
		deceasedID = event.Participants[0]
	}
	deceased := findKnownPoble(gameWorld, deceasedID)
	if deceased == nil {
		return "Le sobreviven los que todavia pueden decir su nombre."
	}

	people := []string{}
	for _, candidate := range gameWorld.GetAllKnownPobles() {
		if candidate == nil || !candidate.IsAlive || candidate.ID == deceased.ID {
			continue
		}
		if rel, ok := candidate.Relationships[deceased.ID]; ok && (rel.Affection >= 55 || rel.Trust >= 55 || rel.Type == entities.RelationshipChild || rel.Type == entities.RelationshipParent) {
			people = append(people, candidate.Name)
		}
	}
	sort.Strings(people)
	if len(people) == 0 {
		return "No quedo claro quien sobrevive a esa ausencia mejor que el resto."
	}
	if len(people) == 1 {
		return fmt.Sprintf("Le sobrevive %s.", people[0])
	}
	if len(people) == 2 {
		return fmt.Sprintf("Le sobreviven %s y %s.", people[0], people[1])
	}
	return fmt.Sprintf("Le sobreviven %s y %d personas mas.", people[0], len(people)-1)
}

func editorialTopic(event world.GameEvent) string {
	switch {
	case isDeathEvent(event):
		return "que las ceremonias se estan volviendo demasiado teatrales"
	case hasAnyTag(event.Tags, "technology"):
		return "que las maquinas ya reciben mas entusiasmo que los vecinos"
	case hasAnyTag(event.Tags, "trade", "currency", "inheritance"):
		return "que el pan sube de precio y el orgullo nunca baja"
	case hasAnyTag(event.Tags, "war", "violence"):
		return "que discutir fuerte no te convierte automaticamente en estratega"
	default:
		return "si el pueblo deberia tener un mejor nombre para su plaza principal"
	}
}

func randomLivingPoble(gameWorld *world.World, day entities.GameTime) *entities.Poble {
	if gameWorld == nil {
		return nil
	}
	living := gameWorld.GetAllPobles()
	if len(living) == 0 {
		return nil
	}
	rng := rand.New(rand.NewSource(int64(day.ToMinutes() + len(living)*17 + 3)))
	return living[rng.Intn(len(living))]
}

func cleanHeadline(description, fallback string) string {
	desc := strings.TrimSpace(description)
	if desc == "" {
		return fallback
	}
	desc = strings.TrimSuffix(desc, ".")
	if len(desc) > 72 {
		desc = desc[:72]
	}
	return desc
}

func cleanSentence(description string) string {
	desc := strings.TrimSpace(description)
	if desc == "" {
		return ""
	}
	desc = strings.TrimSuffix(desc, ".")
	return desc + "."
}

func punchyEconomyTitle(description string) string {
	desc := strings.TrimSpace(description)
	if desc == "" {
		return "alguien encontro otra forma emocional de hablar de plata"
	}
	return strings.TrimSuffix(desc, ".")
}

func isDeathEvent(event world.GameEvent) bool {
	return event.Type == "DEATH" || hasAnyTag(event.Tags, "death", "murder", "suicide", "war")
}

func isClimateEvent(event world.GameEvent) bool {
	return strings.Contains(strings.ToLower(event.ID), "storm") ||
		strings.Contains(strings.ToLower(event.ID), "drought") ||
		hasAnyTag(event.Tags, "weather", "storm", "drought", "flood", "fire")
}

func hasAnyTag(tags []string, targets ...string) bool {
	for _, tag := range tags {
		lowered := strings.ToLower(tag)
		for _, target := range targets {
			if lowered == strings.ToLower(target) {
				return true
			}
		}
	}
	return false
}

func findKnownPoble(gameWorld *world.World, id string) *entities.Poble {
	if gameWorld == nil || id == "" {
		return nil
	}
	for _, poble := range gameWorld.GetAllKnownPobles() {
		if poble != nil && poble.ID == id {
			return poble
		}
	}
	return nil
}

func eventPrimaryName(gameWorld *world.World, event world.GameEvent) string {
	if len(event.Participants) == 0 {
		return "el pueblo"
	}
	if target := findKnownPoble(gameWorld, event.Participants[0]); target != nil {
		return target.Name
	}
	return "alguien"
}

func eventSecondaryName(gameWorld *world.World, event world.GameEvent) string {
	if len(event.Participants) < 2 {
		return ""
	}
	if target := findKnownPoble(gameWorld, event.Participants[1]); target != nil {
		return target.Name
	}
	return ""
}

func fallbackPerson(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func closingTone(event world.GameEvent) string {
	if hasAnyTag(event.Tags, "murder", "suicide", "war") {
		return "Nadie salio ligero de esa noticia."
	}
	return "La noticia peso distinto segun quien la escucho, pero peso."
}

func trimArticles(articles []NewsArticle, limit int) []NewsArticle {
	if len(articles) <= limit {
		return articles
	}
	return articles[:limit]
}
