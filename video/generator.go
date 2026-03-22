package video

import (
	"crypto/rand"
	"fmt"
	"io"
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
	silencePadding    = 0.3
	fadeInDuration    = 1.5
	crossfadeDuration = 0.5
	fadeOutDuration   = 2.0
)

func GenerateVideo(
	surahNum int,
	verses []quran.Verse,
	images [][]byte,
	reciterFolder string,
	style config.Style,
	fontPath string,
	userID int64,
) ([]byte, error) {
	if len(verses) > 3 {
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

		if err := downloadFile(audioURL, audioPath); err != nil {
			return nil, fmt.Errorf("فشل في تحميل التلاوة للآية %d: %w", v.ID, err)
		}

		dur, err := getAudioDuration(audioPath)
		if err != nil {
			return nil, fmt.Errorf("فشل في قراءة مدة التلاوة: %w", err)
		}
		durations = append(durations, dur)
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

	for i := 0; i < numVerses; i++ {
		framePath := filepath.Join(tempDir, fmt.Sprintf("frame_%03d.png", i))
		args = append(args, "-loop", "1", "-t", fmt.Sprintf("%.3f", durations[i]+silencePadding), "-i", framePath)
	}

	for i := 0; i < numVerses; i++ {
		audioPath := filepath.Join(tempDir, fmt.Sprintf("audio_%03d.mp3", i))
		args = append(args, "-i", audioPath)
	}

	totalDur := 0.0
	for _, d := range durations {
		totalDur += d + silencePadding
	}
	totalDur -= silencePadding

	filterComplex, audioConcat := buildFilterGraph(numVerses, durations, totalDur)
	args = append(args, "-filter_complex", filterComplex)
	args = append(args, "-map", "[vout]", "-map", fmt.Sprintf("[%s]", audioConcat))
	args = append(args, "-c:v", "libx264", "-preset", "fast", "-crf", "23")
	args = append(args, "-c:a", "aac", "-b:a", "128k")
	args = append(args, "-pix_fmt", "yuv420p")
	args = append(args, "-movflags", "+faststart")
	args = append(args, "-y", outputPath)

	cmd := exec.Command("ffmpeg", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ffmpeg error: %s", string(output))
	}

	return nil
}

func buildFilterGraph(numVerses int, durations []float64, totalDur float64) (string, string) {
	if numVerses == 1 {
		fadeOutStart := totalDur - fadeOutDuration
		filter := fmt.Sprintf(
			"[0:v]fade=t=in:st=0:d=%.1f,fade=t=out:st=%.3f:d=%.1f[vout];[1:a]anull[aout]",
			fadeInDuration, fadeOutStart, fadeOutDuration)
		return filter, "aout"
	}

	var videoParts []string
	prevOutput := "[0:v]"

	videoParts = append(videoParts, fmt.Sprintf("[0:v]fade=t=in:st=0:d=%.1f[v0fade]", fadeInDuration))
	prevOutput = "[v0fade]"

	offset := durations[0] + silencePadding - crossfadeDuration
	for i := 1; i < numVerses; i++ {
		videoParts = append(videoParts, fmt.Sprintf("%s[%d:v]xfade=transition=fade:duration=%.1f:offset=%.3f[vout]", prevOutput, i, crossfadeDuration, offset))
		offset += durations[i] + silencePadding - crossfadeDuration
		prevOutput = "[vout]"
	}

	fadeOutStart := totalDur - fadeOutDuration
	videoParts = append(videoParts, fmt.Sprintf("[vout]fade=t=out:st=%.3f:d=%.1f[vout]", fadeOutStart, fadeOutDuration))

	var audioInputs []string
	for i := 0; i < numVerses; i++ {
		audioInputs = append(audioInputs, fmt.Sprintf("[%d:a]", numVerses+i))
	}
	audioConcat := "aout"
	audioChain := fmt.Sprintf("%sconcat=n=%d:v=0:a=1[%s]", strings.Join(audioInputs, ""), numVerses, audioConcat)

	filterComplex := strings.Join(videoParts, ";") + ";" + audioChain

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
