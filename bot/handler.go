package bot

import (
	"bytes"
	"fmt"
	"log/slog"
	"regexp"
	"strconv"
	"strings"

	"github.com/zakaria-chahboun/AyatDesingBot/config"
	"github.com/zakaria-chahboun/AyatDesingBot/image"
	"github.com/zakaria-chahboun/AyatDesingBot/quran"
	tele "gopkg.in/telebot.v3"
)

type RequestData struct {
	SurahNum  int
	StartAyah int
	EndAyah   int
}

var pendingRequests = make(map[int64]RequestData)

func userAttrs(c tele.Context) []any {
	u := c.Sender()
	return []any{
		slog.String("user_id", fmt.Sprintf("%d", u.ID)),
		slog.String("user_name", u.FirstName+" "+u.LastName),
		slog.String("username", u.Username),
	}
}

func RegisterHandlers(b *tele.Bot, _, fontPath string) {
	b.Handle("/start", func(c tele.Context) error {
		slog.Info("User started bot", userAttrs(c)...)
		msg := "مرحباً بك في روبوت آيات 📖\n\nأرسل طلبك بالصيغة التالية:\n[اسم السورة] [رقم الآية أو من-إلى]\n\nمثال:\nالبقرة 2-3\nآل عمران 7"
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

		totalChars := 0
		for _, v := range verses {
			totalChars += len([]rune(v.Text))
		}

		warning := ""
		if totalChars > 600 {
			warning = "\n\n⚠️ ملاحظة: النص طويل جداً وقد يظهر بخط صغير. يُفضل تقليل عدد الآيات."
		}

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

		slog.Info("User requested verses",
			append(userAttrs(c),
				slog.String("surah", surahName),
				slog.Int("surah_num", surahNum),
				slog.Int("start_ayah", startAyah),
				slog.Int("end_ayah", endAyah),
				slog.Int("verse_count", len(verses)),
			)...,
		)

		return c.Send(confirmMsg, menu)
	})

	b.Handle("\fselect_style", func(c tele.Context) error {
		req, ok := pendingRequests[c.Chat().ID]
		if !ok {
			slog.Warn("Expired request", userAttrs(c)...)
			return c.Respond(&tele.CallbackResponse{Text: "انتهت صلاحية الطلب، يرجى إرساله مجدداً."})
		}

		styleID := c.Callback().Data
		selectedStyle := config.GetStyleByID(styleID)

		delete(pendingRequests, c.Chat().ID)

		c.Edit(c.Message().Text + "\n\n⏳ جاري التصميم (" + selectedStyle.Name + ")...")

		slog.Info("User selected style, generating image",
			append(userAttrs(c),
				slog.String("style_id", styleID),
				slog.String("style_name", selectedStyle.Name),
				slog.Int("surah_num", req.SurahNum),
				slog.Int("start_ayah", req.StartAyah),
				slog.Int("end_ayah", req.EndAyah),
			)...,
		)

		verses, surahName, err := quran.FetchAyat(req.SurahNum, req.StartAyah, req.EndAyah)
		if err != nil {
			slog.Error("Failed to fetch verses",
				append(userAttrs(c), slog.String("error", err.Error()))...,
			)
			return c.Send("⚠️ خطأ غير متوقع: " + err.Error())
		}

		imgBytes, err := image.GenerateImage(req.SurahNum, surahName, req.StartAyah, req.EndAyah, verses, selectedStyle, fontPath)
		if err != nil {
			slog.Error("Failed to generate image",
				append(userAttrs(c),
					slog.String("style", selectedStyle.Name),
					slog.String("error", err.Error()),
				)...,
			)
			return c.Send("⚠️ حدث خطأ أثناء إنشاء الصورة.")
		}

		slog.Info("Image generated and sent",
			append(userAttrs(c),
				slog.String("surah", surahName),
				slog.String("style", selectedStyle.Name),
				slog.Int("image_size_bytes", len(imgBytes)),
			)...,
		)

		photo := &tele.Photo{File: tele.FromReader(bytes.NewReader(imgBytes))}
		c.Send(photo)

		return c.Respond()
	})
}
