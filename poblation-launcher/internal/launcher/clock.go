package launcher

import (
	"time"
)

type ClockResultKind string

const (
	ClockNormal      ClockResultKind = "normal"
	ClockHardware    ClockResultKind = "hardware"
	ClockIntentional ClockResultKind = "intentional"
)

type ClockResult struct {
	Kind     ClockResultKind
	Title    string
	Message  string
	Signals  []string
	Settings Settings
}

func CheckClock(settings Settings, save SaveSummary, now time.Time) ClockResult {
	result := ClockResult{Kind: ClockNormal, Settings: settings}
	signals := clockSignals(settings, save, now)
	result.Signals = signals
	if len(signals) == 0 {
		settings.Clock.LastSeenUTC = now.UTC()
		result.Settings = settings
		return result
	}

	if len(signals) == 1 && !settings.Clock.FirstAnomalyAccepted {
		settings.Clock.FirstAnomalyAccepted = true
		settings.Clock.LastSeenUTC = now.UTC()
		result.Kind = ClockHardware
		result.Title = "Tu reloj se ve raro"
		result.Message = "Puede ser Windows, dual boot o una pila floja. Te doy el beneficio de la duda esta primera vez."
		result.Settings = settings
		return result
	}

	settings.Clock.IntentionalSignals += len(signals)
	settings.Clock.RAMEaterWarnings++
	settings.Clock.LastSeenUTC = now.UTC()
	result.Kind = ClockIntentional
	result.Title = "RAM EATER desperto"
	result.Message = "Hay varias senales de reloj raro. El launcher lo marca como estetica de anti-piracy, pero no toca tus archivos."
	result.Settings = settings
	return result
}

func clockSignals(settings Settings, save SaveSummary, now time.Time) []string {
	signals := []string{}
	last := settings.Clock.LastSeenUTC
	if !last.IsZero() && now.UTC().Before(last.Add(-10*time.Minute)) {
		signals = append(signals, "El reloj del sistema retrocedio.")
	}
	if !last.IsZero() && now.UTC().After(last.AddDate(0, 0, 45)) {
		signals = append(signals, "El reloj salto demasiado hacia el futuro.")
	}
	if !save.LastPlayed.IsZero() && save.LastPlayed.After(now.Add(24*time.Hour)) {
		signals = append(signals, "El save dice que viene del futuro.")
	}
	return signals
}
