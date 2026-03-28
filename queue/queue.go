package queue

import (
	"sync"

	"github.com/zakaria-chahboun/AyatDesingBot/quran"
)

// ─── Job types ────────────────────────────────────────────────────────────────

type TextJob struct {
	ChatID    int64
	MsgID     int
	SurahNum  int
	SurahName string
	StartAyah int
	EndAyah   int
	Verses    []quran.Verse
	UserID    int64
	Username  string
	FullName  string
}

type ImageJob struct {
	ChatID    int64
	MsgID     int
	SurahNum  int
	SurahName string
	StartAyah int
	EndAyah   int
	Verses    []quran.Verse
	StyleID   string
	FontPath  string
	UserID    int64
	Username  string
	FullName  string
}

type VideoJob struct {
	ChatID    int64
	MsgID     int
	SurahNum  int
	SurahName string
	StartAyah int
	EndAyah   int
	Verses    []quran.Verse
	StyleID   string
	ReciterID string
	FontPath  string
	UserID    int64
	Username  string
	FullName  string
	Bypass    bool
}

// ─── Job result ───────────────────────────────────────────────────────────────

type JobType string

const (
	JobTypeText  JobType = "text"
	JobTypeImage JobType = "image"
	JobTypeVideo JobType = "video"
)

type JobResult struct {
	ChatID    int64
	MsgID     int
	SurahName string
	StartAyah int
	EndAyah   int
	Type      JobType
	Text      string // text jobs only
	FileBytes []byte // image/video jobs only
	Err       error
	UserID    int64
	Username  string
	FullName  string
	StartTime int64 // Unix timestamp in milliseconds when job was submitted
}

// ─── Active-user guard ────────────────────────────────────────────────────────

var (
	activeMu    sync.Mutex
	activeUsers = make(map[int64]struct{})
)

// TryAcquire marks the user as having an active job.
// Returns false if the user already has one in flight — caller must reject.
func TryAcquire(chatID int64) bool {
	activeMu.Lock()
	defer activeMu.Unlock()
	if _, busy := activeUsers[chatID]; busy {
		return false
	}
	activeUsers[chatID] = struct{}{}
	return true
}

// Release clears the user's active-job flag when a job finishes or errors.
func Release(chatID int64) {
	activeMu.Lock()
	defer activeMu.Unlock()
	delete(activeUsers, chatID)
}

// ─── Buffered job channels ────────────────────────────────────────────────────

var (
	TextCh  chan TextJob
	ImageCh chan ImageJob
	VideoCh chan VideoJob
)

// Init creates the buffered channels. Call once at startup with sizes from config.
func Init(textSize, imageSize, videoSize int) {
	TextCh = make(chan TextJob, textSize)
	ImageCh = make(chan ImageJob, imageSize)
	VideoCh = make(chan VideoJob, videoSize)
}

// ─── Non-blocking enqueue ─────────────────────────────────────────────────────

func EnqueueText(job TextJob) bool {
	select {
	case TextCh <- job:
		return true
	default:
		return false
	}
}

func EnqueueImage(job ImageJob) bool {
	select {
	case ImageCh <- job:
		return true
	default:
		return false
	}
}

func EnqueueVideo(job VideoJob) bool {
	select {
	case VideoCh <- job:
		return true
	default:
		return false
	}
}

// ─── Queue length helpers ─────────────────────────────────────────────────────

func TextQueueLen() int  { return len(TextCh) }
func ImageQueueLen() int { return len(ImageCh) }
func VideoQueueLen() int { return len(VideoCh) }
