package minigames

import (
	"fmt"
	"strings"

	"github.com/user/poblation/internal/entities"
	"github.com/user/poblation/internal/events"
	"github.com/user/poblation/internal/templates"
	"github.com/user/poblation/internal/world"
)

// FightPhase marks the current stage of the fight.
type FightPhase string

const (
	FightPhaseVerbalClash FightPhase = "VERBAL_CLASH"
	FightPhaseEscalation  FightPhase = "ESCALATION"
	FightPhasePhysical    FightPhase = "PHYSICAL"
	FightPhaseOutcome     FightPhase = "OUTCOME"
)

// FightOutcomeType stores the final state of the conflict.
type FightOutcomeType string

const (
	FightOutcomeStopped   FightOutcomeType = "STOPPED"
	FightOutcomeLoserHurt FightOutcomeType = "LOSER_HURT"
	FightOutcomeBothHurt  FightOutcomeType = "BOTH_HURT"
	FightOutcomeFatal     FightOutcomeType = "FATAL"
	FightOutcomeTruce     FightOutcomeType = "TRUCE"
	FightOutcomeCatharsis FightOutcomeType = "CATHARSIS"
)

// FightBeat is one visible step of the fight script.
type FightBeat struct {
	Phase        FightPhase
	Text         string
	Delay        int
	Danger       bool
	CanIntervene bool
}

// FightOutcome stores the consequences decided for one fight.
type FightOutcome struct {
	Type           FightOutcomeType
	WinnerID       string
	LoserID        string
	Public         bool
	Witnesses      []string
	EventTriggered *events.GameEvent
	EventSummary   string
}

// FightState stores one escalating fight session.
type FightState struct {
	Attacker          *entities.Poble
	Defender          *entities.Poble
	Trigger           string
	Phase             FightPhase
	Intensity         int
	CanDeescalate     bool
	FatalChance       float32
	Weapon            *entities.Item
	Outcome           *FightOutcome
	StartedAt         entities.GameTime
	Public            bool
	WitnessIDs        []string
	RelationshipDelta map[string]int
	Beats             []FightBeat
	CurrentBeat       int
	ResultApplied     bool
}

// NewFightState creates the base state for a conflict that may turn physical.
func NewFightState(attacker, defender *entities.Poble, w *world.World, trigger string) FightState {
	state := FightState{
		Attacker:          attacker,
		Defender:          defender,
		Trigger:           strings.TrimSpace(trigger),
		Phase:             FightPhaseVerbalClash,
		RelationshipDelta: map[string]int{},
		Beats:             []FightBeat{},
	}
	if w != nil {
		state.StartedAt = w.Calendar
	}
	if state.Trigger == "" {
		state.Trigger = fallbackFightTrigger(attacker, defender)
	}
	state.Weapon = selectFightWeapon(attacker, defender)
	state.WitnessIDs, state.Public = detectFightWitnesses(w, attacker, defender)
	state.Intensity = fightStartingIntensity(attacker, defender, state.Trigger, state.Weapon)
	state.CanDeescalate = state.Intensity < 5
	state.FatalChance = fightFatalChance(attacker, defender, state.Weapon, len(state.WitnessIDs))
	state.Outcome = resolveFightOutcome(state)
	state.RelationshipDelta = fightRelationshipDelta(state)
	state.Beats = buildFightBeats(state, nil, w)
	return state
}

// BuildFightBeats generates the text script for the fight.
func BuildFightBeats(state FightState, engine *templates.TemplateEngine, w *world.World) []FightBeat {
	return buildFightBeats(state, engine, w)
}

// DeescalateFight lets the player or director pull the fight back before it breaks further.
func DeescalateFight(state *FightState) {
	if state == nil || state.Attacker == nil || state.Defender == nil {
		return
	}
	state.Phase = FightPhaseOutcome
	state.CanDeescalate = false
	state.Intensity = max(1, state.Intensity-2)
	state.Outcome = &FightOutcome{
		Type:         FightOutcomeStopped,
		Public:       state.Public,
		Witnesses:    append([]string(nil), state.WitnessIDs...),
		EventSummary: "La pelea se corta antes de romper hueso.",
		EventTriggered: &events.GameEvent{
			ID:           fightEventID(*state, "stopped"),
			Type:         events.EventFightVerbal,
			Timestamp:    state.StartedAt,
			Participants: fightParticipants(*state),
			IsPublic:     state.Public,
			Description:  "La pelea se corta antes de volverse fisica.",
		},
	}
	state.RelationshipDelta = map[string]int{
		relationshipKey(state.Attacker.ID, state.Defender.ID): -6,
		relationshipKey(state.Defender.ID, state.Attacker.ID): -6,
	}
}

// ApplyFightResult commits injuries, memories, rumours, and death fallout into the world.
func ApplyFightResult(state FightState, w *world.World) {
	if w == nil || state.ResultApplied {
		return
	}
	if state.Outcome == nil {
		state.Outcome = resolveFightOutcome(state)
	}

	attacker := worldParticipant(w, state.Attacker)
	defender := worldParticipant(w, state.Defender)
	if attacker == nil || defender == nil {
		return
	}

	applyFightRelationshipDelta(attacker, defender, state.RelationshipDelta)
	addFightMemory(attacker, defender, state, false)
	addFightMemory(defender, attacker, state, false)
	addWitnessFightMemories(w, attacker, defender, state)

	switch state.Outcome.Type {
	case FightOutcomeLoserHurt:
		applyFightDamage(fightLoser(state, attacker, defender), fightDamageFor(state, false))
	case FightOutcomeBothHurt, FightOutcomeCatharsis:
		damage := fightDamageFor(state, true)
		applyFightDamage(attacker, damage)
		applyFightDamage(defender, damage)
	case FightOutcomeFatal:
		loser := fightLoser(state, attacker, defender)
		if loser != nil {
			_ = events.HandleDeath(loser, events.DeathCauseMurder, w)
		}
	}

	if state.Public || len(state.WitnessIDs) > 0 {
		w.RumourPool = append(w.RumourPool, world.Rumour{
			ID:          fightEventID(state, "rumour"),
			SourceID:    attacker.ID,
			AboutID:     defender.ID,
			Content:     fightRumourText(state, attacker, defender),
			Truthiness:  0.92,
			SpreadLevel: 0.66,
			Tags:        []string{"fight", "violence", strings.ToLower(string(state.Outcome.Type))},
		})
	}
}

func buildFightBeats(state FightState, engine *templates.TemplateEngine, w *world.World) []FightBeat {
	if state.Attacker == nil || state.Defender == nil {
		return nil
	}

	beats := []FightBeat{
		{
			Phase:        FightPhaseVerbalClash,
			Text:         renderFightTemplate(engine, w, state, "dialogues/argument/petty", fallbackFightVerbal(state, 0)),
			Delay:        760,
			CanIntervene: state.CanDeescalate,
		},
		{
			Phase:        FightPhaseVerbalClash,
			Text:         renderFightTemplate(engine, w, state, "dialogues/argument/escalating", fallbackFightVerbal(state, 1)),
			Delay:        860,
			CanIntervene: state.CanDeescalate,
		},
		{
			Phase:        FightPhaseEscalation,
			Text:         fightEscalationText(state),
			Delay:        900,
			CanIntervene: state.CanDeescalate,
		},
	}

	switch state.Outcome.Type {
	case FightOutcomeStopped, FightOutcomeTruce:
		beats = append(beats, FightBeat{
			Phase: FightPhaseOutcome,
			Text:  fightOutcomeLine(state),
			Delay: 900,
		})
	default:
		beats = append(beats,
			FightBeat{
				Phase:  FightPhasePhysical,
				Text:   renderFightTemplate(engine, w, state, "reactions/violence/fight_physical", fallbackFightPhysical(state, 0)),
				Delay:  980,
				Danger: true,
			},
			FightBeat{
				Phase:  FightPhasePhysical,
				Text:   fallbackFightPhysical(state, 1),
				Delay:  1000,
				Danger: true,
			},
			FightBeat{
				Phase: FightPhaseOutcome,
				Text:  fightOutcomeLine(state),
				Delay: 900,
			},
		)
	}

	return beats
}

func resolveFightOutcome(state FightState) *FightOutcome {
	if state.Attacker == nil || state.Defender == nil {
		return &FightOutcome{Type: FightOutcomeStopped}
	}

	attackerScore := fightPower(state.Attacker) + float32(state.Intensity)
	defenderScore := fightPower(state.Defender) + float32(max(1, 11-state.Intensity))
	if state.Weapon != nil {
		attackerScore += 14
	}
	diff := attackerScore - defenderScore
	public := state.Public
	witnesses := append([]string(nil), state.WitnessIDs...)

	if state.CanDeescalate && state.Intensity <= 3 {
		return makeFightOutcome(state, FightOutcomeTruce, "", "", public, witnesses, "La furia no desaparece, pero se queda sin ultimo paso.")
	}

	if state.Weapon != nil && state.FatalChance > 0 && encounterRoll(state.Attacker, state.Defender, "fight:fatal", state.StartedAt) < state.FatalChance {
		winner, loser := fightWinnerLoser(diff, state.Attacker, state.Defender)
		return makeFightOutcome(state, FightOutcomeFatal, winner.ID, loser.ID, public, witnesses, "La pelea cruza una linea que ya no se puede deshacer.")
	}

	if state.Intensity >= 8 && encounterRoll(state.Attacker, state.Defender, "fight:catharsis", state.StartedAt) < 0.08 {
		winner, loser := fightWinnerLoser(diff, state.Attacker, state.Defender)
		return makeFightOutcome(state, FightOutcomeCatharsis, winner.ID, loser.ID, public, witnesses, "Se lastiman, pero el odio termina diciendo una verdad rara.")
	}

	if diff > 9 || diff < -9 {
		winner, loser := fightWinnerLoser(diff, state.Attacker, state.Defender)
		return makeFightOutcome(state, FightOutcomeLoserHurt, winner.ID, loser.ID, public, witnesses, fmt.Sprintf("%s sale peor parado.", loser.Name))
	}

	return makeFightOutcome(state, FightOutcomeBothHurt, state.Attacker.ID, state.Defender.ID, public, witnesses, "Los dos terminan marcados.")
}

func makeFightOutcome(state FightState, outcomeType FightOutcomeType, winnerID, loserID string, public bool, witnesses []string, summary string) *FightOutcome {
	eventType := events.EventFightVerbal
	if outcomeType != FightOutcomeStopped && outcomeType != FightOutcomeTruce {
		eventType = events.EventFightPhysical
	}
	description := summary
	if strings.TrimSpace(description) == "" {
		description = "Una pelea deja grietas nuevas."
	}
	return &FightOutcome{
		Type:         outcomeType,
		WinnerID:     winnerID,
		LoserID:      loserID,
		Public:       public,
		Witnesses:    witnesses,
		EventSummary: summary,
		EventTriggered: &events.GameEvent{
			ID:           fightEventID(state, strings.ToLower(string(outcomeType))),
			Type:         eventType,
			Timestamp:    state.StartedAt,
			Participants: fightParticipants(state),
			IsPublic:     public,
			Description:  description,
		},
	}
}

func renderFightTemplate(engine *templates.TemplateEngine, w *world.World, state FightState, category, fallback string) string {
	if engine == nil || w == nil || state.Attacker == nil || state.Defender == nil {
		return fallback
	}

	relationship := relationBetween(state.Attacker, state.Defender)
	recent, old := fightRelevantMemories(state.Attacker, state.Defender.ID)
	ctx := templates.TemplateContext{
		Speaker:                state.Attacker,
		Target:                 state.Defender,
		Location:               fightLocationLabel(w, state.Attacker),
		GameTime:               state.StartedAt,
		RecentMemory:           recent,
		OldMemory:              old,
		RelationshipWithTarget: &relationship,
		WorldState:             w.GetWorldState(),
		ExtraVars:              map[string]string{},
	}
	selected, err := engine.Select(category, ctx)
	if err != nil {
		return fallback
	}
	rendered, err := engine.Render(selected, ctx)
	if err != nil {
		return fallback
	}
	rendered = strings.TrimSpace(rendered)
	if rendered == "" {
		return fallback
	}
	return rendered
}

func fallbackFightTrigger(attacker, defender *entities.Poble) string {
	if attacker == nil || defender == nil {
		return "una herida vieja"
	}
	relationship := relationBetween(attacker, defender)
	switch {
	case relationship.Resentment >= 75:
		return "una herida vieja que nadie quiso cerrar"
	case relationship.Attraction >= 55 && relationship.Resentment >= 45:
		return "celos mezclados con cosas que nadie admite"
	default:
		return "algo pequeno que ya venia cargado"
	}
}

func selectFightWeapon(attacker, defender *entities.Poble) *entities.Item {
	for _, owner := range []*entities.Poble{attacker, defender} {
		if owner == nil {
			continue
		}
		for index := range owner.Inventory {
			item := owner.Inventory[index]
			label := strings.ToLower(item.Type + " " + item.Name + " " + strings.Join(item.Tags, " "))
			if strings.Contains(label, "weapon") || strings.Contains(label, "knife") ||
				strings.Contains(label, "blade") || strings.Contains(label, "club") ||
				strings.Contains(label, "sharp") {
				copy := item
				return &copy
			}
		}
	}
	return nil
}

func detectFightWitnesses(w *world.World, attacker, defender *entities.Poble) ([]string, bool) {
	if w == nil || attacker == nil || defender == nil {
		return nil, false
	}
	attackerLoc, ok := w.GetLocation(attacker.ID)
	if !ok {
		return nil, false
	}
	witnesses := []string{}
	for _, candidate := range w.GetAllPobles() {
		if candidate == nil || candidate.ID == attacker.ID || candidate.ID == defender.ID {
			continue
		}
		loc, ok := w.GetLocation(candidate.ID)
		if !ok || loc.IslandID != attackerLoc.IslandID {
			continue
		}
		if loc.BuildingID != "" && loc.BuildingID == attackerLoc.BuildingID {
			witnesses = append(witnesses, candidate.ID)
			continue
		}
		if absInt(loc.X-attackerLoc.X) <= 2 && absInt(loc.Y-attackerLoc.Y) <= 2 {
			witnesses = append(witnesses, candidate.ID)
		}
	}
	return witnesses, len(witnesses) > 0 || attackerLoc.BuildingID == ""
}

func fightStartingIntensity(attacker, defender *entities.Poble, trigger string, weapon *entities.Item) int {
	base := 3
	if attacker != nil {
		base += int(attacker.Personality.Cruelty / 22)
		base += int(attacker.Personality.Neuroticism / 28)
	}
	if defender != nil {
		base += int(defender.Personality.Cruelty / 35)
		base += int(defender.Personality.Neuroticism / 35)
	}
	triggerLower := strings.ToLower(trigger)
	if strings.Contains(triggerLower, "cel") || strings.Contains(triggerLower, "traic") || strings.Contains(triggerLower, "ment") {
		base++
	}
	if weapon != nil {
		base += 2
	}
	if base < 1 {
		base = 1
	}
	if base > 10 {
		base = 10
	}
	return base
}

func fightFatalChance(attacker, defender *entities.Poble, weapon *entities.Item, witnessCount int) float32 {
	chance := float32(0.01)
	if attacker != nil {
		chance += attacker.Personality.Cruelty / 1500
	}
	if defender != nil {
		chance += defender.Personality.Cruelty / 2400
	}
	if weapon != nil {
		chance += 0.07
	}
	if witnessCount > 0 {
		chance -= 0.03
	}
	if chance < 0.005 {
		chance = 0.005
	}
	if chance > 0.22 {
		chance = 0.22
	}
	return chance
}

func fightRelationshipDelta(state FightState) map[string]int {
	if state.Attacker == nil || state.Defender == nil {
		return map[string]int{}
	}
	base := -6 - state.Intensity
	if state.Outcome != nil && state.Outcome.Type == FightOutcomeCatharsis {
		base = 4
	}
	if state.Outcome != nil && state.Outcome.Type == FightOutcomeTruce {
		base = -3
	}
	return map[string]int{
		relationshipKey(state.Attacker.ID, state.Defender.ID): base,
		relationshipKey(state.Defender.ID, state.Attacker.ID): base,
	}
}

func fightPower(poble *entities.Poble) float32 {
	if poble == nil {
		return 0
	}
	power := float32(poble.Health.HP) * 0.45
	power += float32(100-poble.Needs.Sleep) * 0.12
	power += poble.Personality.Cruelty * 0.25
	power -= poble.Personality.Neuroticism * 0.12
	return power
}

func fightWinnerLoser(diff float32, attacker, defender *entities.Poble) (*entities.Poble, *entities.Poble) {
	if diff >= 0 {
		return attacker, defender
	}
	return defender, attacker
}

func fightLoser(state FightState, attacker, defender *entities.Poble) *entities.Poble {
	if state.Outcome == nil {
		return defender
	}
	if attacker != nil && state.Outcome.LoserID == attacker.ID {
		return attacker
	}
	if defender != nil && state.Outcome.LoserID == defender.ID {
		return defender
	}
	return defender
}

func fightParticipants(state FightState) []string {
	ids := []string{}
	if state.Attacker != nil {
		ids = append(ids, state.Attacker.ID)
	}
	if state.Defender != nil {
		ids = append(ids, state.Defender.ID)
	}
	return ids
}

func fightEscalationText(state FightState) string {
	if state.Attacker == nil || state.Defender == nil {
		return "El aire se parte y nadie sabe retroceder."
	}
	switch {
	case state.Intensity >= 8:
		return fmt.Sprintf("%s da un paso demasiado cerca. %s ya no contesta para entender: contesta para herir.", state.Attacker.Name, state.Defender.Name)
	case state.Intensity >= 5:
		return fmt.Sprintf("La discusion deja de sonar domestica. %s y %s ya estan midiendo distancia, no razones.", state.Attacker.Name, state.Defender.Name)
	default:
		return fmt.Sprintf("%s todavia podria bajar la voz, pero %s ya tiene los hombros tensos.", state.Attacker.Name, state.Defender.Name)
	}
}

func fallbackFightVerbal(state FightState, index int) string {
	if state.Attacker == nil || state.Defender == nil {
		return "La pelea arranca en seco."
	}
	if index == 0 {
		return fmt.Sprintf("[%s]: No me hables como si esto fuera pequeno.\n[%s]: Pequeño era antes de que abrieras la boca asi.", state.Attacker.Name, state.Defender.Name)
	}
	return fmt.Sprintf("[%s]: Ya cruzaste una linea.\n[%s]: Entonces deja de fingir que no querias verla.", state.Defender.Name, state.Attacker.Name)
}

func fallbackFightPhysical(state FightState, index int) string {
	if state.Attacker == nil || state.Defender == nil {
		return "La pelea se vuelve fisica."
	}
	switch index {
	case 0:
		return fmt.Sprintf("El primer empujon no suena heroico. Suena torpe, rabioso, demasiado humano entre %s y %s.", state.Attacker.Name, state.Defender.Name)
	default:
		return fmt.Sprintf("Ya no estan discutiendo el tema. Estan dejando que el cuerpo traduzca lo peor de la discusion.")
	}
}

func fightOutcomeLine(state FightState) string {
	if state.Outcome == nil {
		return "La pelea no deja nada limpio."
	}
	switch state.Outcome.Type {
	case FightOutcomeStopped:
		return "Alguien logra cortar el impulso antes del daño serio."
	case FightOutcomeTruce:
		return "Se separan con la respiracion rota y una tregua miserable."
	case FightOutcomeLoserHurt:
		return state.Outcome.EventSummary
	case FightOutcomeBothHurt:
		return "Nadie gana. Solo cambia quien duele primero."
	case FightOutcomeFatal:
		return "Esta vez la pelea no termina en pelea. Termina en muerte."
	case FightOutcomeCatharsis:
		return "Lo absurdo: de la violencia sale una verdad que los deja temblando distinto."
	default:
		return "La pelea deja su propia resaca."
	}
}

func fightDamageFor(state FightState, both bool) int {
	damage := 10 + state.Intensity*2
	if state.Weapon != nil {
		damage += 8
	}
	if both {
		damage -= 6
	}
	if damage < 6 {
		damage = 6
	}
	if damage > 48 {
		damage = 48
	}
	return damage
}

func applyFightDamage(poble *entities.Poble, damage int) {
	if poble == nil || !poble.IsAlive {
		return
	}
	poble.Health.HP -= damage
	if poble.Health.HP < 1 {
		poble.Health.HP = 1
	}
	if !hasCondition(poble, entities.ConditionInjured) {
		poble.Health.Conditions = append(poble.Health.Conditions, entities.ConditionInjured)
	}
	poble.CurrentMood = entities.MoodAngry
	poble.EmotionalState.CurrentMood = entities.MoodAngry
}

func applyFightRelationshipDelta(attacker, defender *entities.Poble, deltas map[string]int) {
	for _, pair := range [][2]*entities.Poble{{attacker, defender}, {defender, attacker}} {
		source := pair[0]
		target := pair[1]
		if source == nil || target == nil {
			continue
		}
		relation := ensureRelationship(source, target.ID)
		delta := float32(deltas[relationshipKey(source.ID, target.ID)])
		relation.Resentment = clampPercent(relation.Resentment + float32(max(2, -int(delta))))
		relation.Trust = clampPercent(relation.Trust + (delta * 0.45))
		relation.Affection = clampPercent(relation.Affection + (delta * 0.25))
		relation.Respect = clampPercent(relation.Respect + (delta * 0.18))
		if delta > 0 {
			relation.Resentment = clampPercent(relation.Resentment - (delta * 0.35))
		}
		source.Relationships[target.ID] = relation
	}
}

func addFightMemory(owner, other *entities.Poble, state FightState, witness bool) {
	if owner == nil {
		return
	}
	memoryType := entities.MemoryViolent
	if state.Outcome != nil && state.Outcome.Type == FightOutcomeFatal {
		memoryType = entities.MemoryTraumatic
	}
	summary := fightMemorySummary(owner, other, state, witness)
	memory := entities.NewMemory(
		fightEventID(state, "memory:"+owner.ID),
		state.StartedAt,
		memoryType,
		summary,
	)
	memory.EmotionIntensity = 82
	memory.Tags = []string{"fight", "violence", strings.ToLower(string(state.Outcome.Type))}
	memory.Participants = fightMemoryParticipants(owner, other, state)
	owner.Memories = append(owner.Memories, memory)
}

func addWitnessFightMemories(w *world.World, attacker, defender *entities.Poble, state FightState) {
	if w == nil {
		return
	}
	for _, witnessID := range state.WitnessIDs {
		witness := w.GetPoble(witnessID)
		if witness == nil {
			continue
		}
		addFightMemory(witness, attacker, state, true)
		if defender != nil {
			witness.EmotionalState.ActiveEmotions = append(witness.EmotionalState.ActiveEmotions, entities.EmotionFear)
		}
	}
}

func fightMemorySummary(owner, other *entities.Poble, state FightState, witness bool) string {
	if owner == nil {
		return "Una pelea deja el pulso raro."
	}
	if witness {
		if other == nil {
			return fmt.Sprintf("%s vio una pelea que no penso olvidar rapido.", owner.Name)
		}
		return fmt.Sprintf("%s vio a %s y %s cruzar de palabras a violencia demasiado rapido.", owner.Name, state.Attacker.Name, state.Defender.Name)
	}
	if other == nil {
		return fmt.Sprintf("%s recuerda la pelea como una puerta rota dentro del dia.", owner.Name)
	}
	switch state.Outcome.Type {
	case FightOutcomeFatal:
		return fmt.Sprintf("%s recuerda el instante en que la pelea con %s dejo de parecer reversible.", owner.Name, other.Name)
	case FightOutcomeCatharsis:
		return fmt.Sprintf("%s no puede sacarse de encima que la pelea con %s dolio y aclaro cosas al mismo tiempo.", owner.Name, other.Name)
	default:
		return fmt.Sprintf("%s recuerda la pelea con %s como una discusion que el cuerpo termino por ellos.", owner.Name, other.Name)
	}
}

func fightMemoryParticipants(owner, other *entities.Poble, state FightState) []string {
	ids := []string{owner.ID}
	if other != nil {
		ids = append(ids, other.ID)
	}
	if state.Attacker != nil && state.Attacker.ID != owner.ID && (other == nil || state.Attacker.ID != other.ID) {
		ids = append(ids, state.Attacker.ID)
	}
	if state.Defender != nil && state.Defender.ID != owner.ID && (other == nil || state.Defender.ID != other.ID) {
		ids = append(ids, state.Defender.ID)
	}
	return ids
}

func fightRumourText(state FightState, attacker, defender *entities.Poble) string {
	if attacker == nil || defender == nil {
		return "Alguien dice que hubo una pelea."
	}
	if state.Outcome != nil && state.Outcome.Type == FightOutcomeFatal {
		return fmt.Sprintf("Corre el rumor de que la pelea entre %s y %s termino en algo irreparable.", attacker.Name, defender.Name)
	}
	return fmt.Sprintf("Ya corre el rumor de la pelea entre %s y %s.", attacker.Name, defender.Name)
}

func fightRelevantMemories(source *entities.Poble, targetID string) (*entities.Memory, *entities.Memory) {
	if source == nil || targetID == "" {
		return nil, nil
	}
	var recent *entities.Memory
	var old *entities.Memory
	for index := range source.Memories {
		memory := source.Memories[index]
		if !memoryInvolves(memory, targetID) {
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

func fightLocationLabel(w *world.World, attacker *entities.Poble) string {
	if w == nil || attacker == nil {
		return "sin lugar"
	}
	location, ok := w.GetLocation(attacker.ID)
	if !ok {
		return "sin lugar"
	}
	if island := w.GetIsland(location.IslandID); island != nil {
		return fmt.Sprintf("%s (%d,%d)", island.Name, location.X, location.Y)
	}
	return fmt.Sprintf("%s (%d,%d)", location.IslandID, location.X, location.Y)
}

func fightEventID(state FightState, suffix string) string {
	left := "unknown"
	right := "unknown"
	if state.Attacker != nil {
		left = state.Attacker.ID
	}
	if state.Defender != nil {
		right = state.Defender.ID
	}
	return fmt.Sprintf("fight:%s:%s:%d:%s", left, right, state.StartedAt.Day, suffix)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
