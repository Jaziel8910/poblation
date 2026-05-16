# POBLATION — Gemini Context

## Read First
→ Read QUALITY.md in project root before any task.
→ Read GDD.md in project root before any task.
→ Read AGENTS.md in project root for all project rules.
This file only adds Gemini-specific context on top of AGENTS.md.

## Gemini-Specific Behavior
- Prefer reading existing files before generating new code. Use file reading tools to understand what's already implemented.
- When uncertain about a design decision, GDD.md is the answer. Do not invent solutions that aren't in the GDD - the design is complete.
- Long context is an advantage here: use it to cross-reference multiple files simultaneously before making changes. Consistency across files matters more than local elegance within a single file.
- When generating template text (the .txt files in templates/): quality of writing is the primary metric. These templates are what players actually read - they determine if the game feels alive or robotic. Prioritize voice, specificity, and avoiding cliches over template count.

## The Templates Are the Game
This project has 15,000+ text templates in the templates/ directory. They are the most important content in the entire project. When generating templates, the bar is:
- Would this line be surprising the first time you read it? Good.
- Does it sound like something a specific kind of person would think? Good.
- Could it appear in any generic life sim without changes? Bad. Rewrite it.
- Does it use a metaphor about hearts, storms, or broken things? Bad. Cut it.
- Is it a sentence a real person would say or think? Good.
- Is it a sentence that only exists in fiction? Bad.

## Gotchas
- Go is not Python. No implicit returns, no dynamic typing, no None. Every function signature is explicit. Follow existing patterns in the codebase.
- The Charm/Bubbletea architecture is strictly Elm-style MVC. Model -> Update -> View. Side effects only through tea.Cmd. Never directly.
- Template files use [TEMPLATE:ID] header format, not any standard data format. The parser is in internal/templates/engine.go - read it.
- Poble orientation uses float32 (0.0-1.0 scale), not enums. Avoid adding orientation enums - the continuous scale is intentional.
