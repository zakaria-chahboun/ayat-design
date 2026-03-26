package queue

import (
	"bytes"
	"fmt"
	"log/slog"
	"strings"

	tele "gopkg.in/telebot.v3"
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
		if _, err := b.Send(chat, "⚠️ "+result.Err.Error()); err != nil {
			slog.Error("Error msg send failed", slog.String("err", err.Error()))
		}
		return
	}

	switch result.Type {
	case JobTypeText:
		escapedCaption := escapeMarkdownV2(caption)
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
