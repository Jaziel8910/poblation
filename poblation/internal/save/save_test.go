package save

import (
	"os"
	"testing"
	"time"

	"github.com/user/poblation/internal/entities"
	"github.com/user/poblation/internal/events"
	"github.com/user/poblation/internal/world"
)

func TestSaveLoadRoundTrip(t *testing.T) {
	withTempHome(t)

	gameWorld := sampleWorld()
	data := SaveData{
		Version:         currentSaveVersion,
		SaveDate:        time.Now(),
		WorldState:      gameWorld,
		AllPobles:       gameWorld.GetAllKnownPobles(),
		EventHistory:    []events.GameEvent{{ID: "event_1", Type: events.EventBirth, Timestamp: gameWorld.Calendar, Description: "A birth changed the settlement."}},
		CurrentTime:     entities.GameTime(gameWorld.Calendar),
		Seed:            77,
		PlaytimeMinutes: gameWorld.Calendar.ToMinutes(),
	}

	if err := Save(data, 1); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loaded, err := Load(1)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if loaded.WorldState == nil || loaded.WorldState.GetPopulation() != gameWorld.GetPopulation() {
		t.Fatalf("expected world population preserved, got %+v", loaded.WorldState)
	}
	if len(loaded.AllPobles) == 0 {
		t.Fatal("expected all pobles restored")
	}
	if loaded.Metadata.Population != gameWorld.GetPopulation() {
		t.Fatalf("expected metadata population %d, got %d", gameWorld.GetPopulation(), loaded.Metadata.Population)
	}
}

func TestAutoSaveAndListSaves(t *testing.T) {
	withTempHome(t)

	gameWorld := sampleWorld()
	gameWorld.Calendar = gameWorld.Calendar.Add(10)
	if err := AutoSave(gameWorld); err != nil {
		t.Fatalf("AutoSave failed: %v", err)
	}

	entries := ListSaves()
	if len(entries) < maxPlayerSlots {
		t.Fatalf("expected at least %d entries, got %d", maxPlayerSlots, len(entries))
	}
	foundAutosave := false
	for _, entry := range entries {
		if len(entry.MostDramaticEvent) >= 10 && entry.MostDramaticEvent[:10] == "[AUTOSAVE]" {
			foundAutosave = true
			break
		}
	}
	if !foundAutosave {
		t.Fatal("expected autosave metadata to be listed")
	}
}

func TestExportNewspaperWritesFile(t *testing.T) {
	withTempHome(t)

	gameWorld := sampleWorld()
	content := ExportNewspaper(gameWorld)
	if content == "" {
		t.Fatal("expected newspaper content")
	}

	path := newspaperPath(gameWorld.Calendar.Day)
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected newspaper file at %s: %v", path, err)
	}
}

func withTempHome(t *testing.T) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
}

func sampleWorld() *world.World {
	gameWorld := world.NewWorld(77)
	poble := entities.NewPoble("p1", "Ami", 24, entities.Female)
	poble.IsAlive = true
	poble.Money = 15
	poble.Inventory = []entities.Item{{ID: "fish", Name: "Fish", Type: "food", Quantity: 2, Value: 3}}
	gameWorld.AddPoble(&poble, world.Location{IslandID: "island_0", BuildingID: "island_0_home_0"})
	gameWorld.Calendar = entities.NewGameTime(3, 12, 0)
	gameWorld.EventHistory = append(gameWorld.EventHistory, world.GameEvent{
		ID:          "world_event",
		Type:        "GOAL_COMPLETE",
		Time:        gameWorld.Calendar,
		Description: "The settlement finally named itself.",
		Severity:    0.8,
		Tags:        []string{"era_transition"},
	})
	return gameWorld
}
