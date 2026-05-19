package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

const (
	distDir      = "dist"
	launcherEXE  = "poblation-launcher.exe"
	installerEXE = "poblation_v1.0.0-beta.1_launcher_installer.exe"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "release:", err)
		os.Exit(1)
	}
}

func run() error {
	if err := os.MkdirAll(distDir, 0o755); err != nil {
		return fmt.Errorf("create dist: %w", err)
	}
	if err := command("go", "build", "-ldflags", "-s -w", "-o", filepath.Join(distDir, launcherEXE), "./cmd/poblation-launcher"); err != nil {
		return err
	}
	payloadDir := filepath.Join("tools", "installer", "payload")
	if err := os.MkdirAll(payloadDir, 0o755); err != nil {
		return fmt.Errorf("create installer payload: %w", err)
	}
	if err := copyFile(filepath.Join(distDir, launcherEXE), filepath.Join(payloadDir, launcherEXE)); err != nil {
		return err
	}
	if err := command("go", "build", "-tags", "installer", "-ldflags", "-s -w", "-o", filepath.Join(distDir, installerEXE), "./tools/installer"); err != nil {
		return err
	}
	fmt.Println("Built", filepath.Join(distDir, installerEXE))
	return nil
}

func command(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s %v: %w", name, args, err)
	}
	return nil
}

func copyFile(from, to string) error {
	src, err := os.Open(from)
	if err != nil {
		return fmt.Errorf("open %s: %w", from, err)
	}
	defer src.Close()
	dst, err := os.Create(to)
	if err != nil {
		return fmt.Errorf("create %s: %w", to, err)
	}
	defer dst.Close()
	if _, err := io.Copy(dst, src); err != nil {
		return fmt.Errorf("copy launcher payload: %w", err)
	}
	return nil
}
