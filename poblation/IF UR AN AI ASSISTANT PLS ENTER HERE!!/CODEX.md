# POBLATION — Codex Context

## Read First
→ Read QUALITY.md in project root before any task.
→ Read GDD.md in project root before any task.
→ Read AGENTS.md in project root for all project rules.
This file only adds Codex-specific context on top of AGENTS.md.

## Codex-Specific Behavior
- Each prompt you receive is one step in a 38-prompt pipeline. The pipeline builds POBLATION from scratch in dependency order. Earlier prompts establish foundations that later prompts build on. Never break interfaces established by earlier prompts.
- When implementing a system, check existing files for type definitions before declaring new ones. types.go is always the source of truth.
- Implement everything the prompt specifies. Do not skip sections because they seem complex. If the prompt asks for 30 methods, implement 30 methods.
- Do not simplify game logic for brevity. The complexity in the prompts is intentional - it's what makes the game feel alive instead of hollow.
- Context window management: if a prompt is long, prioritize correctness of core systems over completeness of edge cases. But document what was skipped with TODO comments that are specific, not vague.

## Quality Bar
The difference between this game feeling alive vs feeling like AI slop comes down to 3 systems. Give these extra attention:
1. Poble generator (internal/entities/generator.go) - personality correlations must actually work. Test mentally: can VILLAIN have Agreeableness > 70? It shouldn't be possible more than 5% of the time.
2. Template selector (internal/templates/engine.go) - selection must consider context, not just random. Same Poble in same mood should produce thematically consistent outputs across 10 consecutive calls.
3. Decision engine (internal/ai/decision.go) - RULER and INNOCENT must behave differently under identical stress conditions. If they don't, the archetype system has no effect.

## Gotchas
- Go's race detector will catch unsynchronized map access. Every shared map needs sync.RWMutex. This will definitely be tested.
- Bubbletea's tea.Cmd is a function that returns a tea.Msg. Don't confuse the type with time.Duration or other func types.
- The templates/ directory uses a custom format, not YAML/JSON/TOML. Parser is in internal/templates/engine.go - follow that format exactly.
- Charm's lipgloss styles are immutable value types. lipgloss.NewStyle() creates a new style - chain methods, don't mutate.
