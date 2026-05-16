━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
SECTION 1: THE SLOP DEFINITION
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Title: ## What Slop Looks Like In This Project

Write this as a specific, ruthless list of what slop means 
for POBLATION specifically. Not generic advice — concrete 
patterns that would appear in THIS codebase if quality fails.

Include all of these and explain each one in 2-3 lines:

CODE SLOP:
- A Poble generator that produces stats independently at random
  with no correlation between traits (the single most important 
  slop signal in this project)
- Template selection that is random.Choose() with extra steps
- Decision engine where RULER and INNOCENT behave identically
  under the same conditions
- Functions over 80 lines that do more than one thing
- Any use of interface{} or any instead of concrete types
- TODO comments that don't specify what exactly needs doing
- Error handling that discards context: if err != nil { return err }
  without wrapping the error with additional information
- Duplicate logic across files instead of shared functions
- Goroutines without context.Context parameter
- Tests that test that 1 == 1 or that a struct initializes without panic

NARRATIVE/TEMPLATE SLOP:
- Thoughts that could belong to any character in any life sim game
- Dialogue that has a narrator explaining what's happening
- Dreams that follow a clear narrative logic (dreams are illogical)
- Any template using these exact phrases or close variants:
  "felt a wave of", "couldn't help but", "deep down", 
  "a part of me", "something inside", "eyes filled with",
  "heart racing", "took a deep breath", "couldn't shake the feeling"
- Reconciliation dialogues that resolve cleanly and completely
- Arguments where both parties are clearly wrong in equal measure
  (real arguments are asymmetric and messy)
- Thoughts that are too self-aware for the character's archetype
  (a GHOST doesn't think "I am emotionally unavailable" —
   they think about nothing and notice the nothing)

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
SECTION 2: THE SELF-AUDIT PROTOCOL
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Title: ## Before You Output Anything

Write this as a checklist that the agent runs on its own output
before considering it done. Frame it as internal questions the
agent asks itself. Make it unambiguous — every question has a
yes/no answer, and "no" means rewrite.

Include these exact checks, written as direct questions:

ARCHITECTURE CHECKS:
□ Does every new type I defined already exist in types.go?
  If yes → delete mine, use the existing one.
□ Does any game logic file import from internal/ui/?
  If yes → restructure immediately, this is a hard violation.
□ Does any system directly reference another system without
  going through the Orchestrator?
  If yes → this creates hidden coupling that will break at prompt 20+.
□ Does every goroutine I created accept context.Context?
  If no → add it. No exceptions.
□ Does every shared map have sync.RWMutex protecting it?
  If no → add it. Race conditions are invisible until they aren't.

CODE QUALITY CHECKS:
□ Is any function longer than 80 lines?
  If yes → it does more than one thing. Split it.
□ Does any function have more than 4 parameters?
  If yes → group related params into a struct.
□ Did I duplicate logic that already exists elsewhere in the codebase?
  If yes → extract to shared function. AI generates duplication by default.
  Actively resist this tendency.
□ Are there any TODO comments in my output?
  If yes → either implement it now or delete the comment and document
  the gap explicitly in the output message to the user. 
  Never leave silent TODOs.
□ Does my error handling add context at every layer?
  fmt.Errorf("generator.GeneratePople: %w", err) not just return err.

GAME DESIGN CHECKS:
□ If I generated a Poble with Archetype:VILLAIN, is Agreeableness < 50
  at least 90% of the time statistically?
  If not → the correlation system has no effect.
□ If I generated 10 thoughts for the same Poble in the same mood,
  would they be thematically consistent without being identical?
  If not → template selection is broken.
□ If I placed RULER and INNOCENT in identical high-stress situations,
  do they choose different actions?
  If not → the decision engine ignores archetype.
□ Does the template selector penalize recently-used templates?
  If not → players will see the same thought twice in a row.
□ Does any user-facing string exist hardcoded in a .go file?
  If yes → move it to templates/. Always.

NARRATIVE QUALITY CHECKS (for template generation only):
□ Could this template appear unchanged in The Sims, Stardew Valley,
  or any other life sim without anyone noticing?
  If yes → it's not specific enough. Rewrite it.
□ Does this template use any phrase from the banned list in Section 1?
  If yes → cut it. Find the specific, concrete version of what 
  the character is actually experiencing.
□ Does this dialogue have a narrator voice explaining the subtext?
  ("She said it casually but meant it deeply")
  If yes → remove the narrator. Show through word choice alone.
□ Does this dream follow cause-and-effect logic?
  If yes → it's not a dream. Make it weirder.
□ Does this argument resolve?
  If yes → add one line that reopens it, or cut the resolution entirely.
  Arguments in POBLATION end, they don't resolve.

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
SECTION 3: THE DEGRADATION SIGNALS
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Title: ## How To Know The Codebase Is Degrading

Write this section based on real research findings about how
AI-generated code degrades over iterative sessions. This is
information for the agent to recognize its own failure modes.

Include these degradation patterns, each with a detection method:

VERBOSITY CREEP
What it is: Functions grow longer with each edit. Helper functions
multiply without consolidating. The same logic appears in 3 places
slightly differently because it was added in 3 different sessions.
How to detect: If a function is longer than when you last touched it
and the feature didn't require the extra lines — trim it.
Rule: When extending existing code, leave it shorter or equal length
to what you found. Never longer unless the feature demands it.

INTERFACE EROSION  
What it is: Clean interfaces between systems get bypassed because
it's faster to add a direct reference than to use the established path.
The Orchestrator gets circumvented. Types get duplicated.
How to detect: Any file that imports more packages than it did in
the previous prompt is a warning sign. Check if the new imports 
bypass the established architecture.
Rule: If adding a feature requires a new cross-system import, 
stop and find the architecturally correct path first.

CORRELATION COLLAPSE
What it is: Specific to POBLATION. The personality correlation system
degrades silently — each edit to the generator slightly loosens the
constraints until eventually stats are effectively random again.
This is the most critical degradation signal in this codebase.
How to detect: Generate 20 Pobles with Archetype:VILLAIN mentally.
If any of them could pass as INNOCENT based on stats alone — collapsed.
Rule: Every edit to the generator must preserve all existing correlations.
Correlations are not optional features. They are the game.

TEMPLATE HOMOGENIZATION
What it is: New templates start to sound like each other because
the agent defaults to its training distribution when generating text.
After enough sessions, all RULER thoughts sound the same. All 
JESTER thoughts hit the same beats. The variety collapses.
How to detect: Read 5 templates from the same category. If you can 
predict the shape of the 6th before reading it — homogenized.
Rule: Each new template must differ from existing ones in at least
one of: length, emotional register, resolution/non-resolution,
or which aspect of the situation it focuses on.

PHANTOM IMPLEMENTATION
What it is: Functions exist, compile, and have no TODO markers,
but do not actually implement the specified behavior. They return
zero values, empty slices, or hardcoded results.
How to detect: A function called SelectTemplate that always returns
templates[0] is phantom implementation. It compiles. It lies.
Rule: If you cannot fully implement something in this session,
output an explicit message: "INCOMPLETE: [function] returns stub.
Needs: [specific what is missing]." Never leave silent phantoms.

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
SECTION 4: THE THREE SYSTEMS THAT 
CANNOT BE FAKED
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Title: ## The Three Systems That Cannot Be Faked

Write this section as an explicit warning about the three systems
where phantom implementation is most tempting and most damaging.
For each one, describe what a real implementation does vs what
a fake implementation looks like.

SYSTEM 1: PERSONALITY CORRELATION (internal/entities/generator.go)

Real implementation:
- Has a correlation matrix or equivalent data structure
- VILLAIN archetype pulls Cruelty up, Agreeableness down,
  with weighted probability — not guaranteed, but statistically enforced
- Generating 100 Pobles with the same archetype produces a 
  recognizable statistical distribution, not uniform random noise
- Openness correlates with orientation Fluidity
- Neuroticism correlates with Jealousy
- These correlations compound: high Neuroticism AND high Jealousy
  AND LOVER archetype produces a recognizable character type

Fake implementation (DO NOT SHIP):
- rand.Float32() * 100 for every stat independently
- Archetype applied as a label only, not as a statistical pull
- "Correlation" implemented as a single += modifier with no weight
- Stats that hit their correlation targets for obvious cases
  but not for compound interactions

SYSTEM 2: TEMPLATE SELECTION (internal/templates/engine.go)

Real implementation:
- Builds a scored candidate list, not a filtered random list
- Context signals (archetype, mood, recent memory tags, relationship
  valence) each contribute to candidate scores
- Anti-repetition penalizes templates used in the last N selections
  for this Poble — the penalty decays over time
- The same Poble in the same mood across 20 ticks produces
  a consistent thematic voice with visible variety
- Rare templates (weight: 2) appear roughly 1/5 as often as
  common templates (weight: 10) — measurable, not approximate

Fake implementation (DO NOT SHIP):
- Filter by archetype, then random.Choose() from filtered list
- Anti-repetition as a comment or TODO
- Weight field parsed but not used in selection
- Context that is loaded but not consulted during scoring

SYSTEM 3: DECISION ENGINE (internal/ai/decision.go)

Real implementation:
- RULER and INNOCENT produce different action priorities
  under identical need states. This is measurable.
- Archetype modifies the decision pipeline at multiple points:
  goal evaluation, target selection, AND approach style
- The 15% random action chance creates genuine surprise
  without overriding the archetype character
- A poble mid-grief chooses different actions than the same
  poble at baseline, regardless of archetype
- Compound states (grief + obsession + hunger) produce
  a priority resolution that reflects the character's nature

Fake implementation (DO NOT SHIP):
- switch archetype { case RULER: priority += 1 } and nothing else
- Emotional state read but not applied to decision weights
- The same top action chosen for all archetypes 70%+ of the time
- Random action that fires too often (>25%) effectively replacing
  archetype-driven behavior with noise

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
SECTION 5: THE OUTPUT CONTRACT
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Title: ## What Every Output Must Include

Write this as the final checklist — what the agent appends
to every output before it's considered complete.

Every time an agent outputs code for POBLATION, the output
must end with this block, filled in honestly:

---
QUALITY REPORT
--------------
Systems touched: [list every file modified]
Self-audit passed: [YES / NO — if NO, list what failed and why it 
                   was shipped anyway or what was done instead]
Phantom implementations: [NONE / list any stubs with explanation]
Correlations preserved: [YES / NOT APPLICABLE / list what changed]
Templates added: [N new templates / NOT APPLICABLE]
Template homogenization check: [PASSED / list any that sound too similar]
Degradation signals present: [NONE DETECTED / list any found]
go build ./... result: [PASSES / list what would fail]
go test ./... result: [PASSES / list what would fail]
Confidence level: [HIGH / MEDIUM / LOW — be honest]
If LOW: [what specifically is uncertain and why]
---

This block is not optional. An output without a QUALITY REPORT
is an incomplete output. The QUALITY REPORT is how the human
who cannot read the code knows whether to trust what was generated.

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
SECTION 6: THE ESCALATION RULE
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Title: ## When To Stop And Say So

Write this section about when the agent should refuse to generate
code and instead report the problem. This is the section that
prevents silent phantom implementations.

An agent working on POBLATION must stop and report — instead of
generating code — when any of these conditions are true:

STOP IF: The prompt requires implementing one of the Three Systems
That Cannot Be Faked, AND the correct implementation would require
more context about existing code than is currently available.
REPORT: "Cannot implement [system] without reading [specific files].
Please provide their current content before proceeding."

STOP IF: Implementing this prompt correctly requires changing a type
defined in types.go in a way that would break other systems.
REPORT: "This prompt requires changing [type] in types.go.
This affects: [list of systems]. Proceeding would require updating
all of them. Confirm this is intended before I continue."

STOP IF: The prompt asks for behavior that contradicts GDD.md.
REPORT: "This prompt asks for [X]. GDD.md specifies [Y] in section [N].
I will follow GDD.md unless you explicitly override it."

STOP IF: The agent has generated the same phantom implementation
for this function in a previous session.
REPORT: "This function [name] was previously implemented as a stub.
A real implementation requires [specific what]. I cannot generate
a real implementation without [specific missing context]."

STOP IF: Confidence in the output would be LOW for any system
in The Three Systems That Cannot Be Faked.
REPORT: "I cannot generate a high-confidence implementation of
[system] in this session. Here is what I can guarantee: [list].
Here is what needs verification: [list]."

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
OUTPUT INSTRUCTIONS
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Output QUALITY.md complete and ready to save.
Do not summarize it after. Do not explain what you wrote.
The file is the output.

The file must end with this line, verbatim:
"This file was written to be read by agents, not humans.
 If you are a human reading this: the QUALITY REPORT at the
 end of every agent output is the only signal you need.
 HIGH confidence + no phantom implementations = trust it.
 Everything else = ask the agent to redo it."