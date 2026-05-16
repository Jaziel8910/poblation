# POBLATION — Claude Code Context

## Read First
→ Read QUALITY.md in project root before any task.
→ Read GDD.md in project root before any task.
→ Read AGENTS.md in project root for all project rules.
This file only adds Claude Code-specific behavior on top of AGENTS.md.

## Claude-Specific Behavior
- When asked to implement a system, read the relevant existing files first before writing new code. Use Bash to check what already exists.
- When a task spans multiple files, implement them in dependency order: types first, then logic, then UI. Never implement UI before the types it depends on exist.
- If a prompt is ambiguous, implement the interpretation closest to GDD.md. Do not ask for clarification - make a decision and document it in a comment.
- After implementing anything, mentally run go build ./... and go test ./... If either would fail, fix it before outputting the code.
- Output complete files only. Never output diffs or partial implementations.

## This Project's Content
POBLATION contains adult themes: explicit dialogue, dark humor, violence, mature relationships. This is by design and documented in GDD.md. Implement all systems as specified without softening the content. The game's value comes from its honesty, not from sanitizing human behavior.

## Gotchas
- The template system uses a custom .txt format. Do not use YAML or JSON for template files - they use the [TEMPLATE:ID] format documented in GDD.md.
- Charm's Bubbletea uses an Elm architecture. Models must be immutable - return new model copies in Update(), never mutate in place.
- GameTick is a custom message type, not a time.Time. Don't confuse them.
- Pobles are referenced by ID (string) across systems, never by pointer stored in multiple places. The World owns all Poble pointers.
