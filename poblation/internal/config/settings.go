package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
)

const (
	// GameVersion is the current desktop build string shown in UI surfaces.
	GameVersion = "1.0.0.2"
)

// GameMode controls which views and powers the player can access.
type GameMode string

const (
	GameModeObserver   GameMode = "observer"
	GameModeDirector   GameMode = "director"
	GameModeGod        GameMode = "god"
	GameModeJournalist GameMode = "journalist"
	GameModeKnown      GameMode = "known"
)

// ParseGameMode validates a user-facing mode string.
func ParseGameMode(value string) (GameMode, bool) {
	switch GameMode(value) {
	case GameModeObserver, GameModeDirector, GameModeGod, GameModeJournalist, GameModeKnown:
		return GameMode(value), true
	default:
		return "", false
	}
}

// Profile stores player-facing persistent preferences and unlocks.
type Profile struct {
	Version         string             `json:"version"`
	Settings        Settings           `json:"settings"`
	AdultContent    AdultContentConfig `json:"adult_content"`
	UnlockedEndings []string           `json:"unlocked_endings"`
}

// Settings groups every adjustable player option.
type Settings struct {
	Graphics      GraphicsSettings      `json:"graphics"`
	Audio         AudioSettings         `json:"audio"`
	Gameplay      GameplaySettings      `json:"gameplay"`
	Content       ContentSettings       `json:"content"`
	Interface     InterfaceSettings     `json:"interface"`
	Accessibility AccessibilitySettings `json:"accessibility"`
	Controls      ControlsSettings      `json:"controls"`
}

// GraphicsSettings controls visual presentation.
type GraphicsSettings struct {
	ColorMode     string `json:"color_mode"`
	UIScale       int    `json:"ui_scale"`
	ReduceMotion  bool   `json:"reduce_motion"`
	ShowWeatherFX bool   `json:"show_weather_fx"`
	VSync         bool   `json:"vsync"`
	TargetFPS     int    `json:"target_fps"`
}

// AudioSettings stores audio mix values for future sound support.
type AudioSettings struct {
	MuteAll        bool `json:"mute_all"`
	MasterVolume   int  `json:"master_volume"`
	MusicVolume    int  `json:"music_volume"`
	AmbienceVolume int  `json:"ambience_volume"`
	UIVolume       int  `json:"ui_volume"`
}

// GameplaySettings stores pacing and safety preferences.
type GameplaySettings struct {
	AutoSaveEnabled        bool     `json:"autosave_enabled"`
	AutoSaveMinutes        int      `json:"autosave_minutes"`
	PauseOnFocusLoss       bool     `json:"pause_on_focus_loss"`
	DefaultSimulationSpeed float64  `json:"default_simulation_speed"`
	TutorialHints          bool     `json:"tutorial_hints"`
	ConfirmDangerousActs   bool     `json:"confirm_dangerous_acts"`
	DramaDensity           string   `json:"drama_density"`
	ActiveMode             GameMode `json:"active_mode"`
}

// ContentSettings stores adult and hard-theme toggles.
type ContentSettings struct {
	AdultContentEnabled bool `json:"adult_content_enabled"`
	ExplicitLanguage    bool `json:"explicit_language"`
	GraphicViolence     bool `json:"graphic_violence"`
	IncestContent       bool `json:"incest_content"`
	BodyHorror          bool `json:"body_horror"`
	PregnancyDetail     bool `json:"pregnancy_detail"`
}

// InterfaceSettings controls presentation density and HUD noise.
type InterfaceSettings struct {
	ShowTimestamps     bool `json:"show_timestamps"`
	CompactFeed        bool `json:"compact_feed"`
	ShowPobleBadges    bool `json:"show_poble_badges"`
	PersistentNotices  bool `json:"persistent_notices"`
	MapLabels          bool `json:"map_labels"`
	ShowSaveThumbnails bool `json:"show_save_thumbnails"`
}

// AccessibilitySettings stores alternate presentation options.
type AccessibilitySettings struct {
	HighContrast      bool `json:"high_contrast"`
	ScreenReaderMode  bool `json:"screen_reader_mode"`
	ReduceFlashing    bool `json:"reduce_flashing"`
	DyslexiaFriendly  bool `json:"dyslexia_friendly"`
	BiggerLineSpacing bool `json:"bigger_line_spacing"`
	HoldToConfirm     bool `json:"hold_to_confirm"`
}

// ControlsSettings stores input preferences.
type ControlsSettings struct {
	VimNavigation bool `json:"vim_navigation"`
	WASDMapPan    bool `json:"wasd_map_pan"`
	InvertScroll  bool `json:"invert_scroll"`
	ConfirmOnQuit bool `json:"confirm_on_quit"`
	QuickPause    bool `json:"quick_pause"`
}

// DefaultProfile returns the shipped baseline profile.
func DefaultProfile() Profile {
	return Profile{
		Version: GameVersion,
		Settings: Settings{
			Graphics: GraphicsSettings{
				ColorMode:     "theme",
				UIScale:       100,
				ReduceMotion:  false,
				ShowWeatherFX: true,
				VSync:         true,
				TargetFPS:     60,
			},
			Audio: AudioSettings{
				MuteAll:        false,
				MasterVolume:   80,
				MusicVolume:    70,
				AmbienceVolume: 75,
				UIVolume:       70,
			},
			Gameplay: GameplaySettings{
				AutoSaveEnabled:        true,
				AutoSaveMinutes:        10,
				PauseOnFocusLoss:       true,
				DefaultSimulationSpeed: 1.0,
				TutorialHints:          true,
				ConfirmDangerousActs:   true,
				DramaDensity:           "normal",
				ActiveMode:             GameModeDirector,
			},
			Content: ContentSettings{
				AdultContentEnabled: false,
				ExplicitLanguage:    false,
				GraphicViolence:     true,
				IncestContent:       false,
				BodyHorror:          true,
				PregnancyDetail:     true,
			},
			Interface: InterfaceSettings{
				ShowTimestamps:     true,
				CompactFeed:        false,
				ShowPobleBadges:    true,
				PersistentNotices:  false,
				MapLabels:          true,
				ShowSaveThumbnails: true,
			},
			Accessibility: AccessibilitySettings{
				HighContrast:      false,
				ScreenReaderMode:  false,
				ReduceFlashing:    true,
				DyslexiaFriendly:  false,
				BiggerLineSpacing: false,
				HoldToConfirm:     false,
			},
			Controls: ControlsSettings{
				VimNavigation: true,
				WASDMapPan:    false,
				InvertScroll:  false,
				ConfirmOnQuit: true,
				QuickPause:    true,
			},
		},
		AdultContent: AdultContentConfig{
			Enabled:   false,
			Confirmed: false,
		},
		UnlockedEndings: []string{},
	}
}

// Load reads the profile from disk.
func Load() (*Profile, error) {
	data, err := os.ReadFile(profilePath())
	if err != nil {
		return nil, err
	}

	var profile Profile
	if err := json.Unmarshal(data, &profile); err != nil {
		return nil, fmt.Errorf("config.Load: %w", err)
	}
	normalized := normalizeProfile(profile)
	return &normalized, nil
}

// LoadOrDefault returns disk data when present or the defaults otherwise.
func LoadOrDefault() Profile {
	profile, err := Load()
	if err != nil {
		return DefaultProfile()
	}
	return *profile
}

// Save writes the profile back to disk.
func Save(profile Profile) error {
	normalized := normalizeProfile(profile)
	if err := os.MkdirAll(filepath.Dir(profilePath()), 0o755); err != nil {
		return fmt.Errorf("config.Save mkdir: %w", err)
	}

	data, err := json.MarshalIndent(normalized, "", "  ")
	if err != nil {
		return fmt.Errorf("config.Save marshal: %w", err)
	}
	if err := os.WriteFile(profilePath(), append(data, '\n'), 0o644); err != nil {
		return fmt.Errorf("config.Save write: %w", err)
	}
	return nil
}

// UpdateSettings loads, mutates, and saves settings atomically.
func UpdateSettings(mutator func(*Settings)) (Profile, error) {
	profile := LoadOrDefault()
	mutator(&profile.Settings)
	if err := Save(profile); err != nil {
		return Profile{}, err
	}
	return profile, nil
}

// UnlockEnding persists one newly earned ending id.
func UnlockEnding(endingID string) error {
	if endingID == "" {
		return nil
	}
	profile := LoadOrDefault()
	if !slices.Contains(profile.UnlockedEndings, endingID) {
		profile.UnlockedEndings = append(profile.UnlockedEndings, endingID)
	}
	return Save(profile)
}

func normalizeProfile(profile Profile) Profile {
	defaults := DefaultProfile()

	if profile.Version == "" {
		profile.Version = defaults.Version
	}
	if profile.Settings.Graphics.ColorMode == "" {
		profile.Settings.Graphics.ColorMode = defaults.Settings.Graphics.ColorMode
	}
	if profile.Settings.Graphics.UIScale == 0 {
		profile.Settings.Graphics.UIScale = defaults.Settings.Graphics.UIScale
	}
	if profile.Settings.Graphics.TargetFPS == 0 {
		profile.Settings.Graphics.TargetFPS = defaults.Settings.Graphics.TargetFPS
	}
	if profile.Settings.Audio.MasterVolume < 0 {
		profile.Settings.Audio.MasterVolume = defaults.Settings.Audio.MasterVolume
	}
	if profile.Settings.Audio.MusicVolume < 0 {
		profile.Settings.Audio.MusicVolume = defaults.Settings.Audio.MusicVolume
	}
	if profile.Settings.Audio.AmbienceVolume < 0 {
		profile.Settings.Audio.AmbienceVolume = defaults.Settings.Audio.AmbienceVolume
	}
	if profile.Settings.Audio.UIVolume < 0 {
		profile.Settings.Audio.UIVolume = defaults.Settings.Audio.UIVolume
	}
	if profile.Settings.Gameplay.AutoSaveMinutes == 0 {
		profile.Settings.Gameplay.AutoSaveMinutes = defaults.Settings.Gameplay.AutoSaveMinutes
	}
	if profile.Settings.Gameplay.DefaultSimulationSpeed == 0 {
		profile.Settings.Gameplay.DefaultSimulationSpeed = defaults.Settings.Gameplay.DefaultSimulationSpeed
	}
	if profile.Settings.Gameplay.DramaDensity == "" {
		profile.Settings.Gameplay.DramaDensity = defaults.Settings.Gameplay.DramaDensity
	}
	if _, ok := ParseGameMode(string(profile.Settings.Gameplay.ActiveMode)); !ok {
		profile.Settings.Gameplay.ActiveMode = defaults.Settings.Gameplay.ActiveMode
	}
	if profile.UnlockedEndings == nil {
		profile.UnlockedEndings = []string{}
	}
	if profile.AdultContent.Enabled && profile.AdultContent.Confirmed {
		profile.Settings.Content.AdultContentEnabled = true
	}
	return profile
}

func configRoot() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".poblation"
	}
	return filepath.Join(home, ".poblation")
}

func profilePath() string {
	return filepath.Join(configRoot(), "config", "settings.json")
}
