# POBLATION

![POBLATION](assets/branding/poblation-wordmark.png)

Version `1.0.0.1` - THE LAUNCHER UPDATE.

Drama de terminal, pobles raros, secretos privados y una civilizacion que se arma con decisiones feas.

## UI Snapshot

```text
+----------------------+  +------------------+  +----------------------+
| MAPA                 |  | MENTE            |  | FEED                 |
| island, pobles, map  |  | mood, traumas    |  | events, console      |
| selected Poble       |  | relationships    |  | event icons          |
+----------------------+  +------------------+  +----------------------+
```

## Install

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

### Launcher installer

Windows players should use:

```text
poblation_1.0.0.1_launcher_installer.exe
```

The installer places the launcher under `~/.poblation/launcher/bin/`, creates launcher settings, and the launcher downloads playable versions from GitHub Releases.

### Binary releases

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
