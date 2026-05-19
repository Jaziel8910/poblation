//go:build installer

package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const (
	launcherFile = "poblation-launcher.exe"
	version      = "v1.0.0.2"
)

//go:embed payload/poblation-launcher.exe
var launcherEXE []byte

type installSettings struct {
	Repository        string `json:"repository"`
	DefaultVersion    string `json:"default_version"`
	InstallDir        string `json:"install_dir"`
	BackgroundProcess bool   `json:"background_process"`
	Notifications     bool   `json:"notifications"`
}

type dependencyManifest struct {
	Version           string   `json:"version"`
	Runtime           string   `json:"runtime"`
	GitHubReleasesAPI string   `json:"github_releases_api"`
	InstalledFolders  []string `json:"installed_folders"`
}

func main() {
	root, err := poblationRoot()
	if err != nil {
		fatal(err)
	}
	launcherDir := filepath.Join(root, "launcher", "bin")
	if err := os.MkdirAll(launcherDir, 0o755); err != nil {
		fatal(fmt.Errorf("crear carpeta del launcher: %w", err))
	}
	target := filepath.Join(launcherDir, launcherFile)
	if err := os.WriteFile(target, launcherEXE, 0o755); err != nil {
		fatal(fmt.Errorf("instalar launcher: %w", err))
	}
	if err := installRuntimeLayout(root); err != nil {
		fatal(err)
	}
	if err := writeDefaultSettings(root); err != nil {
		fatal(err)
	}
	if err := writeDependencyManifest(root); err != nil {
		fatal(err)
	}
	if err := writeLaunchScripts(root); err != nil {
		fatal(err)
	}
	fmt.Println("POBLATION Launcher instalado.")
	fmt.Println("Version:", version)
	fmt.Println("Ruta:", target)
	fmt.Println("Dependencias listas: runtime self-contained, carpetas, cache y GitHub releases API.")
	fmt.Println("Puedes abrirlo ejecutando ese .exe.")
}

func writeLaunchScripts(root string) error {
	launcherDir := filepath.Join(root, "launcher")
	launcherEXEPath := filepath.Join(launcherDir, "bin", launcherFile)
	openScript := filepath.Join(launcherDir, "OPEN POBLATION.bat")
	menuScript := filepath.Join(launcherDir, "POBLATION Launcher.cmd")
	playScript := filepath.Join(launcherDir, "Play POBLATION Beta.cmd")
	openBody := fmt.Sprintf("@echo off\r\n\"%s\"\r\npause\r\n", launcherEXEPath)
	menuBody := openBody
	playBody := "@echo off\r\n\"%~dp0bin\\poblation-launcher.exe\" play\r\npause\r\n"
	if err := os.WriteFile(openScript, []byte(openBody), 0o755); err != nil {
		return fmt.Errorf("crear acceso principal: %w", err)
	}
	if err := os.WriteFile(menuScript, []byte(menuBody), 0o755); err != nil {
		return fmt.Errorf("crear acceso del launcher: %w", err)
	}
	if err := os.WriteFile(playScript, []byte(playBody), 0o755); err != nil {
		return fmt.Errorf("crear acceso para jugar: %w", err)
	}
	if desktop, err := os.UserHomeDir(); err == nil {
		desktopDir := filepath.Join(desktop, "Desktop")
		_ = os.WriteFile(filepath.Join(desktopDir, "OPEN POBLATION.bat"), []byte(openBody), 0o755)
		_ = os.WriteFile(filepath.Join(desktopDir, "POBLATION Launcher.cmd"), []byte(openBody), 0o755)
	}
	return nil
}

func poblationRoot() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("leer home del usuario: %w", err)
	}
	return filepath.Join(home, ".poblation"), nil
}

func writeDefaultSettings(root string) error {
	path := filepath.Join(root, "launcher", "settings.json")
	settings := installSettings{
		Repository:        "Jaziel8910/poblation",
		DefaultVersion:    version,
		InstallDir:        filepath.Join(root, "versions"),
		BackgroundProcess: false,
		Notifications:     true,
	}
	if existing, err := os.ReadFile(path); err == nil {
		var current installSettings
		if json.Unmarshal(trimBOM(existing), &current) == nil {
			settings = current
			if settings.Repository == "" {
				settings.Repository = "Jaziel8910/poblation"
			}
			if settings.InstallDir == "" {
				settings.InstallDir = filepath.Join(root, "versions")
			}
			settings.DefaultVersion = version
		}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("crear carpeta de ajustes: %w", err)
	}
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("preparar ajustes: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("guardar ajustes: %w", err)
	}
	return nil
}

func trimBOM(data []byte) []byte {
	if len(data) >= 3 && data[0] == 0xEF && data[1] == 0xBB && data[2] == 0xBF {
		return data[3:]
	}
	return data
}

func installRuntimeLayout(root string) error {
	for _, path := range []string{
		filepath.Join(root, "versions"),
		filepath.Join(root, "saves"),
		filepath.Join(root, "cache"),
		filepath.Join(root, "launcher", "logs"),
	} {
		if err := os.MkdirAll(path, 0o755); err != nil {
			return fmt.Errorf("crear carpeta runtime %s: %w", path, err)
		}
	}
	return nil
}

func writeDependencyManifest(root string) error {
	path := filepath.Join(root, "launcher", "dependencies.json")
	manifest := dependencyManifest{
		Version:           version,
		Runtime:           "self-contained Windows launcher binary",
		GitHubReleasesAPI: "https://api.github.com/repos/Jaziel8910/poblation/releases",
		InstalledFolders: []string{
			filepath.Join(root, "versions"),
			filepath.Join(root, "saves"),
			filepath.Join(root, "cache"),
			filepath.Join(root, "launcher", "logs"),
		},
	}
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("preparar manifiesto de dependencias: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("guardar manifiesto de dependencias: %w", err)
	}
	return nil
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "Installer error:", err)
	os.Exit(1)
}
