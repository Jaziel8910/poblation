package ai

import (
	"math"
	"math/rand"
	"sort"
	"strings"

	"github.com/user/poblation/internal/entities"
)

// Poble reuses the core entity type while keeping decision signatures local.
type Poble = entities.Poble

// World is the minimal view the decision engine needs. internal/world.World
// satisfies this without creating an import cycle back into internal/ai.
type World interface {
	GetAllPobles() []*Poble
}

type activeEventsProvider interface {
	GetActiveEvents() []GameEvent
}

type currentTimeProvider interface {
	GetCurrentTime() entities.GameTime
}

type proximityProvider interface {
	GetProximityScore(fromID, targetID string) float32
}

// ActionType identifies one procedural behavior a Poble can choose.
type ActionType string

const (
	ActionSleep           ActionType = "SLEEP"
	ActionEat             ActionType = "EAT"
	ActionDrink           ActionType = "DRINK"
	ActionWork            ActionType = "WORK"
	ActionExplore         ActionType = "EXPLORE"
	ActionTalkTo          ActionType = "TALK_TO"
	ActionArgueWith       ActionType = "ARGUE_WITH"
	ActionFlirtWith       ActionType = "FLIRT_WITH"
	ActionConfessTo       ActionType = "CONFESS_TO"
	ActionGossipWith      ActionType = "GOSSIP_WITH"
	ActionThreaten        ActionType = "THREATEN"
	ActionObserveSecretly ActionType = "OBSERVE_SECRETLY"
	ActionWriteDiary      ActionType = "WRITE_DIARY"
	ActionSendLetter      ActionType = "SEND_LETTER"
	ActionPlanRevenge     ActionType = "PLAN_REVENGE"
	ActionFormAlliance    ActionType = "FORM_ALLIANCE"
	ActionBetray          ActionType = "BETRAY"
	ActionHaveSex         ActionType = "HAVE_SEX"
	ActionPropose         ActionType = "PROPOSE"
	ActionBreakUp         ActionType = "BREAK_UP"
	ActionFight           ActionType = "FIGHT"
	ActionMurder          ActionType = "MURDER"
	ActionPray            ActionType = "PRAY"
	ActionGovern          ActionType = "GOVERN"
	ActionTrade           ActionType = "TRADE"
	ActionBuild           ActionType = "BUILD"
	ActionResearch        ActionType = "RESEARCH"
	ActionHaveBreakdown   ActionType = "HAVE_BREAKDOWN"
	ActionIsolate         ActionType = "ISOLATE"
	ActionParty           ActionType = "PARTY"
	ActionRest            ActionType = "REST"
)

var allActionTypes = []ActionType{
	ActionSleep,
	ActionEat,
	ActionDrink,
	ActionWork,
	ActionExplore,
	ActionTalkTo,
	ActionArgueWith,
	ActionFlirtWith,
	ActionConfessTo,
	ActionGossipWith,
	ActionThreaten,
	ActionObserveSecretly,
	ActionWriteDiary,
	ActionSendLetter,
	ActionPlanRevenge,
	ActionFormAlliance,
	ActionBetray,
	ActionHaveSex,
	ActionPropose,
	ActionBreakUp,
	ActionFight,
	ActionMurder,
	ActionPray,
	ActionGovern,
	ActionTrade,
	ActionBuild,
	ActionResearch,
	ActionHaveBreakdown,
	ActionIsolate,
	ActionParty,
	ActionRest,
}

// Action stores one chosen or candidate behavior.
type Action struct {
	Type          ActionType
	TargetID      string
	Priority      int
	Duration      int
	FailureChance float32
	OnComplete    func()
	OnFail        func()
	Tags          []string
}

// DecisionEngine chooses procedural actions from needs, emotions, memory,
// relationships, archetype, world state, and a controlled dose of chaos.
type DecisionEngine struct {
	poble         *Poble
	world         World
	rng           *rand.Rand
	currentAction *Action
	actionQueue   []Action
}

// NewDecisionEngine binds a decision engine to one Poble.
func NewDecisionEngine(poble *Poble, world World, rng *rand.Rand) *DecisionEngine {
	if rng == nil {
		rng = rand.New(rand.NewSource(1))
	}

	return &DecisionEngine{
		poble:       poble,
		world:       world,
		rng:         rng,
		actionQueue: []Action{},
	}
}

// Decide runs the full decision pipeline and returns current candidate actions.
func (e *DecisionEngine) Decide(deltaHours int) []Action {
	if e == nil || e.poble == nil || !e.poble.IsAlive {
		return nil
	}

	actions := make([]Action, 0, 12)
	survival := e.CheckSurvivalNeeds()
	actions = append(actions, survival...)
	actions = append(actions, e.CheckEmotionalUrgency()...)
	actions = append(actions, e.CheckRelationshipGoals()...)
	actions = append(actions, e.CheckArchetypeGoal()...)
	actions = append(actions, e.CheckWorldState()...)

	if len(actions) == 0 {
		actions = append(actions, e.DefaultBehavior()...)
	}
	if len(survival) == 0 {
		actions = append(actions, e.AddRandomness()...)
	}

	actions = e.normalizeActions(actions, deltaHours)
	if len(actions) == 0 {
		e.actionQueue = nil
		return nil
	}

	e.actionQueue = actions
	best := actions[0]
	if e.currentAction == nil || e.ShouldInterruptCurrentAction(best) {
		e.setCurrentAction(best)
	}

	return append([]Action(nil), actions...)
}

// CheckSurvivalNeeds creates maximum-priority food and water actions.
func (e *DecisionEngine) CheckSurvivalNeeds() []Action {
	if e == nil || e.poble == nil {
		return nil
	}

	actions := make([]Action, 0, 2)
	if e.poble.Needs.Thirst > 90 {
		actions = append(actions, newAction(ActionDrink, "", 100, 1, 0.01, "survival", "need:thirst"))
	}
	if e.poble.Needs.Hunger > 90 {
		actions = append(actions, newAction(ActionEat, "", 100, 1, 0.02, "survival", "need:hunger"))
	}

	sortActions(actions)
	return actions
}

// CheckEmotionalUrgency maps intense active emotions into immediate behavior.
func (e *DecisionEngine) CheckEmotionalUrgency() []Action {
	if e == nil || e.poble == nil {
		return nil
	}

	emotion, intensity, ok := e.dominantEmotion()
	if !ok || intensity <= 85 {
		return nil
	}

	priority := 78 + int((intensity-85)/3)
	switch emotion {
	case entities.EmotionGrief, entities.EmotionLoneliness:
		return []Action{newAction(ActionIsolate, "", priority+8, 4, 0.08, "emotion:"+emotion.String(), "solitary", "negative_valence")}
	case entities.EmotionAnger, entities.EmotionContempt, entities.EmotionResentment:
		target := e.firstTarget(ActionArgueWith)
		if e.canAttemptMurder(target, intensity) {
			return []Action{newAction(ActionMurder, target, 99, 2, 0.72, "emotion:"+emotion.String(), "violence", "rare", "extreme")}
		}
		if intensity > 94 && e.rng.Float32() < 0.35 {
			return []Action{newAction(ActionFight, target, priority+8, 1, 0.42, "emotion:"+emotion.String(), "violence", "high_arousal")}
		}
		if e.rng.Float32() < 0.45 {
			return []Action{newAction(ActionThreaten, target, priority+5, 1, 0.28, "emotion:"+emotion.String(), "threat", "high_arousal")}
		}
		return []Action{newAction(ActionArgueWith, target, priority+3, 1, 0.22, "emotion:"+emotion.String(), "argument", "high_arousal")}
	case entities.EmotionFear, entities.EmotionAnxiety:
		if e.poble.Archetype == entities.ArchetypeWarrior && intensity > 92 {
			return []Action{newAction(ActionFight, e.firstTarget(ActionFight), priority+4, 1, 0.44, "emotion:"+emotion.String(), "survival", "violence")}
		}
		return []Action{newAction(ActionIsolate, "", priority+4, 3, 0.10, "emotion:"+emotion.String(), "safety", "negative_valence")}
	case entities.EmotionGuilt, entities.EmotionShame:
		if target := e.firstTarget(ActionConfessTo); target != "" && e.hasBurningSecret() {
			return []Action{newAction(ActionConfessTo, target, priority+6, 1, 0.35, "emotion:"+emotion.String(), "confession", "secret")}
		}
		return []Action{newAction(ActionWriteDiary, "", priority, 1, 0.04, "emotion:"+emotion.String(), "diary", "private")}
	case entities.EmotionLove:
		if target := e.firstTarget(ActionFlirtWith); target != "" && e.canRomance() {
			return []Action{newAction(ActionFlirtWith, target, priority+3, 1, 0.24, "emotion:love", "flirt", "romance")}
		}
	case entities.EmotionLust:
		if target := e.firstTarget(ActionHaveSex); target != "" && e.canHaveSexWith(target) {
			return []Action{newAction(ActionHaveSex, target, priority+5, 2, 0.35, "emotion:lust", "sex", "private")}
		}
	case entities.EmotionJoy, entities.EmotionPride, entities.EmotionRelief:
		if e.rng.Float32() < 0.50 {
			return []Action{newAction(ActionParty, "", priority, 3, 0.12, "emotion:"+emotion.String(), "celebration", "positive_valence")}
		}
		return []Action{newAction(ActionTalkTo, e.firstTarget(ActionTalkTo), priority, 1, 0.08, "emotion:"+emotion.String(), "social", "positive_valence")}
	case entities.EmotionBoredom, entities.EmotionCuriosity, entities.EmotionConfusion:
		return []Action{newAction(ActionExplore, "", priority, 3, 0.20, "emotion:"+emotion.String(), "curiosity", "world")}
	}

	if e.poble.Mental.Stability <= 12 {
		return []Action{newAction(ActionHaveBreakdown, "", 96, 4, 0.18, "mental", "breakdown", "high_intensity")}
	}

	return nil
}

// CheckRelationshipGoals turns active relationship flags into social actions.
func (e *DecisionEngine) CheckRelationshipGoals() []Action {
	if e == nil || e.poble == nil {
		return nil
	}

	actions := make([]Action, 0, len(e.poble.Relationships)+1)
	if e.hasBurningSecret() {
		if target := e.firstTarget(ActionConfessTo); target != "" && e.roll(0.38) {
			actions = append(actions, newAction(ActionConfessTo, target, 78, 1, 0.34, "relationship", "confession", "secret"))
		}
	}

	targetIDs := e.sortedRelationshipIDs()
	for _, targetID := range targetIDs {
		relationship := e.poble.Relationships[targetID]
		switch {
		case e.isCrush(relationship):
			if !e.canRomanceWith(targetID) {
				continue
			}
			priority := 62 + int(relationship.Attraction/10)
			if e.roll(0.58) {
				actions = append(actions, newAction(ActionFlirtWith, targetID, priority, 1, 0.26, "relationship:crush", "flirt", "romance"))
			} else if e.roll(0.45) {
				actions = append(actions, newAction(ActionObserveSecretly, targetID, priority-4, 2, 0.40, "relationship:crush", "obsession", "secret"))
			}
		case e.isHighResentment(relationship):
			priority := 64 + int(relationship.Resentment/8)
			if e.roll(0.40) {
				actions = append(actions, newAction(ActionGossipWith, e.firstTargetExcept(ActionGossipWith, targetID), priority-3, 1, 0.22, "relationship:resentment", "gossip", "social_negative"))
			}
			if e.roll(0.34) {
				actions = append(actions, newAction(ActionPlanRevenge, targetID, priority, 4, 0.30, "relationship:resentment", "revenge", "private"))
			}
			if relationship.Resentment > 92 && e.roll(0.12) {
				actions = append(actions, newAction(ActionThreaten, targetID, priority+5, 1, 0.32, "relationship:resentment", "threat", "escalation"))
			}
		case e.isRecentPartner(relationship):
			if !e.canRomanceWith(targetID) {
				continue
			}
			priority := 60 + int(maxFloat32(relationship.Affection, relationship.Attraction)/12)
			if e.roll(0.45) {
				actions = append(actions, newAction(ActionTalkTo, targetID, priority, 1, 0.08, "relationship:partner", "social", "intimacy"))
			}
			if e.canHaveSexWith(targetID) && e.roll(0.30+(e.poble.Needs.Sex/250.0)) {
				actions = append(actions, newAction(ActionHaveSex, targetID, priority+4, 2, 0.30, "relationship:partner", "sex", "private"))
			}
		case hasTag(relationship.Tags, "alliance_opportunity"):
			actions = append(actions, newAction(ActionFormAlliance, targetID, 66, 2, 0.18, "relationship", "alliance", "political"))
		}
	}

	sortActions(actions)
	return actions
}

// CheckArchetypeGoal adds recurrent archetype-specific motives.
func (e *DecisionEngine) CheckArchetypeGoal() []Action {
	if e == nil || e.poble == nil {
		return nil
	}

	switch e.poble.Archetype {
	case entities.ArchetypeRuler:
		if e.poble.Needs.Power > 55 || e.poble.Personality.Ambition > 70 {
			if target := e.firstTarget(ActionFormAlliance); target != "" && e.roll(0.35) {
				return []Action{newAction(ActionFormAlliance, target, 70, 2, 0.18, "archetype:ruler", "political", "control")}
			}
			return []Action{newAction(ActionGovern, "", 68, 3, 0.14, "archetype:ruler", "govern", "control")}
		}
	case entities.ArchetypeLover:
		if !e.canRomance() {
			return nil
		}
		if target := e.firstTarget(ActionFlirtWith); target != "" {
			return []Action{newAction(ActionFlirtWith, target, 69, 1, 0.24, "archetype:lover", "flirt", "romance")}
		}
		return []Action{newAction(ActionWriteDiary, "", 57, 1, 0.04, "archetype:lover", "diary", "longing")}
	case entities.ArchetypeSchemer:
		if target := e.firstTarget(ActionObserveSecretly); target != "" && e.roll(0.45) {
			return []Action{newAction(ActionObserveSecretly, target, 70, 2, 0.38, "archetype:schemer", "information", "secret")}
		}
		if target := e.firstTarget(ActionFormAlliance); target != "" {
			return []Action{newAction(ActionFormAlliance, target, 66, 2, 0.20, "archetype:schemer", "alliance", "secret")}
		}
	case entities.ArchetypeProphet:
		if target := e.firstTarget(ActionTalkTo); target != "" && e.roll(0.42) {
			return []Action{newAction(ActionTalkTo, target, 64, 2, 0.18, "archetype:prophet", "religious", "conversion")}
		}
		return []Action{newAction(ActionPray, "", 66, 2, 0.06, "archetype:prophet", "religious", "ritual")}
	case entities.ArchetypeWarrior:
		if e.poble.Needs.Safety > 70 {
			return []Action{newAction(ActionFight, e.firstTarget(ActionFight), 67, 1, 0.40, "archetype:warrior", "safety", "violence")}
		}
	case entities.ArchetypeSage:
		if e.poble.Needs.Purpose > 55 || e.poble.Personality.Openness > 70 {
			return []Action{newAction(ActionResearch, "", 64, 4, 0.12, "archetype:sage", "research", "purpose")}
		}
	case entities.ArchetypeRebel:
		return []Action{newAction(ActionExplore, "", 58, 3, 0.22, "archetype:rebel", "freedom", "world")}
	case entities.ArchetypeCaretaker:
		if target := e.firstTarget(ActionTalkTo); target != "" {
			return []Action{newAction(ActionTalkTo, target, 60, 1, 0.06, "archetype:caretaker", "care", "social")}
		}
	case entities.ArchetypeGhost:
		return []Action{newAction(ActionIsolate, "", 58, 4, 0.08, "archetype:ghost", "solitary", "low_arousal")}
	case entities.ArchetypeVillain:
		if target := e.firstTarget(ActionThreaten); target != "" && e.roll(0.32) {
			return []Action{newAction(ActionThreaten, target, 64, 1, 0.28, "archetype:villain", "threat", "control")}
		}
	case entities.ArchetypeDrifter:
		return []Action{newAction(ActionExplore, "", 56, 4, 0.24, "archetype:drifter", "world", "wandering")}
	case entities.ArchetypeJester:
		return []Action{newAction(ActionParty, "", 55, 2, 0.18, "archetype:jester", "humor", "social")}
	case entities.ArchetypeInnocent:
		if target := e.firstTarget(ActionTalkTo); target != "" {
			return []Action{newAction(ActionTalkTo, target, 55, 1, 0.05, "archetype:innocent", "trust", "social")}
		}
	case entities.ArchetypeMirror:
		if target := e.firstTarget(ActionObserveSecretly); target != "" {
			return []Action{newAction(ActionObserveSecretly, target, 56, 2, 0.32, "archetype:mirror", "reflection", "social")}
		}
	case entities.ArchetypeAddict:
		return []Action{newAction(ActionParty, "", 54, 3, 0.25, "archetype:addict", "escape", "impulse")}
	}

	return nil
}

// CheckWorldState reacts to recent world events when the world exposes them.
func (e *DecisionEngine) CheckWorldState() []Action {
	if e == nil || e.poble == nil || e.world == nil {
		return nil
	}

	provider, ok := e.world.(activeEventsProvider)
	if !ok {
		return nil
	}

	actions := make([]Action, 0, 4)
	for _, event := range provider.GetActiveEvents() {
		if !e.eventMatters(event) {
			continue
		}

		priority := 55 + int(clampPercent(event.Severity)/5)
		target := e.eventOtherPoble(event)
		switch event.Type {
		case GameEventDeath:
			if e.relationshipSentiment(target) >= 20 {
				actions = append(actions, newAction(ActionIsolate, "", priority+12, 5, 0.08, "world:death", "grief", "negative_valence"))
			} else {
				actions = append(actions, newAction(ActionGossipWith, e.firstTargetExcept(ActionGossipWith, target), priority, 1, 0.20, "world:death", "gossip", "social"))
			}
		case GameEventThreat:
			if e.poble.Archetype == entities.ArchetypeRuler {
				actions = append(actions, newAction(ActionGovern, "", priority+8, 2, 0.20, "world:threat", "control", "political"))
			} else if e.poble.Archetype == entities.ArchetypeWarrior {
				actions = append(actions, newAction(ActionFight, target, priority+7, 1, 0.42, "world:threat", "violence", "safety"))
			} else {
				actions = append(actions, newAction(ActionIsolate, "", priority+5, 2, 0.10, "world:threat", "safety", "fear"))
			}
		case GameEventConflict, GameEventBetrayal:
			if target == "" {
				target = e.firstTarget(ActionArgueWith)
			}
			if event.Type == GameEventBetrayal || e.poble.Archetype == entities.ArchetypeSchemer {
				actions = append(actions, newAction(ActionPlanRevenge, target, priority+9, 4, 0.34, "world:betrayal", "revenge", "private"))
			} else {
				actions = append(actions, newAction(ActionArgueWith, target, priority+4, 1, 0.24, "world:conflict", "argument", "social_negative"))
			}
		case GameEventSocialPositive:
			actions = append(actions, newAction(ActionTalkTo, target, priority, 1, 0.08, "world:social_positive", "social", "positive_valence"))
		case GameEventSocialNegative:
			actions = append(actions, newAction(ActionArgueWith, target, priority+3, 1, 0.24, "world:social_negative", "social_negative", "high_arousal"))
		case GameEventGoalComplete:
			actions = append(actions, newAction(ActionParty, "", priority, 3, 0.14, "world:goal_complete", "celebration", "positive_valence"))
		case GameEventIntimacy:
			if e.canHaveSexWith(target) && e.roll(0.35) {
				actions = append(actions, newAction(ActionHaveSex, target, priority+3, 2, 0.32, "world:intimacy", "sex", "private"))
			} else {
				actions = append(actions, newAction(ActionTalkTo, target, priority, 1, 0.08, "world:intimacy", "social", "intimacy"))
			}
		default:
			if event.Valence < -0.2 {
				actions = append(actions, newAction(ActionRest, "", priority, 2, 0.08, "world:stress", "rest", "negative_valence"))
			} else if event.Valence > 0.2 {
				actions = append(actions, newAction(ActionTalkTo, e.firstTarget(ActionTalkTo), priority, 1, 0.08, "world:positive", "social", "positive_valence"))
			}
		}
	}

	sortActions(actions)
	return actions
}

// DefaultBehavior returns habitual archetype-shaped behavior when nothing burns.
func (e *DecisionEngine) DefaultBehavior() []Action {
	if e == nil || e.poble == nil {
		return nil
	}

	if e.poble.Needs.Sleep > 78 {
		return []Action{newAction(ActionSleep, "", 64, 8, 0.03, "need:sleep", "rest")}
	}
	if e.poble.Needs.Belonging > 74 {
		return []Action{newAction(ActionTalkTo, e.firstTarget(ActionTalkTo), 60, 1, 0.08, "need:belonging", "social")}
	}
	if e.poble.Needs.Sex > 82 && e.canRomance() {
		if target := e.firstTarget(ActionHaveSex); target != "" && e.canHaveSexWith(target) {
			return []Action{newAction(ActionHaveSex, target, 62, 2, 0.35, "need:sex", "sex", "private")}
		}
	}

	switch e.poble.Archetype {
	case entities.ArchetypeRuler:
		return []Action{newAction(ActionGovern, "", 52, 3, 0.12, "default", "archetype:ruler", "control")}
	case entities.ArchetypeLover:
		if target := e.firstTarget(ActionTalkTo); target != "" && e.canRomance() {
			return []Action{newAction(ActionTalkTo, target, 52, 1, 0.08, "default", "archetype:lover", "social")}
		}
		return []Action{newAction(ActionWriteDiary, "", 48, 1, 0.04, "default", "archetype:lover", "diary")}
	case entities.ArchetypeSchemer:
		return []Action{newAction(ActionWriteDiary, "", 50, 1, 0.04, "default", "archetype:schemer", "planning")}
	case entities.ArchetypeProphet:
		return []Action{newAction(ActionPray, "", 52, 2, 0.06, "default", "archetype:prophet", "religious")}
	case entities.ArchetypeSage:
		return []Action{newAction(ActionResearch, "", 51, 4, 0.12, "default", "archetype:sage", "research")}
	case entities.ArchetypeRebel, entities.ArchetypeDrifter:
		return []Action{newAction(ActionExplore, "", 50, 3, 0.22, "default", "world", "wandering")}
	case entities.ArchetypeCaretaker:
		return []Action{newAction(ActionBuild, "", 49, 4, 0.18, "default", "archetype:caretaker", "care")}
	case entities.ArchetypeWarrior:
		return []Action{newAction(ActionWork, "", 49, 4, 0.12, "default", "archetype:warrior", "discipline")}
	case entities.ArchetypeGhost:
		return []Action{newAction(ActionRest, "", 49, 2, 0.05, "default", "archetype:ghost", "quiet")}
	case entities.ArchetypeJester, entities.ArchetypeAddict:
		return []Action{newAction(ActionParty, "", 48, 2, 0.18, "default", "archetype:jester", "social")}
	case entities.ArchetypeVillain:
		if target := e.firstTarget(ActionObserveSecretly); target != "" {
			return []Action{newAction(ActionObserveSecretly, target, 49, 2, 0.32, "default", "archetype:villain", "information")}
		}
	case entities.ArchetypeInnocent:
		if target := e.firstTarget(ActionTalkTo); target != "" {
			return []Action{newAction(ActionTalkTo, target, 48, 1, 0.05, "default", "archetype:innocent", "social")}
		}
	case entities.ArchetypeMirror:
		if target := e.firstTarget(ActionTalkTo); target != "" {
			return []Action{newAction(ActionTalkTo, target, 48, 1, 0.08, "default", "archetype:mirror", "social")}
		}
	}

	return []Action{newAction(ActionWork, "", 45, 4, 0.12, "default", "work")}
}

// AddRandomness injects a 15% chaos candidate when survival is not urgent.
func (e *DecisionEngine) AddRandomness() []Action {
	if e == nil || e.poble == nil || !e.roll(0.15) {
		return nil
	}

	actionType := allActionTypes[e.rng.Intn(len(allActionTypes))]
	for actionType == ActionMurder || actionType == ActionHaveSex || actionType == ActionFlirtWith || actionType == ActionPropose {
		actionType = allActionTypes[e.rng.Intn(len(allActionTypes))]
	}

	target := ""
	if actionNeedsTarget(actionType) {
		target = e.firstTarget(actionType)
		if target == "" {
			return nil
		}
	}

	return []Action{newAction(actionType, target, 59, defaultDuration(actionType), defaultFailureChance(actionType), "chaos", "random")}
}

// EvaluateTarget ranks possible targets for the given action.
func (e *DecisionEngine) EvaluateTarget(action ActionType) []string {
	if e == nil || e.poble == nil || e.world == nil {
		return nil
	}

	candidates := make([]targetScore, 0)
	for _, target := range e.world.GetAllPobles() {
		if target == nil || !target.IsAlive || target.ID == "" || target.ID == e.poble.ID {
			continue
		}
		if !e.targetAllowed(action, target) {
			continue
		}

		score := e.scoreTarget(action, target)
		if score <= -90 {
			continue
		}
		candidates = append(candidates, targetScore{id: target.ID, score: score})
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].score == candidates[j].score {
			return candidates[i].id < candidates[j].id
		}
		return candidates[i].score > candidates[j].score
	})

	ids := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		ids = append(ids, candidate.id)
	}
	return ids
}

// ShouldInterruptCurrentAction checks if newAction deserves immediate focus.
func (e *DecisionEngine) ShouldInterruptCurrentAction(newAction Action) bool {
	if e == nil || e.currentAction == nil {
		return true
	}
	if newAction.Type == "" {
		return false
	}

	current := *e.currentAction
	if newAction.Priority >= 98 {
		return true
	}
	if current.Type == ActionSleep {
		return newAction.Priority >= 95 || hasTag(newAction.Tags, "survival")
	}
	if current.Type == ActionHaveSex || current.Type == ActionFight {
		return newAction.Priority >= current.Priority+18
	}

	requiredGap := 8
	if isSocialAction(newAction.Type) && !isSocialAction(current.Type) {
		requiredGap = 2
	}
	if !isSocialAction(newAction.Type) && isSocialAction(current.Type) {
		requiredGap = 12
	}
	if isSolitaryAction(current.Type) && isSocialAction(newAction.Type) {
		requiredGap = 0
	}

	return newAction.Priority >= current.Priority+requiredGap
}

// GetCurrentIntent returns a compact UI-safe intent key, not narrative prose.
func (e *DecisionEngine) GetCurrentIntent() string {
	if e == nil || e.currentAction == nil {
		return "intent:IDLE"
	}
	if e.currentAction.TargetID == "" {
		return "intent:" + string(e.currentAction.Type)
	}
	return "intent:" + string(e.currentAction.Type) + " target:" + e.currentAction.TargetID
}

type targetScore struct {
	id    string
	score float32
}

func newAction(actionType ActionType, targetID string, priority, duration int, failureChance float32, tags ...string) Action {
	if duration <= 0 {
		duration = defaultDuration(actionType)
	}
	return Action{
		Type:          actionType,
		TargetID:      targetID,
		Priority:      priority,
		Duration:      duration,
		FailureChance: clampPercent(failureChance*100) / 100,
		Tags:          append([]string(nil), tags...),
	}
}

func (e *DecisionEngine) normalizeActions(actions []Action, deltaHours int) []Action {
	if len(actions) == 0 {
		return nil
	}

	normalized := actions[:0]
	for _, action := range actions {
		if action.Type == "" {
			continue
		}
		if deltaHours > 1 && action.Type != ActionSleep {
			action.Priority += minInt(deltaHours, 4) - 1
		}
		if action.Duration <= 0 {
			action.Duration = defaultDuration(action.Type)
		}
		if action.FailureChance < 0 {
			action.FailureChance = 0
		}
		if action.FailureChance > 1 {
			action.FailureChance = 1
		}
		if actionNeedsTarget(action.Type) && action.TargetID == "" {
			continue
		}
		normalized = append(normalized, action)
	}

	sortActions(normalized)
	return normalized
}

func (e *DecisionEngine) setCurrentAction(action Action) {
	copyAction := action
	e.currentAction = &copyAction
}

func sortActions(actions []Action) {
	sort.SliceStable(actions, func(i, j int) bool {
		if actions[i].Priority == actions[j].Priority {
			if actions[i].Duration == actions[j].Duration {
				return actions[i].Type < actions[j].Type
			}
			return actions[i].Duration < actions[j].Duration
		}
		return actions[i].Priority > actions[j].Priority
	})
}

func (e *DecisionEngine) dominantEmotion() (entities.EmotionType, float32, bool) {
	bestEmotion := entities.EmotionType("")
	bestIntensity := float32(0)
	for _, emotion := range e.poble.EmotionalState.ActiveEmotions {
		intensity := e.emotionIntensity(emotion)
		if intensity > bestIntensity {
			bestEmotion = emotion
			bestIntensity = intensity
		}
	}

	if bestEmotion == "" {
		return "", 0, false
	}
	return bestEmotion, bestIntensity, true
}

func (e *DecisionEngine) emotionIntensity(emotion entities.EmotionType) float32 {
	state := e.poble.EmotionalState
	intensity := (decisionAbs(state.Valence) * 38) + (decisionAbs(state.Arousal) * 42) + (decisionAbs(state.Dominance) * 20)

	switch emotion {
	case entities.EmotionAnger, entities.EmotionFear, entities.EmotionLust:
		intensity += maxFloat32(0, state.Arousal) * 18
	case entities.EmotionGrief, entities.EmotionLoneliness:
		intensity += maxFloat32(0, -state.Valence) * 18
	case entities.EmotionPride:
		intensity += maxFloat32(0, state.Dominance) * 14
	case entities.EmotionShame, entities.EmotionGuilt:
		intensity += maxFloat32(0, -state.Dominance) * 14
	}

	switch e.poble.CurrentMood {
	case entities.MoodAngry:
		if emotion == entities.EmotionAnger || emotion == entities.EmotionResentment || emotion == entities.EmotionContempt {
			intensity += 18
		}
	case entities.MoodDepressed, entities.MoodSad:
		if emotion == entities.EmotionGrief || emotion == entities.EmotionLoneliness || emotion == entities.EmotionShame {
			intensity += 16
		}
	case entities.MoodAnxious:
		if emotion == entities.EmotionFear || emotion == entities.EmotionAnxiety {
			intensity += 16
		}
	case entities.MoodEuphoric, entities.MoodHappy:
		if emotion == entities.EmotionJoy || emotion == entities.EmotionPride || emotion == entities.EmotionLove {
			intensity += 12
		}
	case entities.MoodObsessive:
		if emotion == entities.EmotionLove || emotion == entities.EmotionLust || emotion == entities.EmotionJealousy {
			intensity += 18
		}
	}

	return clampPercent(intensity)
}

func (e *DecisionEngine) sortedRelationshipIDs() []string {
	ids := make([]string, 0, len(e.poble.Relationships))
	for id := range e.poble.Relationships {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func (e *DecisionEngine) isCrush(relationship entities.Relationship) bool {
	return relationship.Type == entities.RelationshipCrush ||
		relationship.Type == entities.RelationshipObsession ||
		hasTag(relationship.Tags, "crush") ||
		hasTag(relationship.Tags, "obsession") ||
		(relationship.Attraction >= 76 && relationship.Resentment < 55)
}

func (e *DecisionEngine) isHighResentment(relationship entities.Relationship) bool {
	return relationship.Type == entities.RelationshipEnemy ||
		relationship.Type == entities.RelationshipRival ||
		relationship.Type == entities.RelationshipBetrayer ||
		hasTag(relationship.Tags, "resentment") ||
		relationship.Resentment >= 72
}

func (e *DecisionEngine) isRecentPartner(relationship entities.Relationship) bool {
	if relationship.Type != entities.RelationshipLover &&
		relationship.Type != entities.RelationshipSpouse &&
		relationship.Type != entities.RelationshipFriendsWithBenefits {
		return false
	}
	if hasTag(relationship.Tags, "recent_partner") || hasTag(relationship.Tags, "new_partner") {
		return true
	}

	provider, ok := e.world.(currentTimeProvider)
	if !ok {
		return true
	}
	return provider.GetCurrentTime().Diff(relationship.LastInteraction) <= 72
}

func (e *DecisionEngine) hasBurningSecret() bool {
	for _, secret := range e.poble.Secrets {
		if secret.IsRevealed {
			continue
		}
		trigger := strings.ToLower(secret.RevealTrigger)
		if strings.Contains(trigger, "burn") ||
			strings.Contains(trigger, "quema") ||
			strings.Contains(trigger, "confess") ||
			strings.Contains(trigger, "pressure") {
			return true
		}
		switch secret.Type {
		case entities.SecretPlannedBetrayal, entities.SecretCriminalAct, entities.SecretObsession, entities.SecretTrueOrientation:
			if e.hasAnyActiveEmotion(entities.EmotionGuilt, entities.EmotionShame, entities.EmotionAnxiety, entities.EmotionLove) {
				return true
			}
		}
	}
	return false
}

func (e *DecisionEngine) hasAnyActiveEmotion(emotions ...entities.EmotionType) bool {
	for _, current := range e.poble.EmotionalState.ActiveEmotions {
		for _, target := range emotions {
			if current == target {
				return true
			}
		}
	}
	return false
}

func (e *DecisionEngine) canAttemptMurder(targetID string, intensity float32) bool {
	if targetID == "" || e.poble.Personality.Cruelty < 86 || e.poble.Mental.Stability > 18 || intensity < 96 {
		return false
	}
	relationship, ok := e.poble.Relationships[targetID]
	if !ok || relationship.Resentment < 90 {
		return false
	}
	return e.roll(0.035)
}

func (e *DecisionEngine) canRomance() bool {
	return e.poble.Age >= 18
}

func (e *DecisionEngine) canRomanceWith(targetID string) bool {
	if !e.canRomance() {
		return false
	}
	target := e.pobleByID(targetID)
	return target != nil && target.Age >= 18
}

func (e *DecisionEngine) canHaveSexWith(targetID string) bool {
	if !e.canRomanceWith(targetID) {
		return false
	}
	relationship := e.poble.Relationships[targetID]
	if relationship.Type == entities.RelationshipFamily ||
		relationship.Type == entities.RelationshipParent ||
		relationship.Type == entities.RelationshipChild ||
		relationship.Type == entities.RelationshipSibling {
		return false
	}

	return relationship.Type == entities.RelationshipLover ||
		relationship.Type == entities.RelationshipSpouse ||
		relationship.Type == entities.RelationshipFriendsWithBenefits ||
		relationship.Attraction >= 72 ||
		(e.poble.Needs.Sex >= 80 && relationship.Attraction >= 58 && relationship.Resentment < 45)
}

func (e *DecisionEngine) firstTarget(action ActionType) string {
	targets := e.EvaluateTarget(action)
	if len(targets) == 0 {
		return ""
	}
	return targets[0]
}

func (e *DecisionEngine) firstTargetExcept(action ActionType, excluded string) string {
	targets := e.EvaluateTarget(action)
	for _, target := range targets {
		if target != excluded {
			return target
		}
	}
	return ""
}

func (e *DecisionEngine) targetAllowed(action ActionType, target *Poble) bool {
	switch action {
	case ActionFlirtWith, ActionHaveSex, ActionPropose:
		return e.poble.Age >= 18 && target.Age >= 18
	case ActionMurder:
		if target.Age < 18 {
			return false
		}
	case ActionConfessTo:
		return true
	}
	return true
}

func (e *DecisionEngine) scoreTarget(action ActionType, target *Poble) float32 {
	relationship, hasRelationship := e.poble.Relationships[target.ID]
	score := float32(0)
	if hasRelationship {
		score += relationship.Familiarity * 0.14
		score += relationship.Respect * 0.08
		score += relationship.Dependency * 0.04
	} else {
		score -= 8
	}

	switch action {
	case ActionTalkTo, ActionSendLetter:
		score += relationship.Affection*0.35 + relationship.Trust*0.25 - relationship.Resentment*0.22
		if relationship.Type == entities.RelationshipFriend || relationship.Type == entities.RelationshipBestFriend || relationship.Type == entities.RelationshipLover || relationship.Type == entities.RelationshipSpouse {
			score += 25
		}
	case ActionFlirtWith, ActionHaveSex, ActionPropose:
		score += relationship.Attraction*0.52 + relationship.Affection*0.28 + relationship.Trust*0.12 - relationship.Resentment*0.40 - relationship.Fear*0.20
		if relationship.Type == entities.RelationshipCrush || relationship.Type == entities.RelationshipLover || relationship.Type == entities.RelationshipSpouse || relationship.Type == entities.RelationshipFriendsWithBenefits {
			score += 28
		}
		if action == ActionHaveSex {
			score += e.poble.Needs.Sex * 0.20
		}
	case ActionArgueWith, ActionThreaten, ActionFight, ActionMurder:
		score += relationship.Resentment*0.48 + relationship.Fear*0.12 - relationship.Affection*0.22 - relationship.Trust*0.18
		if relationship.Type == entities.RelationshipEnemy || relationship.Type == entities.RelationshipRival || relationship.Type == entities.RelationshipBetrayer {
			score += 30
		}
		if action == ActionMurder {
			score += e.poble.Personality.Cruelty*0.25 + float32(100-e.poble.Mental.Stability)*0.35
		}
	case ActionGossipWith:
		score += relationship.Familiarity*0.25 + relationship.Trust*0.30 + relationship.Resentment*0.12
		if relationship.Type == entities.RelationshipFriend || relationship.Type == entities.RelationshipBestFriend || relationship.Type == entities.RelationshipAlly {
			score += 24
		}
	case ActionObserveSecretly:
		score += relationship.Attraction*0.32 + relationship.Resentment*0.22 + relationship.Familiarity*0.12
		if relationship.Type == entities.RelationshipCrush || relationship.Type == entities.RelationshipObsession || relationship.IsSecret {
			score += 30
		}
	case ActionConfessTo:
		score += relationship.Trust*0.45 + relationship.Affection*0.28 - relationship.Resentment*0.35
		if relationship.Type == entities.RelationshipLover || relationship.Type == entities.RelationshipBestFriend || relationship.Type == entities.RelationshipSpouse {
			score += 25
		}
	case ActionFormAlliance, ActionTrade:
		score += relationship.Trust*0.30 + relationship.Respect*0.30 + relationship.Familiarity*0.15 - relationship.Resentment*0.25
		if relationship.Type == entities.RelationshipAlly || relationship.Type == entities.RelationshipFriend || relationship.Type == entities.RelationshipBoss || relationship.Type == entities.RelationshipEmployee {
			score += 22
		}
	case ActionPlanRevenge, ActionBetray, ActionBreakUp:
		score += relationship.Resentment*0.50 - relationship.Trust*0.20
		if relationship.Type == entities.RelationshipBetrayer || relationship.Type == entities.RelationshipEnemy {
			score += 24
		}
	default:
		score += relationship.Familiarity * 0.20
	}

	score += e.targetEmotionalScore(action, target)
	if provider, ok := e.world.(proximityProvider); ok {
		score += clampRange(provider.GetProximityScore(e.poble.ID, target.ID), -20, 20)
	}

	return score
}

func (e *DecisionEngine) targetEmotionalScore(action ActionType, target *Poble) float32 {
	state := target.EmotionalState
	switch action {
	case ActionTalkTo, ActionConfessTo:
		if state.Valence < -0.25 {
			return 12 + (decisionAbs(state.Valence) * 10)
		}
	case ActionArgueWith, ActionFight, ActionThreaten:
		if state.Arousal > 0.45 {
			return 10 + (state.Arousal * 12)
		}
	case ActionFlirtWith, ActionHaveSex:
		if state.Valence > 0.15 && state.Arousal > 0 {
			return 8 + (state.Valence * 8) + (state.Arousal * 8)
		}
	case ActionObserveSecretly:
		if state.Arousal < -0.2 || state.Valence < -0.3 {
			return 8
		}
	}
	return 0
}

func (e *DecisionEngine) eventMatters(event GameEvent) bool {
	if event.Severity >= 82 {
		return true
	}
	if event.PrimaryActor == e.poble.ID || event.TargetID == e.poble.ID {
		return true
	}
	for _, participant := range event.Participants {
		if participant == e.poble.ID {
			return true
		}
		if _, ok := e.poble.Relationships[participant]; ok {
			return true
		}
	}
	if _, ok := e.poble.Relationships[event.PrimaryActor]; ok {
		return true
	}
	if _, ok := e.poble.Relationships[event.TargetID]; ok {
		return true
	}
	return false
}

func (e *DecisionEngine) eventOtherPoble(event GameEvent) string {
	if event.PrimaryActor != "" && event.PrimaryActor != e.poble.ID {
		return event.PrimaryActor
	}
	if event.TargetID != "" && event.TargetID != e.poble.ID {
		return event.TargetID
	}
	for _, participant := range event.Participants {
		if participant != e.poble.ID {
			return participant
		}
	}
	return ""
}

func (e *DecisionEngine) relationshipSentiment(targetID string) float32 {
	relationship, ok := e.poble.Relationships[targetID]
	if !ok {
		return 0
	}
	return clampRange(
		(relationship.Affection*0.35)+(relationship.Trust*0.25)+(relationship.Respect*0.18)-
			(relationship.Resentment*0.38)-(relationship.Fear*0.14),
		-100,
		100,
	)
}

func (e *DecisionEngine) pobleByID(id string) *Poble {
	if e.world == nil || id == "" {
		return nil
	}
	for _, candidate := range e.world.GetAllPobles() {
		if candidate != nil && candidate.ID == id {
			return candidate
		}
	}
	return nil
}

func (e *DecisionEngine) roll(chance float32) bool {
	if e == nil || e.rng == nil {
		return false
	}
	if chance <= 0 {
		return false
	}
	if chance >= 1 {
		return true
	}
	return e.rng.Float32() < chance
}

func actionNeedsTarget(action ActionType) bool {
	switch action {
	case ActionTalkTo, ActionArgueWith, ActionFlirtWith, ActionConfessTo, ActionGossipWith,
		ActionThreaten, ActionObserveSecretly, ActionSendLetter, ActionPlanRevenge,
		ActionFormAlliance, ActionBetray, ActionHaveSex, ActionPropose, ActionBreakUp,
		ActionFight, ActionMurder, ActionTrade:
		return true
	default:
		return false
	}
}

func isSocialAction(action ActionType) bool {
	switch action {
	case ActionTalkTo, ActionArgueWith, ActionFlirtWith, ActionConfessTo, ActionGossipWith,
		ActionThreaten, ActionSendLetter, ActionFormAlliance, ActionBetray, ActionHaveSex,
		ActionPropose, ActionBreakUp, ActionFight, ActionMurder, ActionTrade, ActionParty:
		return true
	default:
		return false
	}
}

func isSolitaryAction(action ActionType) bool {
	switch action {
	case ActionSleep, ActionEat, ActionDrink, ActionWork, ActionExplore, ActionObserveSecretly,
		ActionWriteDiary, ActionPlanRevenge, ActionPray, ActionResearch, ActionHaveBreakdown,
		ActionIsolate, ActionRest:
		return true
	default:
		return false
	}
}

func defaultDuration(action ActionType) int {
	switch action {
	case ActionSleep:
		return 8
	case ActionEat, ActionDrink, ActionTalkTo, ActionArgueWith, ActionFlirtWith,
		ActionConfessTo, ActionGossipWith, ActionThreaten, ActionFight, ActionTrade:
		return 1
	case ActionHaveSex, ActionObserveSecretly, ActionSendLetter, ActionFormAlliance,
		ActionPray, ActionRest:
		return 2
	case ActionWork, ActionExplore, ActionPlanRevenge, ActionGovern, ActionBuild,
		ActionResearch, ActionHaveBreakdown, ActionParty:
		return 4
	default:
		return 2
	}
}

func defaultFailureChance(action ActionType) float32 {
	switch action {
	case ActionEat, ActionDrink, ActionSleep, ActionRest:
		return 0.03
	case ActionTalkTo, ActionWork, ActionPray:
		return 0.08
	case ActionFlirtWith, ActionArgueWith, ActionGossipWith, ActionBuild, ActionResearch:
		return 0.20
	case ActionThreaten, ActionObserveSecretly, ActionPlanRevenge, ActionHaveSex, ActionBetray:
		return 0.35
	case ActionFight:
		return 0.45
	case ActionMurder:
		return 0.78
	default:
		return 0.15
	}
}

func hasTag(tags []string, target string) bool {
	for _, tag := range tags {
		if strings.EqualFold(tag, target) {
			return true
		}
	}
	return false
}

func decisionAbs(value float32) float32 {
	return float32(math.Abs(float64(value)))
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
