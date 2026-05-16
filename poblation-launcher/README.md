# POBLATION Launcher

![POBLATION](assets/branding/poblation-wordmark.png)

Version `1.0.0.1` - THE LAUNCHER UPDATE.

Native Go + Fyne launcher for POBLATION. It downloads public GitHub releases, keeps local versions in `~/.poblation/versions`, shows save previews, and can launch offline using the last installed build.

## Features

- Minecraft-style launcher layout with game art, save summary, news, version selector, and play button.
- GitHub Releases API support without auth for public repos.
- Download progress, SHA256 verification, and cleanup that keeps the latest 3 local versions.
- Offline launch from the newest downloaded build.
- Save preview from `~/.poblation/saves` without opening the game.
- Clock anomaly handling and purely aesthetic anti-piracy sequence.
- Settings for repo, install directory, default version, background mode, notifications, and cache cleanup.

## Local Run

```bash
go run ./cmd/poblation-launcher
```

## Build

```bash
make launcher-release
```

The Windows installer is written to:

```text
dist/poblation_1.0.0.1_launcher_installer.exe
```
