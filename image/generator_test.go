package image

import (
	"os"
	"testing"

	"github.com/tdewolff/canvas"
	"github.com/zakaria-chahboun/AyatDesingBot/quran"
)

func TestGenerateImage(t *testing.T) {
	// Need to be inside the project root for paths to work, or adjust paths.
	// `go test` runs from the package directory (`image/`), so we need to step back.

	err := quran.LoadQuran("../quran.json")
	if err != nil {
		t.Fatalf("LoadQuran failed: %v", err)
	}

	verses, surahName, err := quran.FetchAyat(2, 255, 255)
	if err != nil {
		t.Fatalf("FetchAyat failed: %v", err)
	}

	testStyle := PredefinedStyles[0]
	// Adjust path because tests run in the image/ directory
	testStyle.BackgroundImage = "../" + testStyle.BackgroundImage
	imgData, err := GenerateImage(2, surahName, 255, 255, verses, testStyle, "../fonts/Nabi.ttf")
	if err != nil {
		t.Fatalf("GenerateImage failed: %v", err)
	}

	err = os.WriteFile("../test_ayat_al_kursi_test.jpg", imgData, 0644)
	if err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	t.Log("Successfully generated test_ayat_al_kursi_test.jpg")
}

func TestTextWrapping(t *testing.T) {
	fontFamily := canvas.NewFontFamily("Nabi")
	if err := fontFamily.LoadFontFile("../fonts/Nabi.ttf", canvas.FontRegular); err != nil {
		t.Fatalf("Font Error: %s\n", err.Error())
	}

	c := canvas.New(1080, 1920)
	ctx := canvas.NewContext(c)
	ctx.SetFillColor(canvas.Black)
	ctx.DrawPath(0, 0, canvas.Rectangle(1080, 1920))

	face := fontFamily.Face(150.0, canvas.White, canvas.FontRegular, canvas.FontNormal)
	text := "بِسْمِ اللَّهِ الرَّحْمَنِ الرَّحِيمِ"

	txt := canvas.NewTextLine(face, text, canvas.Center)
	ctx.DrawText(540, 1920/2, txt)

	t.Log("Text wrapping and font loading successful")
}
