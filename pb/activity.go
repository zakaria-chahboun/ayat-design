package pb

import (
	"log/slog"

	"github.com/chrisbrocklesby/pbclient"
	"github.com/zakaria-chahboun/AyatDesingBot/config"
)

type ActivityData struct {
	UserID       int64
	Username     string
	FullName     string
	Action       string
	Status       string
	ErrorMessage string
	SurahName    string
	AyahRange    string
	DurationMs   int64
	StyleName    string
	ReciterName  string
	IsHindiNum   bool
}

type AyatActivity struct {
	UserID       int64  `json:"user_id"`
	Username     string `json:"username"`
	FullName     string `json:"fullname"`
	Action       string `json:"action"`
	Status       string `json:"status"`
	ErrorMessage string `json:"error_message"`
	SurahName    string `json:"surah_name"`
	AyahRange    string `json:"ayah_range"`
	DurationMs   int64  `json:"duration_ms"`
	StyleName    string `json:"style_name"`
	ReciterName  string `json:"reciter_name"`
	IsHindiNum   bool   `json:"is_hindi_num"`
}

func RecordActivity(data ActivityData) {
	if !IsEnabled() {
		return
	}

	activity := AyatActivity{
		UserID:      data.UserID,
		Username:    data.Username,
		FullName:    data.FullName,
		Action:      data.Action,
		Status:      data.Status,
		SurahName:   data.SurahName,
		AyahRange:   data.AyahRange,
		DurationMs:  data.DurationMs,
		StyleName:   data.StyleName,
		ReciterName: data.ReciterName,
		IsHindiNum:  data.IsHindiNum,
	}

	if data.ErrorMessage != "" {
		activity.ErrorMessage = data.ErrorMessage
	}

	_, err := pbclient.Collection[AyatActivity](config.PocketBaseCollection).Create(activity)
	if err != nil {
		slog.Warn("Failed to record activity", "error", err)
	}
}
