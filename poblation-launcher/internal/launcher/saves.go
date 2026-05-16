package launcher

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type SaveSummary struct {
	Slot        int
	Path        string
	WorldName   string
	Day         int
	Population  int
	LastPlayed  time.Time
	IsAutosave  bool
	Description string
}

type savePreview struct {
	SaveDate     time.Time    `json:"save_date"`
	WorldState   worldPreview `json:"world_state"`
	AllPobles    []pobleStub  `json:"all_pobles"`
	CurrentTime  gameTime     `json:"current_time"`
	Metadata     metadataStub `json:"metadata"`
}

type worldPreview struct {
	State    worldStatePreview `json:"state"`
	Calendar gameTime          `json:"calendar"`
	Pobles   []pobleStub       `json:"pobles"`
}

type worldStatePreview struct {
	Population  int              `json:"population"`
	Day         gameTime         `json:"day"`
	Settlements []settlementStub `json:"settlements"`
	Islands     []islandStub     `json:"islands"`
}

type metadataStub struct {
	Population        int       `json:"population"`
	Day               int       `json:"day"`
	MostDramaticEvent string    `json:"most_dramatic_event"`
	SavedAt           time.Time `json:"saved_at"`
	Slot              int       `json:"slot"`
	IsAutoSave        bool      `json:"is_autosave"`
}

type settlementStub struct {
	Name       string `json:"name"`
	Population int    `json:"population"`
}

type islandStub struct {
	Name string `json:"name"`
}

type pobleStub struct {
	ID string `json:"id"`
}

type gameTime struct {
	Day int `json:"day"`
}

func ListSaveSummaries() ([]SaveSummary, error) {
	entries, err := os.ReadDir(SavesDir())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read saves directory: %w", err)
	}
	summaries := make([]SaveSummary, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sav") {
			continue
		}
		path := filepath.Join(SavesDir(), entry.Name())
		summary, err := ReadSaveSummary(path)
		if err == nil {
			summaries = append(summaries, summary)
		}
	}
	sort.SliceStable(summaries, func(i, j int) bool {
		return summaries[i].LastPlayed.After(summaries[j].LastPlayed)
	})
	return summaries, nil
}

func ReadSaveSummary(path string) (SaveSummary, error) {
	file, err := os.Open(path)
	if err != nil {
		return SaveSummary{}, fmt.Errorf("open save: %w", err)
	}
	defer file.Close()
	reader, err := gzip.NewReader(file)
	if err != nil {
		return SaveSummary{}, fmt.Errorf("open gzip save: %w", err)
	}
	defer reader.Close()

	var preview savePreview
	if err := json.NewDecoder(reader).Decode(&preview); err != nil {
		return SaveSummary{}, fmt.Errorf("decode save preview: %w", err)
	}
	summary := SaveSummary{
		Slot:        slotFromPath(path, preview.Metadata.Slot),
		Path:        path,
		WorldName:   worldName(preview),
		Day:         saveDay(preview),
		Population:  savePopulation(preview),
		LastPlayed:  saveDate(preview),
		IsAutosave:  strings.Contains(strings.ToLower(filepath.Base(path)), "autosave"),
		Description: strings.TrimSpace(preview.Metadata.MostDramaticEvent),
	}
	if summary.Description == "" {
		summary.Description = "Todavia no hay drama registrado."
	}
	return summary, nil
}

func LatestSaveSummary() (SaveSummary, bool) {
	saves, err := ListSaveSummaries()
	if err != nil || len(saves) == 0 {
		return SaveSummary{}, false
	}
	return saves[0], true
}

func worldName(preview savePreview) string {
	for _, settlement := range preview.WorldState.State.Settlements {
		if strings.TrimSpace(settlement.Name) != "" {
			return settlement.Name
		}
	}
	for _, island := range preview.WorldState.State.Islands {
		if strings.TrimSpace(island.Name) != "" {
			return island.Name
		}
	}
	return "El Origen"
}

func saveDay(preview savePreview) int {
	if preview.Metadata.Day > 0 {
		return preview.Metadata.Day
	}
	if preview.WorldState.Calendar.Day > 0 {
		return preview.WorldState.Calendar.Day
	}
	if preview.WorldState.State.Day.Day > 0 {
		return preview.WorldState.State.Day.Day
	}
	return preview.CurrentTime.Day
}

func savePopulation(preview savePreview) int {
	if preview.Metadata.Population > 0 {
		return preview.Metadata.Population
	}
	if preview.WorldState.State.Population > 0 {
		return preview.WorldState.State.Population
	}
	if len(preview.WorldState.Pobles) > 0 {
		return len(preview.WorldState.Pobles)
	}
	return len(preview.AllPobles)
}

func saveDate(preview savePreview) time.Time {
	if !preview.Metadata.SavedAt.IsZero() {
		return preview.Metadata.SavedAt
	}
	return preview.SaveDate
}

func slotFromPath(path string, fallback int) int {
	if fallback > 0 {
		return fallback
	}
	base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	base = strings.TrimPrefix(strings.ToLower(base), "slot_")
	var slot int
	if _, err := fmt.Sscanf(base, "%d", &slot); err == nil {
		return slot
	}
	return 0
}
