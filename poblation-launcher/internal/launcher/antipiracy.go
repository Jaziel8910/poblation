package launcher

import (
	"math/rand"
	"time"
)

type AntiPiracyTrigger string

const (
	APNone       AntiPiracyTrigger = "AP_NONE"
	APRandom     AntiPiracyTrigger = "AP_RANDOM"
	APOffline    AntiPiracyTrigger = "AP_OFFLINE"
	APClock      AntiPiracyTrigger = "AP_CLOCK"
	APRAMEater   AntiPiracyTrigger = "AP_RAM_EATER"
)

type AntiPiracySequence struct {
	Trigger AntiPiracyTrigger
	Lines   []string
	Delay   time.Duration
}

func NewLaunchRNG() *rand.Rand {
	return rand.New(rand.NewSource(time.Now().UnixNano()))
}

func RandomAntiPiracyTriggered(rng *rand.Rand) bool {
	if rng == nil {
		rng = NewLaunchRNG()
	}
	return rng.Intn(1000) == 0
}

func BuildAntiPiracySequence(trigger AntiPiracyTrigger, save SaveSummary) AntiPiracySequence {
	if trigger == APNone {
		return AntiPiracySequence{}
	}
	name := save.WorldName
	if name == "" {
		name = "El Origen"
	}
	day := save.Day
	pop := save.Population
	lines := map[AntiPiracyTrigger][]string{
		APRandom: {
			"POBLATION te vio abrir la puerta.",
			"En " + name + ", dia " + itoa(day) + ", alguien conto " + itoa(pop) + " respiraciones.",
			"No es un castigo. Solo una coincidencia muy especifica.",
		},
		APOffline: {
			"GitHub no contesto.",
			name + " sigue guardado en esta maquina.",
			"El fin del mundo tambien funciona sin internet.",
		},
		APClock: {
			"El reloj hizo algo raro.",
			"Primera vez: beneficio de la duda.",
			"Los Pobles no necesitan saberlo.",
		},
		APRAMEater: {
			"RAM EATER esta mirando el calendario.",
			"Detecto demasiadas senales juntas.",
			"No va a borrar nada. Solo te juzga teatralmente.",
		},
	}
	return AntiPiracySequence{
		Trigger: trigger,
		Lines:   lines[trigger],
		Delay:   28 * time.Millisecond,
	}
}

func itoa(value int) string {
	if value == 0 {
		return "0"
	}
	buf := [20]byte{}
	i := len(buf)
	negative := value < 0
	if negative {
		value = -value
	}
	for value > 0 {
		i--
		buf[i] = byte('0' + value%10)
		value /= 10
	}
	if negative {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
