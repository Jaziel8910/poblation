package views

import (
	"encoding/base64"
	"os"
	"strings"
)

const kittyImageChunkSize = 4096

func terminalPNG(data []byte) string {
	if len(data) == 0 {
		return ""
	}
	switch terminalImageProtocol() {
	case "iterm":
		return itermInlineImage(data)
	case "kitty":
		return kittyInlinePNG(data)
	default:
		return ""
	}
}

func terminalImageProtocol() string {
	term := strings.ToLower(os.Getenv("TERM"))
	program := strings.ToLower(os.Getenv("TERM_PROGRAM"))
	if strings.Contains(program, "iterm") {
		return "iterm"
	}
	if strings.Contains(term, "xterm-kitty") || strings.Contains(program, "kitty") || strings.Contains(program, "wezterm") {
		return "kitty"
	}
	return ""
}

func kittyInlinePNG(data []byte) string {
	encoded := base64.StdEncoding.EncodeToString(data)
	var builder strings.Builder
	for len(encoded) > 0 {
		part := encoded
		if len(part) > kittyImageChunkSize {
			part = encoded[:kittyImageChunkSize]
		}
		encoded = encoded[len(part):]
		more := "0"
		if len(encoded) > 0 {
			more = "1"
		}
		builder.WriteString("\x1b_Ga=T,f=100,q=2,m=")
		builder.WriteString(more)
		builder.WriteString(";")
		builder.WriteString(part)
		builder.WriteString("\x1b\\")
	}
	return builder.String()
}

func itermInlineImage(data []byte) string {
	encoded := base64.StdEncoding.EncodeToString(data)
	return "\x1b]1337;File=inline=1;width=42;height=9;preserveAspectRatio=1:" + encoded + "\a"
}
