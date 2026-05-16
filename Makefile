.PHONY: launcher-release game-test launcher-test

launcher-release:
	cd poblation-launcher && go run ./tools/release

game-test:
	cd poblation && go test ./...

launcher-test:
	cd poblation-launcher && go test ./internal/launcher
