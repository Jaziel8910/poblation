# Release Notes

`make launcher-release` builds the Windows launcher and installer from this checkout.

Full release targets expected later:

- Windows: `poblation_v1.0.0.2_launcher_installer.exe`
- macOS: `.app` bundle, signed/notarized outside this repo
- Linux: AppImage and `.deb`

Fyne can cross-compile app binaries, but native packaging for macOS/Linux needs host-specific tools. This repo keeps one command for the local Windows release and documents the remaining packaging boundary honestly instead of faking artifacts.
