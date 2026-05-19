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
	loadedPoble := loaded.WorldState.GetPoble("p1")
	if loadedPoble == nil || len(loadedPoble.Memories) != 1 || loadedPoble.Memories[0].ID != "memory_storm" {
		t.Fatalf("expected poble memory to survive save/load, got %+v", loadedPoble)
	}
	if len(loadedPoble.Thoughts) != 1 || loadedPoble.Thoughts[0].ID != "thought_storm" {
		t.Fatalf("expected poble thoughts to survive save/load, got %+v", loadedPoble.Thoughts)
	}
	if len(loadedPoble.Dreams) != 1 || loadedPoble.Dreams[0].ID != "dream_storm" {
		t.Fatalf("expected poble dreams to survive save/load, got %+v", loadedPoble.Dreams)
	}
	if len(loadedPoble.DiaryEntries) != 1 || loadedPoble.DiaryEntries[0].ID != "diary_storm" {
		t.Fatalf("expected poble diary entries to survive save/load, got %+v", loadedPoble.DiaryEntries)
	}
	if len(loadedPoble.Letters) != 1 || loadedPoble.Letters[0].ID != "letter_storm" {
		t.Fatalf("expected poble letters to survive save/load, got %+v", loadedPoble.Letters)
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
	memory := entities.NewMemory("memory_storm", entities.NewGameTime(2, 20, 0), entities.MemoryNegative, "Ami remembered the first bad storm.")
	memory.Participants = []string{"p1"}
	memory.EmotionIntensity = 72
	memory.Tags = []string{"storm", "weather", "unresolved"}
	poble.Memories = []entities.Memory{memory}
	poble.Thoughts = []entities.Thought{{
		ID:        "thought_storm",
		Timestamp: entities.NewGameTime(2, 21, 0),
		Text:      "The storm still sounds wrong in Ami's head.",
		Tags:      []string{"thought", "weather"},
	}}
	poble.Dreams = []entities.Dream{{
		ID:        "dream_storm",
		Timestamp: entities.NewGameTime(2, 23, 0),
		Text:      "Ami dreamed the settlement had no roof.",
		Category:  "dreams/nightmare/general",
		IsPrivate: true,
		Tags:      []string{"dream", "storm"},
	}}
	poble.DiaryEntries = []entities.DiaryEntry{{
		ID:        "diary_storm",
		Timestamp: entities.NewGameTime(3, 6, 0),
		Text:      "I wrote it down because everyone else kept pretending.",
		Mood:      entities.MoodAnxious,
		Tags:      []string{"diary", "storm"},
	}}
	poble.Letters = []entities.Letter{{
		ID:        "letter_storm",
		Timestamp: entities.NewGameTime(3, 7, 0),
		ToID:      "p2",
		Text:      "I should have told you about the storm.",
		IsSent:    false,
		Tags:      []string{"letter", "unsent"},
	}}
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
