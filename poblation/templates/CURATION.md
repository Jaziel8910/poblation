# POBLATION Template Curation

Goal: keep only templates that can make Pobles feel like people with memory, pressure, desire, shame, family, power, and contradiction.

## Removed

These packs were deleted because they inflated the count with repeated generated molds, numeric suffixes, and weak variation:

- `templates/dialogues/generated/`
- `templates/diary/generated/`
- `templates/dreams/generated/`
- `templates/letters/generated/`
- `templates/narrator/generated/`
- `templates/newspaper/`
- `templates/reactions/generated/`
- `templates/rumours/`
- `templates/thoughts/generated/`

Decision: removed.

Reason: valid format is not enough. These packs passed the loader, but they did not pass human reading.

## Kept

These curated packs stay in the game:

- `templates/thoughts/about_other/`
- `templates/thoughts/about_self/`
- `templates/thoughts/by_archetype/`
- `templates/thoughts/morning/`
- `templates/thoughts/night/`
- `templates/thoughts/random/`
- `templates/dialogues/argument/`
- `templates/dialogues/confession/`
- `templates/dialogues/encounter/`
- `templates/dialogues/flirt/`
- `templates/dialogues/gossip/`
- `templates/dialogues/greeting/`
- `templates/dialogues/political/`
- `templates/dialogues/reconciliation/`
- `templates/dialogues/religious/`
- `templates/dialogues/sex/`
- `templates/dialogues/threat/`
- `templates/dreams/erotic/`
- `templates/dreams/nightmare/`
- `templates/dreams/nonsense/`
- `templates/dreams/prophetic/`
- `templates/dreams/wish_fulfillment/`
- `templates/diary/daily/`
- `templates/diary/family/`
- `templates/diary/heartbreak/`
- `templates/diary/planning/`
- `templates/diary/secret_keeping/`
- `templates/letters/apology/`
- `templates/letters/hate/`
- `templates/letters/love/`
- `templates/narrator/birth/`
- `templates/narrator/death/`
- `templates/narrator/disaster/`
- `templates/narrator/end_of_game/`
- `templates/reactions/betrayal`
- `templates/reactions/birth`
- `templates/reactions/death_of_loved_one`
- `templates/reactions/discovery`
- `templates/reactions/illness/`
- `templates/reactions/violence/`
- `templates/reactions/war/`
- `templates/reactions/weather/`

Decision: kept.

Reason: these are smaller, category-specific, load cleanly, and fit the current engine without fake count inflation.

## 15k Rule

The 15k goal stays, but it must be reached by curated packs only.

A template counts toward the 15k goal only if:

- it has a unique emotional angle, not just a number or swapped tag;
- it uses valid variables in a way that reacts to the world;
- it can be read aloud without sounding like filler;
- it avoids banned QUALITY.md phrases;
- it belongs to a concrete category the game actually selects;
- it gives Pobles memory, contradiction, or consequence.

Current curated count after cleanup and first hand-curated repair packs: 1,814.

Target curated count: 15,000.

Gap: 13,186.
