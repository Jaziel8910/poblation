package launcher

import (
	"strings"
	"time"
)

type NewsItem struct {
	Version string    `json:"version"`
	Title   string    `json:"title"`
	Date    time.Time `json:"date"`
	Summary string    `json:"summary"`
	URL     string    `json:"url"`
}

func EmbeddedNews() []NewsItem {
	return []NewsItem{
		{
			Version: "v1.0.0.2",
			Title:   "PLAYABLE RELEASE",
			Date:    time.Date(2026, 5, 19, 0, 0, 0, 0, time.UTC),
			Summary: "Portable game exe, lightweight launcher, fixed Desktop shortcut, friendlier GitHub page, and verified local startup.",
		},
		{
			Version: "v1.0.0-beta.1",
			Title:   "BETA V1.0.0",
			Date:    time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC),
			Summary: "Beta jugable: memoria social visible, settlement profundo, finales ampliados, 1814 templates curados y launcher actualizado.",
		},
		{
			Version: "v1.0.0.1",
			Title:   "THE LAUNCHER UPDATE",
			Date:    time.Date(2026, 5, 16, 0, 0, 0, 0, time.UTC),
			Summary: "Launcher grafico, logo PNG, versiones descargables, saves visibles y modo offline.",
		},
		{
			Version: "v1.0.0",
			Title:   "POBLATION",
			Date:    time.Date(2026, 5, 15, 0, 0, 0, 0, time.UTC),
			Summary: "La humanidad se extinguio. Quedan los Pobles, sus secretos y tus malas decisiones.",
		},
	}
}

func releaseToNews(release Release) NewsItem {
	return NewsItem{
		Version: release.TagName,
		Title:   release.DisplayName(),
		Date:    release.PublishedAt,
		Summary: summarizeReleaseBody(release.Body),
		URL:     release.HTMLURL,
	}
}

func summarizeReleaseBody(body string) string {
	clean := strings.TrimSpace(body)
	if clean == "" {
		return "Release disponible para descargar desde GitHub."
	}
	lines := strings.Split(clean, "\n")
	parts := make([]string, 0, 2)
	for _, line := range lines {
		line = strings.TrimSpace(strings.TrimLeft(line, "-#* "))
		if line == "" {
			if len(parts) > 0 {
				break
			}
			continue
		}
		parts = append(parts, line)
		if len(parts) == 2 {
			break
		}
	}
	summary := strings.Join(parts, " ")
	if len(summary) > 180 {
		return summary[:177] + "..."
	}
	return summary
}
