# POBLATION Beta v1.0.0

This is the first public beta meant to be played, tested, and broken by real people.

## Download

Recommended for Windows:

1. Download `poblation_v1.0.0-beta.1_launcher_installer.exe`.
2. Run it.
3. Open `POBLATION Launcher.cmd` from your Desktop or from `~/.poblation/launcher`.
4. Install the latest release from the launcher.
5. Press Play.

Manual download:

- `poblation_windows_amd64.exe`

The game executable is now portable. It includes its template content and prepares local runtime files under `~/.poblation`.

## What Is New

- Lightweight launcher replacing the old heavy GUI launcher.
- Launcher commands: install, play, list, saves, news, doctor, folders, settings.
- Portable Windows game exe that no longer dies because `templates/` is missing beside it.
- Social memory now affects more decisions, arguments, reconciliation, jealousy, and visible intent.
- Settlement UI shows government, institutions, economy, families, technology, and consequences.
- More endings and ending chapters.
- Template cleanup: generated filler packs removed; `1,814` curated templates validated.

## Honest Beta Notes

This is a beta, not the full 15k-template final GDD dream. The game is playable now, but the roadmap still includes more curated text, deeper generations, more UI surfacing, and longer civilization arcs.

## Verified

- `go test ./...` for the game.
- `go build ./...` for the game.
- `go test ./...` for the launcher.
- `go build ./...` for the launcher.
- Windows game smoke startup from the installed version folder.
