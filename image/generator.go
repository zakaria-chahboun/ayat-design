package image

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"

	"github.com/disintegration/imaging"
	"github.com/zakaria-chahboun/AyatDesingBot/quran"

	"github.com/tdewolff/canvas"
	"github.com/tdewolff/canvas/renderers"
)

// GenerateImage creates a stylized image of Quran verses natively
func GenerateImage(surahNum int, surahName string, startAyah, endAyah int, verses []quran.Verse, style Style, fontPath string) ([]byte, error) {
	openingText := fmt.Sprintf("surah%03d", surahNum)

	var versesText string
	for _, v := range verses {
		versesText += fmt.Sprintf("%s ﴿%s﴾ ", v.Text, quran.ConvertToArabicIndic(fmt.Sprintf("%d", v.ID)))
	}

	footerText := "تيليجرام AyatDesignBot"

	// 1. Setup Fonts
	fontDir := filepath.Dir(fontPath)
	nabiFontPath := filepath.Join(fontDir, "Nabi.ttf")
	openingFontPath := filepath.Join(fontDir, "surah-name-v2.ttf")
	footerFontPath := filepath.Join(fontDir, "Tajawal-Regular.ttf")

	// Use Nabi for verses
	fontFamily := canvas.NewFontFamily("Nabi")
	if err := fontFamily.LoadFontFile(nabiFontPath, canvas.FontRegular); err != nil {
		return nil, fmt.Errorf("could not load font %s: %w", nabiFontPath, err)
	}

	// Font for opening text
	openingFontFamily := canvas.NewFontFamily("SurahName")
	if err := openingFontFamily.LoadFontFile(openingFontPath, canvas.FontRegular); err != nil {
		return nil, fmt.Errorf("could not load opening font: %w", err)
	}

	// Font for footer
	footerFontFamily := canvas.NewFontFamily("Tajawal")
	if err := footerFontFamily.LoadFontFile(footerFontPath, canvas.FontRegular); err != nil {
		return nil, fmt.Errorf("could not load footer font: %w", err)
	}

	// 2. Setup Canvas
	const width = 1080.0
	const height = 1920.0
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

	// Apply blur if applicable
	if style.BlurValue > 0 {
		bgImg = imaging.Blur(bgImg, style.BlurValue)
	}

	// Calculate aspect-fill scale
	bgW := float64(bgImg.Bounds().Dx())
	bgH := float64(bgImg.Bounds().Dy())
	scale := width / bgW
	if (height / bgH) > scale {
		scale = height / bgH
	}

	// Draw image scaled from bottom-left origin
	// To center it, offset X and Y
	scaledW := bgW * scale
	scaledH := bgH * scale
	offsetX := (width - scaledW) / 2
	offsetY := (height - scaledH) / 2

	// Render Image using a DrawImage or similar API.
	// Actually `tdewolff/canvas` DrawImage is straightforward:
	// Let's create an affine transform to scale and center.
	ctx.SetFillColor(canvas.Black) // Background color fallback
	ctx.DrawPath(0, 0, canvas.Rectangle(width, height))

	// Create an Image element
	// Wait, tdewolff/canvas doesn't use standard DrawImage matrix right away,
	// let's do Push/SetMatrix/DrawImage/Pop or just draw.
	// `ctx.DrawImage(offsetX, offsetY, img, resolution)` - Actually resolution is dpi/dpm.
	// Since we want `img` to fit exactly `scaledW` and `scaledH`,
	// The resolution specifies pixels per mm. Default canvas unit is mm.
	// If canvas width is 1080 mm, resolution = bgW / scaledW.
	imgResolution := canvas.Resolution(bgW / scaledW)
	ctx.DrawImage(offsetX, offsetY, bgImg, imgResolution)

	// 4. Draw Overlay (50% opacity)
	var textColor color.Color = canvas.White
	if style.TextColor == "black" {
		textColor = canvas.Black
		ctx.SetFillColor(canvas.RGBA(255, 255, 255, 128)) // Light overlay for black text
	} else {
		ctx.SetFillColor(canvas.RGBA(0, 0, 0, 128)) // Dark overlay for white text
	}
	ctx.DrawPath(0, 0, canvas.Rectangle(width, height))

	// 5. Draw Texts
	// tdewolff/canvas v3 handles BiDi automatically

	// --- Opening Text ---
	openingFace := openingFontFamily.Face(350.0, textColor, canvas.FontRegular, canvas.FontNormal)
	openingTxt := canvas.NewTextLine(openingFace, openingText, canvas.Center)
	ctx.DrawText(width/2, height*0.87, openingTxt)

	// --- Main Verses ---
	// Start with a very large font size and scale down until the text fits in the vertical box bounds
	boxWidth := width * 0.85
	maxBoxHeight := height * 0.70 // Maximum vertical space for verses

	fontSize := 250.0
	var versesTxt *canvas.Text
	var versesBounds canvas.Rect

	// Dynamic line spacing: Add more space between lines for shorter text
	var lineStretch float64 = 0.0
	charsCount := len([]rune(versesText))
	if charsCount < 80 {
		lineStretch = 0.8
	} else if charsCount < 200 {
		lineStretch = 0.4
	} else if charsCount < 400 {
		lineStretch = 0.2
	}
	opts := &canvas.TextOptions{LineStretch: lineStretch}

	for fontSize > 30.0 {
		verseFace := fontFamily.Face(fontSize, textColor, canvas.FontRegular, canvas.FontNormal)

		// Create a text box. This auto-wraps text if it exceeds boxWidth.
		versesTxt = canvas.NewTextBox(verseFace, versesText, boxWidth, 0.0, canvas.Center, canvas.Top, opts)
		versesBounds = versesTxt.Bounds()

		// A text box fits if its visual Height is less than our allowed maxBoxHeight
		if versesBounds.H() <= maxBoxHeight && versesBounds.W() <= boxWidth {
			break
		}

		fontSize -= 5.0
	}

	// Center the text box physically
	// versesTxt uses Top-Left origin natively when instantiated via NewTextBox with canvas.Top.
	// We want its center to be at the center of the canvas
	versesX := (width - versesBounds.W()) / 2
	versesY := (height + versesBounds.H()) / 2
	ctx.DrawText(versesX, versesY, versesTxt)

	// --- Footer Text ---
	footerFace := footerFontFamily.Face(120.0, textColor, canvas.FontRegular, canvas.FontNormal)
	footerTxt := canvas.NewTextLine(footerFace, footerText, canvas.Center)
	ctx.DrawText(width/2, height*0.06, footerTxt)

	// 6. Encode Output
	buf := new(bytes.Buffer)

	// renderers.JPEG returns a Writer function
	writerFn := renderers.JPEG(canvas.DPMM(1.0))
	if err := writerFn(buf, c); err != nil {
		return nil, fmt.Errorf("failed to encode jpeg: %w", err)
	}

	return buf.Bytes(), nil
}
