package launcher

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
)

type PlayRequest struct {
	Version VersionRecord
	Save    SaveSummary
}

func LaunchGame(request PlayRequest) error {
	if request.Version.Path == "" {
		return fmt.Errorf("no downloaded version selected")
	}
	args := []string{}
	if request.Save.Slot > 0 {
		args = append(args, "--slot", strconv.Itoa(request.Save.Slot))
	}
	cmd := exec.Command(request.Version.Path, args...)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start POBLATION: %w", err)
	}
	store := NewVersionStoreFromRecord(request.Version)
	return store.MarkPlayed(request.Version.Version)
}

func NewVersionStoreFromRecord(record VersionRecord) VersionStore {
	return NewVersionStore(versionRootFromPath(record.Path))
}

func versionRootFromPath(path string) string {
	dir := filepath.Dir(path)
	return filepath.Dir(dir)
}
