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
}

func RecordActivity(data ActivityData) {
	if !IsEnabled() {
		return
	}

	activity := AyatActivity{
		UserID:     data.UserID,
		Username:   data.Username,
		FullName:   data.FullName,
		Action:     data.Action,
		Status:     data.Status,
		SurahName:  data.SurahName,
		AyahRange:  data.AyahRange,
		DurationMs: data.DurationMs,
	}

	if data.ErrorMessage != "" {
		activity.ErrorMessage = data.ErrorMessage
	}

	_, err := pbclient.Collection[AyatActivity](config.PocketBaseCollection).Create(activity)
	if err != nil {
		slog.Warn("Failed to record activity", "error", err)
	}
}
