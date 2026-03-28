package queue

import (
	"bytes"
	"fmt"
	"log/slog"
	"strings"
	"time"

	tele "gopkg.in/telebot.v3"

	"github.com/zakaria-chahboun/AyatDesingBot/config"
	"github.com/zakaria-chahboun/AyatDesingBot/pb"
	"github.com/zakaria-chahboun/AyatDesingBot/utils"
)

const telegramCharLimit = 4000

// Results is the shared channel workers push finished jobs into.
var Results chan JobResult

// InitResults creates the results channel. Call once at startup.
func InitResults(size int) {
	// Buffer = total possible concurrent jobs across all queues.
	Results = make(chan JobResult, size)
}

// ─── Public: submit a job and park a goroutine waiting for its result ─────────

// SubmitText validates the one-active-per-user guard, enqueues the job,
// and spawns a goroutine that blocks until the worker returns a result,
// then delivers it to the user via the bot.
//
// Returns false if the user already has an active job or the queue is full.
func SubmitText(b *tele.Bot, job TextJob) bool {
	if !TryAcquire(job.ChatID) {
		return false
	}

	resultCh := make(chan JobResult, 1)
	go waitAndDeliver(b, job.ChatID, resultCh)

	if !EnqueueText(job) {
		Release(job.ChatID)
		close(resultCh)
		return false
	}

	go routeResult(job.ChatID, resultCh, job.UserID, job.Username, job.FullName, "", "", job.IsHindiNum)
	return true
}

func SubmitImage(b *tele.Bot, job ImageJob) bool {
	if !TryAcquire(job.ChatID) {
		return false
	}

	resultCh := make(chan JobResult, 1)
	go waitAndDeliver(b, job.ChatID, resultCh)

	if !EnqueueImage(job) {
		Release(job.ChatID)
		close(resultCh)
		return false
	}

	styleName := config.GetStyleByID(job.StyleID).Name
	go routeResult(job.ChatID, resultCh, job.UserID, job.Username, job.FullName, styleName, "", job.IsHindiNum)
	return true
}

func SubmitVideo(b *tele.Bot, job VideoJob) bool {
	if !TryAcquire(job.ChatID) {
		return false
	}

	resultCh := make(chan JobResult, 1)
	go waitAndDeliver(b, job.ChatID, resultCh)

	if !EnqueueVideo(job) {
		Release(job.ChatID)
		close(resultCh)
		return false
	}

	styleName := config.GetStyleByID(job.StyleID).Name
	reciterName := config.GetReciterByID(job.ReciterID).Name
	go routeResult(job.ChatID, resultCh, job.UserID, job.Username, job.FullName, styleName, reciterName, job.IsHindiNum)
	return true
}

// ─── Internal: route result from global Results channel to per-job channel ───

// routeResult reads from the shared Results channel until it finds the result
// belonging to chatID, then forwards it. Other results are re-queued.
//
// This is safe because only one job per user can be active at a time,
// so at most one goroutine is waiting for any given chatID.
func routeResult(chatID int64, dest chan<- JobResult, userID int64, username, fullname, styleName, reciterName string, isHindiNum bool) {
	startTime := time.Now().UnixMilli()
	for result := range Results {
		if result.ChatID == chatID {
			result.UserID = userID
			result.Username = username
			result.FullName = fullname
			result.StartTime = startTime
			result.StyleName = styleName
			result.ReciterName = reciterName
			result.IsHindiNum = isHindiNum
			dest <- result
			return
		}
		// Not ours — put it back for the right goroutine to pick up.
		Results <- result
	}
}

// ─── Internal: wait for result and deliver to user ───────────────────────────

func waitAndDeliver(b *tele.Bot, chatID int64, resultCh <-chan JobResult) {
	defer Release(chatID)

	result, ok := <-resultCh
	if !ok {
		// Channel closed — queue was full, nothing to deliver.
		return
	}

	durationMs := time.Now().UnixMilli() - result.StartTime
	ayahRange := fmt.Sprintf("%d", result.StartAyah)
	if result.StartAyah != result.EndAyah {
		ayahRange = fmt.Sprintf("%d-%d", result.StartAyah, result.EndAyah)
	}

	chat := &tele.Chat{ID: chatID}

	// Delete the "waiting…" message.
	if result.MsgID != 0 {
		_ = b.Delete(&tele.Message{ID: result.MsgID, Chat: chat})
	}

	caption := buildCaption(result.SurahName, result.StartAyah, result.EndAyah)

	if result.Err != nil {
		slog.Error("Job failed",
			slog.Int64("chat_id", chatID),
			slog.String("type", string(result.Type)),
			slog.String("err", result.Err.Error()),
		)
		if _, err := b.Send(chat, "⚠️ "+result.Err.Error()); err != nil {
			slog.Error("Error msg send failed", slog.String("err", err.Error()))
		}

		pb.RecordActivity(pb.ActivityData{
			UserID:       result.UserID,
			Username:     result.Username,
			FullName:     result.FullName,
			Action:       string(result.Type),
			Status:       "error",
			ErrorMessage: result.Err.Error(),
			SurahName:    result.SurahName,
			AyahRange:    ayahRange,
			DurationMs:   durationMs,
			StyleName:    result.StyleName,
			ReciterName:  result.ReciterName,
			IsHindiNum:   result.IsHindiNum,
		})
		return
	}

	switch result.Type {
	case JobTypeText:
		escapedCaption := utils.EscapeMarkdownV2(caption)
		textLen := len(escapedCaption) + 10 + len(result.Text)

		if textLen <= telegramCharLimit {
			msg := fmt.Sprintf("%s\n\n```\n%s\n```", escapedCaption, result.Text)
			if _, err := b.Send(chat, msg, tele.ModeMarkdownV2); err != nil {
				slog.Error("Text send failed", slog.String("err", err.Error()))
			}
		} else {
			verses := strings.Split(result.Text, "\n")
			chunks := splitVersesByLimit(verses, telegramCharLimit-20)

			for i, chunk := range chunks {
				if len(chunks) > 1 {
					header := ""
					if i == 0 {
						header = fmt.Sprintf("%s\n(%d/%d)\n\n", caption, i+1, len(chunks))
					} else {
						header = fmt.Sprintf("(%d/%d)\n\n", i+1, len(chunks))
					}
					msg := fmt.Sprintf("%s```\n%s\n```", header, chunk)
					if _, err := b.Send(chat, msg, tele.ModeMarkdown); err != nil {
						slog.Error("Chunk send failed", slog.String("err", err.Error()))
					}
				}
			}
		}

	case JobTypeImage:
		photo := &tele.Photo{
			File:    tele.FromReader(bytes.NewReader(result.FileBytes)),
			Caption: caption,
		}
		if _, err := b.Send(chat, photo); err != nil {
			slog.Error("Image send failed", slog.String("err", err.Error()))
		}

	case JobTypeVideo:
		video := &tele.Video{
			File:    tele.FromReader(bytes.NewReader(result.FileBytes)),
			Caption: caption,
		}
		if _, err := b.Send(chat, video); err != nil {
			slog.Error("Video send failed", slog.String("err", err.Error()))
		}
	}

	pb.RecordActivity(pb.ActivityData{
		UserID:      result.UserID,
		Username:    result.Username,
		FullName:    result.FullName,
		Action:      string(result.Type),
		Status:      "success",
		SurahName:   result.SurahName,
		AyahRange:   ayahRange,
		DurationMs:  durationMs,
		StyleName:   result.StyleName,
		ReciterName: result.ReciterName,
		IsHindiNum:  result.IsHindiNum,
	})

	slog.Info("Job delivered",
		slog.Int64("chat_id", chatID),
		slog.String("type", string(result.Type)),
	)
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func buildCaption(surahName string, startAyah, endAyah int) string {
	if startAyah == endAyah {
		return fmt.Sprintf("سورة %s، الآية (%d)", surahName, startAyah)
	}
	return fmt.Sprintf("سورة %s، الآيات (%d-%d)", surahName, startAyah, endAyah)
}

func splitVersesByLimit(verses []string, limit int) []string {
	var parts []string
	var current strings.Builder

	for _, verse := range verses {
		if current.Len()+len(verse)+1 > limit && current.Len() > 0 {
			parts = append(parts, current.String())
			current.Reset()
		}
		if current.Len() > 0 {
			current.WriteString("\n")
		}
		current.WriteString(verse)
	}

	if current.Len() > 0 {
		parts = append(parts, current.String())
	}
	return parts
}
