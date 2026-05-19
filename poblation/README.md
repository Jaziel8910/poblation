# POBLATION

![POBLATION](assets/branding/poblation-wordmark.png)

**Beta v1.0.0** is the first public, playable beta of POBLATION: a terminal life sim about the last humans, their secrets, their bad choices, and the tiny civilization they may or may not ruin.

> Adult game for players 18+. Dark comedy, explicit systems, death, obsession, betrayal, grief, messy relationships, and procedural drama.

## Download

The easiest Windows path is the launcher:

1. Open the latest GitHub Release.
2. Download `poblation_v1.0.0-beta.1_launcher_installer.exe`.
3. Run it.
4. Open `POBLATION Launcher.cmd` from your Desktop or from `C:\Users\<you>\.poblation\launcher`.
5. Choose `Install/update latest release`, then `Play`.

Manual players can download `poblation_windows_amd64.exe`. The beta exe is portable: it carries the text templates it needs and prepares its local runtime under `~/.poblation`.

## Why It Exists

POBLATION starts with almost nothing: a few Pobles, an island, private needs, public consequences, and a simulation that keeps asking one rude question: what does a society become when everyone remembers?

Pobles have moods, needs, relationships, memories, secrets, letters, dreams, diary fragments, health, family pressure, power struggles, and grudges that can come back later. The game does not use runtime LLMs; everything is procedural code plus curated text templates.

## What Is In This Beta

- Procedural Pobles with correlated personality and archetype behavior.
- Memory-weighted decisions: old events can affect jealousy, arguments, distance, reconciliation, and intent.
- Settlement view with family, economy, institutions, technology, government pressure, and long consequences.
- Event feed, mind view, dialogue, house view, newspaper/export surfaces, endings, and debug console.
- Sex/fight/social systems with privacy rules and consequence hooks.
- Save/load under `~/.poblation`.
- Lightweight launcher with install, play, list, saves, news, folders, settings, and doctor commands.
- `1,814` curated templates after deleting generated filler packs.

## UI Snapshot

```text
+----------------------+  +------------------+  +----------------------+
| MAPA                 |  | MENTE            |  | FEED                 |
| island, pobles, map  |  | mood, traumas    |  | events, console      |
| selected Poble       |  | relationships    |  | event icons          |
+----------------------+  +------------------+  +----------------------+
```

## Install From Source

### Go

```bash
go install github.com/user/poblation@latest
```

For a local checkout:

```bash
go install .
```

### Homebrew

There is no official Homebrew formula yet. If one ships later, it will live in the official release notes or repo docs.

## Launcher Installer

Windows players should use:

```text
poblation_v1.0.0-beta.1_launcher_installer.exe
```

The installer places the launcher under `~/.poblation/launcher/bin/`, creates launcher settings, and the launcher downloads playable versions from GitHub Releases.

## Binary Releases

Download the latest release from GitHub Releases and run the binary directly.

## Controls

| Key | Action |
| --- | --- |
| `M` | Open mind view |
| `W` | Open world exploration |
| `P` | Open pobles list |
| `E` | Open events feed |
| `S` | Open settlement |
| `SPACE` | Pause / resume |
| `+` / `=` | Speed up |
| `-` | Slow down |
| `ENTER` | Select / confirm |
| `ESC` | Back / close current screen |
| `` ` `` | Debug console |
| `Q` | Quit |
| `TAB` | Cycle local selection in compact views |
| `SHIFT+TAB` | Reverse cycle in compact views |
| `ARROWS` | Move inside map, feed, and list views |
| `R` | Refresh newspaper / journal views |
| `X` | Exit private encounter views |
| `D` | De-escalate a fight when the view allows it |

## Your First Poble

1. Start the game.
2. Pick `New Civilization`.
3. Fill the form with a name, sex, archetype, and history seed.
4. Confirm the character.
5. The world starts around that first Poble and the simulation takes over from there.

## Systems

### Mind

The mind view shows the selected Poble at a glance: mood, stability, therapy level, traumas, and the pressure coming from relationships. It is meant for quick emotional reading, not deep menus.

### Dialogues

Dialogue views reveal conversation line by line. The tone changes with context, relationship, and the current dramatic beat. Some talks stay messy on purpose.

### Exploration

Exploration is the map and the feed. You move around the island, inspect pobles, open houses, and watch the event feed while the console lets you poke the simulation.

## Content Warning

POBLATION is an adult game for players 18+. It includes explicit sexual content, dark humor, harsh emotional situations, death, betrayal, violence, grief, obsession, and other extreme themes. It is not meant for minors.

## Credits

Built with the Charmbracelet stack:

- `bubbletea`
- `bubbles`
- `lipgloss`
- `huh`
- `glamour`

## License

Source-available under the POBLATION Game-Source License (PGSL) 1.0.
