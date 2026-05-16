## Read First
- Read QUALITY.md in project root before any task.
- Read GDD.md in project root before any task.

## Project
POBLATION is a Go terminal life simulation built on the Charm stack that starts with the last humans and lets a procedural world of behavior, drama, and survival unfold. What makes it unusual is that its behavior AI is not LLM-based at runtime: it uses decision trees, state systems, memory, and 15k+ text templates to generate dialogue, thoughts, dreams, rumors, and reactions with dark humor, adult content, and real narrative consequence.

## Architecture
- Source of truth for all types: internal/entities/types.go
- UI layer: internal/ui/ (Bubbletea models only)
- Game logic: internal/engine/ and internal/ai/ (zero UI imports)
- Templates: templates/ directory, loaded at startup by internal/templates/engine.go
- Save system: internal/save/
- Entry point: main.go -> internal/engine/orchestrator.go

## Critical Rules (the non-negotiables)
1. internal/entities/types.go is ALWAYS the source of truth for types. If there is a conflict between any file and types.go, types.go wins.
2. Game logic systems (internal/ai/, internal/engine/) must NEVER import from internal/ui/. The dependency goes one way only: UI imports engine.
3. The Orchestrator (internal/engine/orchestrator.go) is the only system that knows about all other systems. Do not create new cross-system dependencies outside of it.
4. Every goroutine must accept a context.Context and respect cancellation.
5. All map reads/writes shared across goroutines require sync.RWMutex.
6. Template text is NEVER hardcoded in Go files. All user-facing strings come from templates/ directory loaded at runtime.
7. The behavior AI is procedural - no LLM calls, no external AI API calls, ever. Intelligence comes from decision trees + template selection only.
8. All files must compile with: go build ./... All tests must pass with: go test ./... Run these mentally before outputting any code.
9. Never output partial files. Always output the complete file content.
10. Never change architecture or game design decisions. If a prompt asks you to do something that contradicts GDD.md, follow GDD.md.

## Game Design Non-Negotiables
(These protect the soul of the game - breaking these breaks the game)
- Pobles must feel like real, flawed, contradictory humans. A Poble with Archetype:INNOCENT and Cruelty:80 is a bug, not a feature.
- Personality correlations in the generator are mandatory, not optional. Random stats with no correlation = characters that feel hollow.
- Template selection must consider: archetype, mood, relationship context, and anti-repetition. Pure random selection = AI slop output.
- Rumours must mutate when they spread. Identical rumours after 5 spreadings = the rumour system is broken.
- The sex minigame stores nothing in public history. This is a hard privacy rule, not a preference.

## Build Commands
go build ./...          -> must always pass
go test ./...           -> must always pass
go run main.go          -> starts the game
go run main.go --debug  -> debug mode with console
go run main.go --seed X -> deterministic world with seed X

## Terminology (domain vocabulary)
- Poble: a character/inhabitant (plural: Pobles). Never call them "character", "NPC", "agent", or "entity" in user-facing text. Only in Go type names.
- GameTick: one tick of the time engine = 1 real minute = 1 in-game hour
- Template: a text file entry in templates/ used to generate dialogue/thoughts
- Archetype: one of 15 personality templates (see GDD.md section 4.3)
- Era: civilization stage 0-4 (see GDD.md section 13)
- Orchestrator: the central coordinator that owns all subsystem references
- Drama Score: internal metric for how narratively interesting a situation is

## What NOT to do
- Do not add logging frameworks. Use Go's standard log package.
- Do not add ORMs or databases. Save system uses JSON + gzip.
- Do not add HTTP servers or REST APIs (exception: Wish SSH server in v3.3).
- Do not install dependencies not already in go.mod without asking.
- Do not create new files outside the established directory structure without a clear reason documented in a comment.
- Do not use interface{} or any. Use concrete types or defined interfaces.
- Do not hardcode strings that players will see. Use templates.
- Do not generate placeholder implementations. If you can't fully implement something, say so explicitly instead of writing fake code.

## If Something Is Broken
Before fixing anything:
1. Read the error completely
2. Find the root cause (not just the symptom)
3. Check if types.go is the conflict source
4. Fix the root cause, not the symptom
5. Verify fix doesn't break other systems
Never suppress errors with _ unless the error is genuinely irrelevant and document why with a comment.
