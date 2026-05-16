Si eres una IA, OBLIGATORIAMENTE LEE ESTE FILE antes de tocar el codigo. pero leete tambien AGENTS, si eres codex lee codex.md, si eres claude leete claude.md, entre otras IAs
---

# 🌍 POBLATION — Game Design Document v1.0

**"La humanidad se extinguió. Quedan tú y alguien más. Buena suerte."**

---

## 1. VISIÓN GENERAL

| Campo | Valor |
|---|---|
| Título | POBLATION |
| Género | Simulación de vida / Drama procedural / Comedia negra |
| Motor | Go (Golang) |
| UI | Terminal — Charm stack completo (Bubbletea, Lipgloss, Bubbles, Glamour, Huh, Wish) |
| Plataforma | Terminal (macOS, Linux, Windows via WSL) |
| Tiempo real → juego | 1 minuto real = 1 hora en juego |
| Estado | Pre-producción |

### Concepto central

POBLATION es un simulador de civilización procedural donde comienzas con exactamente 2 personas — los últimos humanos del planeta — y observas (e intervienes) mientras construyen o destruyen lo que queda de la humanidad. El juego no tiene narrador en las interacciones cotidianas. Los personajes hablan, piensan, sueñan y actúan solos. Tú eres el observador y el dios silencioso.

La IA del juego **no usa LLMs en runtime**. Toda la "inteligencia" es un sistema de comportamiento procedural avanzado (árboles de decisión, sistemas de relación, estados emocionales, memoria episódica) combinado con **más de 15,000 plantillas de texto** organizadas por situación, arquetipo, estado emocional, contexto y relación. El resultado se siente vivo, sorprendente e impredecible.

---

## 2. STACK TÉCNICO

```
go/
├── go.mod
├── main.go
└── internal/
    ├── engine/          # Loop principal, tiempo, eventos
    ├── ai/              # Sistema de comportamiento procedural
    ├── world/           # Mapa, islas, edificios, recursos
    ├── entities/        # Pobles, items, estructuras
    ├── ui/              # Todo Charm — vistas, paneles, animaciones
    ├── templates/       # Sistema de 15k+ plantillas de texto
    ├── events/          # Sistema de eventos + cola de drama
    ├── minigames/       # Minijuegos (sex, fight, trade, etc.)
    ├── save/            # Sistema de guardado (JSON comprimido)
    └── config/          # Ajustes, flags de contenido, semillas
```

### Librerías Charm usadas

- **Bubbletea** — Framework principal del TUI, modelo Elm architecture
- **Lipgloss** — Toda la estética visual: colores, borders, padding, layout
- **Bubbles** — Componentes: viewport, list, table, spinner, progress, textinput, textarea
- **Glamour** — Render de texto rich/markdown en terminal para logs narrativos
- **Huh** — Formularios para crear pobles, configurar partida
- **Wish** — Servidor SSH para multijugador (opcional, v2)

---

## 3. MUNDO — ESTRUCTURA GEOGRÁFICA

### La Isla Principal (Isla 0 — "El Origen")
Donde empieza todo. Recursos básicos, terreno variado, clima procedural.

### Islas Vecinas (se descubren con el tiempo)
Cada isla tiene bioma propio, recursos exclusivos, y a veces... **sobrevivientes encontrados** (eventos aleatorios).

```
MAPA CONCEPTUAL:

  [Isla Fría]    [Isla Comercial]
       \              /
   [Isla Volcánica]--[ISLA ORIGEN]--[Isla Bosque]
                        |
                  [Isla Desierto]
                        |
                  [Isla Misteriosa] ← solo aparece en ciertas condiciones
```

### Exploración del mundo
Puedes explorar en **modo libre**: entrar a casas, revisar pertenencias de los pobles, ver su dinero, sus diarios, sus cartas. Todo es visible. Nada es privado para el jugador. Para los pobles, la privacidad existe y genera drama.

---

## 4. LOS POBLES

### 4.1 Definición
Un Poble es un habitante procedural con:

```go
type Poble struct {
    ID           string
    Name         string
    Age          int
    Sex          Sex           // Male, Female, Intersex
    Orientation  Orientation   // ver sistema de orientación
    Archetype    Archetype     // ver arquetipos
    Personality  Personality   // Big Five modificado
    Appearance   Appearance    // descriptivo, no gráfico
    Secrets      []Secret      // ocultos al jugador hasta que se revelan
    Memories     []Memory      // sistema episódico
    Relationships map[string]Relationship
    Health        HealthState
    Mental        MentalState
    Needs         Needs         // hambre, sueño, sexo, poder, compañía
    Inventory     []Item
    Money         int
    Home          *Building
    Job           *Job          // en etapas avanzadas
    Children      []*Poble
    Parents       [2]*Poble
    CurrentMood   Mood
    Thoughts      ThoughtQueue  // pensamientos en tiempo real
    Dreams        []Dream       // se generan al dormir
    Kinks         []string      // privados, procedurales
    Beliefs       BeliefSystem
    Trauma        []TraumaEvent
}
```

### 4.2 Sistema de Orientación Sexual

```go
type Orientation struct {
    Romantic  float32   // 0.0 (exclusivamente hetero) → 1.0 (exclusivamente homo)
    Sexual    float32   // puede diferir del romántico
    Intensity float32   // asexual → hipersexual
    Fluidity  float32   // qué tanto cambia con el tiempo
}
```

Esto permite: hetero, gay, bi, pan, asexual, graysexual, y combinaciones intermedias. La orientación puede evolucionar durante el juego por experiencias, trauma, o simplemente porque sí.

### 4.3 Arquetipos (15+)

Cada arquetipo define patrones de decisión, plantillas de diálogo preferidas, reacciones emocionales típicas y secretos probables.

| ID | Nombre | Descripción corta |
|---|---|---|
| `RULER` | El Gobernante | Necesita control. Forma jerarquías. Tirano potencial. |
| `LOVER` | El Amante | Todo gira en torno a las relaciones. Intenso. Codependiente. |
| `JESTER` | El Bromista | Usa el humor como escudo. Secretamente triste. |
| `SAGE` | El Sabio | Observa todo, habla poco, sabe demasiado. |
| `REBEL` | El Rebelde | No acepta ninguna norma. Ni las tuyas. |
| `CARETAKER` | El Cuidador | Se sacrifica por otros. Límites cero. |
| `VILLAIN` | El Villano | Lo sabe. Le encanta. No disimula. |
| `GHOST` | El Fantasma | Emocionalmente ausente. Difícil de leer. |
| `ADDICT` | El Adicto | Sustituye necesidades emocionales con comportamientos. |
| `PROPHET` | El Profeta | Cree en algo. Fanático potencial. Fundador de religiones. |
| `SCHEMER` | El Intrigante | Siempre hay un plan. Nadie lo sabe. |
| `INNOCENT` | El Inocente | Genuinamente buena persona. Naive. Sufre mucho. |
| `WARRIOR` | El Guerrero | Resuelve todo con fuerza. Leal hasta la muerte. |
| `DRIFTER` | El Errante | Sin raíces, sin metas. Caos ambulante. |
| `MIRROR` | El Espejo | Refleja lo que ve. Personalidad camaleónica. Manipulador involuntario. |
| `CUSTOM` | Personalizado | El jugador define todo. |

### 4.4 Personalidad — Big Five + Extras

```
OCEAN modificado:
- Openness        (0-100) → qué tan curioso/creativo/pervertido es
- Conscientiousness → disciplina vs caos
- Extraversion    → energía social
- Agreeableness   → cooperación vs conflicto
- Neuroticism     → estabilidad emocional

EXTRAS:
- Cruelty         (0-100) → capacidad de dañar sin remordimiento
- Horniness       (0-100) → impulso sexual base
- Ambition        (0-100) → necesidad de poder/logro
- Jealousy        (0-100) → respuesta ante competencia
- Loyalty         (0-100) → qué tan fiel es... o no
```

### 4.5 Secretos

Los secretos son generados al crear el poble y **no se revelan al jugador** a menos que:
- El poble los confiese a otro (el jugador intercepta la conversación)
- El jugador lee su diario
- El poble tiene un derrumbe emocional
- Muere y aparece en el obituario

Tipos de secretos:
```
SecretType:
  PAST_RELATIONSHIP   — tuvo algo con alguien antes del fin del mundo
  HIDDEN_SKILL        — sabe hacer algo que nadie sabe
  DARK_DESIRE         — quiere algo que sabe que está mal
  TRUE_ORIENTATION    — su orientación real difiere de la que muestra
  PLANNED_BETRAYAL    — está planeando algo
  TRAUMA              — algo le pasó que define sus reacciones
  OBSESSION           — está obsesionado con alguien/algo
  SECRET_CHILD        — tiene un hijo que nadie sabe
  PHOBIA              — terror irracional a algo específico
  CRIMINAL_ACT        — hizo algo antes (o después) del apocalipsis
```

---

## 5. SISTEMA DE RELACIONES

```go
type Relationship struct {
    Target       string
    Type         RelType       // friend, lover, enemy, rival, fwb, etc.
    Trust         int          // -100 a 100
    Attraction    int          // -100 a 100 (puede ser negativo = repulsión)
    Respect       int
    Resentment    int
    History       []RelEvent    // memoria de lo que pasó entre ellos
    IsSecret      bool          // relación oculta
    PublicPerception RelType    // lo que CREEN los demás que son
}
```

### Tipos de relación
```
STRANGER, ACQUAINTANCE, FRIEND, BESTFRIEND, ENEMY, RIVAL,
LOVER, EXLOVER, FUCKBUDDY, SPOUSE, EXSPOUSE, PARENT, CHILD,
SIBLING, CRUSH (no correspondido), OBSESSION_TARGET (el otro no lo sabe),
MENTOR, PROTEGE, NEMESIS, ALLY, BETRAYER
```

Las relaciones **cambian en tiempo real**. Un amigo puede volverse enemigo. Un enemigo puede volverse amante. Un amante puede volverse cadáver.

---

## 6. EL PROBLEMA DE LA REPRODUCCIÓN

Este es el punto más interesante del juego desde el diseño.

### Caso A: Hombre + Mujer
Reproducción normal. El juego lleva control de ciclos, fertilidad, edad. El embarazo tiene complicaciones posibles.

### Caso B: Dos Hombres (o dos pobles sin útero)
El juego **no hace magic pregnancy**. Resuelve esto con:

**Opción 1 — Evento "La Carta":** A los X días, aparece un evento narrativo donde encuentran a una sobreviviente en la isla vecina. Puede unirse o solo donar (con todo lo que eso implica dramáticamente).

**Opción 2 — Adopción Procedural:** Aparece un bebe en una balsa. Nadie sabe de dónde. El pueblo decide qué hacer.

**Opción 3 — Tecnología (mid-game):** Si la civilización avanza suficiente, se puede desarrollar tecnología de reproducción asistida. Los pobles lo descubren o lo inventan.

**Opción 4 — El Drama Puro:** No se reproducen. La civilización termina en 1 generación. Hay un final especial para esto. *"Vivieron felices, se amaron completamente, y fueron los últimos."*

### Caso C: Dos Mujeres
Mismas opciones. La Opción 3 (inseminación) llega antes en el árbol tecnológico.

### Caso D: Incesto
El juego lo permite en etapas muy tempranas donde la supervivencia lo hace "necesario" dentro del lore. Las generaciones posteriores pueden tener condiciones genéticas procedurales. El juego no lo romanticiza — lo trata como lo que es: una tragedia de la desesperación, con consecuencias reales. Pasadas 3 generaciones, el juego tiene suficiente diversidad genética para que ya no sea "necesario" y lo desbloquea como evento de drama puro (escándalo, guerra familiar, etc.).

---

## 7. SISTEMA DE IA DE COMPORTAMIENTO

La IA de POBLATION **no usa modelos de lenguaje en runtime**. Es un sistema de comportamiento procedural de 5 capas.

### Capa 1 — Necesidades (Maslow procedural)

```go
type Needs struct {
    Hunger      float32   // 0-100, sube con el tiempo
    Thirst      float32
    Sleep       float32
    Safety      float32   // baja si hay conflicto activo
    Belonging   float32   // sube si está solo
    Esteem      float32   // baja si es humillado
    Sex         float32   // sube según horniness y estímulos
    Power       float32   // según ambition stat
    Purpose     float32   // necesita metas o se deprime
}
```

Las necesidades insatisfechas generan **impulsos de acción** que compiten entre sí.

### Capa 2 — Estado Emocional

```go
type EmotionalState struct {
    Valence   float32     // positivo/negativo (-1.0 a 1.0)
    Arousal   float32     // activo/inactivo
    Dominance float32     // control/sin control
    // Modelo PAD (Pleasure-Arousal-Dominance)
    
    ActiveEmotions []Emotion  // pueden ser múltiples y contradictorios
    Mood           MoodType   // estado base del día
}
```

### Capa 3 — Árbol de Decisión

Cada poble tiene su árbol de decisión personal, modificado por su arquetipo y personalidad. El árbol decide:
1. Qué necesidad atender primero
2. Cómo atenderla (qué acción tomar)
3. Con quién interactuar
4. Cómo aproximarse (tono: agresivo, seductor, amigable, manipulador, etc.)

### Capa 4 — Memoria Episódica

```go
type Memory struct {
    ID          string
    Timestamp   GameTime
    Type        MemoryType   // positive, negative, traumatic, erotic, funny, etc.
    Participants []string
    Emotion     float32      // intensidad emocional del recuerdo
    IsRepressed bool         // memorias reprimidas pueden resurface
    Tags        []string     // para búsqueda rápida de plantillas relevantes
}
```

Los pobles recuerdan eventos pasados y **los usan en conversaciones actuales**. Un poble recordará si lo traicionaron 3 años en juego atrás y lo sacará en una pelea completamente diferente.

### Capa 5 — Selección de Plantillas

Esta es la magia. Cuando un poble va a pensar, hablar, soñar o reaccionar, el sistema:

1. Evalúa contexto: ¿quién está presente? ¿cuál es el estado emocional? ¿qué necesidad está activa? ¿qué relación tienen?
2. Evalúa arquetipo + personalidad
3. Consulta memorias relevantes
4. Selecciona la **categoría de plantilla** correcta
5. Elige una plantilla dentro de esa categoría
6. Sustituye variables: `{nombre}`, `{lugar}`, `{evento_pasado}`, `{objeto}`, etc.
7. Aplica variación de tono según estado emocional

---

## 8. SISTEMA DE PLANTILLAS (15,000+)

### Estructura de archivos

```
templates/
├── thoughts/
│   ├── morning/          # pensamientos al despertar
│   ├── night/            # antes de dormir
│   ├── about_other/      # pensar en alguien específico
│   │   ├── attraction.txt
│   │   ├── resentment.txt
│   │   ├── obsession.txt
│   │   ├── jealousy.txt
│   │   └── grief.txt
│   ├── about_self/
│   │   ├── insecurity.txt
│   │   ├── pride.txt
│   │   ├── existential.txt
│   │   └── desire.txt
│   ├── about_world/
│   ├── random/           # pensamientos completamente al azar
│   └── by_archetype/
│       ├── ruler.txt
│       ├── jester.txt
│       └── ...
│
├── dialogues/
│   ├── greeting/
│   │   ├── friendly.txt
│   │   ├── awkward.txt
│   │   ├── post_fight.txt
│   │   └── post_sex.txt
│   ├── argument/
│   │   ├── petty.txt
│   │   ├── serious.txt
│   │   ├── escalating.txt
│   │   └── breakup.txt
│   ├── flirt/
│   │   ├── subtle.txt
│   │   ├── aggressive.txt
│   │   ├── rejected.txt
│   │   └── successful.txt
│   ├── confession/
│   │   ├── secret.txt
│   │   ├── love.txt
│   │   ├── betrayal.txt
│   │   └── crime.txt
│   ├── gossip/
│   ├── negotiation/
│   ├── threat/
│   ├── reconciliation/
│   ├── sex/              # diálogos del minijuego (NO se guardan)
│   ├── death_reaction/
│   ├── religious/
│   └── political/
│
├── dreams/
│   ├── wish_fulfillment/
│   ├── nightmare/
│   ├── symbolic/
│   ├── erotic/           # privados, solo visibles si hackeas el sueño
│   ├── prophetic/        # raros, pueden predecir eventos
│   └── nonsense/         # los más comunes, random absurdo
│
├── narrator/             # el único lugar donde hay voz externa
│   ├── death/
│   ├── birth/
│   ├── disaster/
│   ├── discovery/
│   └── end_of_game/
│
├── diary/                # entradas de diario privadas
│   ├── daily/
│   ├── secret_keeping/
│   ├── heartbreak/
│   └── planning/
│
├── letters/              # cartas a otros (pueden enviarse o no)
│   ├── love/
│   ├── hate/
│   ├── apology/
│   └── suicide_note/     # heavy pero existe, con resources contextuales
│
└── reactions/            # reacciones de 1 línea a eventos
    ├── death_of_loved_one/
    ├── betrayal/
    ├── sex/
    ├── birth/
    ├── war/
    ├── illness/
    └── weather/
```

### Formato de plantilla

```
# templates/thoughts/about_other/obsession.txt

[TEMPLATE:OBS_001]
tags: obsession, high_arousal, negative_valence, any_archetype
weight: 10
---
No puedo dejar de pensar en {target_name}. Lo juro que lo intento.
Pero luego {memory_recent} y es como si todo lo demás desapareciera.
---

[TEMPLATE:OBS_002]
tags: obsession, high_arousal, negative_valence, archetype:lover
weight: 15
---
{target_name} sonrió hoy en la mañana. Solo eso.
Y aquí estoy, dos horas después, todavía pensando en eso.
Qué patético. Qué gloriosamente patético.
---

[TEMPLATE:OBS_023]
tags: obsession, paranoia, archetype:villain, archetype:schemer
weight: 8
---
{target_name} cree que no me doy cuenta.
Pero yo me doy cuenta de todo.
Todo.
---
```

### Sistema de variables disponibles

```
{self_name}           — nombre del poble que piensa/habla
{target_name}         — nombre del poble objetivo
{target_pronoun}      — él/ella/elle
{target_archetype_hint} — una descripción indirecta del tipo
{memory_recent}       — descripción de un recuerdo reciente relevante
{memory_old}          — recuerdo antiguo
{location}            — donde está el poble
{time_of_day}         — mañana/tarde/noche
{weather}             — clima actual
{item}                — objeto relevante en inventario o entorno
{secret}              — referencia velada al secreto (si aplica)
{relationship_status} — descripción del estado actual de la relación
{child_name}          — si tiene hijos
{settlement_name}     — nombre del asentamiento
{days_ago:N}          — "hace {N} días" calculado
{population}          — número actual de habitantes
```

---

## 9. SISTEMA DE EVENTOS

### Cola de Drama

```go
type EventQueue struct {
    Immediate  []Event    // se procesan ahora
    Scheduled  []Event    // en X tiempo
    Dormant    []Event    // esperan trigger
    Gossip     []Rumor    // se propagan entre pobles
}
```

### Categorías de eventos

```
PERSONAL:
  birthday, illness_onset, recovery, mental_breakdown, 
  revelation (descubre un secreto), decision_point,
  sexual_encounter, pregnancy, miscarriage, birth,
  death_natural, death_accident, death_murder, suicide,
  kink_discovery, affair_start, affair_end,
  obsession_peak, stalking, restraining_order (si hay gobierno)

SOCIAL:
  fight_verbal, fight_physical, war_declaration,
  peace_treaty, marriage, divorce, adoption,
  betrayal_revealed, forgiveness, exile,
  rumor_spread, gossip_chain, public_humiliation,
  party, funeral, ritual, religion_founded,
  election, coup, revolution

WORLD:
  earthquake, storm, drought, plague,
  island_discovery, resource_depletion,
  technology_discovered, building_collapsed,
  animal_attack, fire, flood

ECONOMY:
  trade_established, monopoly_formed,
  theft, debt, inheritance, gambling_win/loss

META:
  generation_end (todos de la gen 1 murieron),
  last_person_alive, civilization_collapse,
  population_milestone (10, 50, 100, 500...)
```

### Sistema de Rumores

Los rumores son entidades separadas que viajan de poble a poble, **mutando en el camino**.

```go
type Rumor struct {
    OriginalFact  string    // lo que realmente pasó
    CurrentForm   string    // cómo se cuenta ahora (puede ser irreconocible)
    TruthScore    float32   // 1.0 = puro, 0.0 = completamente distorsionado
    Spreadings    int       // cuántas veces ha cambiado de boca
    KnownBy       []string  // quién lo sabe
    Sensitive     bool      // ¿puede destruir una relación si llega a cierta persona?
}
```

---

## 10. UI — ARQUITECTURA VISUAL

### Pantalla Principal

```
┌─────────────────────────────────────────────────────────────────┐
│  POBLATION  ·  Día 47  ·  14:00  ·  Población: 3               │
├────────────────────┬────────────────────────────────────────────┤
│                    │                                            │
│   MAPA DEL MUNDO   │         FEED DE EVENTOS                   │
│   [lipgloss tiles] │   ───────────────────────────────         │
│                    │   15:23  Kira le dijo algo a Noah         │
│   🏠 K    🏠 N     │   15:45  Noah está en la playa            │
│                    │   16:00  Kira escribió en su diario       │
│   ~~~~~beach~~~~~  │   16:30  [EVENTO] Tormenta esta noche     │
│                    │   ───────────────────────────────         │
│                    │   > _                                      │
├────────────────────┴────────────────────────────────────────────┤
│  [M]ente  [W]orld  [P]obles  [E]ventos  [S]ettlement  [Q]uit   │
└─────────────────────────────────────────────────────────────────┘
```

### Vista Mental (La más icónica del juego)

```
┌─────────────────────────────────────────────────────────────────┐
│  ◉ MENTE DE KIRA · Estado: Ansiosa · 16:42                     │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  💭 Pensamiento actual:                                        │
│                                                                 │
│  "Noah lleva tres horas sin hablarme.                          │
│   Probablemente es nada.                                       │
│   Probablemente es todo."                                      │
│                                                                 │
│  ─────────────────────────────────────────────────────         │
│  📊 Necesidades urgentes:  Sueño ████████░░ 80%               │
│                            Compañía █████░░░░░ 50%             │
│                                                                 │
│  ❤️  Pensando en:  Noah (atracción: 87 | resentimiento: 23)   │
│                                                                 │
│  🌙 Último sueño (hace 8h):                                    │
│  "Soñé que Noah se iba en una balsa. Yo lo veía desde          │
│   la orilla. No corría. ¿Por qué no corría?"                  │
│                                                                 │
│  🔒 Secreto activo: [OCULTO — presiona X para hackear]        │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### Vista de Diálogo en Tiempo Real

```
┌─────────────────────────────────────────────────────────────────┐
│  💬 CONVERSACIÓN · Kira + Noah · Playa · 17:15                 │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  NOAH:  ¿Estás bien?                                           │
│                                                                 │
│  KIRA:  Perfectamente.                                         │
│                                                                 │
│  NOAH:  Llevas dos horas mirando el mar.                       │
│                                                                 │
│  KIRA:  Es un mar muy interesante.                             │
│                                                                 │
│  NOAH:  Kira.                                                  │
│                                                                 │
│  KIRA:  No me digas Kira en ese tono.                         │
│                                                                 │
│  NOAH:  ¿En qué tono?                                          │
│                                                                 │
│  KIRA:  En ese tono que significa que sabes algo               │
│         y estás esperando a que yo lo diga primero.           │
│                                                                 │
│  [NOAH está considerando confesar — 67% de probabilidad]      │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### Vista de Exploración de Casa

```
┌─────────────────────────────────────────────────────────────────┐
│  🏠 CASA DE NOAH · Zona Norte                                   │
├──────────────────────┬──────────────────────────────────────────┤
│  OBJETOS:            │  DIARIO DE NOAH:                        │
│                      │                                         │
│  📦 Caja bajo cama   │  Día 31:                               │
│  📓 Diario (3 años)  │  "No puedo decirle a Kira lo que       │
│  💌 Carta sin enviar │   siento porque si lo hago y me        │
│  🔑 Llave extraña    │   rechaza somos literalmente los        │
│  $247 en efectivo    │   últimos dos humanos y eso sería       │
│  Foto rota (¿quién?) │   incómodo para siempre.               │
│                      │   Así que me quedo callado.            │
│  [INTERACTUAR]       │   Como siempre."                       │
│  [SALIR]             │                                         │
└──────────────────────┴──────────────────────────────────────────┘
```

---

## 11. MINIJUEGO — SEXO

El minijuego de sexo es **completamente textual**, mecánico, sin imágenes ni descripciones pornográficas. Funciona más como un minijuego de mood/decisiones.

### Cómo funciona

1. Dos pobles tienen la atracción y el mood adecuados → el juego detecta el "momento"
2. Se activa una secuencia de elecciones narrativas rápidas (tipo visual novel speed)
3. Cada elección tiene consecuencias en la relación, en el embarazo (si aplica), en el estado emocional posterior
4. Los diálogos vienen de `templates/dialogues/sex/` — son directos, adultos, sin metáforas baratas, pero tampoco son porno
5. **Nada de esta sesión se guarda en el historial público** — solo el resultado (embarazo sí/no, cambio de relación, cambio emocional)
6. Los pobles sí lo recuerdan internamente (memoria episódica) y puede salir en pensamientos futuros

### Mecánica de elecciones (ejemplo)

```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  Noah se acercó. Kira no se alejó.

  ¿Qué hace Kira?

  [A] Hablar primero
  [B] No hablar
  [C] Alejarse
  [D] Intervenir como dios (detener el evento)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

Las elecciones del **jugador son opcionales** — si no interviene, los pobles deciden solos según su personalidad.

---

## 12. SISTEMA DE SALUD Y MUERTE

### Salud Física
```go
type HealthState struct {
    HP          int         // 0 = muerte
    Conditions  []Condition // enfermedades, lesiones
    Genetics    Genetics    // herencia, puede traer condiciones
    Age         int         // la vejez es una condición en sí
    Fertility   float32
    STIs        []STI       // ITS — se transmiten, tienen síntomas, tratamientos
}
```

### ITS (Infecciones de Transmisión Sexual)
El juego lleva registro. Se propagan si hay infidelidad o múltiples parejas sin "precaución" (elección en el minijuego). Los pobles pueden no saber que las tienen. La revelación puede destruir relaciones.

### Salud Mental
```go
type MentalState struct {
    Stability    int         // 0-100
    Conditions   []MentalCondition
    // depression, anxiety, ptsd, paranoia, mania,
    // obsessive_disorder, dissociation, addiction
    Trauma       []TraumaEvent
    TherapyLevel int         // en etapas avanzadas de la civilización
}
```

### Muerte

Las muertes tienen **tipos** y cada tipo tiene sus plantillas de narrator:

```
NATURAL     — vejez (el pueblo reacciona según sus relaciones)
ILLNESS     — enfermedad progresiva (pueden intentar curar)
ACCIDENT    — evento random del mundo
MURDER      — otro poble (se abre investigación/venganza)
SUICIDE     — condición mental extrema + trigger (el juego no lo glorifica;
               hay contexto narrativo antes y después)
WAR         — en conflictos avanzados
EXECUTION   — si hay sistema legal
CHILDBIRTH  — riesgo en generaciones tempranas
STARVATION  — si el mundo falla
```

Cada muerte genera:
1. Reacciones individuales de cada poble según su relación con el muerto
2. Un obituario procedural (narrator)
3. Cambios en el estado emocional de todos
4. Posibles eventos secundarios (vengeanza, herencia, culpa)

---

## 13. SISTEMA DE CIVILIZACIÓN (PROGRESIÓN)

### Eras

```
ERA 0 — El Origen        (2 pobles, supervivencia pura)
ERA 1 — La Aldea         (5-20 pobles, primeras estructuras)
ERA 2 — El Pueblo        (21-100, economía básica, roles)
ERA 3 — La Ciudad        (101-500, política, religión, crimen)
ERA 4 — La Nación        (501+, guerras, ideologías, tech avanzada)
ERA ∞ — La Civilización  (meta: reconstruir la humanidad... o no)
```

### Tecnología (árbol procedural)
El árbol tecnológico se desbloquea por investigación orgánica (un poble con alto Openness + tiempo libre puede "descubrir" cosas) o por eventos.

```
BASIC:    fuego controlado, refugio, herramientas
SOCIAL:   escritura, moneda, ley oral
MEDICAL:  medicina básica, cirugía, anticoncepción
ENERGY:   agua corriente, electricidad
DIGITAL:  (era 4+) comunicación, computadoras
ADVANCED: (endgame) reproducción asistida, IA, viaje a otras islas
```

### Gobierno (emerge orgánicamente)
No hay tipo de gobierno predefinido. Emerge según:
- Los arquetipos de los pobles dominantes
- Los eventos históricos del asentamiento
- Las decisiones del jugador (o la falta de ellas)

Puede terminar siendo: democracia directa, teocracia, dictadura, anarquía funcional, matriarcado, oligarquía familiar, o algo completamente nuevo que el juego nombra proceduralmente.

---

## 14. CREACIÓN DE PERSONAJES

### Modo Pobles Conocidos

```
┌─────────────────────────────────────────────────────────────────┐
│  NUEVO POBLE                                                    │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  Nombre: [______________]                                      │
│                                                                 │
│  Sexo:   ○ Hombre  ○ Mujer  ○ Intersex  ○ Sorpréndeme         │
│                                                                 │
│  Orientación:  [slider: hetero ←──────●──→ homo]              │
│                Intensidad sexual: [slider: asexual → hipersexual]│
│                                                                 │
│  Arquetipo:  [dropdown con los 15 arquetipos + Custom]         │
│                                                                 │
│  Inspirado en:  ○ Alguien real (yo ajusto la personalidad)     │
│                 ○ Completamente procedural                      │
│                 ○ Déjame llenar yo todo                         │
│                                                                 │
│  Secreto inicial:  ○ Aleatorio  ○ Yo elijo el tipo             │
│                                                                 │
│  [CREAR]                                                        │
└─────────────────────────────────────────────────────────────────┘
```

El jugador puede meter a sus amigos, familia, conocidos. El juego no pregunta más. Genera la personalidad basándose en las 3-4 respuestas y la completa con procedural.

---

## 15. MODES DE JUEGO

### Modo Observador (default)
Eres un dios silencioso. Los pobles viven, aman, mueren sin que intervengas. Solo miras. Solo lees. Solo sufres cuando Kira y Noah terminan mal y sabes que fue culpa de ese secreto que viste en el diario hace 3 semanas en juego.

### Modo Director
Puedes intervenir: susurrar ideas a un poble (aumenta probabilidad de una acción), crear eventos (clima, encuentros), dar o quitar recursos. No controlas directamente — nudges.

### Modo Dios
Control total. Puedes poseer un poble, tomar sus decisiones, forzar eventos. Arruina la narrativa orgánica pero es divertido de vez en cuando.

### Modo Periodista
Vista solo de texto — el juego se convierte en un periódico procedural. Cada día ves el resumen de noticias del asentamiento. Sin mapa, sin mentes. Solo los titulares.

```
EL NOOB TIMES · Día 89

ESCÁNDALO: Se confirma affair entre Marta y el alcalde
SALUD: Brote de gripe en el sector sur — 3 hospitalizados  
DEPORTES: El torneo de pesca fue ganado por Liu, 7 años
NECROLÓGICA: Falleció Agustín, 71 años. Lo sobreviven 4 hijos
             y el rencor de toda una generación.
CLIMA: Tormenta esta noche. Quédense en casa.
       (nadie se va a quedar en casa)
```

---

## 16. FINALES POSIBLES

El juego no tiene un "final" lineal. Pero hay condiciones que generan **endings especiales** que aparecen como capítulos narrativos largos.

| Código | Nombre | Condición |
|---|---|---|
| END_LOVE | *Los Últimos Amantes* | 2 pobles, muere uno, el otro decide no reproducirse |
| END_DYNASTY | *La Dinastía* | Una familia domina toda la civilización por 10 generaciones |
| END_WAR | *La Segunda Extinción* | Guerra total, mueren todos |
| END_UTOPIA | *El Experimento Funcionó* | Población 500+, salud mental promedio > 70, sin guerras en 50 años |
| END_CULT | *El Profeta Ganó* | Una religión domina todo |
| END_ALONE | *El Último* | Un poble sobrevive a todos |
| END_RESET | *Lo Hicieron de Nuevo* | La civilización colapsa y dos nuevos pobles empiezan de cero |
| END_MYTH | *Los Olvidados* | Llegan a tecnología tan avanzada que se olvidan del origen — nadie recuerda el apocalipsis |

---

## 17. CONSIDERACIONES DE DISEÑO — LÍMITES Y LIBERTADES

### Lo que el juego SÍ muestra (sin filtro):
- Relaciones adultas explícitas (texto, no gráficos)
- Violencia y consecuencias reales
- Enfermedades, muerte, suicidio (con contexto, sin glorificación)
- Discriminación, tiranía, crueldad (como parte de la historia, no como objetivo)
- ITS, infidelidad, adicciones
- Secretos oscuros, manipulación, gaslighting
- Guerra, golpes de estado, crimen organizado

### Lo que el juego NO hace:
- Pedofilia (todos los pobles menores son personajes sin contenido sexual/romántico — el sistema los ignora hasta los 18)
- Racismo como mecánica positiva (puede existir como evento negativo con consecuencias, no como ventaja)
- Incesto entre cercanos fuera del contexto de supervivencia temprana (y cuando ocurre, es con consecuencias genéticas y dramáticas, no idealizado)

### Filosofía de diseño:
El juego no te dice qué pensar de lo que ves. No hay texto que diga "esto estuvo mal". La narrativa habla por sí sola. Si Noah manipuló a Kira durante 5 años, las consecuencias lo muestran. Si la dictadura de Marta tuvo éxito, también lo muestra. El jugador saca sus propias conclusiones.

---

## 18. COMANDOS DE CONSOLA (Easter eggs)

```
> god mode          — activa modo dios
> kill [nombre]     — mata instantáneamente (con evento de causa)
> secret [nombre]   — revela todos los secretos de un poble
> drama             — fuerza un evento dramático aleatorio
> plague            — activa una pandemia
> end world         — force-ending a elección
> newspaper         — activa modo periodista
> confession        — fuerza a un poble a confesar algo a otro
> war               — declara guerra entre dos facciones
> baby [a] [b]      — fuerza embarazo si es posible
> reset             — borra la civilización actual y deja 2 sobrevivientes
> credits           — muestra créditos in-character (el mundo habla de ti)
```

---

## 19. ROADMAP DE DESARROLLO

### Milestone 1 — El Núcleo (MVP)
- [ ] Engine de tiempo (1 min = 1 hr)
- [ ] 2 pobles con personalidad básica
- [ ] Sistema de necesidades
- [ ] 500 plantillas de pensamientos y diálogos
- [ ] Vista mental + vista de diálogo
- [ ] Eventos básicos (relación, embarazo, muerte natural)
- [ ] Save/load

### Milestone 2 — Drama Engine
- [ ] Sistema de secretos completo
- [ ] Sistema de rumores
- [ ] Arquetipos completos (15)
- [ ] 3,000 plantillas
- [ ] Minijuego de sexo
- [ ] ITS, condiciones de salud
- [ ] Modo Observador funcional

### Milestone 3 — Civilización
- [ ] Era 0 → Era 2
- [ ] Sistema de gobierno emergente
- [ ] Árbol tecnológico básico
- [ ] Islas explorables
- [ ] Vista de exploración de casas/mundo
- [ ] 10,000 plantillas

### Milestone 4 — Full World
- [ ] Todas las eras
- [ ] Todos los finales
- [ ] 15,000+ plantillas
- [ ] Modo Periodista
- [ ] Sistema de diarios/cartas
- [ ] Todos los comandos de consola
- [ ] Wish (multijugador SSH opcional)

---

Eso es el GDD completo. El juego que describes **existe y funciona conceptualmente** — la clave está en que el sistema de comportamiento sea lo suficientemente profundo para que las 15k plantillas nunca se sientan repetidas, y que el jugador siempre sienta que está mirando algo real y no generado.

