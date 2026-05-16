package config

import "time"

// ContentLevel controls how encounter content is exposed.
type ContentLevel string

const (
	ContentRestricted ContentLevel = "RESTRICTED"
	ContentFull       ContentLevel = "FULL"
)

// AdultContentConfig stores one-time consent for explicit systems.
type AdultContentConfig struct {
	Enabled   bool      `json:"enabled"`
	EnabledAt time.Time `json:"enabled_at"`
	Confirmed bool      `json:"confirmed"`
}

// IsAdultContentEnabled returns true only when adult content is explicitly enabled.
func IsAdultContentEnabled() bool {
	profile := LoadOrDefault()
	return profile.Settings.Content.AdultContentEnabled &&
		profile.AdultContent.Enabled &&
		profile.AdultContent.Confirmed
}

// EnableAdultContent turns on full adult content and persists the consent.
func EnableAdultContent() error {
	profile := LoadOrDefault()
	profile.Settings.Content.AdultContentEnabled = true
	profile.AdultContent.Enabled = true
	profile.AdultContent.Confirmed = true
	if profile.AdultContent.EnabledAt.IsZero() {
		profile.AdultContent.EnabledAt = time.Now()
	}
	return Save(profile)
}

// GetContentLevel returns the current effective content gate.
func GetContentLevel() ContentLevel {
	if IsAdultContentEnabled() {
		return ContentFull
	}
	return ContentRestricted
}
