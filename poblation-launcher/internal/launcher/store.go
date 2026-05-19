package launcher

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

type VersionStore struct {
	Root string
}

type VersionManifest struct {
	Versions []VersionRecord `json:"versions"`
}

type VersionRecord struct {
	Version      string    `json:"version"`
	Name         string    `json:"name"`
	Path         string    `json:"path"`
	SHA256       string    `json:"sha256"`
	SourceURL    string    `json:"source_url"`
	InstalledAt  time.Time `json:"installed_at"`
	LastPlayedAt time.Time `json:"last_played_at"`
}

func NewVersionStore(root string) VersionStore {
	return VersionStore{Root: root}
}

func (s VersionStore) List() ([]VersionRecord, error) {
	manifest, err := s.loadManifest()
	if err != nil {
		return nil, err
	}
	records := make([]VersionRecord, 0, len(manifest.Versions))
	for _, record := range manifest.Versions {
		if record.Path == "" {
			continue
		}
		if _, err := os.Stat(record.Path); err == nil {
			records = append(records, record)
		}
	}
	sortRecords(records)
	return records, nil
}

func (s VersionStore) Find(version string) (VersionRecord, bool, error) {
	records, err := s.List()
	if err != nil {
		return VersionRecord{}, false, err
	}
	for _, record := range records {
		if record.Version == version || version == "latest" {
			return record, true, nil
		}
	}
	return VersionRecord{}, false, nil
}

func (s VersionStore) Latest() (VersionRecord, bool, error) {
	records, err := s.List()
	if err != nil {
		return VersionRecord{}, false, err
	}
	if len(records) == 0 {
		return VersionRecord{}, false, nil
	}
	return records[0], true, nil
}

func (s VersionStore) Upsert(record VersionRecord) error {
	manifest, err := s.loadManifest()
	if err != nil {
		return err
	}
	replaced := false
	for i := range manifest.Versions {
		if manifest.Versions[i].Version == record.Version {
			manifest.Versions[i] = record
			replaced = true
			break
		}
	}
	if !replaced {
		manifest.Versions = append(manifest.Versions, record)
	}
	sortRecords(manifest.Versions)
	return s.saveManifest(manifest)
}

func (s VersionStore) MarkPlayed(version string) error {
	manifest, err := s.loadManifest()
	if err != nil {
		return err
	}
	for i := range manifest.Versions {
		if manifest.Versions[i].Version == version {
			manifest.Versions[i].LastPlayedAt = time.Now().UTC()
			return s.saveManifest(manifest)
		}
	}
	return nil
}

func (s VersionStore) Cleanup(keep int) error {
	records, err := s.List()
	if err != nil {
		return err
	}
	if len(records) <= keep {
		return nil
	}
	for _, record := range records[keep:] {
		if err := os.Remove(record.Path); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove old version %s: %w", record.Version, err)
		}
		_ = os.Remove(filepath.Dir(record.Path))
	}
	manifest := VersionManifest{Versions: records[:keep]}
	return s.saveManifest(manifest)
}

func (s VersionStore) manifestPath() string {
	return filepath.Join(s.Root, "manifest.json")
}

func (s VersionStore) loadManifest() (VersionManifest, error) {
	path := s.manifestPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return VersionManifest{}, nil
		}
		return VersionManifest{}, fmt.Errorf("open version manifest: %w", err)
	}
	var manifest VersionManifest
	data = bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF})
	if err := json.Unmarshal(data, &manifest); err != nil {
		return VersionManifest{}, fmt.Errorf("decode version manifest: %w", err)
	}
	return manifest, nil
}

func (s VersionStore) saveManifest(manifest VersionManifest) error {
	if err := os.MkdirAll(s.Root, 0o755); err != nil {
		return fmt.Errorf("create version store: %w", err)
	}
	file, err := os.Create(s.manifestPath())
	if err != nil {
		return fmt.Errorf("create version manifest: %w", err)
	}
	defer file.Close()
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(manifest); err != nil {
		return fmt.Errorf("encode version manifest: %w", err)
	}
	return nil
}

func sortRecords(records []VersionRecord) {
	sort.SliceStable(records, func(i, j int) bool {
		left := records[i].InstalledAt
		right := records[j].InstalledAt
		if left.Equal(right) {
			return records[i].Version > records[j].Version
		}
		return left.After(right)
	})
}
