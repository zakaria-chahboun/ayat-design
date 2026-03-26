package queue

import (
	"log/slog"
	"strings"

	"github.com/zakaria-chahboun/AyatDesingBot/config"
	imageGen "github.com/zakaria-chahboun/AyatDesingBot/image"
	videoGen "github.com/zakaria-chahboun/AyatDesingBot/video"
)

// StartWorkers launches all worker pools. Call once at startup.
// results is the shared channel that dispatch.go reads from.
func StartWorkers(results chan<- JobResult, imageWorkers, videoWorkers, textWorkers int) {
	for i := 0; i < textWorkers; i++ {
		go textWorker(results)
	}
	for i := 0; i < imageWorkers; i++ {
		go imageWorker(results)
	}
	for i := 0; i < videoWorkers; i++ {
		go videoWorker(results)
	}
	slog.Info("Workers started",
		slog.Int("text", textWorkers),
		slog.Int("image", imageWorkers),
		slog.Int("video", videoWorkers),
	)
}

// ─── Text worker ──────────────────────────────────────────────────────────────

func textWorker(results chan<- JobResult) {
	for job := range TextCh {
		slog.Info("Text job started", slog.Int64("chat_id", job.ChatID))

		var lines []string
		for _, v := range job.Verses {
			lines = append(lines, v.Text)
		}

		results <- JobResult{
			ChatID:    job.ChatID,
			MsgID:     job.MsgID,
			SurahName: job.SurahName,
			StartAyah: job.StartAyah,
			EndAyah:   job.EndAyah,
			Type:      JobTypeText,
			Text:      strings.Join(lines, "\n"),
		}
	}
}

// ─── Image worker ─────────────────────────────────────────────────────────────

func imageWorker(results chan<- JobResult) {
	for job := range ImageCh {
		slog.Info("Image job started", slog.Int64("chat_id", job.ChatID))

		style := config.GetStyleByID(job.StyleID)

		imgBytes, err := imageGen.GenerateImage(
			job.SurahNum,
			job.SurahName,
			job.StartAyah,
			job.EndAyah,
			job.Verses,
			style,
			job.FontPath,
		)

		results <- JobResult{
			ChatID:    job.ChatID,
			MsgID:     job.MsgID,
			SurahName: job.SurahName,
			StartAyah: job.StartAyah,
			EndAyah:   job.EndAyah,
			Type:      JobTypeImage,
			FileBytes: imgBytes,
			Err:       err,
		}
	}
}

// ─── Video worker ─────────────────────────────────────────────────────────────

func videoWorker(results chan<- JobResult) {
	for job := range VideoCh {
		slog.Info("Video job started", slog.Int64("chat_id", job.ChatID))

		style := config.GetStyleByID(job.StyleID)
		reciter := config.GetReciterByID(job.ReciterID)

		// Generate one image frame per verse.
		var images [][]byte
		var genErr error
		for i, v := range job.Verses {
			imgBytes, err := imageGen.GenerateImage(
				job.SurahNum,
				job.SurahName,
				v.ID, v.ID,
				job.Verses[i:i+1],
				style,
				job.FontPath,
			)
			if err != nil {
				genErr = err
				break
			}
			images = append(images, imgBytes)
		}

		var videoBytes []byte
		if genErr == nil {
			videoBytes, genErr = videoGen.GenerateVideo(
				job.SurahNum,
				job.Verses,
				images,
				reciter.Folder,
				style,
				job.FontPath,
				job.UserID,
				job.Bypass,
			)
		}

		results <- JobResult{
			ChatID:    job.ChatID,
			MsgID:     job.MsgID,
			SurahName: job.SurahName,
			StartAyah: job.StartAyah,
			EndAyah:   job.EndAyah,
			Type:      JobTypeVideo,
			FileBytes: videoBytes,
			Err:       genErr,
		}
	}
}
