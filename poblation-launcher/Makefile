APP_NAME := poblation-launcher
VERSION := v1.0.0.2
DIST := dist
INSTALLER := poblation_$(VERSION)_launcher_installer.exe

.PHONY: run build installer launcher-release clean

run:
	go run ./cmd/poblation-launcher

build:
	go build -o $(DIST)/$(APP_NAME).exe ./cmd/poblation-launcher

installer:
	go run ./tools/release

launcher-release: installer
	@echo Built $(DIST)/$(INSTALLER)
	@echo Cross-platform packaging notes live in docs/release.md

clean:
	go clean ./...
