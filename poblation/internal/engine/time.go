package engine

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/user/poblation/internal/entities"
)

// GameTime is the shared simulation time contract.
type GameTime = entities.GameTime

// GameTick describes one time advancement event.
type GameTick struct {
	// CurrentTime stores the current game time after the tick.
	CurrentTime GameTime `json:"current_time"`
	// DeltaHours stores how many game hours passed in this tick.
	DeltaHours int `json:"delta_hours"`
	// IsNewDay marks day boundary crossings.
	IsNewDay bool `json:"is_new_day"`
	// IsNewWeek marks every seventh game day.
	IsNewWeek bool `json:"is_new_week"`
	// IsNewMonth marks every thirtieth game day.
	IsNewMonth bool `json:"is_new_month"`
	// IsNewYear marks every three hundred sixty fifth game day.
	IsNewYear bool `json:"is_new_year"`
}

// TimeEngine advances game time and publishes clock events.
type TimeEngine struct {
	// ticker fires every 60 real seconds adjusted by Speed.
	ticker *time.Ticker
	// GameTime stores current game time.
	GameTime GameTime
	// Speed stores clock multiplier such as 0.5x, 1x, 2x, or 4x.
	Speed float64
	// IsPaused marks whether time advancement is paused.
	IsPaused bool
	// subscribers stores channels that receive time ticks.
	subscribers []chan GameTick

	// mu protects time state, subscribers, and ticker changes.
	mu sync.Mutex
	// ctx controls clean cancellation for the time loop.
	ctx context.Context
	// cancel stops the active time loop context.
	cancel context.CancelFunc
	// debug receives every emitted tick for diagnostics.
	debug chan GameTick
	// done closes when the active loop exits.
	done chan struct{}
	// running marks whether Start has an active goroutine.
	running bool
}

// NewTimeEngine creates a stopped time engine at day zero.
func NewTimeEngine() *TimeEngine {
	ctx, cancel := context.WithCancel(context.Background())

	return &TimeEngine{
		ticker:      nil,
		GameTime:    entities.NewGameTime(0, 0, 0),
		Speed:       1.0,
		IsPaused:    false,
		subscribers: []chan GameTick{},
		ctx:         ctx,
		cancel:      cancel,
		debug:       make(chan GameTick, 128),
		done:        nil,
		running:     false,
	}
}

// Start begins the clock loop in a goroutine.
func (e *TimeEngine) Start(ctxs ...context.Context) {
	e.mu.Lock()
	if e.running {
		e.mu.Unlock()
		return
	}
	if len(ctxs) > 0 && ctxs[0] != nil {
		e.ctx, e.cancel = context.WithCancel(ctxs[0])
	} else if e.ctx == nil || e.ctx.Err() != nil {
		e.ctx, e.cancel = context.WithCancel(context.Background())
	}
	if e.ticker == nil {
		e.ticker = time.NewTicker(e.intervalLocked())
	} else {
		e.ticker.Reset(e.intervalLocked())
	}
	e.done = make(chan struct{})
	e.running = true
	ticker := e.ticker
	ctx := e.ctx
	done := e.done
	e.mu.Unlock()

	go func() {
		defer func() {
			e.mu.Lock()
			e.running = false
			e.mu.Unlock()
			close(done)
		}()
		for {
			select {
			case <-ctx.Done():
				e.mu.Lock()
				if ticker != nil {
					ticker.Stop()
				}
				e.mu.Unlock()
				return
			case <-ticker.C:
				e.AdvanceBy(1)
			}
		}
	}()
}

// Stop cancels the time loop and stops the ticker.
func (e *TimeEngine) Stop() {
	e.mu.Lock()
	done := e.done
	running := e.running
	if e.cancel != nil {
		e.cancel()
	}
	if e.ticker != nil {
		e.ticker.Stop()
	}
	e.mu.Unlock()

	if running && done != nil {
		<-done
	}
}

// Pause stops advancement while keeping the engine alive.
func (e *TimeEngine) Pause() {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.IsPaused = true
}

// Resume allows advancement after Pause.
func (e *TimeEngine) Resume() {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.IsPaused = false
}

// SetSpeed changes ticker interval from the speed multiplier.
func (e *TimeEngine) SetSpeed(multiplier float64) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if multiplier <= 0 {
		e.Speed = 1.0
		e.IsPaused = true
		return
	}

	e.Speed = multiplier
	e.IsPaused = false
	if e.ticker != nil {
		e.ticker.Reset(e.intervalLocked())
	}
}

// Subscribe returns a channel that receives future GameTick events.
func (e *TimeEngine) Subscribe() chan GameTick {
	e.mu.Lock()
	defer e.mu.Unlock()

	ch := make(chan GameTick, 16)
	e.subscribers = append(e.subscribers, ch)
	return ch
}

// Unsubscribe removes and closes a subscription channel.
func (e *TimeEngine) Unsubscribe(ch chan GameTick) {
	e.mu.Lock()
	defer e.mu.Unlock()

	for i, subscriber := range e.subscribers {
		if subscriber == ch {
			e.subscribers = append(e.subscribers[:i], e.subscribers[i+1:]...)
			close(subscriber)
			return
		}
	}
}

// Debug returns the debug tick channel.
func (e *TimeEngine) Debug() <-chan GameTick {
	e.mu.Lock()
	defer e.mu.Unlock()

	return e.debug
}

// GetCurrentTime returns current game time.
func (e *TimeEngine) GetCurrentTime() GameTime {
	e.mu.Lock()
	defer e.mu.Unlock()

	return e.GameTime
}

// SetTime forces the simulation clock to an exact in-game time.
func (e *TimeEngine) SetTime(target GameTime) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.GameTime = target
}

// FormatTime returns localized display time.
func (e *TimeEngine) FormatTime() string {
	e.mu.Lock()
	defer e.mu.Unlock()

	return fmt.Sprintf("Día %d, %02d:%02d", e.GameTime.Day, e.GameTime.Hour, e.GameTime.Minute)
}

// AdvanceBy advances game time by hours for tests and debug commands.
func (e *TimeEngine) AdvanceBy(hours int) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.IsPaused || hours <= 0 {
		return
	}

	previous := e.GameTime
	e.GameTime = e.GameTime.Add(hours)
	tick := newGameTick(previous, e.GameTime, hours)
	e.publishLocked(tick)
}

func (e *TimeEngine) intervalLocked() time.Duration {
	if e.Speed <= 0 {
		return time.Minute
	}
	return time.Duration(float64(time.Minute) / e.Speed)
}

func (e *TimeEngine) publishLocked(tick GameTick) {
	for _, subscriber := range e.subscribers {
		select {
		case subscriber <- tick:
		default:
		}
	}

	select {
	case e.debug <- tick:
	default:
	}
}

func newGameTick(previous, current GameTime, deltaHours int) GameTick {
	isNewDay := current.Day != previous.Day

	return GameTick{
		CurrentTime: current,
		DeltaHours:  deltaHours,
		IsNewDay:    isNewDay,
		IsNewWeek:   crossesDayMultiple(previous.Day, current.Day, 7),
		IsNewMonth:  crossesDayMultiple(previous.Day, current.Day, 30),
		IsNewYear:   crossesDayMultiple(previous.Day, current.Day, 365),
	}
}

func crossesDayMultiple(previousDay, currentDay, size int) bool {
	if currentDay <= previousDay || size <= 0 {
		return false
	}

	for day := previousDay + 1; day <= currentDay; day++ {
		if day > 0 && day%size == 0 {
			return true
		}
	}
	return false
}
