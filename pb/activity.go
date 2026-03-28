package pb

import (
	"log/slog"
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

	body := map[string]any{
		"user_id":     data.UserID,
		"username":    data.Username,
		"fullname":    data.FullName,
		"action":      data.Action,
		"status":      data.Status,
		"surah_name":  data.SurahName,
		"ayah_range":  data.AyahRange,
		"duration_ms": data.DurationMs,
	}

	if data.ErrorMessage != "" {
		body["error_message"] = data.ErrorMessage
	}

	if err := doRequestWithRetry(body); err != nil {
		slog.Warn("Failed to record activity", "error", err)
	}
}
