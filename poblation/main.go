package main

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/user/poblation/internal/engine"
	"github.com/user/poblation/internal/ui"
)

//go:embed templates
var embeddedTemplates embed.FS

func main() {
	seed := flag.Int64("seed", 0, "world seed")
	debug := flag.Bool("debug", false, "enable debug console")
	speed := flag.Float64("speed", 1.0, "simulation speed multiplier")
	slot := flag.Int("slot", 0, "save slot to load directly")
	smoke := flag.Bool("smoke", false, "verify startup and exit")
	agentPlay := flag.Bool("agent-play", false, "run an automated user-style play session with UI screenshots")
	agentOut := flag.String("agent-out", "", "output directory for --agent-play artifacts")
	agentSteps := flag.Int("agent-steps", 10, "simulation hours to advance during --agent-play")
	flag.Parse()

	ctx, stopSignals := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopSignals()

	if err := preparePortableTemplates(); err != nil {
		fmt.Fprintf(os.Stderr, "poblation templates: %v\n", err)
		os.Exit(1)
	}

	orchestrator := engine.NewOrchestrator(engine.OrchestratorOptions{
		Debug: *debug,
		Speed: *speed,
		Slot:  *slot,
	})
	if err := orchestrator.Init(*seed); err != nil {
		fmt.Fprintf(os.Stderr, "poblation init: %v\n", err)
		os.Exit(1)
	}
	defer orchestrator.Stop()
	if *smoke {
		fmt.Println("POBLATION startup ok")
		return
	}
	if *agentPlay {
		out := *agentOut
		if out == "" {
			out = filepath.Join("..", "agent-runs", time.Now().Format("20060102-150405"))
		}
		if err := ui.RunAgentPlay(orchestrator, ui.AgentPlayOptions{
			OutputDir: out,
			Width:     120,
			Height:    36,
			Steps:     *agentSteps,
		}); err != nil {
			fmt.Fprintf(os.Stderr, "poblation agent play: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("POBLATION agent play ok: %s\n", out)
		return
	}

	model := ui.NewAppModelWithOrchestrator(orchestrator)
	program := tea.NewProgram(model, tea.WithAltScreen())
	go func() {
		<-ctx.Done()
		orchestrator.Stop()
		program.Quit()
	}()
	if _, err := program.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "poblation: %v\n", err)
		os.Exit(1)
	}
}

func preparePortableTemplates() error {
	if _, err := os.Stat("templates"); err == nil {
		return nil
	}
	root, err := portableTemplateRoot()
	if err != nil {
		return err
	}
	if err := extractEmbeddedTemplates(root); err != nil {
		return err
	}
	return os.Setenv("POBLATION_TEMPLATES_DIR", root)
}

func portableTemplateRoot() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("find user home: %w", err)
	}
	return filepath.Join(home, ".poblation", "runtime", "templates"), nil
}

func extractEmbeddedTemplates(targetRoot string) error {
	if err := os.MkdirAll(targetRoot, 0o755); err != nil {
		return fmt.Errorf("create portable template root: %w", err)
	}
	return fs.WalkDir(embeddedTemplates, "templates", func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel("templates", filepath.FromSlash(path))
		if err != nil {
			return fmt.Errorf("template relative path: %w", err)
		}
		if relative == "." {
			return nil
		}
		target := filepath.Join(targetRoot, relative)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := embeddedTemplates.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read embedded template %s: %w", path, err)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return fmt.Errorf("create template folder: %w", err)
		}
		if err := os.WriteFile(target, data, 0o644); err != nil {
			return fmt.Errorf("write template %s: %w", target, err)
		}
		return nil
	})
}
