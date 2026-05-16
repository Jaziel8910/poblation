package events

import (
	"fmt"
	"math/rand"
	"sort"
	"sync"

	"github.com/user/poblation/internal/entities"
)

// EventTiming controls when a pushed event fires.
type EventTiming struct {
	// Mode is IMMEDIATE, IN_HOURS, or TRIGGERED_BY.
	Mode TimingMode
	// DelayHours is used when Mode is IN_HOURS.
	DelayHours int
	// TriggerCondition is evaluated each tick when Mode is TRIGGERED_BY.
	TriggerCondition func(entities.GameTime, World) bool
}

// TimingMode identifies when an event enters active processing.
type TimingMode string

const (
	TimingImmediate TimingMode = "IMMEDIATE"
	TimingInHours   TimingMode = "IN_HOURS"
	TimingTriggered TimingMode = "TRIGGERED_BY"
)

// Immediate returns an EventTiming that fires now.
func Immediate() EventTiming {
	return EventTiming{Mode: TimingImmediate}
}

// InHours returns an EventTiming that fires after n in-game hours.
func InHours(n int) EventTiming {
	if n < 0 {
		n = 0
	}
	return EventTiming{Mode: TimingInHours, DelayHours: n}
}

// TriggeredBy returns an EventTiming that fires when the condition is met.
func TriggeredBy(condition func(entities.GameTime, World) bool) EventTiming {
	return EventTiming{Mode: TimingTriggered, TriggerCondition: condition}
}

// ScheduledEvent wraps an event with its target fire time.
type ScheduledEvent struct {
	Event  GameEvent
	FireAt entities.GameTime
}

// DormantEvent wraps an event with its trigger condition.
type DormantEvent struct {
	Event     GameEvent
	Condition func(entities.GameTime, World) bool
}

// EventQueue manages four priority lanes per GDD section 9.
type EventQueue struct {
	mu sync.Mutex

	Immediate []GameEvent
	Scheduled []ScheduledEvent
	Dormant   []DormantEvent
	Gossip    []GameEvent

	rng       *rand.Rand
	processed map[string]struct{}
}

// NewEventQueue creates an empty event queue.
func NewEventQueue(rng *rand.Rand) *EventQueue {
	if rng == nil {
		rng = rand.New(rand.NewSource(1))
	}
	return &EventQueue{
		Immediate: make([]GameEvent, 0, 16),
		Scheduled: make([]ScheduledEvent, 0, 16),
		Dormant:   make([]DormantEvent, 0, 8),
		Gossip:    make([]GameEvent, 0, 8),
		rng:       rng,
		processed: map[string]struct{}{},
	}
}

// Push adds an event to the appropriate lane based on timing.
func (q *EventQueue) Push(event GameEvent, timing EventTiming) {
	if q == nil || event.ID == "" {
		return
	}

	q.mu.Lock()
	defer q.mu.Unlock()

	if _, seen := q.processed[event.ID]; seen {
		return
	}

	switch timing.Mode {
	case TimingImmediate:
		q.Immediate = append(q.Immediate, event)
	case TimingInHours:
		q.Scheduled = append(q.Scheduled, ScheduledEvent{
			Event:  event,
			FireAt: event.Timestamp.Add(timing.DelayHours),
		})
	case TimingTriggered:
		if timing.TriggerCondition != nil {
			q.Dormant = append(q.Dormant, DormantEvent{
				Event:     event,
				Condition: timing.TriggerCondition,
			})
		}
	}
}

// PushGossip adds a gossip-lane event (rumour propagation, social chain events).
func (q *EventQueue) PushGossip(event GameEvent) {
	if q == nil || event.ID == "" {
		return
	}

	q.mu.Lock()
	defer q.mu.Unlock()

	if _, seen := q.processed[event.ID]; seen {
		return
	}
	q.Gossip = append(q.Gossip, event)
}

// Process evaluates all lanes and returns events ready for this tick.
// Order: Immediate → Scheduled (if time reached) → Dormant (if condition met) → Gossip.
func (q *EventQueue) Process(currentTime entities.GameTime, world World) []GameEvent {
	if q == nil {
		return nil
	}

	q.mu.Lock()
	defer q.mu.Unlock()

	fired := make([]GameEvent, 0, len(q.Immediate)+4)

	// 1. Drain immediate queue completely.
	fired = append(fired, q.Immediate...)
	q.Immediate = q.Immediate[:0]

	// 2. Check scheduled events.
	remaining := q.Scheduled[:0]
	for _, entry := range q.Scheduled {
		if isTimeReached(currentTime, entry.FireAt) {
			fired = append(fired, entry.Event)
		} else {
			remaining = append(remaining, entry)
		}
	}
	q.Scheduled = remaining

	// 3. Evaluate dormant triggers.
	stillDormant := q.Dormant[:0]
	for _, entry := range q.Dormant {
		if entry.Condition != nil && entry.Condition(currentTime, world) {
			fired = append(fired, entry.Event)
		} else {
			stillDormant = append(stillDormant, entry)
		}
	}
	q.Dormant = stillDormant

	// 4. Process gossip (one per tick to simulate spread speed).
	if len(q.Gossip) > 0 {
		fired = append(fired, q.Gossip[0])
		q.Gossip = q.Gossip[1:]
	}

	// Mark all as processed to prevent duplicates.
	for i := range fired {
		q.processed[fired[i].ID] = struct{}{}
	}

	// Sort by timestamp for deterministic ordering.
	sort.SliceStable(fired, func(i, j int) bool {
		return fired[i].Timestamp.ToMinutes() < fired[j].Timestamp.ToMinutes()
	})

	return fired
}

// Len returns total pending events across all lanes.
func (q *EventQueue) Len() int {
	if q == nil {
		return 0
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.Immediate) + len(q.Scheduled) + len(q.Dormant) + len(q.Gossip)
}

// Stats returns counts per lane for debugging.
func (q *EventQueue) Stats() (immediate, scheduled, dormant, gossip int) {
	if q == nil {
		return
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.Immediate), len(q.Scheduled), len(q.Dormant), len(q.Gossip)
}

// ProcessTick is the main per-tick entry point. It:
// 1. Generates natural/social/world events.
// 2. Pushes them as immediate.
// 3. Processes the full queue.
// 4. Applies consequences (deferred consequences become scheduled events).
// 5. Returns all events processed this tick.
func ProcessTick(q *EventQueue, currentTime entities.GameTime, world World, rng *rand.Rand, renderer TemplateRenderer) []GameEvent {
	if q == nil || world == nil {
		return nil
	}

	// Generate events from world state.
	generated := make([]GameEvent, 0, 8)
	generated = append(generated, CheckNaturalEvents(world, rng)...)
	generated = append(generated, CheckSocialEvents(world, rng)...)
	generated = append(generated, CheckWorldEvents(world, rng)...)

	// Push all generated events as immediate.
	for i := range generated {
		if generated[i].Timestamp.Day == 0 && generated[i].Timestamp.Hour == 0 {
			generated[i].Timestamp = currentTime
		}
		q.Push(generated[i], Immediate())
	}

	// Process queue.
	fired := q.Process(currentTime, world)

	// Apply consequences and collect deferred events.
	for i := range fired {
		if fired[i].Description == "" {
			fired[i].Description = GenerateEventDescription(fired[i], TemplateContext{
				Renderer:   renderer,
				WorldState: world.GetWorldState(),
			})
		}
		deferred := ApplyConsequences(fired[i], world)
		for _, d := range deferred {
			q.Push(d, InHours(0))
		}
	}

	return fired
}

// ClearProcessed resets the duplicate-prevention set.
// Call between save/load cycles.
func (q *EventQueue) ClearProcessed() {
	if q == nil {
		return
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	q.processed = map[string]struct{}{}
}

// PendingImmediateCount returns how many immediate events await processing.
func (q *EventQueue) PendingImmediateCount() int {
	if q == nil {
		return 0
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.Immediate)
}

// DrainGossip returns and removes all gossip events at once.
func (q *EventQueue) DrainGossip() []GameEvent {
	if q == nil {
		return nil
	}
	q.mu.Lock()
	defer q.mu.Unlock()

	drained := append([]GameEvent{}, q.Gossip...)
	q.Gossip = q.Gossip[:0]
	return drained
}

func isTimeReached(current, target entities.GameTime) bool {
	return current.ToMinutes() >= target.ToMinutes()
}

// DebugDump returns a summary string for debugging.
func (q *EventQueue) DebugDump() string {
	if q == nil {
		return "EventQueue: nil"
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	return fmt.Sprintf("EventQueue: immediate=%d scheduled=%d dormant=%d gossip=%d processed=%d",
		len(q.Immediate), len(q.Scheduled), len(q.Dormant), len(q.Gossip), len(q.processed))
}
