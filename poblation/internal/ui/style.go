package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/lipgloss"
	uiviews "github.com/user/poblation/internal/ui/views"
)

type Theme = uiviews.Theme
type LayoutManager = uiviews.LayoutManager

var DefaultTheme = uiviews.DefaultTheme

var (
	appBackgroundStyle = lipgloss.NewStyle().
				Background(DefaultTheme.Background).
				Foreground(DefaultTheme.Text)

	statusBarStyle = lipgloss.NewStyle().
			Background(DefaultTheme.Surface).
			Foreground(DefaultTheme.Text).
			Padding(0, 1)

	statusTitleStyle = lipgloss.NewStyle().
				Background(DefaultTheme.Surface).
				Foreground(DefaultTheme.Accent).
				Bold(true)

	navBarStyle = lipgloss.NewStyle().
			Background(DefaultTheme.Surface).
			Foreground(DefaultTheme.Muted).
			Padding(0, 1)

	viewFrameStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(DefaultTheme.Border).
			Background(DefaultTheme.Background).
			Foreground(DefaultTheme.Text).
			Padding(1, 2)

	viewTitleStyle = lipgloss.NewStyle().
			Foreground(DefaultTheme.SecondaryAccent).
			Bold(true)

	mutedTextStyle = lipgloss.NewStyle().
			Foreground(DefaultTheme.Muted)

	consoleStyle = lipgloss.NewStyle().
			Foreground(DefaultTheme.Warning)
)

type templateLoadingState struct {
	Active           bool
	Err              error
	StartedAt        time.Time
	UpdatedAt        time.Time
	CurrentCategory  string
	LoadedTemplates  int
	TotalTemplates   int
	LoadedCategories int
	TotalCategories  int
	CategoryTotals   map[string]int
	CategoryLoaded   map[string]int
	TaglineFrame     int
	Spinner          spinner.Model
}

var loadingTaglines = []string{
	"El mundo todavia no existe. Los chismes ya casi.",
	"Cargando ruinas, deseos, hambre y malas decisiones.",
	"Armando la isla una plantilla a la vez.",
	"Pobles, secretos y drama entrando en calor.",
}

func newTemplateLoadingState() templateLoadingState {
	spin := spinner.New()
	spin.Spinner = spinner.Dot
	spin.Style = lipgloss.NewStyle().Foreground(DefaultTheme.Accent)

	return templateLoadingState{
		Active:         true,
		StartedAt:      time.Now(),
		UpdatedAt:      time.Now(),
		CategoryTotals: map[string]int{},
		CategoryLoaded: map[string]int{},
		Spinner:        spin,
	}
}

func (s templateLoadingState) percent() float64 {
	if s.TotalTemplates <= 0 {
		return 0
	}
	value := float64(s.LoadedTemplates) / float64(s.TotalTemplates)
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}

func (s templateLoadingState) eta() string {
	if !s.Active || s.LoadedTemplates <= 0 || s.TotalTemplates <= 0 {
		return "calculando"
	}
	progress := s.percent()
	if progress <= 0 || progress >= 1 {
		return "casi listo"
	}
	elapsed := time.Since(s.StartedAt)
	totalEstimate := time.Duration(float64(elapsed) / progress)
	remaining := totalEstimate - elapsed
	if remaining < 0 {
		remaining = 0
	}
	if remaining < time.Second {
		return "<1s"
	}
	return remaining.Round(time.Second).String()
}

func (s templateLoadingState) tagline() string {
	if len(loadingTaglines) == 0 {
		return ""
	}
	return loadingTaglines[s.TaglineFrame%len(loadingTaglines)]
}

func (s templateLoadingState) spinnerFrame() string {
	frames := []string{"-", "\\", "|", "/"}
	return frames[s.TaglineFrame%len(frames)]
}

func (s templateLoadingState) render(width, height int) string {
	layout := LayoutManager{Width: width, Height: height}
	cardWidth := minInt(maxInt(48, width-8), 86)
	if layout.IsCompactHeight() {
		cardWidth = minInt(cardWidth, maxInt(34, width-4))
	}

	lines := []string{
		uiviews.HeaderStyle.Render("POBLATION"),
		uiviews.SubheaderStyle.Render(s.spinnerFrame() + " " + s.tagline()),
		"",
		uiviews.BodyStyle.Render("Levantando templates, eventos y voces del mundo..."),
		renderLoadingBar(s.percent(), minInt(42, maxInt(18, cardWidth-8))),
		uiviews.MutedStyle.Render(fmt.Sprintf("%d / %d templates", s.LoadedTemplates, s.TotalTemplates)),
		uiviews.MutedStyle.Render(fmt.Sprintf("%d / %d categorias", s.LoadedCategories, s.TotalCategories)),
		uiviews.MutedStyle.Render("ETA: " + s.eta()),
	}

	if strings.TrimSpace(s.CurrentCategory) != "" {
		lines = append(lines, "", uiviews.AccentStyle.Render("Categoria actual: "+strings.ToUpper(s.CurrentCategory)))
	}

	categoryLines := make([]string, 0, len(s.CategoryTotals))
	for _, row := range categoryRows(s.CategoryTotals, s.CategoryLoaded) {
		categoryLines = append(categoryLines, uiviews.BodyStyle.Render(row))
	}
	if len(categoryLines) > 0 {
		lines = append(lines, "", uiviews.SubheaderStyle.Render("Carga por categoria"))
		lines = append(lines, categoryLines...)
	}

	if s.Err != nil {
		lines = append(lines, "", uiviews.DangerStyle.Render("Error: "+s.Err.Error()))
	}

	card := uiviews.BorderAccent.
		Width(cardWidth).
		Padding(1, 2).
		Render(strings.Join(lines, "\n"))

	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, card)
}

func renderLoadingBar(percent float64, width int) string {
	width = maxInt(8, width)
	filled := int(percent * float64(width))
	if filled > width {
		filled = width
	}
	if filled < 0 {
		filled = 0
	}
	bar := strings.Repeat("=", filled) + strings.Repeat("-", width-filled)
	return uiviews.AccentStyle.Render("[" + bar + "]")
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func categoryRows(totals, loaded map[string]int) []string {
	keys := make([]string, 0, len(totals))
	for key := range totals {
		keys = append(keys, key)
	}
	if len(keys) == 0 {
		return nil
	}
	// tiny stable order without importing sort here
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			if keys[j] < keys[i] {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}

	rows := make([]string, 0, len(keys))
	for _, key := range keys {
		rows = append(rows, fmt.Sprintf("%-10s %4d / %4d", strings.ToUpper(key), loaded[key], totals[key]))
	}
	return rows
}

func notificationStyle(kind NotificationType) lipgloss.Style {
	color := DefaultTheme.SecondaryAccent
	switch kind {
	case NotificationDeath:
		color = DefaultTheme.Danger
	case NotificationBirth:
		color = DefaultTheme.Success
	case NotificationDrama:
		color = DefaultTheme.Warning
	}

	return lipgloss.NewStyle().
		Background(DefaultTheme.Surface).
		Foreground(color).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(color).
		Padding(0, 1)
}
