package image

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/zakaria-chahboun/AyatDesingBot/config"
	"github.com/zakaria-chahboun/AyatDesingBot/quran"

	"github.com/tdewolff/canvas"
	"github.com/tdewolff/canvas/renderers"
)

// Unicode BiDi control characters
const (
	RLM = "\u200F" // Right-to-Left Mark — forces RTL context around a token
	RLE = "\u202B" // Right-to-Left Embedding
	PDF = "\u202C" // Pop Directional Formatting
)

// isQuranicAnnotation returns true for standalone Quranic pause/annotation marks
// in the range U+06D6–U+06ED that appear as isolated space-separated tokens.
func isQuranicAnnotation(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if unicode.IsSpace(r) {
			continue
		}
		if r >= 0x06D6 && r <= 0x06ED {
			return true
		}
		if r >= 0x0750 && r <= 0x077F {
			return true
		}
		if r >= 0x08A0 && r <= 0x08FF {
			return true
		}
	}
	return false
}

// normalizeVerseText:
//  1. Merges standalone Quranic annotation marks into the preceding word.
//  2. Wraps ornamental verse-number brackets ﴿N﴾ with RLM marks so BiDi
//     renders them correctly (closing bracket on the right in RTL context).
func normalizeVerseText(text string) string {
	parts := strings.Split(text, " ")
	var merged []string
	for _, p := range parts {
		if p == "" {
			continue
		}
		if isQuranicAnnotation(p) && len(merged) > 0 {
			merged[len(merged)-1] = merged[len(merged)-1] + " " + p
		} else {
			// Wrap tokens that start with ﴿ (U+FD3F) with RLM on both sides.
			// Without this, the Unicode BiDi algorithm treats the opening bracket
			// as a neutral character and may mirror it, placing it on the wrong side.
			if strings.ContainsRune(p, '\uFD3F') || strings.ContainsRune(p, '\uFD3E') {
				p = RLM + p + RLM
			}
			merged = append(merged, p)
		}
	}
	return strings.Join(merged, " ")
}

// wrapArabicText manually breaks Arabic text into lines that fit within maxWidth,
// always breaking at word (space) boundaries — never mid-word.
// Returns both the wrapped string and the number of lines produced.
func wrapArabicText(face *canvas.FontFace, text string, maxWidth float64) (string, int) {
	text = normalizeVerseText(text)

	words := strings.Fields(text)
	if len(words) == 0 {
		return text, 1
	}

	var lines []string
	currentLine := ""

	for _, word := range words {
		candidate := word
		if currentLine != "" {
			candidate = currentLine + " " + word
		}

		t := canvas.NewTextLine(face, candidate, canvas.Center)
		bounds := t.Bounds()

		if bounds.W() <= maxWidth {
			currentLine = candidate
		} else {
			if currentLine != "" {
				lines = append(lines, currentLine)
			}
			currentLine = word
		}
	}
	if currentLine != "" {
		lines = append(lines, currentLine)
	}

	return strings.Join(lines, "\n"), len(lines)
}

// estimateTextHeight returns an accurate height estimate for N lines at the given
// font size, using a single measured line height plus line spacing.
// This is used to pre-check fit BEFORE calling NewTextBox, avoiding the silent
// clipping that happens when NewTextBox receives a height smaller than its content.
func estimateTextHeight(face *canvas.FontFace, sampleLine string, numLines int, lineStretch float64) float64 {
	t := canvas.NewTextLine(face, sampleLine, canvas.Center)
	b := t.Bounds()
	lineH := b.H()
	// canvas applies lineStretch as extra spacing between lines
	spacing := lineH * lineStretch
	return float64(numLines)*lineH + float64(numLines-1)*spacing
}

// GenerateImage creates a stylized image of Quran verses natively
func GenerateImage(surahNum int, surahName string, startAyah, endAyah int, verses []quran.Verse, style config.Style, fontPath string) ([]byte, error) {
	openingText := fmt.Sprintf("surah%03d", surahNum)

	var versesText string
	for _, v := range verses {
		versesText += fmt.Sprintf("%s ﴿%s﴾ ", v.Text, quran.ConvertToArabicIndic(fmt.Sprintf("%d", v.ID)))
	}
	versesText = strings.TrimSpace(versesText)

	footerText := "تيليجرام AyatDesignBot"

	// 1. Setup Fonts
	fontDir := filepath.Dir(fontPath)
	nabiFontPath := filepath.Join(fontDir, "Nabi.ttf")
	openingFontPath := filepath.Join(fontDir, "surah-name-v2.ttf")
	footerFontPath := filepath.Join(fontDir, "Tajawal-Regular.ttf")

	fontFamily := canvas.NewFontFamily("Nabi")
	if err := fontFamily.LoadFontFile(nabiFontPath, canvas.FontRegular); err != nil {
		return nil, fmt.Errorf("could not load font %s: %w", nabiFontPath, err)
	}

	openingFontFamily := canvas.NewFontFamily("SurahName")
	if err := openingFontFamily.LoadFontFile(openingFontPath, canvas.FontRegular); err != nil {
		return nil, fmt.Errorf("could not load opening font: %w", err)
	}

	footerFontFamily := canvas.NewFontFamily("Tajawal")
	if err := footerFontFamily.LoadFontFile(footerFontPath, canvas.FontRegular); err != nil {
		return nil, fmt.Errorf("could not load footer font: %w", err)
	}

	// 2. Setup Canvas
	const width = 720.0
	const height = 1280.0
	c := canvas.New(width, height)
	ctx := canvas.NewContext(c)

	// 3. Draw Background Image
	bgFile, err := os.Open(style.BackgroundImage)
	if err != nil {
		return nil, fmt.Errorf("failed to open background image: %w", err)
	}
	defer bgFile.Close()

	bgImg, _, err := image.Decode(bgFile)
	if err != nil {
		return nil, fmt.Errorf("failed to decode background image: %w", err)
	}

	bgW := float64(bgImg.Bounds().Dx())
	bgH := float64(bgImg.Bounds().Dy())
	scale := width / bgW
	if (height / bgH) > scale {
		scale = height / bgH
	}
	scaledW := bgW * scale
	scaledH := bgH * scale
	offsetX := (width - scaledW) / 2
	offsetY := (height - scaledH) / 2

	ctx.SetFillColor(canvas.Black)
	ctx.DrawPath(0, 0, canvas.Rectangle(width, height))
	imgResolution := canvas.Resolution(bgW / scaledW)
	ctx.DrawImage(offsetX, offsetY, bgImg, imgResolution)

	// 4. Draw Overlay
	var textColor color.Color = canvas.White
	if style.TextColor == "black" {
		textColor = canvas.Black
		ctx.SetFillColor(canvas.RGBA(255, 255, 255, 128))
	} else {
		ctx.SetFillColor(canvas.RGBA(0, 0, 0, 128))
	}
	ctx.DrawPath(0, 0, canvas.Rectangle(width, height))

	// 5. Draw Texts

	// --- Opening (Surah glyph) ---
	openingFace := openingFontFamily.Face(235.0, textColor, canvas.FontRegular, canvas.FontNormal)
	openingTxt := canvas.NewTextLine(openingFace, openingText, canvas.Center)
	ctx.DrawText(width/2, height*0.87, openingTxt)

	// --- Main Verses ---
	boxWidth := width * 0.85
	maxBoxHeight := height * 0.68

	lineStretch := 0.0
	charsCount := len([]rune(versesText))
	switch {
	case charsCount < 80:
		lineStretch = 0.8
	case charsCount < 200:
		lineStretch = 0.4
	case charsCount < 400:
		lineStretch = 0.2
	}
	opts := &canvas.TextOptions{LineStretch: lineStretch}

	fontSize := 167.0
	var versesTxt *canvas.Text
	var versesBounds canvas.Rect

	for fontSize > 20.0 {
		verseFace := fontFamily.Face(fontSize, textColor, canvas.FontRegular, canvas.FontNormal)

		// Pre-wrap at word boundaries
		wrappedText, numLines := wrapArabicText(verseFace, versesText, boxWidth)

		// FIX: Estimate the actual rendered height BEFORE passing to NewTextBox.
		// NewTextBox silently clips content that exceeds its height argument —
		// it does NOT return an error or expand. So we must verify fit ourselves
		// using a direct line-height measurement, then only call NewTextBox when
		// we know the content will fit without clipping.
		estimatedH := estimateTextHeight(verseFace, "أ", numLines, lineStretch)
		if estimatedH > maxBoxHeight {
			fontSize -= 5.0
			continue
		}

		// Pass a very large height to NewTextBox so it never clips — we've already
		// ensured the content fits via our estimate above.
		versesTxt = canvas.NewTextBox(verseFace, wrappedText, boxWidth, maxBoxHeight*2, canvas.Center, canvas.Top, opts)
		versesBounds = versesTxt.Bounds()

		if versesBounds.W() <= boxWidth {
			break
		}
		fontSize -= 5.0
	}

	// Center the text block vertically on the canvas.
	// canvas Y=0 is bottom, DrawText places bottom-left of text at (x,y).
	versesX := (width - boxWidth) / 2
	versesY := height/2 + versesBounds.H()/2
	ctx.DrawText(versesX, versesY, versesTxt)

	// --- Footer ---
	footerFace := footerFontFamily.Face(80.0, textColor, canvas.FontRegular, canvas.FontNormal)
	footerTxt := canvas.NewTextLine(footerFace, footerText, canvas.Center)
	ctx.DrawText(width/2, height*0.06, footerTxt)

	// 6. Encode Output
	buf := new(bytes.Buffer)
	jpegOpts := &jpeg.Options{Quality: 90}
	writerFn := renderers.JPEG(canvas.DPMM(1.0), jpegOpts)
	if err := writerFn(buf, c); err != nil {
		return nil, fmt.Errorf("failed to encode jpeg: %w", err)
	}

	return buf.Bytes(), nil
}
