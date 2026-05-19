package engine

import (
	"fmt"
	"math"
	"math/rand"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/user/poblation/internal/ai"
	"github.com/user/poblation/internal/entities"
	"github.com/user/poblation/internal/templates"
	simworld "github.com/user/poblation/internal/world"
)

// World is the concrete runtime world the ending checker reads.
type World = simworld.World

// EndingType identifies every special ending from the GDD.
type EndingType string

const (
	END_LOVE       EndingType = "END_LOVE"
	END_DYNASTY    EndingType = "END_DYNASTY"
	END_WAR        EndingType = "END_WAR"
	END_UTOPIA     EndingType = "END_UTOPIA"
	END_CULT       EndingType = "END_CULT"
	END_ALONE      EndingType = "END_ALONE"
	END_RESET      EndingType = "END_RESET"
	END_MYTH       EndingType = "END_MYTH"
	END_PLAGUE     EndingType = "END_PLAGUE"
	END_MONOPOLY   EndingType = "END_MONOPOLY"
	END_QUARANTINE EndingType = "END_QUARANTINE"
	END_SCANDAL    EndingType = "END_SCANDAL"

	END_FAMILY_EMPIRE        EndingType = "END_FAMILY_EMPIRE"
	END_ETERNAL_DICTATORSHIP EndingType = "END_ETERNAL_DICTATORSHIP"
	END_FAILED_REVOLUTION    EndingType = "END_FAILED_REVOLUTION"
	END_ETERNAL_ARCHIVE      EndingType = "END_ETERNAL_ARCHIVE"
	END_ECONOMIC_COLLAPSE    EndingType = "END_ECONOMIC_COLLAPSE"
	END_EXODUS               EndingType = "END_EXODUS"
	END_TECH_ISOLATION       EndingType = "END_TECH_ISOLATION"
	END_LAST_CHILD           EndingType = "END_LAST_CHILD"
)

// EndingChecker owns threshold values for GDD ending conditions.
type EndingChecker struct {
	DynastyLeadershipShare float32
	DynastyGenerations     int
	UtopiaPopulation       int
	UtopiaMentalHealth     float32
	UtopiaPeaceDays        int
	UtopiaTechLevel        int
	CultAdherentShare      float32
	MonopolyWealthShare    float32
	QuarantineHealthShare  float32
}

// Ending stores the complete final screen payload.
type Ending struct {
	Type              EndingType
	Title             string
	NarrativeChapters []string
	Statistics        GameStatistics
	LastWords         string
}

// GameStatistics summarizes the played world at the final moment.
type GameStatistics struct {
	TotalDays            int
	TotalPobles          int
	TotalDeaths          int
	TotalBirths          int
	TotalWars            int
	TotalAffairs         int
	TotalSecretsRevealed int
	MostDramaticEvent    string
	LongestLivedPople    string
	LongestLivedAge      int
	MostLovedPople       string
	MostHatedPople       string
	RicherPople          string
	FirstCoupleNames     [2]string
	TotalRumours         int
	RumourMostMutated    string
}

type endingFacts struct {
	Stats          GameStatistics
	Living         []*entities.Poble
	Known          []*entities.Poble
	Dead           []*entities.Poble
	Subject        string
	SecondSubject  string
	Survivor       string
	Deceased       string
	LastEvent      string
	Settlement     string
	Era            string
	Tech           string
	Belief         string
	Leader         string
	Prophet        string
	Family         string
	TemplateSuffix string
}

// NewEndingChecker returns the GDD default thresholds.
func NewEndingChecker() EndingChecker {
	return EndingChecker{
		DynastyLeadershipShare: 0.60,
		DynastyGenerations:     10,
		UtopiaPopulation:       500,
		UtopiaMentalHealth:     70,
		UtopiaPeaceDays:        50 * 365,
		UtopiaTechLevel:        3,
		CultAdherentShare:      0.80,
		MonopolyWealthShare:    0.72,
		QuarantineHealthShare:  0.70,
	}
}

// CheckEndingConditions returns the first ending currently earned by the world.
func CheckEndingConditions(world *World) *Ending {
	checker := NewEndingChecker()
	return checker.CheckEndingConditions(world)
}

// CheckEndingConditions returns the first ending currently earned by the world.
func (c EndingChecker) CheckEndingConditions(world *World) *Ending {
	endingType, ok := c.DetectEndingType(world)
	if !ok {
		return nil
	}
	ending := GenerateEnding(endingType, world)
	return &ending
}

// DetectEndingType checks all GDD ending conditions without building narrative.
func (c EndingChecker) DetectEndingType(world *World) (EndingType, bool) {
	if world == nil {
		return "", false
	}
	for _, candidate := range []struct {
		ending EndingType
		ok     bool
	}{
		{END_PLAGUE, c.isPlagueEnding(world)},
		{END_QUARANTINE, c.isQuarantineEnding(world)},
		{END_WAR, c.isWarEnding(world)},
		{END_LOVE, c.isLoveEnding(world)},
		{END_LAST_CHILD, c.isLastChildEnding(world)},
		{END_ALONE, c.isAloneEnding(world)},
		{END_RESET, c.isResetEnding(world)},
		{END_ECONOMIC_COLLAPSE, c.isEconomicCollapseEnding(world)},
		{END_EXODUS, c.isExodusEnding(world)},
		{END_TECH_ISOLATION, c.isTechIsolationEnding(world)},
		{END_FAILED_REVOLUTION, c.isFailedRevolutionEnding(world)},
		{END_ETERNAL_DICTATORSHIP, c.isEternalDictatorshipEnding(world)},
		{END_ETERNAL_ARCHIVE, c.isEternalArchiveEnding(world)},
		{END_FAMILY_EMPIRE, c.isFamilyEmpireEnding(world)},
		{END_MYTH, c.isMythEnding(world)},
		{END_SCANDAL, c.isScandalEnding(world)},
		{END_MONOPOLY, c.isMonopolyEnding(world)},
		{END_UTOPIA, c.isUtopiaEnding(world)},
		{END_CULT, c.isCultEnding(world)},
		{END_DYNASTY, c.isDynastyEnding(world)},
	} {
		if candidate.ok {
			return candidate.ending, true
		}
	}
	return "", false
}

// GenerateEnding builds final chapters using concrete facts from the world.
func GenerateEnding(endingType EndingType, world *World) Ending {
	facts := collectEndingFacts(endingType, world)
	title := endingTitle(endingType)
	extra := endingExtraVars(endingType, title, facts)
	chapters := renderEndingChapters(facts.TemplateSuffix, world, extra)
	lastWords := renderEndingLastWords(facts.TemplateSuffix, world, extra)

	return Ending{
		Type:              endingType,
		Title:             title,
		NarrativeChapters: chapters,
		Statistics:        facts.Stats,
		LastWords:         lastWords,
	}
}

func (c EndingChecker) isLoveEnding(world *World) bool {
	living := world.GetAllPobles()
	known := world.GetAllKnownPobles()
	return len(living) == 1 &&
		len(known) == 2 &&
		hasDeathEvent(world) &&
		hasReproductionRejection(world)
}

func (c EndingChecker) isDynastyEnding(world *World) bool {
	leader := currentLeader(world)
	if leader == nil {
		return false
	}
	generation := generationDepth(leader, knownPobleMap(world), map[string]bool{})
	if generation < c.DynastyGenerations {
		return false
	}
	return familyPowerShare(world, leader) >= c.DynastyLeadershipShare
}

func (c EndingChecker) isFamilyEmpireEnding(world *World) bool {
	if world == nil || world.GetPopulation() < 20 {
		return false
	}
	leader := currentLeader(world)
	if leader == nil {
		return false
	}
	return generationDepth(leader, knownPobleMap(world), map[string]bool{}) >= 4 &&
		familyPowerShare(world, leader) >= 0.75 &&
		world.Government != nil &&
		(world.Government.Type == simworld.GovernmentMonarchy || world.Government.Type == simworld.GovernmentOligarchy || world.Government.Type == simworld.GovernmentDictatorship)
}

func (c EndingChecker) isWarEnding(world *World) bool {
	if world.GetPopulation() != 0 {
		return false
	}
	return hasEventMatching(world, "war", "combat", "violence", "fight")
}

func (c EndingChecker) isPlagueEnding(world *World) bool {
	if world.GetPopulation() != 0 {
		return false
	}
	return hasEventMatching(world, "plague", "illness", "infection", "sickness")
}

func (c EndingChecker) isMonopolyEnding(world *World) bool {
	if world.GetPopulation() < 5 || !economyCanMonopolize(world) {
		return false
	}
	_, share := richestWealthShare(world)
	return share >= c.MonopolyWealthShare
}

func (c EndingChecker) isQuarantineEnding(world *World) bool {
	if world == nil || world.GetPopulation() < 5 {
		return false
	}
	if !hasEventMatching(world, "sti", "illness", "plague", "infection", "sickness") {
		return false
	}
	return healthCrisisShare(world) >= c.QuarantineHealthShare
}

func (c EndingChecker) isScandalEnding(world *World) bool {
	if world == nil || world.GetPopulation() < 6 {
		return false
	}
	affairs := countEvents(world, "affair", "encounter", "secret")
	reveals := countEvents(world, "revelation", "secret", "witness")
	return affairs >= maxInt(3, world.GetPopulation()/2) &&
		(reveals >= maxInt(2, world.GetPopulation()/3) || len(world.RumourPool) >= maxInt(3, world.GetPopulation()/2))
}

func (c EndingChecker) isEternalDictatorshipEnding(world *World) bool {
	if world == nil || world.Government == nil || world.Government.Leader == nil {
		return false
	}
	if world.Government.Type != simworld.GovernmentDictatorship && world.Government.Type != simworld.GovernmentMonarchy {
		return false
	}
	daysInPower := world.Calendar.Day - world.Government.EstablishedAt.Day
	return daysInPower >= 365 &&
		world.Government.Stability >= 70 &&
		world.Government.Legitimacy <= 55 &&
		currentLeader(world) != nil
}

func (c EndingChecker) isFailedRevolutionEnding(world *World) bool {
	if world == nil || world.Government == nil {
		return false
	}
	authoritarian := world.Government.Type == simworld.GovernmentDictatorship ||
		world.Government.Type == simworld.GovernmentOligarchy ||
		world.Government.Type == simworld.GovernmentMonarchy
	return authoritarian &&
		hasEventMatching(world, "revolution") &&
		(world.Government.Stability >= 55 || hasEventMatching(world, "coup"))
}

func (c EndingChecker) isEternalArchiveEnding(world *World) bool {
	if world == nil {
		return false
	}
	privatePages := 0
	for _, poble := range world.GetAllKnownPobles() {
		if poble == nil {
			continue
		}
		privatePages += len(poble.DiaryEntries) + len(poble.Letters) + len(poble.Memories)
	}
	return world.GetPopulation() == 0 &&
		techUnlocked(world, simworld.TechWriting) &&
		(len(world.EventHistory) >= 50 || privatePages >= 80)
}

func (c EndingChecker) isEconomicCollapseEnding(world *World) bool {
	if world == nil || world.GetPopulation() < 5 || !techUnlocked(world, simworld.TechCurrency) {
		return false
	}
	collapseSignals := countEvents(world, "scarcity", "debt", "theft", "resource", "monopoly")
	return collapseSignals >= maxInt(5, world.GetPopulation()/2) &&
		totalLivingWealth(world) <= maxInt(1, world.GetPopulation()/3)
}

func (c EndingChecker) isExodusEnding(world *World) bool {
	if world == nil || world.GetPopulation() == 0 || !techUnlocked(world, simworld.TechNavigation) {
		return false
	}
	return countEvents(world, "exodus", "island_travel", "migration", "navigation") >= 3 ||
		discoveredIslandCount(world) >= 4 && world.GetPopulation() <= maxInt(2, len(world.GetAllKnownPobles())/4)
}

func (c EndingChecker) isTechIsolationEnding(world *World) bool {
	if world == nil || world.GetPopulation() == 0 {
		return false
	}
	return (techUnlocked(world, simworld.TechComputers) || techUnlocked(world, simworld.TechAIResearch)) &&
		countEvents(world, "isolation", "isolate", "communication", "digital") >= maxInt(4, world.GetPopulation()/3) &&
		averageBelonging(world) >= 72
}

func (c EndingChecker) isUtopiaEnding(world *World) bool {
	return world.GetPopulation() >= c.UtopiaPopulation &&
		averageMentalHealth(world) > c.UtopiaMentalHealth &&
		daysSinceLastWar(world) >= c.UtopiaPeaceDays &&
		technologyLevelFromWorld(world) >= c.UtopiaTechLevel
}

func (c EndingChecker) isCultEnding(world *World) bool {
	_, share := dominantBelief(world)
	return share >= c.CultAdherentShare &&
		hasReligiousLaw(world) &&
		activeProphet(world) != nil
}

func (c EndingChecker) isAloneEnding(world *World) bool {
	return world.GetPopulation() == 1 && !hasReproductionRoute(world)
}

func (c EndingChecker) isLastChildEnding(world *World) bool {
	if world == nil || world.GetPopulation() != 1 {
		return false
	}
	living := world.GetAllPobles()
	return len(living) == 1 && living[0] != nil && living[0].Age < 18 && len(world.GetAllKnownPobles()) > 2
}

func (c EndingChecker) isResetEnding(world *World) bool {
	if world.GetPopulation() != 2 {
		return false
	}
	known := world.GetAllKnownPobles()
	if len(known) <= 2 {
		return false
	}
	return hasCollapseEvent(world) || float32(world.GetPopulation()) <= float32(len(known))*0.20
}

func (c EndingChecker) isMythEnding(world *World) bool {
	return techUnlocked(world, simworld.TechAIResearch) && !hasOriginalApocalypseMemory(world)
}

func collectEndingFacts(endingType EndingType, world *World) endingFacts {
	stats := BuildGameStatistics(world)
	living := sortedPobles(world.GetAllPobles())
	known := sortedPobles(world.GetAllKnownPobles())
	dead := deadPobles(known)
	leader := currentLeader(world)
	prophet := activeProphet(world)
	belief, _ := dominantBelief(world)

	return endingFacts{
		Stats:          stats,
		Living:         living,
		Known:          known,
		Dead:           dead,
		Subject:        firstPobleName(living, known, "nadie"),
		SecondSubject:  secondPobleName(living, known, "nadie mas"),
		Survivor:       firstPobleName(living, known, "nadie"),
		Deceased:       firstPobleName(dead, known, "alguien"),
		LastEvent:      lastEventDescription(world),
		Settlement:     settlementName(world),
		Era:            world.Era.String(),
		Tech:           strconv.Itoa(technologyLevelFromWorld(world)),
		Belief:         fallbackString(belief, "una fe sin nombre"),
		Leader:         pobleName(leader, "ningun lider"),
		Prophet:        pobleName(prophet, "ningun profeta"),
		Family:         familyLabel(world, leader),
		TemplateSuffix: endingTemplateSuffix(endingType),
	}
}

// BuildGameStatistics extracts the requested final statistics from real state.
func BuildGameStatistics(world *World) GameStatistics {
	if world == nil {
		return GameStatistics{}
	}
	known := sortedPobles(world.GetAllKnownPobles())
	stats := GameStatistics{
		TotalDays:            world.Calendar.Day,
		TotalPobles:          len(known),
		TotalDeaths:          totalDead(known),
		TotalBirths:          totalBirths(world, known),
		TotalWars:            countEvents(world, "war"),
		TotalAffairs:         countEvents(world, "affair"),
		TotalSecretsRevealed: totalSecretsRevealed(world, known),
		MostDramaticEvent:    mostDramaticEvent(world),
		FirstCoupleNames:     firstCoupleNames(known),
		TotalRumours:         len(world.RumourPool),
		RumourMostMutated:    mostMutatedRumour(world),
	}
	stats.LongestLivedPople, stats.LongestLivedAge = longestLived(known)
	stats.MostLovedPople = relationshipExtreme(known, true)
	stats.MostHatedPople = relationshipExtreme(known, false)
	stats.RicherPople = richestPoble(known)
	return stats
}

func renderEndingChapters(suffix string, world *World, extra map[string]string) []string {
	renderer, ok := endingRenderer()
	if !ok {
		return fallbackChapters(extra)
	}
	ctx := templateContext(world, extra)
	chapters := make([]string, 0, 4)
	for i := 1; i <= 4; i++ {
		category := path.Join("narrator/end_of_game", suffix, fmt.Sprintf("chapter%d", i))
		rendered, err := renderOne(renderer, category, ctx)
		if err != nil {
			return fallbackChapters(extra)
		}
		chapters = append(chapters, rendered)
	}
	return chapters
}

func renderEndingLastWords(suffix string, world *World, extra map[string]string) string {
	renderer, ok := endingRenderer()
	if !ok {
		return fallbackString(extra["ending_last_event"], "Y entonces el mundo guardo silencio.")
	}
	category := path.Join("narrator/end_of_game", suffix, "last_words")
	rendered, err := renderOne(renderer, category, templateContext(world, extra))
	if err != nil {
		return fallbackString(extra["ending_last_event"], "Y entonces el mundo guardo silencio.")
	}
	return rendered
}

func endingRenderer() (*templates.TemplateEngine, bool) {
	root, ok := findTemplatesRoot()
	if !ok {
		return nil, false
	}
	renderer := templates.NewTemplateEngine(rand.New(rand.NewSource(time.Now().UnixNano())))
	if err := renderer.LoadTemplates(root); err != nil {
		return nil, false
	}
	return renderer, true
}

func renderOne(renderer *templates.TemplateEngine, category string, ctx templates.TemplateContext) (string, error) {
	tmpl, err := renderer.Select(category, ctx)
	if err != nil {
		return "", fmt.Errorf("engine.renderOne: select %s: %w", category, err)
	}
	rendered, err := renderer.Render(tmpl, ctx)
	if err != nil {
		return "", fmt.Errorf("engine.renderOne: render %s: %w", category, err)
	}
	return rendered, nil
}

func templateContext(world *World, extra map[string]string) templates.TemplateContext {
	state := entities.NewWorldState()
	timeNow := entities.NewGameTime(0, 0, 0)
	if world != nil {
		state = world.GetWorldState()
		timeNow = world.Calendar
	}
	return templates.TemplateContext{
		GameTime:   timeNow,
		WorldState: state,
		ExtraVars:  extra,
	}
}

func findTemplatesRoot() (string, bool) {
	wd, err := os.Getwd()
	if err != nil {
		return "", false
	}
	for i := 0; i < 8; i++ {
		candidate := filepath.Join(wd, "templates")
		if info, statErr := os.Stat(candidate); statErr == nil && info.IsDir() {
			return candidate, true
		}
		next := filepath.Dir(wd)
		if next == wd {
			break
		}
		wd = next
	}
	return "", false
}

func endingExtraVars(endingType EndingType, title string, facts endingFacts) map[string]string {
	stats := facts.Stats
	return map[string]string{
		"ending_title":          title,
		"ending_type":           string(endingType),
		"ending_day":            strconv.Itoa(stats.TotalDays),
		"ending_population":     strconv.Itoa(len(facts.Living)),
		"ending_total_pobles":   strconv.Itoa(stats.TotalPobles),
		"ending_total_deaths":   strconv.Itoa(stats.TotalDeaths),
		"ending_total_births":   strconv.Itoa(stats.TotalBirths),
		"ending_total_wars":     strconv.Itoa(stats.TotalWars),
		"ending_total_affairs":  strconv.Itoa(stats.TotalAffairs),
		"ending_total_secrets":  strconv.Itoa(stats.TotalSecretsRevealed),
		"ending_total_rumours":  strconv.Itoa(stats.TotalRumours),
		"ending_subject":        facts.Subject,
		"ending_second_subject": facts.SecondSubject,
		"ending_survivor":       facts.Survivor,
		"ending_deceased":       facts.Deceased,
		"ending_first_couple":   joinPair(stats.FirstCoupleNames),
		"ending_last_event":     facts.LastEvent,
		"ending_dramatic_event": stats.MostDramaticEvent,
		"ending_longest_lived":  longestLabel(stats),
		"ending_most_loved":     fallbackString(stats.MostLovedPople, "nadie"),
		"ending_most_hated":     fallbackString(stats.MostHatedPople, "nadie"),
		"ending_richest":        fallbackString(stats.RicherPople, "nadie"),
		"ending_mutated_rumour": fallbackString(stats.RumourMostMutated, "ningun rumor sobrevivio"),
		"ending_settlement":     facts.Settlement,
		"ending_era":            facts.Era,
		"ending_tech":           facts.Tech,
		"ending_belief":         facts.Belief,
		"ending_leader":         facts.Leader,
		"ending_prophet":        facts.Prophet,
		"ending_family":         facts.Family,
	}
}

func fallbackChapters(extra map[string]string) []string {
	return []string{
		fmt.Sprintf("%s termino en el dia %s.", extra["ending_title"], extra["ending_day"]),
		fmt.Sprintf("La historia empezo con %s y llego hasta %s pobles.", extra["ending_first_couple"], extra["ending_total_pobles"]),
		fmt.Sprintf("El hecho que nadie pudo ignorar fue: %s", extra["ending_dramatic_event"]),
		fmt.Sprintf("El ultimo nombre que queda en la arena es %s.", extra["ending_survivor"]),
	}
}

func endingTitle(endingType EndingType) string {
	switch endingType {
	case END_LOVE:
		return "Los Ultimos Amantes"
	case END_DYNASTY:
		return "La Dinastia"
	case END_WAR:
		return "La Segunda Extincion"
	case END_UTOPIA:
		return "El Experimento Funciono"
	case END_CULT:
		return "El Profeta Gano"
	case END_ALONE:
		return "El Ultimo"
	case END_RESET:
		return "Lo Hicieron de Nuevo"
	case END_MYTH:
		return "Los Olvidados"
	case END_PLAGUE:
		return "La Fiebre Escribio el Final"
	case END_MONOPOLY:
		return "El Ultimo Mercado"
	case END_QUARANTINE:
		return "La Isla En Cuarentena"
	case END_SCANDAL:
		return "Todos Lo Supieron"
	case END_FAMILY_EMPIRE:
		return "El Imperio Familiar"
	case END_ETERNAL_DICTATORSHIP:
		return "La Dictadura Eterna"
	case END_FAILED_REVOLUTION:
		return "La Revolucion Fallida"
	case END_ETERNAL_ARCHIVE:
		return "El Archivo Eterno"
	case END_ECONOMIC_COLLAPSE:
		return "La Cuenta Final"
	case END_EXODUS:
		return "El Exodo"
	case END_TECH_ISOLATION:
		return "La Isla de Pantallas"
	case END_LAST_CHILD:
		return "El Ultimo Nino"
	default:
		return "Final sin nombre"
	}
}

func endingTemplateSuffix(endingType EndingType) string {
	return strings.TrimPrefix(strings.ToLower(string(endingType)), "end_")
}

func hasDeathEvent(world *World) bool {
	return hasEventMatching(world, "death", "murder", "suicide", "accident", "war")
}

func hasReproductionRejection(world *World) bool {
	return hasEventMatching(world, "decision_point", "reproduction_rejected", "no_reproduction", "reject_reproduce")
}

func hasCollapseEvent(world *World) bool {
	return hasEventMatching(world, "collapse", "civilization_collapse", "reset")
}

func hasEventMatching(world *World, needles ...string) bool {
	for _, event := range allWorldEvents(world) {
		if eventMatches(event, needles...) {
			return true
		}
	}
	return false
}

func countEvents(world *World, needles ...string) int {
	count := 0
	for _, event := range allWorldEvents(world) {
		if eventMatches(event, needles...) {
			count++
		}
	}
	return count
}

func eventMatches(event ai.GameEvent, needles ...string) bool {
	haystack := strings.ToLower(event.ID + " " + string(event.Type) + " " + event.Description + " " + strings.Join(event.Tags, " "))
	for _, needle := range needles {
		if strings.Contains(haystack, strings.ToLower(needle)) {
			return true
		}
	}
	return false
}

func allWorldEvents(world *World) []ai.GameEvent {
	if world == nil {
		return nil
	}
	events := make([]ai.GameEvent, 0, len(world.EventHistory)+len(world.ActiveEvents))
	events = append(events, world.EventHistory...)
	events = append(events, world.ActiveEvents...)
	return events
}

func currentLeader(world *World) *entities.Poble {
	if world == nil || world.Government == nil || world.Government.Leader == nil {
		return nil
	}
	return findKnownPoble(world, *world.Government.Leader)
}

func averageMentalHealth(world *World) float32 {
	living := world.GetAllPobles()
	if len(living) == 0 {
		return 0
	}
	total := 0
	for _, poble := range living {
		total += poble.Mental.Stability
	}
	return float32(total) / float32(len(living))
}

func daysSinceLastWar(world *World) int {
	now := world.Calendar.Day
	lastWar := math.MinInt
	for _, event := range allWorldEvents(world) {
		if eventMatches(event, "war") && event.Time.Day > lastWar {
			lastWar = event.Time.Day
		}
	}
	if lastWar == math.MinInt {
		return math.MaxInt / 4
	}
	return maxInt(0, now-lastWar)
}

func technologyLevelFromWorld(world *World) int {
	if world == nil {
		return 0
	}
	if techUnlocked(world, simworld.TechAIResearch) {
		return 5
	}
	if world.Era == entities.EraFour {
		return 4
	}
	if world.Era == entities.EraThree {
		return 3
	}
	return minInt(2, len(world.TechTree.Unlocked)/3)
}

func economyCanMonopolize(world *World) bool {
	if world == nil {
		return false
	}
	return world.Era == entities.EraTwo || world.Era == entities.EraThree || world.Era == entities.EraFour ||
		techUnlocked(world, simworld.TechCurrency)
}

func richestWealthShare(world *World) (*entities.Poble, float32) {
	if world == nil {
		return nil, 0
	}
	total := 0
	var richest *entities.Poble
	for _, poble := range world.GetAllPobles() {
		if poble == nil || poble.Money <= 0 {
			continue
		}
		total += poble.Money
		if richest == nil || poble.Money > richest.Money {
			richest = poble
		}
	}
	if richest == nil || total == 0 {
		return nil, 0
	}
	return richest, float32(richest.Money) / float32(total)
}

func healthCrisisShare(world *World) float32 {
	if world == nil {
		return 0
	}
	living := world.GetAllPobles()
	if len(living) == 0 {
		return 0
	}
	affected := 0
	for _, poble := range living {
		if poble == nil {
			continue
		}
		if poble.Health.HP <= 35 || len(poble.Health.STIs) > 0 || hasHealthCondition(poble, entities.ConditionSick) {
			affected++
		}
	}
	return float32(affected) / float32(len(living))
}

func hasHealthCondition(poble *entities.Poble, target entities.ConditionID) bool {
	if poble == nil {
		return false
	}
	for _, condition := range poble.Health.Conditions {
		if condition == target {
			return true
		}
	}
	return false
}

func techUnlocked(world *World, techID simworld.TechID) bool {
	return world != nil && world.TechTree.Unlocked != nil && world.TechTree.Unlocked[techID]
}

func dominantBelief(world *World) (string, float32) {
	living := world.GetAllPobles()
	if len(living) == 0 {
		return "", 0
	}
	counts := map[string]int{}
	for _, poble := range living {
		for _, superstition := range poble.Superstitions {
			belief := strings.TrimSpace(superstition.Belief)
			if belief != "" && superstition.Strength >= 50 {
				counts[belief]++
			}
		}
	}
	bestBelief, bestCount := bestStringCount(counts)
	return bestBelief, float32(bestCount) / float32(len(living))
}

func hasReligiousLaw(world *World) bool {
	if world == nil || world.Government == nil {
		return false
	}
	if world.Government.Type == simworld.GovernmentTheocracy {
		return true
	}
	for _, law := range world.Government.Laws {
		text := strings.ToLower(law.ID + " " + law.Description)
		if strings.Contains(text, "relig") || strings.Contains(text, "faith") ||
			strings.Contains(text, "ritual") || strings.Contains(text, "profet") {
			return law.IsEnforced
		}
	}
	return false
}

func activeProphet(world *World) *entities.Poble {
	for _, poble := range world.GetAllPobles() {
		if poble != nil && poble.Archetype == entities.ArchetypeProphet {
			return poble
		}
	}
	return nil
}

func hasReproductionRoute(world *World) bool {
	living := world.GetAllPobles()
	if len(living) != 1 {
		return len(living) >= 2
	}
	return techUnlocked(world, simworld.TechReproductionTech)
}

func hasOriginalApocalypseMemory(world *World) bool {
	for _, poble := range world.GetAllPobles() {
		if poble == nil {
			continue
		}
		if poble.Parents == [2]string{} && poble.DayOfBirth.Day <= 0 {
			return true
		}
	}
	return false
}

func familyPowerShare(world *World, leader *entities.Poble) float32 {
	if leader == nil {
		return 0
	}
	living := world.GetAllPobles()
	if len(living) == 0 {
		return 0
	}
	leaderFamily := rootFamilyKey(leader, knownPobleMap(world), map[string]bool{})
	matches := 0
	for _, poble := range living {
		if rootFamilyKey(poble, knownPobleMap(world), map[string]bool{}) == leaderFamily {
			matches++
		}
	}
	return float32(matches) / float32(len(living))
}

func generationDepth(poble *entities.Poble, known map[string]*entities.Poble, seen map[string]bool) int {
	if poble == nil || seen[poble.ID] {
		return 0
	}
	seen[poble.ID] = true
	bestParent := 0
	for _, parentID := range poble.Parents {
		parent := known[parentID]
		if parent == nil {
			continue
		}
		bestParent = maxInt(bestParent, generationDepth(parent, known, seen))
	}
	return bestParent + 1
}

func rootFamilyKey(poble *entities.Poble, known map[string]*entities.Poble, seen map[string]bool) string {
	if poble == nil {
		return ""
	}
	if seen[poble.ID] {
		return poble.ID
	}
	seen[poble.ID] = true
	roots := []string{}
	for _, parentID := range poble.Parents {
		parent := known[parentID]
		if parent != nil {
			roots = append(roots, rootFamilyKey(parent, known, seen))
		}
	}
	if len(roots) == 0 {
		return poble.ID
	}
	sort.Strings(roots)
	return strings.Join(roots, "+")
}

func knownPobleMap(world *World) map[string]*entities.Poble {
	known := map[string]*entities.Poble{}
	for _, poble := range world.GetAllKnownPobles() {
		if poble != nil {
			known[poble.ID] = poble
		}
	}
	return known
}

func totalDead(known []*entities.Poble) int {
	dead := 0
	for _, poble := range known {
		if poble != nil && !poble.IsAlive {
			dead++
		}
	}
	return dead
}

func totalBirths(world *World, known []*entities.Poble) int {
	fromPeople := 0
	for _, poble := range known {
		if poble != nil && poble.Parents != [2]string{} {
			fromPeople++
		}
	}
	return maxInt(fromPeople, countEvents(world, "birth"))
}

func totalSecretsRevealed(world *World, known []*entities.Poble) int {
	revealed := countEvents(world, "secret", "revelation")
	for _, poble := range known {
		for _, secret := range poble.Secrets {
			if secret.IsRevealed {
				revealed++
			}
		}
	}
	return revealed
}

func mostDramaticEvent(world *World) string {
	best := ai.GameEvent{}
	for _, event := range allWorldEvents(world) {
		if event.Severity > best.Severity {
			best = event
		}
	}
	if best.ID == "" {
		return "ningun evento quedo escrito"
	}
	return eventDescription(best)
}

func lastEventDescription(world *World) string {
	events := allWorldEvents(world)
	if len(events) == 0 {
		return "el silencio fue el ultimo registro"
	}
	sort.SliceStable(events, func(i, j int) bool {
		return events[i].Time.ToMinutes() < events[j].Time.ToMinutes()
	})
	return eventDescription(events[len(events)-1])
}

func eventDescription(event ai.GameEvent) string {
	if strings.TrimSpace(event.Description) != "" {
		return event.Description
	}
	if event.ID != "" {
		return event.ID
	}
	return string(event.Type)
}

func firstCoupleNames(known []*entities.Poble) [2]string {
	sorted := append([]*entities.Poble(nil), known...)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].ID < sorted[j].ID
	})
	var names [2]string
	for i := 0; i < len(sorted) && i < 2; i++ {
		names[i] = sorted[i].Name
	}
	return names
}

func longestLived(known []*entities.Poble) (string, int) {
	name := ""
	age := 0
	for _, poble := range known {
		if poble != nil && poble.Age > age {
			name = poble.Name
			age = poble.Age
		}
	}
	return name, age
}

func relationshipExtreme(known []*entities.Poble, loved bool) string {
	scores := map[string]float32{}
	for _, source := range known {
		for targetID, rel := range source.Relationships {
			if loved {
				scores[targetID] += rel.Affection + rel.Trust + rel.Respect
			} else {
				scores[targetID] += rel.Resentment + rel.Fear - rel.Trust
			}
		}
	}
	id, _ := bestFloatScore(scores)
	return nameByID(known, id)
}

func richestPoble(known []*entities.Poble) string {
	best := ""
	money := -1
	for _, poble := range known {
		if poble != nil && poble.Money > money {
			best = poble.Name
			money = poble.Money
		}
	}
	return best
}

func mostMutatedRumour(world *World) string {
	bestContent := ""
	bestScore := float32(-1)
	for _, rumour := range world.RumourPool {
		score := rumour.SpreadLevel + (1 - rumour.Truthiness)
		if score > bestScore {
			bestContent = rumour.Content
			bestScore = score
		}
	}
	return bestContent
}

func totalLivingWealth(world *World) int {
	total := 0
	for _, poble := range world.GetAllPobles() {
		if poble != nil && poble.Money > 0 {
			total += poble.Money
		}
	}
	return total
}

func discoveredIslandCount(world *World) int {
	count := 0
	if world == nil {
		return count
	}
	for _, island := range world.Islands {
		if island != nil && island.IsDiscovered {
			count++
		}
	}
	return count
}

func averageBelonging(world *World) float32 {
	living := world.GetAllPobles()
	if len(living) == 0 {
		return 0
	}
	total := float32(0)
	for _, poble := range living {
		if poble != nil {
			total += poble.Needs.Belonging
		}
	}
	return total / float32(len(living))
}

func deadPobles(known []*entities.Poble) []*entities.Poble {
	dead := []*entities.Poble{}
	for _, poble := range known {
		if poble != nil && !poble.IsAlive {
			dead = append(dead, poble)
		}
	}
	return dead
}

func sortedPobles(pobles []*entities.Poble) []*entities.Poble {
	result := append([]*entities.Poble(nil), pobles...)
	sort.SliceStable(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})
	return result
}

func firstPobleName(primary []*entities.Poble, fallback []*entities.Poble, empty string) string {
	if len(primary) > 0 && primary[0] != nil {
		return primary[0].Name
	}
	if len(fallback) > 0 && fallback[0] != nil {
		return fallback[0].Name
	}
	return empty
}

func secondPobleName(primary []*entities.Poble, fallback []*entities.Poble, empty string) string {
	if len(primary) > 1 && primary[1] != nil {
		return primary[1].Name
	}
	if len(fallback) > 1 && fallback[1] != nil {
		return fallback[1].Name
	}
	return empty
}

func settlementName(world *World) string {
	if world != nil {
		state := world.GetWorldState()
		if len(state.Settlements) > 0 && strings.TrimSpace(state.Settlements[0].Name) != "" {
			return state.Settlements[0].Name
		}
	}
	return "El Origen"
}

func familyLabel(world *World, leader *entities.Poble) string {
	if leader == nil {
		return "ninguna familia"
	}
	key := rootFamilyKey(leader, knownPobleMap(world), map[string]bool{})
	parts := strings.Split(key, "+")
	names := make([]string, 0, len(parts))
	for _, id := range parts {
		if poble := findKnownPoble(world, id); poble != nil {
			names = append(names, poble.Name)
		}
	}
	if len(names) == 0 {
		return leader.Name
	}
	return strings.Join(names, " / ")
}

func findKnownPoble(world *World, id string) *entities.Poble {
	for _, poble := range world.GetAllKnownPobles() {
		if poble != nil && poble.ID == id {
			return poble
		}
	}
	return nil
}

func nameByID(known []*entities.Poble, id string) string {
	for _, poble := range known {
		if poble != nil && poble.ID == id {
			return poble.Name
		}
	}
	return ""
}

func bestStringCount(counts map[string]int) (string, int) {
	bestKey := ""
	bestCount := 0
	for key, count := range counts {
		if count > bestCount || (count == bestCount && key < bestKey) {
			bestKey = key
			bestCount = count
		}
	}
	return bestKey, bestCount
}

func bestFloatScore(scores map[string]float32) (string, float32) {
	bestKey := ""
	bestScore := float32(math.Inf(-1))
	for key, score := range scores {
		if score > bestScore || (score == bestScore && key < bestKey) {
			bestKey = key
			bestScore = score
		}
	}
	return bestKey, bestScore
}

func pobleName(poble *entities.Poble, fallback string) string {
	if poble == nil || strings.TrimSpace(poble.Name) == "" {
		return fallback
	}
	return poble.Name
}

func joinPair(pair [2]string) string {
	if pair[0] == "" && pair[1] == "" {
		return "nadie"
	}
	if pair[1] == "" {
		return pair[0]
	}
	return pair[0] + " y " + pair[1]
}

func longestLabel(stats GameStatistics) string {
	if stats.LongestLivedPople == "" {
		return "nadie"
	}
	return fmt.Sprintf("%s (%d)", stats.LongestLivedPople, stats.LongestLivedAge)
}

func fallbackString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
