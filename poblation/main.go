package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/user/poblation/internal/engine"
	"github.com/user/poblation/internal/ui"
)

func main() {
	seed := flag.Int64("seed", 0, "world seed")
	debug := flag.Bool("debug", false, "enable debug console")
	speed := flag.Float64("speed", 1.0, "simulation speed multiplier")
	slot := flag.Int("slot", 0, "save slot to load directly")
	flag.Parse()

	ctx, stopSignals := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopSignals()

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
