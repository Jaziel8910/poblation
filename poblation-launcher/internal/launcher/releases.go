package launcher

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"strings"
	"time"
)

type ReleaseClient struct {
	Repository string
	HTTPClient *http.Client
}

type Release struct {
	TagName     string         `json:"tag_name"`
	Name        string         `json:"name"`
	Body        string         `json:"body"`
	HTMLURL     string         `json:"html_url"`
	PublishedAt time.Time      `json:"published_at"`
	Prerelease   bool           `json:"prerelease"`
	Draft        bool           `json:"draft"`
	Assets      []ReleaseAsset `json:"assets"`
}

type ReleaseAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
	Digest             string `json:"digest"`
}

func NewReleaseClient(repository string) ReleaseClient {
	return ReleaseClient{
		Repository: repository,
		HTTPClient: &http.Client{Timeout: 15 * time.Second},
	}
}

func (c ReleaseClient) FetchReleases(ctx context.Context) ([]Release, error) {
	if strings.Count(c.Repository, "/") != 1 {
		return nil, fmt.Errorf("repository must look like owner/name, got %q", c.Repository)
	}
	url := "https://api.github.com/repos/" + c.Repository + "/releases"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create releases request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "poblation-launcher/"+AppVersion)

	client := c.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch GitHub releases: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, fmt.Errorf("fetch GitHub releases: status %s", resp.Status)
	}

	var releases []Release
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, fmt.Errorf("decode GitHub releases: %w", err)
	}
	return publicReleases(releases), nil
}

func publicReleases(releases []Release) []Release {
	result := make([]Release, 0, len(releases))
	for _, release := range releases {
		if release.Draft {
			continue
		}
		result = append(result, release)
	}
	return result
}

func (r Release) DisplayName() string {
	if strings.TrimSpace(r.Name) != "" {
		return r.Name
	}
	if strings.TrimSpace(r.TagName) != "" {
		return r.TagName
	}
	return "Unknown release"
}

func (r Release) SelectPlayableAsset(goos, goarch string) (ReleaseAsset, bool) {
	osNeedle := strings.ToLower(goos)
	archNeedle := strings.ToLower(goarch)
	for _, asset := range r.Assets {
		name := strings.ToLower(asset.Name)
		if strings.Contains(name, ".sha256") {
			continue
		}
		if strings.Contains(name, osNeedle) && strings.Contains(name, archNeedle) {
			return asset, true
		}
		if osNeedle == "windows" && strings.HasSuffix(name, ".exe") && strings.Contains(name, archNeedle) {
			return asset, true
		}
	}
	return ReleaseAsset{}, false
}

func (r Release) CurrentPlatformAsset() (ReleaseAsset, bool) {
	return r.SelectPlayableAsset(runtime.GOOS, runtime.GOARCH)
}

func ReleasesToNews(releases []Release) []NewsItem {
	if len(releases) == 0 {
		return EmbeddedNews()
	}
	items := make([]NewsItem, 0, len(releases))
	for _, release := range releases {
		items = append(items, releaseToNews(release))
		if len(items) == 5 {
			break
		}
	}
	return items
}

func VersionLabels(releases []Release, installed []VersionRecord) []string {
	seen := map[string]bool{}
	labels := make([]string, 0, len(releases)+len(installed)+1)
	for _, release := range releases {
		label := release.TagName
		if label == "" {
			label = release.DisplayName()
		}
		if !seen[label] {
			labels = append(labels, label)
			seen[label] = true
		}
	}
	for _, record := range installed {
		if !seen[record.Version] {
			labels = append(labels, record.Version)
			seen[record.Version] = true
		}
	}
	if len(labels) == 0 {
		labels = append(labels, "offline")
	}
	return labels
}
