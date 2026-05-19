package launcher

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	AppVersion        = "1.0.0.2"
	BuildName         = "PLAYABLE RELEASE"
	MaxVersionsToKeep = 3
)

type Settings struct {
	Repository        string      `json:"repository"`
	UserName          string      `json:"user_name"`
	DefaultVersion    string      `json:"default_version"`
	InstallDir        string      `json:"install_dir"`
	BackgroundProcess bool        `json:"background_process"`
	Notifications     bool        `json:"notifications"`
	LastNews          []NewsItem  `json:"last_news"`
	Clock             ClockMemory `json:"clock"`
}

type ClockMemory struct {
	LastSeenUTC          time.Time `json:"last_seen_utc"`
	FirstAnomalyAccepted bool      `json:"first_anomaly_accepted"`
	IntentionalSignals   int       `json:"intentional_signals"`
	RAMEaterWarnings     int       `json:"ram_eater_warnings"`
}

func DefaultSettings() Settings {
	root := PoblationRoot()
	return Settings{
		Repository:        "Jaziel8910/poblation",
		UserName:          "Observer",
		DefaultVersion:    "latest",
		InstallDir:        filepath.Join(root, "versions"),
		BackgroundProcess: false,
		Notifications:     true,
		LastNews:          EmbeddedNews(),
	}
}

func LoadSettings() (Settings, error) {
	settings := DefaultSettings()
	path := SettingsPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return settings, nil
		}
		return settings, fmt.Errorf("open launcher settings: %w", err)
	}
	data = bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF})
	if err := json.Unmarshal(data, &settings); err != nil {
		return settings, fmt.Errorf("decode launcher settings: %w", err)
	}
	settings = normalizeSettings(settings)
	return settings, nil
}

func SaveSettings(settings Settings) error {
	settings = normalizeSettings(settings)
	path := SettingsPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create launcher settings directory: %w", err)
	}
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create launcher settings: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(settings); err != nil {
		return fmt.Errorf("encode launcher settings: %w", err)
	}
	return nil
}

func PoblationRoot() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".poblation"
	}
	return filepath.Join(home, ".poblation")
}

func LauncherRoot() string {
	return filepath.Join(PoblationRoot(), "launcher")
}

func SettingsPath() string {
	return filepath.Join(LauncherRoot(), "settings.json")
}

func SavesDir() string {
	return filepath.Join(PoblationRoot(), "saves")
}

func normalizeSettings(settings Settings) Settings {
	defaults := DefaultSettings()
	if settings.Repository == "" {
		settings.Repository = defaults.Repository
	}
	if settings.UserName == "" {
		settings.UserName = defaults.UserName
	}
	if settings.DefaultVersion == "" {
		settings.DefaultVersion = defaults.DefaultVersion
	}
	if settings.InstallDir == "" {
		settings.InstallDir = defaults.InstallDir
	}
	if len(settings.LastNews) == 0 {
		settings.LastNews = defaults.LastNews
	}
	return settings
}
