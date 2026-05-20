package ui

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/user/poblation/internal/engine"
	"github.com/user/poblation/internal/entities"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

// AgentPlayOptions controls a deterministic playthrough used by Codex/agents.
type AgentPlayOptions struct {
	OutputDir string
	Width     int
	Height    int
	Steps     int
}

type agentPlayCapture struct {
	Name string
	View ViewType
	Note string
	Text string
	PNG  string
}

// RunAgentPlay drives the real Bubbletea model without a terminal so agents can
// play and inspect the game like a user, while saving screenshot artifacts.
func RunAgentPlay(orchestrator *engine.Orchestrator, options AgentPlayOptions) error {
	if orchestrator == nil {
		return fmt.Errorf("agent play needs an initialized orchestrator")
	}
	if options.OutputDir == "" {
		options.OutputDir = filepath.Join("agent-runs", time.Now().Format("20060102-150405"))
	}
	if options.Width <= 0 {
		options.Width = 120
	}
	if options.Height <= 0 {
		options.Height = 36
	}
	if options.Steps <= 0 {
		options.Steps = 10
	}
	if err := os.MkdirAll(options.OutputDir, 0o755); err != nil {
		return fmt.Errorf("create agent play output: %w", err)
	}

	model := NewAppModelWithOrchestrator(orchestrator)
	model.loading.Active = false
	updated, _ := model.Update(tea.WindowSizeMsg{Width: options.Width, Height: options.Height})
	model = asAppModel(updated)

	captures := []agentPlayCapture{}
	capture := func(name, note string) error {
		item, err := captureAgentFrame(options.OutputDir, len(captures), name, note, model)
		if err != nil {
			return err
		}
		captures = append(captures, item)
		return nil
	}

	if err := capture("menu", "Primera pantalla que ve un jugador al abrir POBLATION."); err != nil {
		return err
	}
	if err := switchAgentView(&model, VIEW_MAIN_MAP); err != nil {
		return err
	}
	if err := capture("main-map", "Mapa principal con feed y estado general."); err != nil {
		return err
	}

	for hour := 1; hour <= options.Steps; hour++ {
		tick := engine.GameTick{
			CurrentTime: entities.NewGameTime(0, hour, 0),
			DeltaHours:  1,
			IsNewDay:    hour%24 == 0,
			IsNewWeek:   false,
			IsNewMonth:  false,
			IsNewYear:   false,
		}
		updated, _ := model.Update(GameTickMsg{Tick: tick})
		model = asAppModel(updated)
	}
	if err := capture("after-time", fmt.Sprintf("Mapa despues de avanzar %d horas de simulacion.", options.Steps)); err != nil {
		return err
	}

	views := []struct {
		view ViewType
		name string
		note string
	}{
		{VIEW_MIND, "mind", "Vista de mente: razones, deseos, recuerdos e intenciones visibles."},
		{VIEW_SETTLEMENT, "settlement", "Vista de settlement: civilizacion, instituciones y estado del pueblo."},
		{VIEW_WORLD_EXPLORE, "world", "Exploracion de mundo como jugador."},
		{VIEW_POBLES_LIST, "pobles", "Lista o entrada hacia Pobles."},
		{VIEW_EVENTS_FEED, "events", "Feed de eventos generado por la partida."},
		{VIEW_NEWSPAPER, "newspaper", "Periodico/lectura social si el modo lo permite."},
	}
	for _, step := range views {
		if err := switchAgentView(&model, step.view); err != nil {
			return err
		}
		if err := capture(step.name, step.note); err != nil {
			return err
		}
	}

	return writeAgentSummary(options.OutputDir, captures, options)
}

func asAppModel(model tea.Model) AppModel {
	if updated, ok := model.(AppModel); ok {
		return updated
	}
	return NewAppModel(nil, nil)
}

func switchAgentView(model *AppModel, view ViewType) error {
	updated, _ := model.switchToView(view, "", "")
	*model = asAppModel(updated)
	return nil
}

func captureAgentFrame(dir string, index int, name, note string, model AppModel) (agentPlayCapture, error) {
	base := fmt.Sprintf("%02d-%s", index, safeAgentFilePart(name))
	raw := model.View()
	clean := cleanANSI(raw)
	textPath := filepath.Join(dir, base+".txt")
	pngPath := filepath.Join(dir, base+".png")
	if err := os.WriteFile(textPath, []byte(clean), 0o644); err != nil {
		return agentPlayCapture{}, fmt.Errorf("write frame text: %w", err)
	}
	if err := renderAgentPNG(clean, pngPath); err != nil {
		return agentPlayCapture{}, fmt.Errorf("write frame png: %w", err)
	}
	return agentPlayCapture{
		Name: name,
		View: model.CurrentView,
		Note: note,
		Text: textPath,
		PNG:  pngPath,
	}, nil
}

func writeAgentSummary(dir string, captures []agentPlayCapture, options AgentPlayOptions) error {
	var b strings.Builder
	b.WriteString("# POBLATION Agent Play\n\n")
	b.WriteString("Run type: playable UI drive, not unit tests.\n")
	b.WriteString(fmt.Sprintf("Viewport: %dx%d\n", options.Width, options.Height))
	b.WriteString(fmt.Sprintf("Simulated hours: %d\n\n", options.Steps))
	for _, capture := range captures {
		b.WriteString(fmt.Sprintf("## %s - %s\n", capture.Name, capture.View.String()))
		b.WriteString(capture.Note)
		b.WriteString("\n")
		b.WriteString(fmt.Sprintf("- Text: `%s`\n", filepath.Base(capture.Text)))
		b.WriteString(fmt.Sprintf("- Screenshot: `%s`\n\n", filepath.Base(capture.PNG)))
	}
	return os.WriteFile(filepath.Join(dir, "summary.md"), []byte(b.String()), 0o644)
}

var ansiPattern = regexp.MustCompile(`\x1b\[[0-9;?]*[ -/]*[@-~]`)

func cleanANSI(text string) string {
	return strings.TrimRight(ansiPattern.ReplaceAllString(text, ""), "\r\n") + "\n"
}

func safeAgentFilePart(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	var b strings.Builder
	lastDash := false
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteRune('-')
			lastDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}

func renderAgentPNG(text, path string) error {
	text = asciiPreviewText(text)
	lines := strings.Split(strings.TrimRight(text, "\n"), "\n")
	if len(lines) == 0 {
		lines = []string{""}
	}
	face := basicfont.Face7x13
	cellWidth := 7
	cellHeight := 15
	padding := 16
	maxCols := 1
	for _, line := range lines {
		if count := len([]rune(line)); count > maxCols {
			maxCols = count
		}
	}
	imgWidth := maxCols*cellWidth + padding*2
	imgHeight := len(lines)*cellHeight + padding*2
	img := image.NewRGBA(image.Rect(0, 0, imgWidth, imgHeight))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: color.RGBA{R: 11, G: 16, B: 24, A: 255}}, image.Point{}, draw.Src)

	drawer := font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(color.RGBA{R: 225, G: 232, B: 241, A: 255}),
		Face: face,
		Dot:  fixed.Point26_6{},
	}
	for i, line := range lines {
		drawer.Dot = fixed.P(padding, padding+13+i*cellHeight)
		drawer.DrawString(line)
	}

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	return png.Encode(file, img)
}

func asciiPreviewText(text string) string {
	var b strings.Builder
	for _, r := range text {
		switch r {
		case 'á', 'à', 'ä', 'â', 'Á', 'À', 'Ä', 'Â':
			b.WriteByte('a')
		case 'é', 'è', 'ë', 'ê', 'É', 'È', 'Ë', 'Ê':
			b.WriteByte('e')
		case 'í', 'ì', 'ï', 'î', 'Í', 'Ì', 'Ï', 'Î':
			b.WriteByte('i')
		case 'ó', 'ò', 'ö', 'ô', 'Ó', 'Ò', 'Ö', 'Ô':
			b.WriteByte('o')
		case 'ú', 'ù', 'ü', 'û', 'Ú', 'Ù', 'Ü', 'Û':
			b.WriteByte('u')
		case 'ñ', 'Ñ':
			b.WriteByte('n')
		case '·', '•':
			b.WriteByte('-')
		case '│', '┃':
			b.WriteByte('|')
		case '─', '━':
			b.WriteByte('-')
		case '╭', '╮', '╰', '╯', '┌', '┐', '└', '┘', '├', '┤', '┬', '┴', '┼':
			b.WriteByte('+')
		default:
			if r == '\n' || r == '\t' || (r >= 32 && r <= 126) {
				b.WriteRune(r)
			} else {
				b.WriteByte(' ')
			}
		}
	}
	return b.String()
}
