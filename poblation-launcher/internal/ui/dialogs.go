package ui

import (
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/user/poblation-launcher/internal/launcher"
)

func (s *screenState) showClockDialog(result launcher.ClockResult, continueLaunch func()) {
	signals := widget.NewLabel(strings.Join(result.Signals, "\n"))
	signals.Wrapping = fyne.TextWrapWord
	message := widget.NewLabel(result.Message)
	message.Wrapping = fyne.TextWrapWord
	var d dialog.Dialog
	buttons := container.NewGridWithColumns(3,
		widget.NewButton("Corregi el reloj", func() {
			d.Hide()
			continueLaunch()
		}),
		widget.NewButton("Fue Windows", func() {
			d.Hide()
			continueLaunch()
		}),
		widget.NewButton("Jugar igual", func() {
			d.Hide()
			continueLaunch()
		}),
	)
	body := container.NewVBox(message, signals, buttons)
	d = dialog.NewCustomWithoutButtons(result.Title, body, s.window)
	d.Show()
}

func (s *screenState) showAntiPiracy(sequence launcher.AntiPiracySequence, continueLaunch func()) {
	if len(sequence.Lines) == 0 {
		continueLaunch()
		return
	}
	text := widget.NewLabel("")
	text.Wrapping = fyne.TextWrapWord
	body := container.NewVBox(widget.NewLabel(string(sequence.Trigger)), text)
	d := dialog.NewCustomWithoutButtons("POBLATION", body, s.window)
	d.Show()
	go func() {
		var builder strings.Builder
		for _, line := range sequence.Lines {
			for _, r := range line {
				builder.WriteRune(r)
				text.SetText(builder.String())
				time.Sleep(sequence.Delay)
			}
			builder.WriteString("\n\n")
			text.SetText(builder.String())
			time.Sleep(240 * time.Millisecond)
		}
		time.Sleep(600 * time.Millisecond)
		d.Hide()
		continueLaunch()
	}()
}
