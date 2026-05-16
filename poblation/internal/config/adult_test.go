package config

import "testing"

func TestAdultContentDefaultsToRestricted(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	if IsAdultContentEnabled() {
		t.Fatalf("adult content should default to disabled")
	}
	if level := GetContentLevel(); level != ContentRestricted {
		t.Fatalf("expected restricted content level, got %s", level)
	}
}

func TestEnableAdultContentPersistsFullMode(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	if err := EnableAdultContent(); err != nil {
		t.Fatalf("enable adult content: %v", err)
	}

	profile, err := Load()
	if err != nil {
		t.Fatalf("load profile after enable: %v", err)
	}
	if !profile.AdultContent.Enabled || !profile.AdultContent.Confirmed || !profile.Settings.Content.AdultContentEnabled {
		t.Fatalf("adult content flags were not fully persisted: %+v", profile.AdultContent)
	}
	if profile.AdultContent.EnabledAt.IsZero() {
		t.Fatalf("enabled timestamp was not stored")
	}
	if level := GetContentLevel(); level != ContentFull {
		t.Fatalf("expected full content level, got %s", level)
	}
}
