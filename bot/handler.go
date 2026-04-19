package bot

import (
	"fmt"
	"log/slog"
	"regexp"
	"strconv"
	"strings"

	"github.com/zakaria-chahboun/AyatDesingBot/config"
	"github.com/zakaria-chahboun/AyatDesingBot/pb"
	"github.com/zakaria-chahboun/AyatDesingBot/queue"
	"github.com/zakaria-chahboun/AyatDesingBot/quran"
	"github.com/zakaria-chahboun/AyatDesingBot/utils"
	tele "gopkg.in/telebot.v3"
)

// RequestData holds the in-progress verse request for a chat session.
type RequestData struct {
	SurahNum     int
	SurahName    string
	StartAyah    int
	EndAyah      int
	StyleID      string
	ReciterID    string
	SelectionMsg string
	Bypass       bool
	IsHindiNum   bool
}

// pendingRequests maps chat ID → active request awaiting further input.
var pendingRequests = make(map[int64]RequestData)

// ─── Helpers ──────────────────────────────────────────────────────────────────

func userAttrs(c tele.Context) []any {
	u := c.Sender()
	return []any{
		slog.String("user_id", fmt.Sprintf("%d", u.ID)),
		slog.String("user_name", u.FirstName+" "+u.LastName),
		slog.String("username", u.Username),
	}
}

func userInfo(c tele.Context) (int64, string, string) {
	u := c.Sender()
	fullName := u.FirstName
	if u.LastName != "" {
		fullName += " " + u.LastName
	}
	username := ""
	if u.Username != "" {
		username = "@" + u.Username
	}
	return u.ID, username, fullName
}

func verseCount(start, end int) int {
	return end - start + 1
}

func buildOutputMenu() *tele.ReplyMarkup {
	menu := &tele.ReplyMarkup{}
	btnText := menu.Data("📝 نص", "output_type", "text")
	btnImage := menu.Data("🖼 صورة", "output_type", "image")
	btnVideo := menu.Data("🎬 فيديو", "output_type", "video")
	menu.Inline(menu.Row(btnText), menu.Row(btnImage), menu.Row(btnVideo))
	return menu
}

func buildStyleMenu() *tele.ReplyMarkup {
	menu := &tele.ReplyMarkup{}
	var rows []tele.Row
	for _, style := range config.AppConfig.Styles {
		name := style.Name
		if style.IsNew {
			name += " ✨"
		}
		btn := menu.Data(name, "select_image_style", style.ID)
		rows = append(rows, menu.Row(btn))
	}
	menu.Inline(rows...)
	return menu
}

func buildVideoStyleMenu() *tele.ReplyMarkup {
	menu := &tele.ReplyMarkup{}
	var rows []tele.Row
	for _, style := range config.AppConfig.Styles {
		name := style.Name
		if style.IsNew {
			name += " ✨"
		}
		btn := menu.Data(name, "select_video_style", style.ID)
		rows = append(rows, menu.Row(btn))
	}
	menu.Inline(rows...)
	return menu
}

func buildReciterMenu() *tele.ReplyMarkup {
	menu := &tele.ReplyMarkup{}
	var rows []tele.Row
	for _, reciter := range config.AppConfig.Reciters {
		name := reciter.Name
		if reciter.IsNew {
			name += " ✨"
		}
		btn := menu.Data(name, "select_reciter", reciter.ID)
		rows = append(rows, menu.Row(btn))
	}
	menu.Inline(rows...)
	return menu
}

// ─── Handler registration ─────────────────────────────────────────────────────

func RegisterHandlers(b *tele.Bot, fontPath string) {

	// /start ──────────────────────────────────────────────────────────────────
	b.Handle("/start", func(c tele.Context) error {
		slog.Info("User started bot", userAttrs(c)...)
		userID, username, fullName := userInfo(c)
		pb.RecordActivity(pb.ActivityData{
			UserID:   userID,
			Username: username,
			FullName: fullName,
			Action:   "start",
			Status:   "success",
		})
		return c.Send(utils.EscapeMarkdownV2(GetStartMessage()), tele.ModeMarkdownV2)
	})

	b.Handle("/notify", func(c tele.Context) error {
		if c.Sender().Username != config.TelegramAdminUsername {
			return nil
		}
		return SendNotification(b, c)
	})

	// Free text: parse surah + ayah range + bypass keyword (optional) ───────────────────────────────
	inputRegex := regexp.MustCompile(`^(.+?)\s+([\d٠-٩]+(?:-[\d٠-٩]+)?)(?:\s+(\S+))?$`)

	b.Handle(tele.OnText, func(c tele.Context) error {
		text := strings.TrimSpace(c.Text())

		matches := inputRegex.FindStringSubmatch(text)
		if len(matches) == 0 {
			return c.Send(GetInvalidFormatMessage())
		}

		surahNameInput := matches[1]
		isHindiNum := utils.ContainsArabicNumerals(matches[2])
		ayahPart := utils.NormalizeNumerals(matches[2])
		bypassKeyword := matches[3]

		surahNum, err := quran.GetSurahByName(surahNameInput)
		if err != nil {
			return c.Send(GetSurahNotFoundMessage(err))
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

		isBypass := config.IsBypassKeyword(bypassKeyword)
		// Check if bypass is detected
		if isBypass {
			slog.Info("Bypass keyword detected", userAttrs(c)...)
		}

		// Reject if the user already has a job in flight.
		if !queue.TryAcquire(c.Chat().ID) {
			return c.Send(GetAlreadyBusyMessage())
		}
		// Release immediately — we only needed to check. The real acquire
		// happens inside queue.Submit* once the user picks an output type.
		queue.Release(c.Chat().ID)

		// Early validation against the most generous limit (text).
		count := verseCount(startAyah, endAyah)
		if count > config.AppConfig.Limits.TextVerses && !isBypass {
			return c.Send(GetTextLimitExceededMessage(config.AppConfig.Limits.TextVerses))
		}

		verses, surahName, err := quran.FetchAyat(surahNum, startAyah, endAyah)
		if err != nil {
			return c.Send(GetVerseRangeErrorMessage(err))
		}

		selectionMsg := GetSelectionMessage(surahName, startAyah, endAyah)

		pendingRequests[c.Chat().ID] = RequestData{
			SurahNum:     surahNum,
			SurahName:    surahName,
			StartAyah:    startAyah,
			EndAyah:      endAyah,
			SelectionMsg: selectionMsg,
			Bypass:       isBypass,
			IsHindiNum:   isHindiNum,
		}

		slog.Info("User requested verses",
			append(userAttrs(c),
				slog.String("surah", surahName),
				slog.Int("surah_num", surahNum),
				slog.Int("start_ayah", startAyah),
				slog.Int("end_ayah", endAyah),
				slog.Int("verse_count", len(verses)),
			)...,
		)

		return c.Send(selectionMsg, buildOutputMenu())
	})

	// Output type selected ────────────────────────────────────────────────────
	b.Handle("\foutput_type", func(c tele.Context) error {
		req, ok := pendingRequests[c.Chat().ID]
		if !ok {
			slog.Warn("Expired request", userAttrs(c)...)
			return c.Respond(&tele.CallbackResponse{Text: GetExpiredRequestMessage()})
		}

		outputType := c.Callback().Data
		count := verseCount(req.StartAyah, req.EndAyah)

		switch outputType {

		case "text":
			if count > config.AppConfig.Limits.TextVerses && !req.Bypass {
				_ = c.Edit(req.SelectionMsg)
				delete(pendingRequests, c.Chat().ID)
				_ = c.Respond()
				return c.Send(GetTextLimitExceededMessage(config.AppConfig.Limits.TextVerses))
			}

			verses, _, err := quran.FetchAyat(req.SurahNum, req.StartAyah, req.EndAyah)
			if err != nil {
				_ = c.Respond()
				return c.Send(GetVerseRangeErrorMessage(err))
			}

			waitMsg, err := b.Edit(c.Message(), GetTextWaitingMessage(req.SelectionMsg))
			if err != nil {
				_ = c.Respond()
				return err
			}

			delete(pendingRequests, c.Chat().ID)
			_ = c.Respond()

			userID, username, fullName := userInfo(c)
			if !queue.SubmitText(b, queue.TextJob{
				ChatID:     c.Chat().ID,
				MsgID:      waitMsg.ID,
				SurahNum:   req.SurahNum,
				SurahName:  req.SurahName,
				StartAyah:  req.StartAyah,
				EndAyah:    req.EndAyah,
				Verses:     verses,
				UserID:     userID,
				Username:   username,
				FullName:   fullName,
				IsHindiNum: req.IsHindiNum,
			}) {
				_ = b.Delete(waitMsg)
				return c.Send(GetQueueFullMessage())
			}

		case "image":
			if count > config.AppConfig.Limits.ImageVerses && !req.Bypass {
				_ = c.Edit(req.SelectionMsg)
				delete(pendingRequests, c.Chat().ID)
				_ = c.Respond()
				return c.Send(GetImageLimitExceededMessage(config.AppConfig.Limits.ImageVerses))
			}
			prompt := req.SelectionMsg + "\n\n" + GetChooseStyleMessage()
			_ = c.Edit(prompt, buildStyleMenu())
			return c.Respond()

		case "video":
			if count > config.AppConfig.Limits.VideoVerses && !req.Bypass {
				_ = c.Edit(GetVideoVerseExceededMessage(req.SelectionMsg, config.AppConfig.Limits.VideoVerses))
				delete(pendingRequests, c.Chat().ID)
				return c.Respond()
			}
			prompt := req.SelectionMsg + "\n\n" + GetChooseStyleMessage()
			_ = c.Edit(prompt, buildVideoStyleMenu())
			return c.Respond()
		}

		return c.Respond()
	})

	// Image style selected ────────────────────────────────────────────────────
	b.Handle("\fselect_image_style", func(c tele.Context) error {
		req, ok := pendingRequests[c.Chat().ID]
		if !ok {
			slog.Warn("Expired request", userAttrs(c)...)
			return c.Respond(&tele.CallbackResponse{Text: GetExpiredRequestMessage()})
		}

		styleID := c.Callback().Data

		verses, _, err := quran.FetchAyat(req.SurahNum, req.StartAyah, req.EndAyah)
		if err != nil {
			_ = c.Respond()
			return c.Send(GetVerseRangeErrorMessage(err))
		}

		waitMsg, err := b.Edit(c.Message(), GetImageWaitingMessage(req.SelectionMsg))
		if err != nil {
			_ = c.Respond()
			return err
		}

		delete(pendingRequests, c.Chat().ID)

		userID, username, fullName := userInfo(c)
		if !queue.SubmitImage(b, queue.ImageJob{
			ChatID:     c.Chat().ID,
			MsgID:      waitMsg.ID,
			SurahNum:   req.SurahNum,
			SurahName:  req.SurahName,
			StartAyah:  req.StartAyah,
			EndAyah:    req.EndAyah,
			Verses:     verses,
			StyleID:    styleID,
			FontPath:   fontPath,
			UserID:     userID,
			Username:   username,
			FullName:   fullName,
			IsHindiNum: req.IsHindiNum,
		}) {
			_ = b.Delete(waitMsg)
			_ = c.Respond()
			return c.Send(GetQueueFullMessage())
		}

		slog.Info("Image job queued",
			append(userAttrs(c),
				slog.String("style_id", styleID),
				slog.Int("queue_pos", queue.ImageQueueLen()),
			)...,
		)
		return c.Respond(&tele.CallbackResponse{Text: GetImageQueuedMessage(queue.ImageQueueLen())})
	})

	// Video style selected ─────────────────────────────────────────────────────
	b.Handle("\fselect_video_style", func(c tele.Context) error {
		req, ok := pendingRequests[c.Chat().ID]
		if !ok {
			slog.Warn("Expired request", userAttrs(c)...)
			return c.Respond(&tele.CallbackResponse{Text: GetExpiredRequestMessage()})
		}

		req.StyleID = c.Callback().Data
		pendingRequests[c.Chat().ID] = req

		prompt := req.SelectionMsg + "\n\n" + GetChooseReciterMessage()
		_ = c.Edit(prompt, buildReciterMenu())
		return c.Respond()
	})

	// Reciter selected ─────────────────────────────────────────────────────────
	b.Handle("\fselect_reciter", func(c tele.Context) error {
		req, ok := pendingRequests[c.Chat().ID]
		if !ok {
			slog.Warn("Expired request", userAttrs(c)...)
			return c.Respond(&tele.CallbackResponse{Text: GetExpiredRequestMessage()})
		}

		reciterID := c.Callback().Data

		verses, _, err := quran.FetchAyat(req.SurahNum, req.StartAyah, req.EndAyah)
		if err != nil {
			_ = c.Respond()
			return c.Send(GetVerseRangeErrorMessage(err))
		}

		waitMsg, err := b.Edit(c.Message(), GetVideoWaitingMessage(req.SelectionMsg))
		if err != nil {
			_ = c.Respond()
			return err
		}

		delete(pendingRequests, c.Chat().ID)

		userID, username, fullName := userInfo(c)
		if !queue.SubmitVideo(b, queue.VideoJob{
			ChatID:     c.Chat().ID,
			MsgID:      waitMsg.ID,
			SurahNum:   req.SurahNum,
			SurahName:  req.SurahName,
			StartAyah:  req.StartAyah,
			EndAyah:    req.EndAyah,
			Verses:     verses,
			StyleID:    req.StyleID,
			ReciterID:  reciterID,
			FontPath:   fontPath,
			UserID:     userID,
			Username:   username,
			FullName:   fullName,
			Bypass:     req.Bypass,
			IsHindiNum: req.IsHindiNum,
		}) {
			_ = b.Delete(waitMsg)
			_ = c.Respond()
			return c.Send(GetQueueFullMessage())
		}

		slog.Info("Video job queued",
			append(userAttrs(c),
				slog.String("reciter_id", reciterID),
				slog.Int("queue_pos", queue.VideoQueueLen()),
			)...,
		)
		return c.Respond(&tele.CallbackResponse{Text: GetVideoQueuedMessage(queue.VideoQueueLen())})
	})
}
