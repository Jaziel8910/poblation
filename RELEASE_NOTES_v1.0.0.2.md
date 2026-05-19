# POBLATION v1.0.0.2

This is the first public build that is meant to be installed and played without repo files beside it.

## Download

Recommended for Windows:

1. Download `POBLATION_v1.0.0.2_WINDOWS_READY.zip`.
2. Unzip it.
3. Double-click `OPEN POBLATION.bat`.
4. Read `TUTORIAL.txt` if you are not used to terminal games.

Inside the zip there is also a direct game exe for technical players:

- `poblation_windows_amd64.exe`

The game executable is portable. It embeds the template content it needs and prepares runtime files under `~/.poblation`.

## Fixed Since The Beta

- Fixed the GitHub game exe failing when `templates/` was not beside it.
- Replaced the heavy GUI launcher with a lightweight terminal launcher.
- Fixed the Desktop launcher shortcut so it points to the installed launcher.
- Added one-click `OPEN POBLATION.bat`.
- Added `TUTORIAL.txt` for non-technical players.
- Added one ready-to-play zip so players do not have to guess which asset to install.
- Added a root README so GitHub shows the project properly.
- Added issue templates for bugs and beta feedback.
- Added launcher smoke testing through `play -smoke`.

## Launcher Commands

```bash
poblation-launcher install v1.0.0.2
poblation-launcher play
poblation-launcher play -smoke
poblation-launcher list
poblation-launcher saves
poblation-launcher news
poblation-launcher doctor
```

## Verified Locally Before Release

- `go test ./...` for the game.
- `go build ./...` for the game.
- `go test ./...` for the launcher.
- `go build ./...` for the launcher.
- Desktop `.cmd` opens the launcher menu.
- Launcher `doctor` sees installed versions.
- Launcher `play -smoke` starts the selected game and exits cleanly.
- Real game process stays running after startup instead of closing immediately.
