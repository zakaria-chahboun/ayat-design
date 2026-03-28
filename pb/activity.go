package pb

import (
	"log/slog"

	"github.com/chrisbrocklesby/pbclient"
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
	if client == nil {
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

	_, err := pbclient.Collection[map[string]any]("ayat_activities", client).Create(map[string]any{
		"user_id":       activity.UserID,
		"username":      activity.Username,
		"fullname":      activity.FullName,
		"action":        activity.Action,
		"status":        activity.Status,
		"error_message": activity.ErrorMessage,
		"surah_name":    activity.SurahName,
		"ayah_range":    activity.AyahRange,
		"duration_ms":   activity.DurationMs,
	})
	if err != nil {
		slog.Warn("Failed to record activity", "error", err)
	}
}
