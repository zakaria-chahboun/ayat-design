package video

import (
	"crypto/rand"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/zakaria-chahboun/AyatDesingBot/config"
	"github.com/zakaria-chahboun/AyatDesingBot/quran"
)

const (
	everyAyahBaseURL  = "https://everyayah.com/data/"
	fetchTimeout      = 15 * time.Second
	leadPadding       = 0.5 // image shown before audio starts (seconds)
	trailPadding      = 0.5 // image shown after audio ends (seconds)
	crossfadeDuration = 0.3
)

func GenerateVideo(
	surahNum int,
	verses []quran.Verse,
	images [][]byte,
	reciterFolder string,
	style config.Style,
	fontPath string,
	userID int64,
	bypass bool,
) ([]byte, error) {
	if len(verses) > 3 && !bypass {
		return nil, fmt.Errorf("الحد الأقصى للآيات في الفيديو هو ٣ آيات")
	}

	randStr := rand.Text()
	tempDir := filepath.Join(os.TempDir(), fmt.Sprintf("ayat_video_%d_%s", userID, randStr))
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return nil, fmt.Errorf("فشل في إنشاء مجلد مؤقت: %w", err)
	}
	defer os.RemoveAll(tempDir)

	for i, img := range images {
		framePath := filepath.Join(tempDir, fmt.Sprintf("frame_%03d.png", i))
		if err := os.WriteFile(framePath, img, 0644); err != nil {
			return nil, fmt.Errorf("فشل في كتابة الصورة المؤقتة: %w", err)
		}
	}

	var durations []float64
	for i, v := range verses {
		surahPadded := strconv.Itoa(surahNum)
		versePadded := strconv.Itoa(v.ID)
		for len(surahPadded) < 3 {
			surahPadded = "0" + surahPadded
		}
		for len(versePadded) < 3 {
			versePadded = "0" + versePadded
		}

		audioURL := fmt.Sprintf("%s%s/%s%s.mp3", everyAyahBaseURL, reciterFolder, surahPadded, versePadded)
		audioPath := filepath.Join(tempDir, fmt.Sprintf("audio_%03d.mp3", i))

		if config.AppConfig.Cache.Audio {
			cachePath := filepath.Join("cache", "audio", reciterFolder, fmt.Sprintf("%s%s.mp3", surahPadded, versePadded))
			if err := getCachedAudio(audioURL, cachePath, audioPath); err != nil {
				return nil, fmt.Errorf("فشل في تحميل التلاوة للآية %d: %w", v.ID, err)
			}
		} else {
			if err := downloadFile(audioURL, audioPath); err != nil {
				return nil, fmt.Errorf("فشل في تحميل التلاوة للآية %d: %w", v.ID, err)
			}
		}

		dur, err := getAudioDuration(audioPath)
		if err != nil {
			return nil, fmt.Errorf("فشل في قراءة مدة التلاوة: %w", err)
		}
		durations = append(durations, dur)
		slog.Info("Audio ready", "verse_id", v.ID, "duration_sec", dur)
	}

	outputPath := filepath.Join(tempDir, "output.mp4")
	if err := buildAndRunFFmpeg(tempDir, len(verses), durations, outputPath); err != nil {
		return nil, fmt.Errorf("فشل في إنشاء الفيديو: %w", err)
	}

	videoBytes, err := os.ReadFile(outputPath)
	if err != nil {
		return nil, fmt.Errorf("فشل في قراءة الفيديو: %w", err)
	}

	return videoBytes, nil
}

func downloadFile(url, destPath string) error {
	client := &http.Client{Timeout: fetchTimeout}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("خطأ في الاتصال: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return fmt.Errorf("التلاوة غير متوفرة")
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("استجابة غير متوقعة: %d", resp.StatusCode)
	}

	outFile, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, resp.Body)
	return err
}

func getCachedAudio(url, cachePath, tempPath string) error {
	if _, err := os.Stat(cachePath); err == nil {
		slog.Info("Audio found in cache", "path", cachePath)
		return copyFile(cachePath, tempPath)
	}

	slog.Info("Audio not in cache, downloading", "url", url)

	if err := os.MkdirAll(filepath.Dir(cachePath), 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	if err := downloadFile(url, cachePath); err != nil {
		return fmt.Errorf("failed to download audio: %w", err)
	}

	return copyFile(cachePath, tempPath)
}

func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}

func getAudioDuration(audioPath string) (float64, error) {
	cmd := exec.Command("ffprobe", "-v", "error", "-show_entries", "format=duration", "-of", "csv=p=0", audioPath)
	output, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("ffprobe failed: %w", err)
	}

	durStr := strings.TrimSpace(string(output))
	dur, err := strconv.ParseFloat(durStr, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse duration: %w", err)
	}

	return dur, nil
}

func buildAndRunFFmpeg(tempDir string, numVerses int, durations []float64, outputPath string) error {
	var args []string

	// Calculate per-frame display durations:
	// first frame gets +leadPadding, last frame gets +trailPadding
	frameDurations := make([]float64, numVerses)
	for i := 0; i < numVerses; i++ {
		frameDurations[i] = durations[i]
		if i == 0 {
			frameDurations[i] += leadPadding
		}
		if i == numVerses-1 {
			frameDurations[i] += trailPadding
		}
	}

	for i := 0; i < numVerses; i++ {
		framePath := filepath.Join(tempDir, fmt.Sprintf("frame_%03d.png", i))
		args = append(args, "-loop", "1", "-t", fmt.Sprintf("%.3f", frameDurations[i]), "-i", framePath)
	}

	for i := 0; i < numVerses; i++ {
		audioPath := filepath.Join(tempDir, fmt.Sprintf("audio_%03d.mp3", i))
		args = append(args, "-i", audioPath)
	}

	totalDur := 0.0
	for _, d := range frameDurations {
		totalDur += d
	}
	if numVerses > 1 {
		totalDur -= float64(numVerses-1) * crossfadeDuration
	}

	filterComplex, audioConcat := buildFilterGraph(numVerses, durations, frameDurations, totalDur)
	args = append(args, "-filter_complex", filterComplex)
	args = append(args, "-map", "[vout]", "-map", fmt.Sprintf("[%s]", audioConcat))
	args = append(args, "-c:v", "libx264", "-preset", "veryfast", "-crf", "27", "-tune", "stillimage")
	args = append(args, "-r", "24")
	args = append(args, "-c:a", "aac", "-b:a", "96k")
	args = append(args, "-pix_fmt", "yuv420p", "-movflags", "+faststart")
	args = append(args, "-y", outputPath)

	cmd := exec.Command("ffmpeg", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ffmpeg error: %s", string(output))
	}

	return nil
}

func buildFilterGraph(numVerses int, durations []float64, frameDurations []float64, totalDur float64) (string, string) {
	scaleFilter := "scale=720:-2:force_original_aspect_ratio=decrease,pad=720:1280:(ow-iw)/2:(oh-ih)/2"

	if numVerses == 1 {
		// Single frame: no crossfade needed, just scale.
		// Audio: prepend leadPadding silence via adelay, append trailPadding via apad.
		filter := fmt.Sprintf(
			"[0:v]%s[vout];"+
				"[1:a]adelay=%.0f:all=1[adelayed];"+
				"[adelayed]apad=pad_dur=%.3f[aout]",
			scaleFilter,
			leadPadding*1000,
			trailPadding,
		)
		return filter, "aout"
	}

	// --- Video: xfade crossfades between frames ---
	var videoParts []string
	prevOutput := "[0:v]"

	offset := frameDurations[0] - crossfadeDuration
	for i := 1; i < numVerses; i++ {
		var outLabel string
		if i == numVerses-1 {
			outLabel = "prescale"
		} else {
			outLabel = fmt.Sprintf("xf%d", i)
		}
		videoParts = append(videoParts, fmt.Sprintf(
			"%s[%d:v]xfade=transition=fade:duration=%.1f:offset=%.3f[%s]",
			prevOutput, i, crossfadeDuration, offset, outLabel,
		))
		offset += frameDurations[i] - crossfadeDuration
		prevOutput = fmt.Sprintf("[%s]", outLabel)
	}
	videoParts = append(videoParts, fmt.Sprintf("[prescale]%s[vout]", scaleFilter))

	// --- Audio: silence pads + concat ---
	// First audio: prepend leadPadding silence via adelay.
	// Last audio: append trailPadding silence via apad.
	// Middle: pass through unchanged.
	var audioParts []string
	var audioLabels []string
	for i := 0; i < numVerses; i++ {
		inputIdx := numVerses + i
		inLabel := fmt.Sprintf("[%d:a]", inputIdx)
		outLabel := fmt.Sprintf("a%d", i)

		switch {
		case i == 0:
			audioParts = append(audioParts, fmt.Sprintf("%sadelay=%.0f:all=1[%s]", inLabel, leadPadding*1000, outLabel))
		case i == numVerses-1:
			audioParts = append(audioParts, fmt.Sprintf("%sapad=pad_dur=%.3f[%s]", inLabel, trailPadding, outLabel))
		default:
			audioParts = append(audioParts, fmt.Sprintf("%sanull[%s]", inLabel, outLabel))
		}
		audioLabels = append(audioLabels, fmt.Sprintf("[%s]", outLabel))
	}
	audioConcat := "aout"
	audioChain := fmt.Sprintf("%sconcat=n=%d:v=0:a=1[%s]", strings.Join(audioLabels, ""), numVerses, audioConcat)

	filterComplex := strings.Join(videoParts, ";") + ";" + strings.Join(audioParts, ";") + ";" + audioChain
	return filterComplex, audioConcat
}

var ffmpegAvailable bool

func CheckFFmpeg() bool {
	cmd := exec.Command("ffmpeg", "-version")
	if err := cmd.Run(); err != nil {
		ffmpegAvailable = false
	} else {
		ffmpegAvailable = true
	}
	return ffmpegAvailable
}

func IsFFmpegAvailable() bool {
	return ffmpegAvailable
}
