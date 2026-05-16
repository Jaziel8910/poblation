package engine

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/user/poblation/internal/entities"
	"github.com/user/poblation/internal/events"
	"github.com/user/poblation/internal/world"
)

// ConsoleViewHint lets the UI switch to a matching view after a command.
type ConsoleViewHint string

const (
	ConsoleViewNone      ConsoleViewHint = ""
	ConsoleViewNewspaper ConsoleViewHint = "NEWSPAPER"
	ConsoleViewEndings   ConsoleViewHint = "ENDINGS"
)

// ConsoleHistoryEntry stores one executed command and its result.
type ConsoleHistoryEntry struct {
	Input      string    `json:"input"`
	Output     string    `json:"output"`
	ExecutedAt time.Time `json:"executed_at"`
	Success    bool      `json:"success"`
}

// ConsoleResult describes what a command changed and what the UI should show.
type ConsoleResult struct {
	Feedback   string          `json:"feedback"`
	ViewHint   ConsoleViewHint `json:"view_hint"`
	Event      *events.GameEvent `json:"event,omitempty"`
	ClearFeed  bool            `json:"clear_feed"`
}

// ConsoleCommand stores metadata and executable behavior for one command.
type ConsoleCommand struct {
	Name        string
	Description string
	Aliases     []string
	Handler     func(args []string) (ConsoleResult, error)
}

// ConsoleSystem owns command registration, aliases, and execution history.
type ConsoleSystem struct {
	world           *world.World
	timeEngine      *TimeEngine
	rng             *rand.Rand
	commands        map[string]ConsoleCommand
	aliases         map[string]string
	history         []ConsoleHistoryEntry
	GodMode         bool
	NewspaperMode   bool
}

// NewConsoleSystem creates the runtime command console.
func NewConsoleSystem(gameWorld *world.World, timeEngine *TimeEngine, rng *rand.Rand) *ConsoleSystem {
	if rng == nil {
		rng = rand.New(rand.NewSource(time.Now().UnixNano()))
	}

	system := &ConsoleSystem{
		world:      gameWorld,
		timeEngine: timeEngine,
		rng:        rng,
		commands:   map[string]ConsoleCommand{},
		aliases:    map[string]string{},
		history:    []ConsoleHistoryEntry{},
	}
	system.registerDefaults()
	return system
}

// Execute validates and runs one console command.
func (c *ConsoleSystem) Execute(raw string) ConsoleResult {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ConsoleResult{Feedback: "Escribe un comando. `help` o `ayuda` te enseña los disponibles."}
	}

	commandName, args := c.resolveCommand(trimmed)
	command, ok := c.commands[commandName]
	if !ok {
		result := ConsoleResult{Feedback: "No entendí ese comando. Usa `help` o `ayuda`."}
		c.pushHistory(trimmed, result.Feedback, false)
		return result
	}

	result, err := command.Handler(args)
	if err != nil {
		feedback := err.Error()
		c.pushHistory(trimmed, feedback, false)
		return ConsoleResult{Feedback: feedback}
	}

	if result.Feedback == "" {
		result.Feedback = "Hecho."
	}
	c.pushHistory(trimmed, result.Feedback, true)
	return result
}

// History returns a copy of the console history.
func (c *ConsoleSystem) History() []ConsoleHistoryEntry {
	if c == nil {
		return nil
	}
	return append([]ConsoleHistoryEntry(nil), c.history...)
}

func (c *ConsoleSystem) registerDefaults() {
	c.register(ConsoleCommand{
		Name:        "god mode",
		Description: "Activa o desactiva el modo Director.",
		Aliases:     []string{"god", "director", "modo dios"},
		Handler: func(args []string) (ConsoleResult, error) {
			c.GodMode = !c.GodMode
			if c.GodMode {
				return ConsoleResult{Feedback: "Modo Director activado. Ahora mandas demasiado."}, nil
			}
			return ConsoleResult{Feedback: "Modo Director desactivado. La isla vuelve a respirar sola."}, nil
		},
	})
	c.register(ConsoleCommand{
		Name:        "kill",
		Description: "Mata a un poble al instante. Uso: kill {nombre} [causa].",
		Aliases:     []string{"k", "matar", "mata"},
		Handler:     c.handleKill,
	})
	c.register(ConsoleCommand{
		Name:        "secret",
		Description: "Revela todos los secretos de un poble.",
		Aliases:     []string{"secretos", "secreto"},
		Handler:     c.handleSecret,
	})
	c.register(ConsoleCommand{
		Name:        "drama",
		Description: "Fuerza un evento dramático fuerte.",
		Aliases:     []string{"d", "dramon", "dramón"},
		Handler:     c.handleDrama,
	})
	c.register(ConsoleCommand{
		Name:        "plague",
		Description: "Inicia una plaga en parte de la población.",
		Aliases:     []string{"plaga", "pandemia"},
		Handler:     c.handlePlague,
	})
	c.register(ConsoleCommand{
		Name:        "end world",
		Description: "Abre el menú de finales forzados.",
		Aliases:     []string{"finales", "fin del mundo"},
		Handler: func(args []string) (ConsoleResult, error) {
			return ConsoleResult{Feedback: "Abriendo finales. Qué ganas de caos.", ViewHint: ConsoleViewEndings}, nil
		},
	})
	c.register(ConsoleCommand{
		Name:        "newspaper",
		Description: "Activa el modo periodista y exporta periódico.",
		Aliases:     []string{"periodico", "periódico", "reportero", "reportera"},
		Handler:     c.handleNewspaper,
	})
	c.register(ConsoleCommand{
		Name:        "confession",
		Description: "Fuerza una confesión incómoda entre dos pobles.",
		Aliases:     []string{"confesion", "confesión", "confesar"},
		Handler:     c.handleConfession,
	})
	c.register(ConsoleCommand{
		Name:        "war",
		Description: "Declara conflicto entre dos grupos dominantes.",
		Aliases:     []string{"guerra"},
		Handler:     c.handleWar,
	})
	c.register(ConsoleCommand{
		Name:        "baby",
		Description: "Fuerza embarazo si la biología lo permite. Uso: baby {a} {b}.",
		Aliases:     []string{"bebe", "bebé", "embarazo"},
		Handler:     c.handleBaby,
	})
	c.register(ConsoleCommand{
		Name:        "reset",
		Description: "Colapsa la civilización y deja 2 sobrevivientes.",
		Aliases:     []string{"reinicio", "colapso"},
		Handler:     c.handleReset,
	})
	c.register(ConsoleCommand{
		Name:        "credits",
		Description: "Muestra créditos in-character.",
		Aliases:     []string{"creditos", "créditos"},
		Handler: func(args []string) (ConsoleResult, error) {
			text := "POBLATION recuerda a quienes jugaron a ser dioses, amantes, cronistas y testigos.\nLa isla puso el drama. Tú pusiste los ojos."
			return ConsoleResult{Feedback: text}, nil
		},
	})
	c.register(ConsoleCommand{
		Name:        "speed",
		Description: "Cambia velocidad. Uso: speed {0.5|1|2|4}.",
		Aliases:     []string{"velocidad"},
		Handler:     c.handleSpeed,
	})
	c.register(ConsoleCommand{
		Name:        "pause",
		Description: "Pausa el tiempo.",
		Aliases:     []string{"pausa"},
		Handler: func(args []string) (ConsoleResult, error) {
			if c.timeEngine != nil {
				c.timeEngine.Pause()
			}
			return ConsoleResult{Feedback: "Tiempo en pausa."}, nil
		},
	})
	c.register(ConsoleCommand{
		Name:        "resume",
		Description: "Reanuda el tiempo.",
		Aliases:     []string{"reanudar", "continuar"},
		Handler: func(args []string) (ConsoleResult, error) {
			if c.timeEngine != nil {
				c.timeEngine.Resume()
			}
			return ConsoleResult{Feedback: "Tiempo reanudado."}, nil
		},
	})
	c.register(ConsoleCommand{
		Name:        "info",
		Description: "Muestra el estado completo de un poble.",
		Aliases:     []string{"ficha", "estado"},
		Handler:     c.handleInfo,
	})
	c.register(ConsoleCommand{
		Name:        "relations",
		Description: "Muestra mapa de relaciones de un poble.",
		Aliases:     []string{"relaciones"},
		Handler:     c.handleRelations,
	})
	c.register(ConsoleCommand{
		Name:        "rumours",
		Description: "Lista rumores activos.",
		Aliases:     []string{"rumores"},
		Handler:     c.handleRumours,
	})
	c.register(ConsoleCommand{
		Name:        "time",
		Description: "Salta a un tiempo. Uso: time {dia} {hora}.",
		Aliases:     []string{"tiempo"},
		Handler:     c.handleTime,
	})
	c.register(ConsoleCommand{
		Name:        "era",
		Description: "Fuerza un cambio de era. Uso: era {0-4}.",
		Aliases:     []string{"epoca", "época"},
		Handler:     c.handleEra,
	})
	c.register(ConsoleCommand{
		Name:        "tech",
		Description: "Descubre una tecnología al instante.",
		Aliases:     []string{"tecnologia", "tecnología"},
		Handler:     c.handleTech,
	})
	c.register(ConsoleCommand{
		Name:        "spawn",
		Description: "Crea un poble random de ese arquetipo.",
		Aliases:     []string{"crear", "invocar"},
		Handler:     c.handleSpawn,
	})
	c.register(ConsoleCommand{
		Name:        "age",
		Description: "Cambia edad. Uso: age {nombre} {n}.",
		Aliases:     []string{"edad"},
		Handler:     c.handleAge,
	})
	c.register(ConsoleCommand{
		Name:        "mood",
		Description: "Cambia humor. Uso: mood {nombre} {mood}.",
		Aliases:     []string{"humor"},
		Handler:     c.handleMood,
	})
	c.register(ConsoleCommand{
		Name:        "think",
		Description: "Fuerza un pensamiento visible ahora mismo.",
		Aliases:     []string{"pensar", "piensa"},
		Handler:     c.handleThink,
	})
	c.register(ConsoleCommand{
		Name:        "help",
		Description: "Lista todos los comandos.",
		Aliases:     []string{"ayuda"},
		Handler:     c.handleHelp,
	})
}

func (c *ConsoleSystem) register(command ConsoleCommand) {
	c.commands[command.Name] = command
	c.aliases[normalizeConsoleToken(command.Name)] = command.Name
	for _, alias := range command.Aliases {
		c.aliases[normalizeConsoleToken(alias)] = command.Name
	}
}

func (c *ConsoleSystem) resolveCommand(raw string) (string, []string) {
	fields := strings.Fields(raw)
	if len(fields) == 0 {
		return "", nil
	}

	if len(fields) >= 2 {
		twoWord := normalizeConsoleToken(fields[0] + " " + fields[1])
		if name, ok := c.aliases[twoWord]; ok {
			return name, fields[2:]
		}
	}

	oneWord := normalizeConsoleToken(fields[0])
	if name, ok := c.aliases[oneWord]; ok {
		return name, fields[1:]
	}
	return "", fields[1:]
}

func (c *ConsoleSystem) handleKill(args []string) (ConsoleResult, error) {
	if len(args) < 1 {
		return ConsoleResult{}, fmt.Errorf("Uso: kill {nombre} [causa]")
	}
	poble := c.findPobleByName(args[0], true)
	if poble == nil {
		return ConsoleResult{}, fmt.Errorf("No encontré a `%s`.", args[0])
	}
	cause := events.DeathCauseAccident
	if len(args) >= 2 {
		parsed, ok := parseDeathCause(args[1])
		if !ok {
			return ConsoleResult{}, fmt.Errorf("Causa inválida. Prueba: accident, murder, illness, war, starvation.")
		}
		cause = parsed
	}
	event := events.HandleDeath(poble, cause, c.world)
	event.Description = fmt.Sprintf("%s murió por %s.", poble.Name, strings.ToLower(string(cause)))
	c.appendEvent(event)
	return ConsoleResult{
		Feedback: fmt.Sprintf("%s cayó. Causa: %s.", poble.Name, strings.ToLower(string(cause))),
		Event:    &event,
	}, nil
}

func (c *ConsoleSystem) handleSecret(args []string) (ConsoleResult, error) {
	if len(args) < 1 {
		return ConsoleResult{}, fmt.Errorf("Uso: secret {nombre}")
	}
	poble := c.findPobleByName(args[0], false)
	if poble == nil {
		return ConsoleResult{}, fmt.Errorf("No encontré a `%s`.", args[0])
	}
	if len(poble.Secrets) == 0 {
		return ConsoleResult{Feedback: fmt.Sprintf("%s no guarda secretos en este momento.", poble.Name)}, nil
	}
	lines := make([]string, 0, len(poble.Secrets)+1)
	lines = append(lines, fmt.Sprintf("Secretos de %s:", poble.Name))
	for i := range poble.Secrets {
		poble.Secrets[i].IsRevealed = true
		lines = append(lines, fmt.Sprintf("%d. %s", i+1, poble.Secrets[i].Content))
	}
	return ConsoleResult{Feedback: strings.Join(lines, "\n")}, nil
}

func (c *ConsoleSystem) handleDrama(args []string) (ConsoleResult, error) {
	living := c.world.GetAllPobles()
	if len(living) == 0 {
		return ConsoleResult{}, fmt.Errorf("No hay pobles vivos para armar drama.")
	}
	actor := living[c.rng.Intn(len(living))]
	target := living[c.rng.Intn(len(living))]
	if len(living) > 1 {
		for target.ID == actor.ID {
			target = living[c.rng.Intn(len(living))]
		}
	}
	options := []events.EventType{
		events.EventBetrayalRevealed,
		events.EventFightPhysical,
		events.EventPublicHumiliation,
		events.EventObsessionPeak,
	}
	event := events.GameEvent{
		ID:           consoleID("drama", c.now().ToMinutes()),
		Type:         options[c.rng.Intn(len(options))],
		Timestamp:    c.now(),
		Participants: uniqueStrings([]string{actor.ID, target.ID}),
		IsPublic:     true,
		Description:  fmt.Sprintf("La calma murió: %s arrastró a %s a un episodio de alto voltaje.", actor.Name, target.Name),
		Consequences: []events.Consequence{
			{TargetID: actor.ID, Type: events.ConsequenceMoodShift, Value: -16},
			{TargetID: target.ID, Type: events.ConsequenceMoodShift, Value: -22},
		},
	}
	applyMood(actor, entities.MoodAngry)
	applyMood(target, entities.MoodAnxious)
	c.appendEvent(event)
	return ConsoleResult{Feedback: "Drama forzado. La isla ya está cuchicheando.", Event: &event}, nil
}

func (c *ConsoleSystem) handlePlague(args []string) (ConsoleResult, error) {
	living := c.world.GetAllPobles()
	if len(living) == 0 {
		return ConsoleResult{}, fmt.Errorf("No hay población viva.")
	}
	infected := 0
	for _, poble := range living {
		if infected >= maxInt(1, len(living)/3) && c.rng.Float32() > 0.18 {
			continue
		}
		if !hasCondition(poble, entities.ConditionSick) {
			poble.Health.Conditions = append(poble.Health.Conditions, entities.ConditionSick)
			poble.Health.HP = clampInt(poble.Health.HP-18, 0, 100)
			applyMood(poble, entities.MoodAnxious)
			infected++
		}
	}
	event := events.GameEvent{
		ID:           consoleID("plague", c.now().ToMinutes()),
		Type:         events.EventPlague,
		Timestamp:    c.now(),
		Participants: nil,
		IsPublic:     true,
		Description:  fmt.Sprintf("Una plaga arrancó con %d personas enfermas.", infected),
	}
	c.appendEvent(event)
	return ConsoleResult{Feedback: fmt.Sprintf("Plaga iniciada. Infectados: %d.", infected), Event: &event}, nil
}

func (c *ConsoleSystem) handleNewspaper(args []string) (ConsoleResult, error) {
	if c.world == nil {
		return ConsoleResult{}, fmt.Errorf("No hay mundo cargado.")
	}
	c.NewspaperMode = true
	return ConsoleResult{
		Feedback: fmt.Sprintf("Modo periodista activo. Titular: Día %d, era %s, población %d.", c.world.Calendar.Day, c.world.Era, c.world.GetPopulation()),
		ViewHint: ConsoleViewNewspaper,
	}, nil
}

func (c *ConsoleSystem) handleConfession(args []string) (ConsoleResult, error) {
	living := c.world.GetAllPobles()
	if len(living) < 2 {
		return ConsoleResult{}, fmt.Errorf("Se necesitan al menos 2 pobles vivos.")
	}
	actor := living[c.rng.Intn(len(living))]
	target := chooseConfessionTarget(actor, living, c.rng)
	if target == nil {
		return ConsoleResult{}, fmt.Errorf("No encontré a quién confesarle algo.")
	}

	confession := "te necesito más de lo que debería"
	if len(actor.Secrets) > 0 {
		actor.Secrets[0].IsRevealed = true
		confession = actor.Secrets[0].Content
	}
	rel := relationOrDefault(actor, target.ID, entities.RelationshipComplicated)
	rel.Trust = clampFloat(rel.Trust+8, 0, 100)
	rel.Familiarity = clampFloat(rel.Familiarity+12, 0, 100)
	actor.Relationships[target.ID] = rel

	event := events.GameEvent{
		ID:           consoleID("confession", c.now().ToMinutes()),
		Type:         events.EventRevelation,
		Timestamp:    c.now(),
		Participants: []string{actor.ID, target.ID},
		IsPublic:     false,
		Description:  fmt.Sprintf("%s le confesó a %s: %s", actor.Name, target.Name, confession),
	}
	c.appendEvent(event)
	return ConsoleResult{
		Feedback: fmt.Sprintf("%s le confesó algo a %s.", actor.Name, target.Name),
		Event:    &event,
	}, nil
}

func (c *ConsoleSystem) handleWar(args []string) (ConsoleResult, error) {
	living := c.world.GetAllPobles()
	if len(living) < 4 {
		return ConsoleResult{}, fmt.Errorf("Hace falta más población para que una guerra se vea como guerra.")
	}
	left, right := dominantArchetypeGroups(living)
	if left == "" || right == "" || left == right {
		return ConsoleResult{}, fmt.Errorf("No encontré dos grupos claros para enfrentar.")
	}

	affected := 0
	for _, poble := range living {
		if poble.Archetype != left && poble.Archetype != right {
			continue
		}
		applyMood(poble, entities.MoodAngry)
		poble.Needs.Safety = clampFloat(poble.Needs.Safety+22, 0, 100)
		affected++
	}

	event := events.GameEvent{
		ID:           consoleID("war", c.now().ToMinutes()),
		Type:         events.EventWarDeclaration,
		Timestamp:    c.now(),
		IsPublic:     true,
		Description:  fmt.Sprintf("Estalló conflicto abierto entre %s y %s.", left, right),
	}
	c.appendEvent(event)
	return ConsoleResult{
		Feedback: fmt.Sprintf("Guerra declarada entre %s y %s. Afectados: %d.", left, right, affected),
		Event:    &event,
	}, nil
}

func (c *ConsoleSystem) handleBaby(args []string) (ConsoleResult, error) {
	if len(args) < 2 {
		return ConsoleResult{}, fmt.Errorf("Uso: baby {a} {b}")
	}
	a := c.findPobleByName(args[0], true)
	b := c.findPobleByName(args[1], true)
	if a == nil || b == nil {
		return ConsoleResult{}, fmt.Errorf("No encontré a ambos pobles.")
	}
	mother, father, chance := biologicalPair(a, b)
	if mother == nil || father == nil || chance <= 0 {
		return ConsoleResult{}, fmt.Errorf("No es biológicamente viable entre %s y %s.", a.Name, b.Name)
	}
	if hasCondition(mother, entities.ConditionPregnant) {
		return ConsoleResult{}, fmt.Errorf("%s ya está embarazada.", mother.Name)
	}

	mother.Health.Conditions = append(mother.Health.Conditions, entities.ConditionPregnant)
	mother.Needs.Safety = clampFloat(mother.Needs.Safety+12, 0, 100)
	rel := relationOrDefault(mother, father.ID, entities.RelationshipLover)
	rel.Familiarity = clampFloat(rel.Familiarity+10, 0, 100)
	mother.Relationships[father.ID] = rel

	event := events.GameEvent{
		ID:           consoleID("pregnancy", c.now().ToMinutes()),
		Type:         events.EventPregnancy,
		Timestamp:    c.now(),
		Participants: []string{mother.ID, father.ID},
		IsPublic:     false,
		Description:  fmt.Sprintf("%s quedó embarazada y %s forma parte de la historia biológica.", mother.Name, father.Name),
	}
	c.appendEvent(event)
	return ConsoleResult{
		Feedback: fmt.Sprintf("Embarazo forzado: %s y %s tenían vía biológica posible.", mother.Name, father.Name),
		Event:    &event,
	}, nil
}

func (c *ConsoleSystem) handleReset(args []string) (ConsoleResult, error) {
	all := c.world.GetAllKnownPobles()
	if len(all) <= 2 {
		return ConsoleResult{Feedback: "Ya quedan dos o menos. El mundo ya está roto."}, nil
	}
	survivors := pickSurvivors(all, c.rng)
	survivorSet := map[string]bool{survivors[0].ID: true, survivors[1].ID: true}
	dead := 0
	for _, poble := range all {
		if poble == nil || !poble.IsAlive || survivorSet[poble.ID] {
			continue
		}
		event := events.HandleDeath(poble, events.DeathCauseWar, c.world)
		event.Description = fmt.Sprintf("%s no sobrevivió al colapso.", poble.Name)
		c.appendEvent(event)
		dead++
	}
	c.world.Era = entities.EraZero
	c.world.Government = nil
	c.world.TechTree = world.NewTechTree()
	_ = c.world.GetWorldState()

	event := events.GameEvent{
		ID:          consoleID("collapse", c.now().ToMinutes()),
		Type:        events.EventCivilizationCollapse,
		Timestamp:   c.now(),
		IsPublic:    true,
		Description: "La civilización cayó y solo quedaron dos sobrevivientes.",
	}
	c.appendEvent(event)
	return ConsoleResult{
		Feedback: fmt.Sprintf("Reset brutal hecho. Muertes: %d. Sobreviven %s y %s.", dead, survivors[0].Name, survivors[1].Name),
		Event:    &event,
		ClearFeed: false,
	}, nil
}

func (c *ConsoleSystem) handleSpeed(args []string) (ConsoleResult, error) {
	if len(args) != 1 {
		return ConsoleResult{}, fmt.Errorf("Uso: speed {0.5|1|2|4}")
	}
	value, err := strconv.ParseFloat(args[0], 64)
	if err != nil {
		return ConsoleResult{}, fmt.Errorf("Velocidad inválida.")
	}
	switch value {
	case 0.5, 1, 2, 4:
	default:
		return ConsoleResult{}, fmt.Errorf("Solo acepto 0.5, 1, 2 o 4.")
	}
	if c.timeEngine != nil {
		c.timeEngine.SetSpeed(value)
	}
	return ConsoleResult{Feedback: fmt.Sprintf("Velocidad puesta en %.1fx.", value)}, nil
}

func (c *ConsoleSystem) handleInfo(args []string) (ConsoleResult, error) {
	if len(args) < 1 {
		return ConsoleResult{}, fmt.Errorf("Uso: info {nombre}")
	}
	poble := c.findPobleByName(args[0], false)
	if poble == nil {
		return ConsoleResult{}, fmt.Errorf("No encontré a `%s`.", args[0])
	}
	data, err := json.MarshalIndent(poble, "", "  ")
	if err != nil {
		return ConsoleResult{}, fmt.Errorf("No pude serializar el estado de %s.", poble.Name)
	}
	return ConsoleResult{Feedback: string(data)}, nil
}

func (c *ConsoleSystem) handleRelations(args []string) (ConsoleResult, error) {
	if len(args) < 1 {
		return ConsoleResult{}, fmt.Errorf("Uso: relations {nombre}")
	}
	poble := c.findPobleByName(args[0], false)
	if poble == nil {
		return ConsoleResult{}, fmt.Errorf("No encontré a `%s`.", args[0])
	}
	if len(poble.Relationships) == 0 {
		return ConsoleResult{Feedback: fmt.Sprintf("%s no tiene relaciones registradas.", poble.Name)}, nil
	}
	lines := []string{fmt.Sprintf("Relaciones de %s:", poble.Name)}
	ids := make([]string, 0, len(poble.Relationships))
	for id := range poble.Relationships {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		rel := poble.Relationships[id]
		targetName := id
		if target := c.findKnownPobleByID(id); target != nil {
			targetName = target.Name
		}
		lines = append(lines, fmt.Sprintf("- %s: %s | afecto %.0f | confianza %.0f | resentimiento %.0f", targetName, rel.Type, rel.Affection, rel.Trust, rel.Resentment))
	}
	return ConsoleResult{Feedback: strings.Join(lines, "\n")}, nil
}

func (c *ConsoleSystem) handleRumours(args []string) (ConsoleResult, error) {
	if len(c.world.RumourPool) == 0 {
		return ConsoleResult{Feedback: "No hay rumores activos."}, nil
	}
	lines := []string{"Rumores activos:"}
	for i, rumour := range c.world.RumourPool {
		lines = append(lines, fmt.Sprintf("%d. %s (%.0f%%)", i+1, rumour.Content, rumour.Truthiness*100))
	}
	return ConsoleResult{Feedback: strings.Join(lines, "\n")}, nil
}

func (c *ConsoleSystem) handleTime(args []string) (ConsoleResult, error) {
	if len(args) != 2 {
		return ConsoleResult{}, fmt.Errorf("Uso: time {dia} {hora}")
	}
	day, err := strconv.Atoi(args[0])
	if err != nil || day < 0 {
		return ConsoleResult{}, fmt.Errorf("Día inválido.")
	}
	hour, err := strconv.Atoi(args[1])
	if err != nil || hour < 0 || hour > 23 {
		return ConsoleResult{}, fmt.Errorf("Hora inválida. Debe ser 0-23.")
	}
	target := entities.NewGameTime(day, hour, 0)
	if c.timeEngine != nil {
		c.timeEngine.SetTime(target)
	}
	c.world.Calendar = target
	_ = c.world.GetWorldState()
	return ConsoleResult{Feedback: fmt.Sprintf("Tiempo saltado a Day %d %02d:00.", day, hour)}, nil
}

func (c *ConsoleSystem) handleEra(args []string) (ConsoleResult, error) {
	if len(args) != 1 {
		return ConsoleResult{}, fmt.Errorf("Uso: era {0-4}")
	}
	index, err := strconv.Atoi(args[0])
	if err != nil || index < 0 || index > 4 {
		return ConsoleResult{}, fmt.Errorf("Era inválida. Usa 0, 1, 2, 3 o 4.")
	}
	target := eraFromInt(index)
	from := c.world.Era
	c.world.Era = target
	_ = c.world.GetWorldState()
	event := events.GameEvent{
		ID:          consoleID("era", c.now().ToMinutes()),
		Type:        events.EventEraChange,
		Timestamp:   c.now(),
		IsPublic:    true,
		Description: fmt.Sprintf("La era cambió de %s a %s por mano del Director.", from, target),
	}
	c.appendEvent(event)
	return ConsoleResult{
		Feedback: fmt.Sprintf("Era forzada: %s -> %s.", from, target),
		Event:    &event,
	}, nil
}

func (c *ConsoleSystem) handleTech(args []string) (ConsoleResult, error) {
	if len(args) != 1 {
		return ConsoleResult{}, fmt.Errorf("Uso: tech {id}")
	}
	techID, ok := parseTechID(args[0])
	if !ok {
		return ConsoleResult{}, fmt.Errorf("Tecnología inválida.")
	}
	c.world.TechTree.Unlocked[techID] = true
	c.world.TechTree.Discovered[techID] = c.now()
	c.world.State.TechTree = c.world.TechTree.ToEntityTechTree()
	event := events.GameEvent{
		ID:          consoleID("tech", c.now().ToMinutes()),
		Type:        events.EventTechDiscovered,
		Timestamp:   c.now(),
		IsPublic:    true,
		Description: fmt.Sprintf("Tecnología descubierta al instante: %s.", techID),
	}
	c.appendEvent(event)
	return ConsoleResult{Feedback: fmt.Sprintf("%s ya está descubierta.", techID), Event: &event}, nil
}

func (c *ConsoleSystem) handleSpawn(args []string) (ConsoleResult, error) {
	if len(args) != 1 {
		return ConsoleResult{}, fmt.Errorf("Uso: spawn {arquetipo}")
	}
	archetype, ok := parseArchetype(args[0])
	if !ok {
		return ConsoleResult{}, fmt.Errorf("Arquetipo inválido.")
	}

	config := entities.PoblConfig{AgeRange: [2]int{18, 45}, Archetype: &archetype}
	poble, err := entities.GeneratePople(config, c.rng)
	if err != nil {
		return ConsoleResult{}, fmt.Errorf("No pude crear el poble: %v", err)
	}
	location := c.defaultSpawnLocation()
	if !c.world.AddPoble(poble, location) {
		return ConsoleResult{}, fmt.Errorf("No pude ubicar al nuevo poble en el mundo.")
	}

	event := events.GameEvent{
		ID:           consoleID("spawn", c.now().ToMinutes()),
		Type:         events.EventPopulationMilestone,
		Timestamp:    c.now(),
		Participants: []string{poble.ID},
		IsPublic:     true,
		Description:  fmt.Sprintf("%s apareció en el mundo con arquetipo %s.", poble.Name, archetype),
	}
	c.appendEvent(event)
	return ConsoleResult{Feedback: fmt.Sprintf("%s apareció. Arquetipo: %s.", poble.Name, archetype), Event: &event}, nil
}

func (c *ConsoleSystem) handleAge(args []string) (ConsoleResult, error) {
	if len(args) != 2 {
		return ConsoleResult{}, fmt.Errorf("Uso: age {nombre} {n}")
	}
	poble := c.findPobleByName(args[0], false)
	if poble == nil {
		return ConsoleResult{}, fmt.Errorf("No encontré a `%s`.", args[0])
	}
	age, err := strconv.Atoi(args[1])
	if err != nil || age < 0 || age > 120 {
		return ConsoleResult{}, fmt.Errorf("Edad inválida.")
	}
	poble.Age = age
	poble.Health.Age = age
	return ConsoleResult{Feedback: fmt.Sprintf("%s ahora tiene %d años.", poble.Name, age)}, nil
}

func (c *ConsoleSystem) handleMood(args []string) (ConsoleResult, error) {
	if len(args) != 2 {
		return ConsoleResult{}, fmt.Errorf("Uso: mood {nombre} {mood}")
	}
	poble := c.findPobleByName(args[0], false)
	if poble == nil {
		return ConsoleResult{}, fmt.Errorf("No encontré a `%s`.", args[0])
	}
	mood, ok := parseMood(args[1])
	if !ok {
		return ConsoleResult{}, fmt.Errorf("Mood inválido.")
	}
	applyMood(poble, mood)
	return ConsoleResult{Feedback: fmt.Sprintf("Humor de %s cambiado a %s.", poble.Name, mood)}, nil
}

func (c *ConsoleSystem) handleThink(args []string) (ConsoleResult, error) {
	if len(args) != 1 {
		return ConsoleResult{}, fmt.Errorf("Uso: think {nombre}")
	}
	poble := c.findPobleByName(args[0], false)
	if poble == nil {
		return ConsoleResult{}, fmt.Errorf("No encontré a `%s`.", args[0])
	}

	thought := buildThought(poble)
	event := events.GameEvent{
		ID:           consoleID("thought", c.now().ToMinutes()),
		Type:         events.EventDecisionPoint,
		Timestamp:    c.now(),
		Participants: []string{poble.ID},
		IsPublic:     false,
		Description:  fmt.Sprintf("%s piensa: %s", poble.Name, thought),
	}
	c.appendEvent(event)
	return ConsoleResult{Feedback: event.Description, Event: &event}, nil
}

func (c *ConsoleSystem) handleHelp(args []string) (ConsoleResult, error) {
	names := make([]string, 0, len(c.commands))
	for name := range c.commands {
		names = append(names, name)
	}
	sort.Strings(names)
	lines := []string{"Comandos disponibles:"}
	for _, name := range names {
		command := c.commands[name]
		lines = append(lines, fmt.Sprintf("- %s: %s", command.Name, command.Description))
	}
	return ConsoleResult{Feedback: strings.Join(lines, "\n")}, nil
}

func (c *ConsoleSystem) pushHistory(input, output string, success bool) {
	c.history = append(c.history, ConsoleHistoryEntry{
		Input:      input,
		Output:     output,
		ExecutedAt: time.Now(),
		Success:    success,
	})
	if len(c.history) > 100 {
		c.history = c.history[len(c.history)-100:]
	}
}

func (c *ConsoleSystem) appendEvent(event events.GameEvent) {
	_ = c
	_ = event
}

func (c *ConsoleSystem) findPobleByName(name string, livingOnly bool) *entities.Poble {
	normalized := normalizeConsoleToken(name)
	pool := c.world.GetAllKnownPobles()
	for _, poble := range pool {
		if poble == nil {
			continue
		}
		if livingOnly && !poble.IsAlive {
			continue
		}
		if normalizeConsoleToken(poble.Name) == normalized {
			return poble
		}
	}
	for _, poble := range pool {
		if poble == nil {
			continue
		}
		if livingOnly && !poble.IsAlive {
			continue
		}
		if strings.Contains(normalizeConsoleToken(poble.Name), normalized) {
			return poble
		}
	}
	return nil
}

func (c *ConsoleSystem) findKnownPobleByID(id string) *entities.Poble {
	for _, poble := range c.world.GetAllKnownPobles() {
		if poble != nil && poble.ID == id {
			return poble
		}
	}
	return nil
}

func (c *ConsoleSystem) defaultSpawnLocation() world.Location {
	for _, island := range c.world.Islands {
		if island == nil || !island.IsDiscovered {
			continue
		}
		buildingID := ""
		if len(island.Buildings) > 0 {
			buildingID = island.Buildings[0].ID
		}
		return world.Location{IslandID: island.ID, BuildingID: buildingID}
	}
	return world.Location{IslandID: "island_0"}
}

func (c *ConsoleSystem) now() entities.GameTime {
	if c.timeEngine != nil {
		return c.timeEngine.GetCurrentTime()
	}
	if c.world != nil {
		return c.world.Calendar
	}
	return entities.NewGameTime(0, 0, 0)
}

func normalizeConsoleToken(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	replacer := strings.NewReplacer(
		"á", "a", "é", "e", "í", "i", "ó", "o", "ú", "u", "ü", "u", "ñ", "n",
	)
	value = replacer.Replace(value)
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsNumber(r) || unicode.IsSpace(r) || r == '_' || r == '.' {
			return r
		}
		return -1
	}, value)
}

func parseDeathCause(value string) (events.DeathCause, bool) {
	switch normalizeConsoleToken(value) {
	case "natural", "old", "age", "vejez":
		return events.DeathCauseNaturalAge, true
	case "illness", "sick", "enfermedad":
		return events.DeathCauseIllness, true
	case "accident", "accidente":
		return events.DeathCauseAccident, true
	case "murder", "asesinato":
		return events.DeathCauseMurder, true
	case "suicide", "suicidio":
		return events.DeathCauseSuicide, true
	case "war", "guerra":
		return events.DeathCauseWar, true
	case "execution", "ejecucion", "ejecución":
		return events.DeathCauseExecution, true
	case "childbirth", "parto":
		return events.DeathCauseChildbirth, true
	case "starvation", "hambre":
		return events.DeathCauseStarvation, true
	default:
		return "", false
	}
}

func parseTechID(value string) (world.TechID, bool) {
	target := normalizeConsoleToken(value)
	for _, node := range allConsoleTechNodes() {
		if normalizeConsoleToken(string(node)) == target {
			return node, true
		}
	}
	return "", false
}

func allConsoleTechNodes() []world.TechID {
	return []world.TechID{
		world.TechFireControl,
		world.TechBasicShelter,
		world.TechStoneTools,
		world.TechWriting,
		world.TechOralLaw,
		world.TechCurrency,
		world.TechAgriculture,
		world.TechBasicMedicine,
		world.TechContraception,
		world.TechSurgery,
		world.TechWaterPipes,
		world.TechElectricity,
		world.TechCommunication,
		world.TechNavigation,
		world.TechReproductionTech,
		world.TechComputers,
		world.TechAIResearch,
		world.TechFishingNets,
		world.TechMasonry,
		world.TechMetalworking,
		world.TechIrrigation,
		world.TechPrinting,
		world.TechAstronomy,
	}
}

func parseArchetype(value string) (entities.ArchetypeID, bool) {
	target := normalizeConsoleToken(value)
	all := []entities.ArchetypeID{
		entities.ArchetypeRuler,
		entities.ArchetypeLover,
		entities.ArchetypeJester,
		entities.ArchetypeSage,
		entities.ArchetypeRebel,
		entities.ArchetypeCaretaker,
		entities.ArchetypeVillain,
		entities.ArchetypeGhost,
		entities.ArchetypeAddict,
		entities.ArchetypeProphet,
		entities.ArchetypeSchemer,
		entities.ArchetypeInnocent,
		entities.ArchetypeWarrior,
		entities.ArchetypeDrifter,
		entities.ArchetypeMirror,
	}
	for _, archetype := range all {
		if normalizeConsoleToken(string(archetype)) == target {
			return archetype, true
		}
	}
	return "", false
}

func parseMood(value string) (entities.MoodType, bool) {
	switch normalizeConsoleToken(value) {
	case "happy", "feliz":
		return entities.MoodHappy, true
	case "content", "contento", "contenta":
		return entities.MoodContent, true
	case "neutral":
		return entities.MoodNeutral, true
	case "anxious", "ansioso", "ansiosa":
		return entities.MoodAnxious, true
	case "sad", "triste":
		return entities.MoodSad, true
	case "angry", "enojado", "enojada", "furioso", "furiosa":
		return entities.MoodAngry, true
	case "depressed", "deprimido", "deprimida":
		return entities.MoodDepressed, true
	case "euphoric", "euforico", "euforica":
		return entities.MoodEuphoric, true
	case "obsessive", "obsesivo", "obsesiva":
		return entities.MoodObsessive, true
	case "numb", "vacio", "vacia":
		return entities.MoodNumb, true
	default:
		return "", false
	}
}

func eraFromInt(index int) entities.Era {
	switch index {
	case 0:
		return entities.EraZero
	case 1:
		return entities.EraOne
	case 2:
		return entities.EraTwo
	case 3:
		return entities.EraThree
	case 4:
		return entities.EraFour
	default:
		return entities.EraZero
	}
}

func biologicalPair(a, b *entities.Poble) (*entities.Poble, *entities.Poble, float32) {
	if a == nil || b == nil || !a.IsAlive || !b.IsAlive {
		return nil, nil, 0
	}
	if canCarryPregnancy(a) && canImpregnate(b) {
		return a, b, fertilityChance(a, b)
	}
	if canCarryPregnancy(b) && canImpregnate(a) {
		return b, a, fertilityChance(b, a)
	}
	return nil, nil, 0
}

func canCarryPregnancy(poble *entities.Poble) bool {
	return poble != nil && poble.Sex == entities.Female && poble.Age >= 16 && poble.Age <= 50
}

func canImpregnate(poble *entities.Poble) bool {
	return poble != nil && poble.Sex == entities.Male && poble.Age >= 16 && poble.Health.Fertility > 0.05
}

func fertilityChance(mother, father *entities.Poble) float32 {
	chance := mother.Health.Fertility * father.Health.Fertility
	if mother.Age >= 20 && mother.Age <= 35 {
		chance *= 1
	} else if mother.Age > 35 {
		chance *= 0.65
	} else {
		chance *= 0.75
	}
	if father.Age > 50 {
		chance *= 0.8
	}
	chance *= 1 - ((100-float32(mother.Mental.Stability))*0.003)
	chance *= 1 - ((100-float32(father.Mental.Stability))*0.002)
	return clampFloat(chance, 0, 1)
}

func hasCondition(poble *entities.Poble, condition entities.ConditionID) bool {
	for _, existing := range poble.Health.Conditions {
		if existing == condition {
			return true
		}
	}
	return false
}

func applyMood(poble *entities.Poble, mood entities.MoodType) {
	if poble == nil {
		return
	}
	poble.CurrentMood = mood
	poble.EmotionalState.CurrentMood = mood
}

func relationOrDefault(poble *entities.Poble, targetID string, relationType entities.RelationshipType) entities.Relationship {
	if poble.Relationships == nil {
		poble.Relationships = map[string]entities.Relationship{}
	}
	if rel, ok := poble.Relationships[targetID]; ok {
		return rel
	}
	return entities.NewRelationship(targetID, relationType)
}

func chooseConfessionTarget(actor *entities.Poble, living []*entities.Poble, rng *rand.Rand) *entities.Poble {
	if actor == nil {
		return nil
	}
	for targetID := range actor.Relationships {
		for _, candidate := range living {
			if candidate != nil && candidate.ID == targetID && candidate.IsAlive {
				return candidate
			}
		}
	}
	candidates := make([]*entities.Poble, 0, len(living))
	for _, candidate := range living {
		if candidate != nil && candidate.ID != actor.ID {
			candidates = append(candidates, candidate)
		}
	}
	if len(candidates) == 0 {
		return nil
	}
	return candidates[rng.Intn(len(candidates))]
}

func dominantArchetypeGroups(living []*entities.Poble) (entities.ArchetypeID, entities.ArchetypeID) {
	counts := map[entities.ArchetypeID]int{}
	for _, poble := range living {
		if poble != nil {
			counts[poble.Archetype]++
		}
	}
	type pair struct {
		id    entities.ArchetypeID
		count int
	}
	pairs := make([]pair, 0, len(counts))
	for id, count := range counts {
		pairs = append(pairs, pair{id: id, count: count})
	}
	sort.Slice(pairs, func(i, j int) bool { return pairs[i].count > pairs[j].count })
	if len(pairs) < 2 {
		return "", ""
	}
	return pairs[0].id, pairs[1].id
}

func pickSurvivors(all []*entities.Poble, rng *rand.Rand) [2]*entities.Poble {
	living := make([]*entities.Poble, 0, len(all))
	for _, poble := range all {
		if poble != nil && poble.IsAlive {
			living = append(living, poble)
		}
	}
	first := living[rng.Intn(len(living))]
	second := living[rng.Intn(len(living))]
	for len(living) > 1 && second.ID == first.ID {
		second = living[rng.Intn(len(living))]
	}
	return [2]*entities.Poble{first, second}
}

func buildThought(poble *entities.Poble) string {
	if poble == nil {
		return "..."
	}
	if len(poble.Secrets) > 0 && !poble.Secrets[0].IsRevealed {
		return "Si digo la verdad, algo se rompe."
	}
	switch poble.CurrentMood {
	case entities.MoodAngry:
		return "No quiero paz. Quiero que alguien admita lo que hizo."
	case entities.MoodSad:
		return "Todo se siente más lejos de lo que estaba ayer."
	case entities.MoodAnxious:
		return "Algo malo viene. Solo no sé por dónde."
	case entities.MoodEuphoric:
		return "Tal vez hoy sí cambie todo."
	default:
		if len(poble.Memories) > 0 {
			return poble.Memories[len(poble.Memories)-1].Summary
		}
		return "Estoy aquí. Eso ya significa algo."
	}
}

func uniqueStrings(values []string) []string {
	seen := map[string]bool{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	return result
}

func consoleID(prefix string, minute int) string {
	return fmt.Sprintf("%s_%d", prefix, minute)
}

func clampFloat(value, min, max float32) float32 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func clampInt(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
