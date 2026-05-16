package ui

import (
	"context"
	"fmt"
	"image/color"
	"path/filepath"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/user/poblation-launcher/assets"
	"github.com/user/poblation-launcher/internal/launcher"
)

type screenState struct {
	app      fyne.App
	window   fyne.Window
	settings launcher.Settings
	releases []launcher.Release
	installed []launcher.VersionRecord
	saves    []launcher.SaveSummary
	selectedSave launcher.SaveSummary

	versionSelect *widget.Select
	userEntry     *widget.Entry
	statusLabel   *widget.Label
	progress      *widget.ProgressBar
	newsBox       *fyne.Container
	saveBox       *fyne.Container
	updateBadge   *widget.Label
}

func Run() error {
	settings, err := launcher.LoadSettings()
	if err != nil {
		return err
	}
	a := app.NewWithID("com.poblation.launcher")
	icon := fyne.NewStaticResource("poblation-icon.png", assets.IconPNG)
	a.SetIcon(icon)

	w := a.NewWindow("POBLATION Launcher")
	w.Resize(fyne.NewSize(1040, 650))
	w.SetIcon(icon)

	state := &screenState{app: a, window: w, settings: settings}
	if err := state.refreshLocal(); err != nil {
		return err
	}
	w.SetContent(state.build())
	state.fetchReleasesAsync()
	w.ShowAndRun()
	return nil
}

func (s *screenState) refreshLocal() error {
	installed, err := launcher.NewVersionStore(s.settings.InstallDir).List()
	if err != nil {
		return err
	}
	saves, err := launcher.ListSaveSummaries()
	if err != nil {
		return err
	}
	s.installed = installed
	s.saves = saves
	if len(saves) > 0 {
		s.selectedSave = saves[0]
	}
	return nil
}

func (s *screenState) build() fyne.CanvasObject {
	wordmark := fyne.NewStaticResource("poblation-wordmark.png", assets.WordmarkPNG)
	logo := canvas.NewImageFromResource(wordmark)
	logo.FillMode = canvas.ImageFillContain
	logo.SetMinSize(fyne.NewSize(420, 130))

	top := container.NewBorder(nil, nil, nil, s.buildTopActions(), logo)
	s.newsBox = container.NewVBox()
	s.saveBox = container.NewVBox()
	s.updateBadge = widget.NewLabel("")
	s.updateBadge.TextStyle = fyne.TextStyle{Bold: true}
	s.renderNews(s.settings.LastNews)
	s.renderSavePanel()

	body := container.NewHSplit(s.leftPanel(), s.rightPanel())
	body.Offset = 0.42
	content := container.NewBorder(top, s.bottomBar(), nil, nil, body)
	bg := canvas.NewRectangle(color.NRGBA{R: 18, G: 20, B: 22, A: 255})
	return container.NewMax(bg, container.NewPadded(content))
}

func (s *screenState) buildTopActions() fyne.CanvasObject {
	settingsButton := widget.NewButtonWithIcon("", theme.SettingsIcon(), s.showSettings)
	settingsButton.Importance = widget.LowImportance
	return container.NewHBox(s.updateBadge, settingsButton)
}

func (s *screenState) leftPanel() fyne.CanvasObject {
	iconRes := fyne.NewStaticResource("poblation-icon.png", assets.IconPNG)
	icon := canvas.NewImageFromResource(iconRes)
	icon.FillMode = canvas.ImageFillContain
	icon.SetMinSize(fyne.NewSize(290, 240))
	launchSave := widget.NewButtonWithIcon("Jugar este save", theme.MediaPlayIcon(), func() {
		s.playSelectedSave()
	})
	launchSave.Importance = widget.HighImportance
	return panel("ULTIMA PARTIDA", container.NewVBox(icon, s.saveBox, launchSave))
}

func (s *screenState) rightPanel() fyne.CanvasObject {
	return panel("NOTICIAS", container.NewVScroll(s.newsBox))
}

func (s *screenState) bottomBar() fyne.CanvasObject {
	s.versionSelect = widget.NewSelect(launcher.VersionLabels(s.releases, s.installed), func(value string) {
		s.settings.DefaultVersion = value
		_ = launcher.SaveSettings(s.settings)
	})
	s.versionSelect.PlaceHolder = "Version"
	s.versionSelect.SetSelected(defaultVersion(s.settings, s.versionSelect.Options))

	s.userEntry = widget.NewEntry()
	s.userEntry.SetPlaceHolder("Usuario")
	s.userEntry.SetText(s.settings.UserName)
	s.userEntry.OnChanged = func(value string) {
		s.settings.UserName = value
		_ = launcher.SaveSettings(s.settings)
	}

	playButton := widget.NewButtonWithIcon("JUGAR", theme.MediaPlayIcon(), s.playSelectedSave)
	playButton.Importance = widget.HighImportance
	s.progress = widget.NewProgressBar()
	s.statusLabel = widget.NewLabel("Listo")
	bar := container.NewVBox(
		container.NewGridWithColumns(4, s.versionSelect, s.userEntry, playButton, s.statusLabel),
		s.progress,
	)
	return panel("", bar)
}

func (s *screenState) renderNews(items []launcher.NewsItem) {
	if s.newsBox == nil {
		return
	}
	s.newsBox.Objects = nil
	for _, item := range items {
		title := widget.NewLabel(item.Version + " - " + item.Title)
		title.TextStyle = fyne.TextStyle{Bold: true}
		date := widget.NewLabel(item.Date.Format("2006-01-02"))
		date.TextStyle = fyne.TextStyle{Italic: true}
		summary := widget.NewLabel(item.Summary)
		summary.Wrapping = fyne.TextWrapWord
		s.newsBox.Add(container.NewVBox(title, date, summary, canvas.NewLine(color.NRGBA{R: 64, G: 68, B: 72, A: 255})))
	}
	s.newsBox.Refresh()
}

func (s *screenState) renderSavePanel() {
	if s.saveBox == nil {
		return
	}
	s.saveBox.Objects = nil
	if len(s.saves) == 0 {
		s.saveBox.Add(widget.NewLabel("No hay saves todavia."))
		s.saveBox.Refresh()
		return
	}
	save := s.selectedSave
	rows := []string{
		save.WorldName + " - Dia " + fmt.Sprint(save.Day),
		"Poblacion: " + fmt.Sprint(save.Population),
		"Ultima sesion: " + friendlyTime(save.LastPlayed),
		save.Description,
	}
	for _, row := range rows {
		label := widget.NewLabel(row)
		label.Wrapping = fyne.TextWrapWord
		s.saveBox.Add(label)
	}
	s.saveBox.Refresh()
}

func (s *screenState) fetchReleasesAsync() {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 16*time.Second)
		defer cancel()
		client := launcher.NewReleaseClient(s.settings.Repository)
		releases, err := client.FetchReleases(ctx)
		if err != nil {
			s.setStatus("Offline: usando noticias guardadas")
			return
		}
		s.releases = releases
		news := launcher.ReleasesToNews(releases)
		s.settings.LastNews = news
		_ = launcher.SaveSettings(s.settings)
		s.renderNews(news)
		s.refreshVersions()
		s.setUpdateBadge()
		s.setStatus("Noticias actualizadas")
	}()
}

func (s *screenState) refreshVersions() {
	if s.versionSelect == nil {
		return
	}
	options := launcher.VersionLabels(s.releases, s.installed)
	s.versionSelect.Options = options
	s.versionSelect.SetSelected(defaultVersion(s.settings, options))
	s.versionSelect.Refresh()
}

func (s *screenState) setUpdateBadge() {
	if s.updateBadge == nil || len(s.releases) == 0 {
		return
	}
	latest := s.releases[0].TagName
	for _, record := range s.installed {
		if record.Version == latest {
			s.updateBadge.SetText("")
			return
		}
	}
	s.updateBadge.SetText("Update disponible: " + latest)
}

func (s *screenState) playSelectedSave() {
	save := s.selectedSave
	if save.WorldName == "" && len(s.saves) > 0 {
		save = s.saves[0]
	}
	go s.prepareAndLaunch(save)
}

func (s *screenState) prepareAndLaunch(save launcher.SaveSummary) {
	s.setStatus("Preparando...")
	record, offline, err := s.resolveVersion()
	if err != nil {
		s.showError(err)
		return
	}
	clock := launcher.CheckClock(s.settings, save, time.Now())
	s.settings = clock.Settings
	_ = launcher.SaveSettings(s.settings)
	if clock.Kind == launcher.ClockHardware {
		s.showClockDialog(clock, func() { s.finishLaunch(record, save, launcher.APClock) })
		return
	}
	trigger := launcher.APNone
	if clock.Kind == launcher.ClockIntentional {
		trigger = launcher.APRAMEater
	} else if offline {
		trigger = launcher.APOffline
	} else if launcher.RandomAntiPiracyTriggered(launcher.NewLaunchRNG()) {
		trigger = launcher.APRandom
	}
	s.finishLaunch(record, save, trigger)
}

func (s *screenState) resolveVersion() (launcher.VersionRecord, bool, error) {
	store := launcher.NewVersionStore(s.settings.InstallDir)
	selected := s.versionSelect.Selected
	if selected == "" {
		selected = s.settings.DefaultVersion
	}
	if record, ok, err := store.Find(selected); err != nil || ok {
		return record, true, err
	}
	for _, release := range s.releases {
		if release.TagName == selected || selected == "latest" {
			return s.download(release)
		}
	}
	record, ok, err := store.Latest()
	if err != nil || !ok {
		return launcher.VersionRecord{}, true, fmt.Errorf("no hay version descargada para jugar offline")
	}
	return record, true, nil
}

func (s *screenState) download(release launcher.Release) (launcher.VersionRecord, bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	record, err := launcher.DownloadRelease(ctx, s.settings, release, func(fraction float64, status string) {
		s.setProgress(fraction)
		s.setStatus(status)
	})
	if err != nil {
		return launcher.VersionRecord{}, false, err
	}
	if err := s.refreshLocal(); err != nil {
		return record, false, err
	}
	s.refreshVersions()
	return record, false, nil
}

func (s *screenState) finishLaunch(record launcher.VersionRecord, save launcher.SaveSummary, trigger launcher.AntiPiracyTrigger) {
	sequence := launcher.BuildAntiPiracySequence(trigger, save)
	if trigger != launcher.APNone {
		s.showAntiPiracy(sequence, func() { s.launchNow(record, save) })
		return
	}
	s.launchNow(record, save)
}

func (s *screenState) launchNow(record launcher.VersionRecord, save launcher.SaveSummary) {
	s.setStatus("Abriendo POBLATION...")
	err := launcher.LaunchGame(launcher.PlayRequest{Version: record, Save: save})
	if err != nil {
		s.showError(err)
		return
	}
	s.setStatus("Juego abierto")
}

func panel(title string, content fyne.CanvasObject) fyne.CanvasObject {
	header := widget.NewLabel(title)
	header.TextStyle = fyne.TextStyle{Bold: true}
	box := container.NewBorder(header, nil, nil, nil, content)
	bg := canvas.NewRectangle(color.NRGBA{R: 30, G: 33, B: 37, A: 245})
	return container.NewMax(bg, container.NewPadded(box))
}

func defaultVersion(settings launcher.Settings, options []string) string {
	for _, option := range options {
		if option == settings.DefaultVersion {
			return option
		}
	}
	if len(options) > 0 {
		return options[0]
	}
	return ""
}

func friendlyTime(value time.Time) string {
	if value.IsZero() {
		return "desconocida"
	}
	delta := time.Since(value)
	switch {
	case delta < time.Hour:
		return "hace minutos"
	case delta < 48*time.Hour:
		return "ayer"
	case delta < 14*24*time.Hour:
		return fmt.Sprintf("hace %d dias", int(delta.Hours()/24))
	default:
		return value.Format("2006-01-02")
	}
}

func (s *screenState) setStatus(value string) {
	if s.statusLabel != nil {
		s.statusLabel.SetText(value)
	}
}

func (s *screenState) setProgress(value float64) {
	if s.progress != nil {
		s.progress.SetValue(value)
	}
}

func (s *screenState) showError(err error) {
	s.setStatus("Error")
	dialog.ShowError(err, s.window)
}

func (s *screenState) showSettings() {
	repo := widget.NewEntry()
	repo.SetText(s.settings.Repository)
	installDir := widget.NewEntry()
	installDir.SetText(s.settings.InstallDir)
	defaultVersion := widget.NewEntry()
	defaultVersion.SetText(s.settings.DefaultVersion)
	background := widget.NewCheck("Proceso en segundo plano", nil)
	background.SetChecked(s.settings.BackgroundProcess)
	notifications := widget.NewCheck("Notificaciones", nil)
	notifications.SetChecked(s.settings.Notifications)
	clearCache := widget.NewButton("Limpiar cache", func() {
		err := launcher.NewVersionStore(s.settings.InstallDir).Cleanup(0)
		if err != nil {
			s.showError(err)
			return
		}
		s.setStatus("Cache limpio")
	})
	form := container.NewVBox(repo, defaultVersion, installDir, background, notifications, clearCache)
	dialog.ShowCustomConfirm("Ajustes", "Guardar", "Cancelar", form, func(ok bool) {
		if !ok {
			return
		}
		s.settings.Repository = strings.TrimSpace(repo.Text)
		s.settings.DefaultVersion = strings.TrimSpace(defaultVersion.Text)
		s.settings.InstallDir = filepath.Clean(strings.TrimSpace(installDir.Text))
		s.settings.BackgroundProcess = background.Checked
		s.settings.Notifications = notifications.Checked
		if err := launcher.SaveSettings(s.settings); err != nil {
			s.showError(err)
			return
		}
		s.setStatus("Ajustes guardados")
	}, s.window)
}
