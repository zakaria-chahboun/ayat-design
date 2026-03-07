package bot

import (
	"bytes"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"

	"github.com/zakaria-chahboun/AyatDesingBot/config"
	"github.com/zakaria-chahboun/AyatDesingBot/image"
	"github.com/zakaria-chahboun/AyatDesingBot/quran"
	tele "gopkg.in/telebot.v3"
)

// RequestData holds user's parsed request pending confirmation
type RequestData struct {
	SurahNum  int
	StartAyah int
	EndAyah   int
}

var pendingRequests = make(map[int64]RequestData)

// RegisterHandlers registers all Telegram bot handlers
func RegisterHandlers(b *tele.Bot, _, fontPath string) {
	b.Handle("/start", func(c tele.Context) error {
		msg := "مرحباً بك في آياتي 📖\n\nأرسل طلبك بالصيغة التالية:\n[اسم السورة] [رقم الآية أو من-إلى]\n\nمثال:\nالبقرة 2-3\nآل عمران 7"
		return c.Send(msg)
	})

	inputRegex := regexp.MustCompile(`^(.+?)\s+(\d+(?:-\d+)?)$`)

	b.Handle(tele.OnText, func(c tele.Context) error {
		text := strings.TrimSpace(c.Text())

		matches := inputRegex.FindStringSubmatch(text)
		if len(matches) == 0 {
			return c.Send("⚠️ صيغة غير صحيحة. يرجى استخدام:\n[اسم السورة] [الآية أو من-إلى]\nمثال: البقرة 2-3")
		}

		surahNameInput := matches[1]
		ayahPart := matches[2]

		surahNum, err := quran.GetSurahByName(surahNameInput)
		if err != nil {
			return c.Send("⚠️ " + err.Error())
		}

		var startAyah, endAyah int
		if strings.Contains(ayahPart, "-") {
			parts := strings.Split(ayahPart, "-")
			startAyah, _ = strconv.Atoi(parts[0])
			endAyah, _ = strconv.Atoi(parts[1])
		} else {
			startAyah, _ = strconv.Atoi(ayahPart)
			endAyah = startAyah
		}

		verses, surahName, err := quran.FetchAyat(surahNum, startAyah, endAyah)
		if err != nil {
			return c.Send("⚠️ " + err.Error())
		}

		// Check complexity warning
		totalChars := 0
		for _, v := range verses {
			totalChars += len([]rune(v.Text))
		}

		warning := ""
		if totalChars > 600 {
			warning = "\n\n⚠️ ملاحظة: النص طويل جداً وقد يظهر بخط صغير. يُفضل تقليل عدد الآيات."
		}

		// Setup confirmation
		pendingRequests[c.Chat().ID] = RequestData{
			SurahNum:  surahNum,
			StartAyah: startAyah,
			EndAyah:   endAyah,
		}

		confirmMsg := fmt.Sprintf("اختر تصميم الصورة لسورة %s من الآية %s إلى الآية %s:",
			surahName,
			quran.ConvertToArabicIndic(strconv.Itoa(startAyah)),
			quran.ConvertToArabicIndic(strconv.Itoa(endAyah)))

		if startAyah == endAyah {
			confirmMsg = fmt.Sprintf("اختر تصميم الصورة لسورة %s الآية %s:",
				surahName,
				quran.ConvertToArabicIndic(strconv.Itoa(startAyah)))
		}

		confirmMsg += warning

		menu := &tele.ReplyMarkup{}
		var rows []tele.Row
		for _, style := range config.AppConfig.Styles {
			btn := menu.Data(style.Name, "select_style", style.ID)
			rows = append(rows, menu.Row(btn))
		}
		menu.Inline(rows...)

		return c.Send(confirmMsg, menu)
	})

	b.Handle("\fselect_style", func(c tele.Context) error {
		req, ok := pendingRequests[c.Chat().ID]
		if !ok {
			return c.Respond(&tele.CallbackResponse{Text: "انتهت صلاحية الطلب، يرجى إرساله مجدداً."})
		}

		styleID := c.Callback().Data
		selectedStyle := config.GetStyleByID(styleID)

		// Clean up state
		delete(pendingRequests, c.Chat().ID)

		// Remove button and edit text
		c.Edit(c.Message().Text + "\n\n⏳ جاري التصميم (" + selectedStyle.Name + ")...")

		verses, surahName, err := quran.FetchAyat(req.SurahNum, req.StartAyah, req.EndAyah)
		if err != nil {
			return c.Send("⚠️ خطأ غير متوقع: " + err.Error())
		}

		imgBytes, err := image.GenerateImage(req.SurahNum, surahName, req.StartAyah, req.EndAyah, verses, selectedStyle, fontPath)
		if err != nil {
			log.Printf("Error generating image %s: %v", selectedStyle.Name, err)
			return c.Send("⚠️ حدث خطأ أثناء إنشاء الصورة.")
		}

		photo := &tele.Photo{File: tele.FromReader(bytes.NewReader(imgBytes))}
		c.Send(photo)

		return c.Respond()
	})
}
