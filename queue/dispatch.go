package queue

import (
	"bytes"
	"fmt"
	"log/slog"
	"strings"

	tele "gopkg.in/telebot.v3"
)

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

	// Wire: once the worker pushes to Results, the router below forwards to resultCh.
	go routeResult(job.ChatID, resultCh)
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

	go routeResult(job.ChatID, resultCh)
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

	go routeResult(job.ChatID, resultCh)
	return true
}

// ─── Internal: route result from global Results channel to per-job channel ───

// routeResult reads from the shared Results channel until it finds the result
// belonging to chatID, then forwards it. Other results are re-queued.
//
// This is safe because only one job per user can be active at a time,
// so at most one goroutine is waiting for any given chatID.
func routeResult(chatID int64, dest chan<- JobResult) {
	for result := range Results {
		if result.ChatID == chatID {
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
		_, _ = b.Send(chat, "⚠️ "+result.Err.Error())
		return
	}

	switch result.Type {
	case JobTypeText:
		msg := fmt.Sprintf("%s\n\n```\n%s\n```", escapeMarkdownV2(caption), result.Text)
		_, _ = b.Send(chat, msg, tele.ModeMarkdownV2)

	case JobTypeImage:
		photo := &tele.Photo{
			File:    tele.FromReader(bytes.NewReader(result.FileBytes)),
			Caption: caption,
		}
		_, _ = b.Send(chat, photo)

	case JobTypeVideo:
		video := &tele.Video{
			File:    tele.FromReader(bytes.NewReader(result.FileBytes)),
			Caption: caption,
		}
		_, _ = b.Send(chat, video)
	}

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

var mdV2Replacer = strings.NewReplacer(
	"_", "\\_",
	"*", "\\*",
	"`", "\\`",
	"[", "\\[",
	"]", "\\]",
	"(", "\\(",
	")", "\\)",
	"~", "\\~",
	">", "\\>",
	"#", "\\#",
	"+", "\\+",
	"-", "\\-",
	"=", "\\=",
	"|", "\\|",
	"{", "\\{",
	"}", "\\}",
	".", "\\.",
	"!", "\\!",
)

func escapeMarkdownV2(s string) string {
	return mdV2Replacer.Replace(s)
}
