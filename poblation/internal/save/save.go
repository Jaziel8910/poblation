package save

import (
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/user/poblation/internal/entities"
	"github.com/user/poblation/internal/events"
	"github.com/user/poblation/internal/world"
)

const (
	currentSaveVersion = "1.0.0.1"
	maxPlayerSlots     = 5
	autosaveInterval   = 10
)

var autosaveState struct {
	mu          sync.Mutex
	lastBucket  int
	initialized bool
}

// SaveData stores the complete serializable game state.
type SaveData struct {
	Version         string             `json:"version"`
	SaveDate        time.Time          `json:"save_date"`
	WorldState      *world.World       `json:"world_state"`
	AllPobles       []*entities.Poble  `json:"all_pobles"`
	EventHistory    []events.GameEvent `json:"event_history"`
	CurrentTime     entities.GameTime  `json:"current_time"`
	Seed            int64              `json:"seed"`
	PlaytimeMinutes int                `json:"playtime_minutes"`
	Metadata        SaveMetadata       `json:"metadata"`
}

// SaveMetadata stores lightweight slot preview information.
type SaveMetadata struct {
	Population        int       `json:"population"`
	Era               world.Era `json:"era"`
	Day               int       `json:"day"`
	MostDramaticEvent string    `json:"most_dramatic_event"`
	Screenshot        string    `json:"screenshot"`
	SavedAt           time.Time `json:"saved_at"`
	Slot              int       `json:"slot"`
	IsAutoSave        bool      `json:"is_autosave"`
	IsEmpty           bool      `json:"is_empty"`
}

// SaveSystem is the runtime facade used by the engine orchestrator.
type SaveSystem struct{}

// NewSaveSystem creates the save facade without adding engine dependencies.
func NewSaveSystem() *SaveSystem {
	return &SaveSystem{}
}

// Save writes one player slot.
func (s *SaveSystem) Save(data SaveData, slot int) error {
	return Save(data, slot)
}

// Load reads one player slot.
func (s *SaveSystem) Load(slot int) (*SaveData, error) {
	return Load(slot)
}

// LoadAutosave reads the autosave slot.
func (s *SaveSystem) LoadAutosave() (*SaveData, error) {
	return LoadAutosave()
}

// AutoSave writes the current world to autosave when due.
func (s *SaveSystem) AutoSave(gameWorld *world.World) error {
	return AutoSave(gameWorld)
}

// Save serializes, compresses, and stores a player save in one of five slots.
func Save(data SaveData, slot int) error {
	if slot < 1 || slot > maxPlayerSlots {
		return fmt.Errorf("save slot %d invalid: use 1-%d", slot, maxPlayerSlots)
	}
	if data.WorldState == nil {
		return errors.New("save requires a world state")
	}

	normalized := normalizeSaveData(data)
	return writeSaveFile(slotPath(slot), normalized)
}

// Load reads, decompresses, validates, and migrates a player save slot.
func Load(slot int) (*SaveData, error) {
	if slot < 1 || slot > maxPlayerSlots {
		return nil, fmt.Errorf("load slot %d invalid: use 1-%d", slot, maxPlayerSlots)
	}

	data, err := readSaveFile(slotPath(slot))
	if err != nil {
		return nil, err
	}
	if err := migrateSaveData(data); err != nil {
		return nil, err
	}
	return data, nil
}

// LoadAutosave reads the special autosave slot.
func LoadAutosave() (*SaveData, error) {
	data, err := readSaveFile(autosavePath())
	if err != nil {
		return nil, err
	}
	if err := migrateSaveData(data); err != nil {
		return nil, err
	}
	return data, nil
}

// AutoSave writes a special autosave every ten simulation ticks.
func AutoSave(gameWorld *world.World) error {
	if gameWorld == nil {
		return errors.New("autosave requires a world")
	}

	bucket := gameWorld.Calendar.ToMinutes() / (autosaveInterval * 60)
	autosaveState.mu.Lock()
	if autosaveState.initialized && autosaveState.lastBucket == bucket {
		autosaveState.mu.Unlock()
		return nil
	}
	autosaveState.lastBucket = bucket
	autosaveState.initialized = true
	autosaveState.mu.Unlock()

	data := normalizeSaveData(SaveData{
		WorldState:      gameWorld,
		AllPobles:       gameWorld.GetAllKnownPobles(),
		EventHistory:    adaptWorldHistory(gameWorld.EventHistory),
		CurrentTime:     entities.GameTime(gameWorld.Calendar),
		Seed:            int64(gameWorld.Calendar.ToMinutes()),
		PlaytimeMinutes: gameWorld.Calendar.ToMinutes(),
	})
	return writeSaveFile(autosavePath(), data)
}

// ListSaves returns metadata for player slots and autosave when present.
func ListSaves() []SaveMetadata {
	metadata := make([]SaveMetadata, 0, maxPlayerSlots+1)
	for slot := 1; slot <= maxPlayerSlots; slot++ {
		data, err := readSaveFile(slotPath(slot))
		if err != nil {
			metadata = append(metadata, SaveMetadata{
				Slot:              slot,
				IsEmpty:           true,
				MostDramaticEvent: fmt.Sprintf("EMPTY SLOT %d", slot),
				Screenshot:        emptyScreenshot(fmt.Sprintf("SLOT %d", slot)),
			})
			continue
		}
		entry := data.Metadata
		entry.Slot = slot
		entry.SavedAt = data.SaveDate
		metadata = append(metadata, entry)
	}

	if data, err := readSaveFile(autosavePath()); err == nil {
		autosaveMeta := data.Metadata
		autosaveMeta.Slot = 0
		autosaveMeta.SavedAt = data.SaveDate
		autosaveMeta.IsAutoSave = true
		autosaveMeta.MostDramaticEvent = "[AUTOSAVE] " + autosaveMeta.MostDramaticEvent
		metadata = append(metadata, autosaveMeta)
	} else {
		metadata = append(metadata, SaveMetadata{
			Slot:              0,
			IsAutoSave:        true,
			IsEmpty:           true,
			MostDramaticEvent: "EMPTY AUTOSAVE",
			Screenshot:        emptyScreenshot("AUTOSAVE"),
		})
	}

	return metadata
}

// ExportNewspaper creates and writes a text newspaper for the current world.
func ExportNewspaper(gameWorld *world.World) string {
	if gameWorld == nil {
		return ""
	}

	content := buildNewspaper(gameWorld)
	path := newspaperPath(gameWorld.Calendar.Day)
	_ = os.MkdirAll(filepath.Dir(path), 0o755)
	_ = os.WriteFile(path, []byte(content), 0o644)
	return content
}

func normalizeSaveData(data SaveData) SaveData {
	if data.Version == "" {
		data.Version = currentSaveVersion
	}
	if data.SaveDate.IsZero() {
		data.SaveDate = time.Now()
	}
	if data.WorldState != nil {
		if len(data.AllPobles) == 0 {
			data.AllPobles = data.WorldState.GetAllKnownPobles()
		}
		if !data.CurrentTime.IsValid() {
			data.CurrentTime = entities.GameTime(data.WorldState.Calendar)
		}
		if data.PlaytimeMinutes == 0 {
			data.PlaytimeMinutes = data.WorldState.Calendar.ToMinutes()
		}
	}
	data.Metadata = buildMetadata(data)
	return data
}

func buildMetadata(data SaveData) SaveMetadata {
	meta := data.Metadata
	if data.WorldState == nil {
		return meta
	}

	meta.Population = data.WorldState.GetPopulation()
	meta.Era = data.WorldState.Era
	meta.Day = data.WorldState.Calendar.Day
	meta.MostDramaticEvent = chooseMostDramaticEvent(data)
	meta.Screenshot = buildScreenshot(data.WorldState)
	meta.SavedAt = data.SaveDate
	return meta
}

// Delete removes one player save slot from disk.
func Delete(slot int) error {
	if slot < 1 || slot > maxPlayerSlots {
		return fmt.Errorf("delete slot %d invalid: use 1-%d", slot, maxPlayerSlots)
	}
	if err := os.Remove(slotPath(slot)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("delete save slot: %w", err)
	}
	return nil
}

// DeleteAutosave removes the autosave file from disk.
func DeleteAutosave() error {
	if err := os.Remove(autosavePath()); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("delete autosave: %w", err)
	}
	return nil
}

func chooseMostDramaticEvent(data SaveData) string {
	if len(data.EventHistory) > 0 {
		for i := len(data.EventHistory) - 1; i >= 0; i-- {
			if strings.TrimSpace(data.EventHistory[i].Description) != "" {
				return data.EventHistory[i].Description
			}
		}
	}
	if data.WorldState != nil && len(data.WorldState.EventHistory) > 0 {
		best := data.WorldState.EventHistory[0]
		for _, event := range data.WorldState.EventHistory[1:] {
			if event.Severity > best.Severity {
				best = event
			}
		}
		if strings.TrimSpace(best.Description) != "" {
			return best.Description
		}
	}
	return "No dramatic event recorded yet."
}

func writeSaveFile(path string, data SaveData) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create save directory: %w", err)
	}

	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create save file: %w", err)
	}
	defer file.Close()

	gzipWriter := gzip.NewWriter(file)
	defer gzipWriter.Close()

	encoder := json.NewEncoder(gzipWriter)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(data); err != nil {
		return fmt.Errorf("encode save data: %w", err)
	}
	return nil
}

func readSaveFile(path string) (*SaveData, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return nil, fmt.Errorf("open gzip save: %w", err)
	}
	defer gzipReader.Close()

	var data SaveData
	if err := json.NewDecoder(gzipReader).Decode(&data); err != nil {
		return nil, fmt.Errorf("decode save data: %w", err)
	}
	return &data, nil
}

func migrateSaveData(data *SaveData) error {
	if data == nil {
		return errors.New("save data missing")
	}

	if data.Version == "" {
		data.Version = currentSaveVersion
	}
	if compareSaveVersions(data.Version, currentSaveVersion) > 0 {
		return fmt.Errorf("save version %s is newer than current version %s", data.Version, currentSaveVersion)
	}

	if data.WorldState == nil {
		return errors.New("save missing world state")
	}

	if len(data.AllPobles) == 0 {
		data.AllPobles = data.WorldState.GetAllKnownPobles()
	}
	if !data.CurrentTime.IsValid() {
		data.CurrentTime = entities.GameTime(data.WorldState.Calendar)
	}
	if data.SaveDate.IsZero() {
		data.SaveDate = time.Now()
	}
	if compareSaveVersions(data.Version, currentSaveVersion) < 0 {
		data.Version = currentSaveVersion
		data.Metadata = buildMetadata(*data)
	}
	return nil
}

func compareSaveVersions(left, right string) int {
	parse := func(version string) []int {
		parts := strings.Split(version, ".")
		result := make([]int, 0, len(parts))
		for _, part := range parts {
			value := 0
			fmt.Sscanf(part, "%d", &value)
			result = append(result, value)
		}
		return result
	}

	a := parse(left)
	b := parse(right)
	size := len(a)
	if len(b) > size {
		size = len(b)
	}
	for i := 0; i < size; i++ {
		av := 0
		bv := 0
		if i < len(a) {
			av = a[i]
		}
		if i < len(b) {
			bv = b[i]
		}
		if av < bv {
			return -1
		}
		if av > bv {
			return 1
		}
	}
	return 0
}

func buildScreenshot(gameWorld *world.World) string {
	label := "POBLATION"
	if gameWorld != nil {
		label = fmt.Sprintf("ERA %s", gameWorld.Era)
	}
	lines := []string{
		"+------------------+",
		fmt.Sprintf("|%-18s|", trimPad(label, 18)),
		fmt.Sprintf("|POP %-14d|", gameWorld.GetPopulation()),
		fmt.Sprintf("|DAY %-14d|", gameWorld.Calendar.Day),
		fmt.Sprintf("|ISL %-14d|", discoveredIslandCount(gameWorld)),
		fmt.Sprintf("|TEC %-14d|", len(gameWorld.TechTree.Discovered)),
		fmt.Sprintf("|GOV %-14s|", trimPad(governmentLabel(gameWorld), 14)),
		fmt.Sprintf("|FOOD %-13d|", totalResources(gameWorld)[world.ResourceFood]),
		fmt.Sprintf("|WATR %-13d|", totalResources(gameWorld)[world.ResourceWater]),
		"+------------------+",
	}
	return strings.Join(lines, "\n")
}

func emptyScreenshot(label string) string {
	lines := []string{
		"+------------------+",
		fmt.Sprintf("|%-18s|", trimPad(label, 18)),
		"|EMPTY             |",
		"|                  |",
		"|                  |",
		"|                  |",
		"|                  |",
		"|                  |",
		"|                  |",
		"+------------------+",
	}
	return strings.Join(lines, "\n")
}

func buildNewspaper(gameWorld *world.World) string {
	resources := totalResources(gameWorld)
	headline := chooseHeadline(gameWorld)
	body := []string{
		"THE POBLATION GAZETTE",
		fmt.Sprintf("Day %d | Era %s | Population %d", gameWorld.Calendar.Day, gameWorld.Era, gameWorld.GetPopulation()),
		"",
		"HEADLINE",
		headline,
		"",
		"STATE OF THE WORLD",
		fmt.Sprintf("Food reserves: %d", resources[world.ResourceFood]),
		fmt.Sprintf("Water reserves: %d", resources[world.ResourceWater]),
		fmt.Sprintf("Known technologies: %d", len(gameWorld.TechTree.Discovered)),
		fmt.Sprintf("Government: %s", governmentLabel(gameWorld)),
		"",
		"RECENT TENSION",
		latestWorldDrama(gameWorld),
	}
	return strings.Join(body, "\n")
}

func chooseHeadline(gameWorld *world.World) string {
	if gameWorld == nil || len(gameWorld.EventHistory) == 0 {
		return "The settlement survives another day, which is never as simple as it sounds."
	}
	best := latestWorldDrama(gameWorld)
	return best
}

func latestWorldDrama(gameWorld *world.World) string {
	if gameWorld == nil || len(gameWorld.EventHistory) == 0 {
		return "Nothing has become public enough to print yet."
	}
	best := gameWorld.EventHistory[0]
	for _, event := range gameWorld.EventHistory[1:] {
		if event.Severity >= best.Severity {
			best = event
		}
	}
	if strings.TrimSpace(best.Description) == "" {
		return "The settlement feels like it is holding its breath."
	}
	return best.Description
}

func adaptWorldHistory(history []world.GameEvent) []events.GameEvent {
	result := make([]events.GameEvent, 0, len(history))
	for _, event := range history {
		result = append(result, events.GameEvent{
			ID:           event.ID,
			Type:         adaptedEventType(event),
			Timestamp:    event.Time,
			Participants: append([]string{}, event.Participants...),
			IsPublic:     true,
			Description:  event.Description,
			Consequences: []events.Consequence{},
			IsResolved:   false,
			ChildEvents:  append([]string{}, event.Tags...),
		})
	}
	return result
}

func adaptedEventType(event world.GameEvent) events.EventType {
	switch {
	case hasTag(event.Tags, "theft"):
		return events.EventTheft
	case hasTag(event.Tags, "inheritance"):
		return events.EventInheritance
	case hasTag(event.Tags, "technology"):
		return events.EventTechDiscovered
	case hasTag(event.Tags, "era_transition"):
		return events.EventEraChange
	case hasTag(event.Tags, "gambling"):
		return events.EventGamblingResult
	case hasTag(event.Tags, "trade"):
		return events.EventTradeEstablished
	default:
		return events.EventDecisionPoint
	}
}

func saveRoot() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".poblation"
	}
	return filepath.Join(home, ".poblation")
}

func slotPath(slot int) string {
	return filepath.Join(saveRoot(), "saves", fmt.Sprintf("slot_%d.sav", slot))
}

func autosavePath() string {
	return filepath.Join(saveRoot(), "saves", "autosave.sav")
}

func newspaperPath(day int) string {
	return filepath.Join(saveRoot(), "exports", fmt.Sprintf("newspaper_day_%d.txt", day))
}

func trimPad(value string, width int) string {
	if len(value) >= width {
		return value[:width]
	}
	return value + strings.Repeat(" ", width-len(value))
}

func discoveredIslandCount(gameWorld *world.World) int {
	if gameWorld == nil {
		return 0
	}
	count := 0
	for _, island := range gameWorld.Islands {
		if island != nil && island.IsDiscovered {
			count++
		}
	}
	return count
}

func governmentLabel(gameWorld *world.World) string {
	if gameWorld == nil || gameWorld.Government == nil {
		return "NONE"
	}
	return string(gameWorld.Government.Type)
}

func totalResources(gameWorld *world.World) map[world.ResourceType]int {
	totals := map[world.ResourceType]int{}
	if gameWorld == nil {
		return totals
	}
	for _, island := range gameWorld.Islands {
		if island == nil || !island.IsDiscovered {
			continue
		}
		for resource, amount := range island.Resources {
			totals[resource] += amount
		}
	}
	return totals
}

func hasTag(tags []string, target string) bool {
	for _, tag := range tags {
		if tag == target {
			return true
		}
	}
	return false
}

func sortMetadataByDay(values []SaveMetadata) {
	sort.SliceStable(values, func(i, j int) bool {
		return values[i].Day > values[j].Day
	})
}
