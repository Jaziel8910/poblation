package views

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/png"
	"os"
	"strings"
)

const kittyImageChunkSize = 4096
const ansiLogoWidth = 62

var ansiLogoBackground = color.RGBA{R: 13, G: 13, B: 13, A: 255}

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
		return ansiRasterPNG(data, ansiLogoWidth)
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

func ansiRasterPNG(data []byte, columns int) string {
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return ""
	}
	bounds := img.Bounds()
	srcW := bounds.Dx()
	srcH := bounds.Dy()
	if srcW <= 0 || srcH <= 0 {
		return ""
	}
	columns = clampImageInt(columns, 24, 90)
	pixelHeight := int(float64(srcH) * (float64(columns) / float64(srcW)))
	pixelHeight = clampImageInt(pixelHeight, 8, 28)
	if pixelHeight%2 != 0 {
		pixelHeight++
	}

	var builder strings.Builder
	for y := 0; y < pixelHeight; y += 2 {
		for x := 0; x < columns; x++ {
			upper := sampleLogoPixel(img, bounds, x, y, columns, pixelHeight)
			lower := sampleLogoPixel(img, bounds, x, y+1, columns, pixelHeight)
			if isLogoBackground(upper) && isLogoBackground(lower) {
				builder.WriteString("\x1b[0m ")
				continue
			}
			builder.WriteString(ansiHalfBlock(upper, lower))
		}
		builder.WriteString("\x1b[0m\n")
	}
	return strings.TrimRight(builder.String(), "\n")
}

func sampleLogoPixel(img image.Image, bounds image.Rectangle, x, y, outW, outH int) color.RGBA {
	if y >= outH {
		return ansiLogoBackground
	}
	srcX := bounds.Min.X + int(float64(x)/float64(outW)*float64(bounds.Dx()))
	srcY := bounds.Min.Y + int(float64(y)/float64(outH)*float64(bounds.Dy()))
	r, g, b, a := img.At(srcX, srcY).RGBA()
	return blendLogoRGBA(color.RGBA{
		R: uint8(r >> 8),
		G: uint8(g >> 8),
		B: uint8(b >> 8),
		A: uint8(a >> 8),
	})
}

func blendLogoRGBA(c color.RGBA) color.RGBA {
	if c.A == 255 {
		return c
	}
	alpha := float64(c.A) / 255.0
	return color.RGBA{
		R: uint8(float64(c.R)*alpha + float64(ansiLogoBackground.R)*(1-alpha)),
		G: uint8(float64(c.G)*alpha + float64(ansiLogoBackground.G)*(1-alpha)),
		B: uint8(float64(c.B)*alpha + float64(ansiLogoBackground.B)*(1-alpha)),
		A: 255,
	}
}

func ansiHalfBlock(upper, lower color.RGBA) string {
	return "\x1b[38;2;" + rgbString(upper) + "m\x1b[48;2;" + rgbString(lower) + "m▀"
}

func rgbString(c color.RGBA) string {
	return itoaImage(int(c.R)) + ";" + itoaImage(int(c.G)) + ";" + itoaImage(int(c.B))
}

func itoaImage(value int) string {
	if value <= 0 {
		return "0"
	}
	if value >= 255 {
		return "255"
	}
	digits := [3]byte{}
	i := len(digits)
	for value > 0 {
		i--
		digits[i] = byte('0' + value%10)
		value /= 10
	}
	return string(digits[i:])
}

func isLogoBackground(c color.RGBA) bool {
	return absImageInt(int(c.R)-int(ansiLogoBackground.R))+
		absImageInt(int(c.G)-int(ansiLogoBackground.G))+
		absImageInt(int(c.B)-int(ansiLogoBackground.B)) < 18
}

func clampImageInt(value, low, high int) int {
	if value < low {
		return low
	}
	if value > high {
		return high
	}
	return value
}

func absImageInt(value int) int {
	if value < 0 {
		return -value
	}
	return value
}
