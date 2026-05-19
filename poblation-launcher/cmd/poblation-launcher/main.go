package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/user/poblation-launcher/internal/launcher"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "Launcher error:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	settings, err := launcher.LoadSettings()
	if err != nil {
		return err
	}
	if err := ensureRuntime(settings); err != nil {
		return err
	}
	if len(args) == 0 {
		return menu(settings)
	}
	switch args[0] {
	case "install", "update", "download":
		return installCommand(settings, args[1:])
	case "play", "run":
		return playCommand(settings, args[1:])
	case "list", "versions":
		return listCommand(settings)
	case "saves":
		return savesCommand()
	case "news", "releases":
		return newsCommand(settings)
	case "doctor":
		return doctorCommand(settings)
	case "folders":
		return foldersCommand(settings)
	case "settings":
		return settingsCommand(settings)
	case "help", "-h", "--help":
		printHelp()
		return nil
	default:
		printHelp()
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func menu(settings launcher.Settings) error {
	reader := bufio.NewReader(os.Stdin)
	for {
		printBanner()
		fmt.Println("1) Play latest installed")
		fmt.Println("2) Install/update latest release")
		fmt.Println("3) List installed versions")
		fmt.Println("4) Saves")
		fmt.Println("5) News/releases")
		fmt.Println("6) Open folders")
		fmt.Println("7) Doctor check")
		fmt.Println("0) Exit")
		fmt.Print("\nChoose: ")
		choice, _ := reader.ReadString('\n')
		switch strings.TrimSpace(choice) {
		case "1":
			return playCommand(settings, nil)
		case "2":
			if err := installCommand(settings, nil); err != nil {
				fmt.Println("Install failed:", err)
			}
			pause(reader)
		case "3":
			_ = listCommand(settings)
			pause(reader)
		case "4":
			_ = savesCommand()
			pause(reader)
		case "5":
			_ = newsCommand(settings)
			pause(reader)
		case "6":
			_ = foldersCommand(settings)
			pause(reader)
		case "7":
			_ = doctorCommand(settings)
			pause(reader)
		case "0", "q", "quit", "exit":
			return nil
		default:
			fmt.Println("Not an option.")
			pause(reader)
		}
	}
}

func installCommand(settings launcher.Settings, args []string) error {
	tag := ""
	flags := flag.NewFlagSet("install", flag.ContinueOnError)
	flags.StringVar(&tag, "version", "", "release tag to install")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() > 0 {
		tag = flags.Arg(0)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	release, err := selectRelease(ctx, settings, tag)
	if err != nil {
		return err
	}
	fmt.Println("Installing:", release.DisplayName(), "("+release.TagName+")")
	record, err := launcher.DownloadRelease(ctx, settings, release, printProgress)
	if err != nil {
		return err
	}
	settings.DefaultVersion = record.Version
	if err := launcher.SaveSettings(settings); err != nil {
		return err
	}
	fmt.Println("\nInstalled:", record.Version)
	fmt.Println("Path:", record.Path)
	return nil
}

func playCommand(settings launcher.Settings, args []string) error {
	version := settings.DefaultVersion
	slot := 0
	smoke := false
	flags := flag.NewFlagSet("play", flag.ContinueOnError)
	flags.StringVar(&version, "version", version, "installed version to run")
	flags.IntVar(&slot, "slot", 0, "save slot to open")
	flags.BoolVar(&smoke, "smoke", false, "verify the selected game starts and exits")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() > 0 {
		version = flags.Arg(0)
	}
	store := launcher.NewVersionStore(settings.InstallDir)
	record, ok, err := store.Find(version)
	if err != nil {
		return err
	}
	if !ok {
		record, ok, err = store.Latest()
		if err != nil {
			return err
		}
	}
	if !ok {
		return errors.New("no installed POBLATION version found; run install first")
	}
	if _, err := os.Stat(record.Path); err != nil {
		return fmt.Errorf("installed executable missing: %w", err)
	}
	return runGameAttached(record, playOptions{slot: slot, smoke: smoke})
}

type playOptions struct {
	slot  int
	smoke bool
}

func runGameAttached(record launcher.VersionRecord, options playOptions) error {
	args := []string{}
	if options.slot > 0 {
		args = append(args, "--slot", strconv.Itoa(options.slot))
	}
	if options.smoke {
		args = append(args, "--smoke")
	}
	cmd := exec.Command(record.Path, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("run POBLATION: %w", err)
	}
	return launcher.NewVersionStoreFromRecord(record).MarkPlayed(record.Version)
}

func listCommand(settings launcher.Settings) error {
	records, err := launcher.NewVersionStore(settings.InstallDir).List()
	if err != nil {
		return err
	}
	if len(records) == 0 {
		fmt.Println("No installed versions yet.")
		return nil
	}
	fmt.Println("Installed versions:")
	for _, record := range records {
		fmt.Printf("- %s | %s\n  %s\n", record.Version, record.Name, record.Path)
	}
	return nil
}

func savesCommand() error {
	saves, err := launcher.ListSaveSummaries()
	if err != nil {
		return err
	}
	if len(saves) == 0 {
		fmt.Println("No saves yet.")
		return nil
	}
	fmt.Println("Saves:")
	for _, save := range saves {
		kind := "slot"
		if save.IsAutosave {
			kind = "autosave"
		}
		fmt.Printf("- %s %d | %s | day %d | pobles %d\n  %s\n", kind, save.Slot, save.WorldName, save.Day, save.Population, save.Description)
	}
	return nil
}

func newsCommand(settings launcher.Settings) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	releases, err := launcher.NewReleaseClient(settings.Repository).FetchReleases(ctx)
	if err != nil {
		return err
	}
	for _, release := range releases {
		fmt.Printf("- %s | %s\n  %s\n", release.TagName, release.DisplayName(), release.HTMLURL)
	}
	return nil
}

func doctorCommand(settings launcher.Settings) error {
	fmt.Println("POBLATION launcher doctor")
	fmt.Println("Repository:", settings.Repository)
	fmt.Println("Install dir:", settings.InstallDir)
	fmt.Println("Default version:", settings.DefaultVersion)
	if err := ensureRuntime(settings); err != nil {
		return err
	}
	records, err := launcher.NewVersionStore(settings.InstallDir).List()
	if err != nil {
		return err
	}
	fmt.Println("Installed versions:", len(records))
	for _, record := range records {
		state := "ok"
		if _, err := os.Stat(record.Path); err != nil {
			state = "missing"
		}
		fmt.Printf("- %s [%s]\n", record.Version, state)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	releases, err := launcher.NewReleaseClient(settings.Repository).FetchReleases(ctx)
	if err != nil {
		fmt.Println("GitHub releases: unavailable:", err)
		return nil
	}
	fmt.Println("GitHub releases:", len(releases))
	return nil
}

func foldersCommand(settings launcher.Settings) error {
	fmt.Println("1) Launcher:", launcher.LauncherRoot())
	fmt.Println("2) Versions:", settings.InstallDir)
	fmt.Println("3) Saves:", launcher.SavesDir())
	openPath(settings.InstallDir)
	return nil
}

func settingsCommand(settings launcher.Settings) error {
	data := []string{
		"Repository: " + settings.Repository,
		"Default version: " + settings.DefaultVersion,
		"Install dir: " + settings.InstallDir,
		"Notifications: " + strconv.FormatBool(settings.Notifications),
	}
	fmt.Println(strings.Join(data, "\n"))
	return nil
}

func selectRelease(ctx context.Context, settings launcher.Settings, tag string) (launcher.Release, error) {
	releases, err := launcher.NewReleaseClient(settings.Repository).FetchReleases(ctx)
	if err != nil {
		return launcher.Release{}, err
	}
	for _, release := range releases {
		if tag != "" && release.TagName != tag {
			continue
		}
		if _, ok := release.CurrentPlatformAsset(); ok {
			return release, nil
		}
	}
	if tag != "" {
		return launcher.Release{}, fmt.Errorf("release %s has no playable asset for %s/%s", tag, runtime.GOOS, runtime.GOARCH)
	}
	return launcher.Release{}, fmt.Errorf("no playable release found for %s/%s", runtime.GOOS, runtime.GOARCH)
}

func ensureRuntime(settings launcher.Settings) error {
	for _, path := range []string{
		settings.InstallDir,
		launcher.SavesDir(),
		filepath.Join(launcher.PoblationRoot(), "cache"),
		filepath.Join(launcher.LauncherRoot(), "logs"),
	} {
		if err := os.MkdirAll(path, 0o755); err != nil {
			return fmt.Errorf("create runtime folder %s: %w", path, err)
		}
	}
	return launcher.SaveSettings(settings)
}

func printProgress(fraction float64, status string) {
	fmt.Printf("\r%s %3.0f%%", status, fraction*100)
}

func openPath(path string) {
	switch runtime.GOOS {
	case "windows":
		_ = exec.Command("explorer", path).Start()
	case "darwin":
		_ = exec.Command("open", path).Start()
	default:
		_ = exec.Command("xdg-open", path).Start()
	}
}

func printBanner() {
	fmt.Print("\033[H\033[2J")
	fmt.Println("POBLATION Launcher")
	fmt.Println("v1.0.0.2 // lightweight edition")
	fmt.Println(strings.Repeat("-", 40))
}

func printHelp() {
	fmt.Println("POBLATION Launcher")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  install [tag]       Download and verify a GitHub release")
	fmt.Println("  play [tag]          Run an installed version")
	fmt.Println("  play -smoke         Verify the selected game starts")
	fmt.Println("  list                Show installed versions")
	fmt.Println("  saves               Show save previews")
	fmt.Println("  news                Show GitHub releases")
	fmt.Println("  doctor              Check local install health")
	fmt.Println("  folders             Open local folders")
	fmt.Println("  settings            Print launcher settings")
}

func pause(reader *bufio.Reader) {
	fmt.Print("\nPress Enter...")
	_, _ = reader.ReadString('\n')
}
