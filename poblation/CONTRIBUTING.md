# Contributing to POBLATION

The most welcome contribution is new template content.

If you add thoughts, dreams, dialogues, endings, or reactions, you usually improve the game more than by adding one more system.

## Template Format

Every template file uses this structure:

```text
[TEMPLATE:UNIQUE_ID]
tags: thought, random, mundane
weight: 10
archetypes: LOVER, GHOST
moods: ANXIOUS, SAD
required_context: target, relationship
min_relationship: -40
max_relationship: 80
---
The body text goes here.
It can span multiple lines.
---
```

Rules:

- `ID` must be unique across the whole template tree.
- `tags` is a comma-separated list.
- `weight` is optional but should stay positive.
- `archetypes`, `moods`, `required_context`, `min_relationship`, and `max_relationship` are optional.
- The body is everything between the two `---` lines.
- Variables must already exist in `internal/templates/engine.go`.

## How To Add Templates

1. Put the file in the right folder under `templates/`.
2. Follow the exact format above.
3. Keep the voice specific to the category.
4. Avoid generic life-sim filler.
5. Run the template checks before opening a PR.

## How To Test Templates

Use these checks:

```bash
go test ./internal/templates ./...
go run ./scripts/generate_templates.go
```

The generator script will flag duplicate IDs, invalid variables, and sparse categories.

## Adding New Archetypes

1. Add the archetype in `internal/entities/types.go`.
2. Wire it into the generator in `internal/entities/generator.go`.
3. Add archetype-specific templates where needed.
4. Add or update tests so the archetype actually changes behavior.

New archetypes should feel different in personality, not just in name.

## Reporting Bugs

If a Poble acts out of character, report it with:

- the exact scene or system
- the seed, if you have it
- what you expected
- what actually happened
- the shortest steps to reproduce

Use a subject like:

`Noah acted out of character`

That helps us find the right behavior fast.
