package launcher

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type ProgressFunc func(fraction float64, status string)

func DownloadRelease(ctx context.Context, settings Settings, release Release, progress ProgressFunc) (VersionRecord, error) {
	asset, ok := release.CurrentPlatformAsset()
	if !ok {
		return VersionRecord{}, fmt.Errorf("release %s has no asset for %s/%s", release.DisplayName(), runtime.GOOS, runtime.GOARCH)
	}
	expectedHash, err := expectedSHA256(ctx, release, asset)
	if err != nil {
		return VersionRecord{}, err
	}
	if expectedHash == "" {
		return VersionRecord{}, fmt.Errorf("release asset %s has no sha256 digest or .sha256 sidecar", asset.Name)
	}
	storeRoot := settings.InstallDir
	versionDir := filepath.Join(storeRoot, cleanVersionDir(release.TagName))
	if err := os.MkdirAll(versionDir, 0o755); err != nil {
		return VersionRecord{}, fmt.Errorf("create version directory: %w", err)
	}

	targetPath := filepath.Join(versionDir, executableName(asset.Name))
	tempPath := targetPath + ".download"
	hash, err := downloadFile(ctx, asset.BrowserDownloadURL, tempPath, asset.Size, progress)
	if err != nil {
		_ = os.Remove(tempPath)
		return VersionRecord{}, err
	}
	if !strings.EqualFold(hash, expectedHash) {
		_ = os.Remove(tempPath)
		return VersionRecord{}, fmt.Errorf("sha256 mismatch for %s: got %s, want %s", asset.Name, hash, expectedHash)
	}
	if err := os.Rename(tempPath, targetPath); err != nil {
		return VersionRecord{}, fmt.Errorf("install downloaded version: %w", err)
	}
	if runtime.GOOS != "windows" {
		if err := os.Chmod(targetPath, 0o755); err != nil {
			return VersionRecord{}, fmt.Errorf("mark version executable: %w", err)
		}
	}

	record := VersionRecord{
		Version:     release.TagName,
		Name:        release.DisplayName(),
		Path:        targetPath,
		SHA256:      hash,
		SourceURL:   asset.BrowserDownloadURL,
		InstalledAt: time.Now().UTC(),
	}
	store := NewVersionStore(settings.InstallDir)
	if err := store.Upsert(record); err != nil {
		return VersionRecord{}, err
	}
	if err := store.Cleanup(MaxVersionsToKeep); err != nil {
		return VersionRecord{}, err
	}
	if progress != nil {
		progress(1, "Version lista")
	}
	return record, nil
}

func expectedSHA256(ctx context.Context, release Release, asset ReleaseAsset) (string, error) {
	digest := strings.TrimSpace(asset.Digest)
	if strings.HasPrefix(strings.ToLower(digest), "sha256:") {
		return digest[len("sha256:"):], nil
	}
	sidecar, ok := shaSidecarAsset(release, asset)
	if !ok {
		return "", nil
	}
	text, err := fetchText(ctx, sidecar.BrowserDownloadURL)
	if err != nil {
		return "", err
	}
	fields := strings.Fields(text)
	if len(fields) == 0 {
		return "", fmt.Errorf("empty sha256 sidecar for %s", asset.Name)
	}
	return fields[0], nil
}

func shaSidecarAsset(release Release, asset ReleaseAsset) (ReleaseAsset, bool) {
	assetName := strings.ToLower(asset.Name)
	for _, candidate := range release.Assets {
		name := strings.ToLower(candidate.Name)
		if name == assetName+".sha256" || strings.TrimSuffix(name, ".sha256") == assetName {
			return candidate, true
		}
	}
	return ReleaseAsset{}, false
}

func downloadFile(ctx context.Context, url, targetPath string, size int64, progress ProgressFunc) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("create download request: %w", err)
	}
	req.Header.Set("User-Agent", "poblation-launcher/"+AppVersion)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("download release asset: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return "", fmt.Errorf("download release asset: status %s", resp.Status)
	}
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return "", fmt.Errorf("create download directory: %w", err)
	}
	file, err := os.Create(targetPath)
	if err != nil {
		return "", fmt.Errorf("create download file: %w", err)
	}
	defer file.Close()

	hasher := sha256.New()
	writer := io.MultiWriter(file, hasher)
	buffer := make([]byte, 64*1024)
	var readTotal int64
	for {
		n, readErr := resp.Body.Read(buffer)
		if n > 0 {
			if _, err := writer.Write(buffer[:n]); err != nil {
				return "", fmt.Errorf("write download file: %w", err)
			}
			readTotal += int64(n)
			reportDownloadProgress(readTotal, size, progress)
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return "", fmt.Errorf("read download stream: %w", readErr)
		}
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func fetchText(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("create sidecar request: %w", err)
	}
	req.Header.Set("User-Agent", "poblation-launcher/"+AppVersion)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("download sha256 sidecar: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return "", fmt.Errorf("download sha256 sidecar: status %s", resp.Status)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8*1024))
	if err != nil {
		return "", fmt.Errorf("read sha256 sidecar: %w", err)
	}
	return string(body), nil
}

func reportDownloadProgress(readTotal, size int64, progress ProgressFunc) {
	if progress == nil {
		return
	}
	if size <= 0 {
		progress(0.15, "Descargando version...")
		return
	}
	fraction := float64(readTotal) / float64(size)
	if fraction > 1 {
		fraction = 1
	}
	progress(fraction, "Descargando version...")
}

func cleanVersionDir(version string) string {
	clean := strings.TrimSpace(version)
	clean = strings.TrimPrefix(clean, "v")
	if clean == "" {
		return "unknown"
	}
	replacer := strings.NewReplacer("/", "-", "\\", "-", ":", "-", "*", "-", "?", "-")
	return replacer.Replace(clean)
}

func executableName(assetName string) string {
	name := filepath.Base(assetName)
	if runtime.GOOS == "windows" && !strings.HasSuffix(strings.ToLower(name), ".exe") {
		return name + ".exe"
	}
	return name
}
