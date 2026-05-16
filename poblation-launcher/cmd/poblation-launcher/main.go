package main

import (
	"log"

	"github.com/user/poblation-launcher/internal/ui"
)

func main() {
	if err := ui.Run(); err != nil {
		log.Fatalf("poblation launcher: %v", err)
	}
}
