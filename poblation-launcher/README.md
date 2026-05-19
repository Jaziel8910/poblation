# POBLATION Launcher

![POBLATION](assets/branding/poblation-wordmark.png)

Version `v1.0.0-beta.1` - lightweight edition.

Small, compatible Go launcher for POBLATION. No Fyne, no CGO, no OpenGL build pain. It downloads public GitHub releases, keeps local versions in `~/.poblation/versions`, shows save previews, and can launch offline using the last installed build.

## Features

- Terminal menu plus direct commands.
- GitHub Releases API support without auth for public repos.
- Download progress, SHA256 verification, and cleanup that keeps the latest 3 local versions.
- Offline launch from the newest downloaded build.
- Save preview from `~/.poblation/saves` without opening the game.
- Settings for repo, install directory, default version, notifications, and local folders.
- Doctor command for checking the local install.

## Commands

```bash
poblation-launcher install v1.0.0-beta.1
poblation-launcher play
poblation-launcher list
poblation-launcher saves
poblation-launcher news
poblation-launcher doctor
poblation-launcher folders
```

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
dist/poblation_v1.0.0-beta.1_launcher_installer.exe
```
