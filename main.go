package main

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	"github.com/zakaria-chahboun/AyatDesingBot/bot"
	"github.com/zakaria-chahboun/AyatDesingBot/config"
	"github.com/zakaria-chahboun/AyatDesingBot/pb"
	"github.com/zakaria-chahboun/AyatDesingBot/queue"
	"github.com/zakaria-chahboun/AyatDesingBot/quran"
	"github.com/zakaria-chahboun/AyatDesingBot/video"
	"github.com/zakaria-chahboun/cute"
	tele "gopkg.in/telebot.v3"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	if err := godotenv.Load(); err != nil {
		logger.Warn("No .env file found, relying on environment variables")
	}

	if err := config.Load("config.json"); err != nil {
		cute.Check("Config Error", err)
	}

	if err := pb.Init(); err != nil {
		logger.Warn("PocketBase init failed", "error", err)
	}

	list := cute.NewList(cute.BrightYellow, "🪏 Config")
	list.Add(cute.DefaultColor, "Image Workers: "+strconv.Itoa(config.AppConfig.Queue.ImageWorkers))
	list.Add(cute.DefaultColor, "Video Workers: "+strconv.Itoa(config.AppConfig.Queue.VideoWorkers))
	list.Add(cute.DefaultColor, "Text Workers: "+strconv.Itoa(config.AppConfig.Queue.TextWorkers))
	list.Add(cute.DefaultColor, "Text Limit: "+strconv.Itoa(config.AppConfig.Limits.TextVerses))
	list.Add(cute.DefaultColor, "Image Limit: "+strconv.Itoa(config.AppConfig.Limits.ImageVerses))
	list.Add(cute.DefaultColor, "Video Limit: "+strconv.Itoa(config.AppConfig.Limits.VideoVerses))

	if bypass := os.Getenv("BYPASS_KEYWORD"); bypass != "" {
		list.Add(cute.BrightGreen, "✓ Bypass Keyword: set")
	}

	if pb.IsEnabled() {
		list.Add(cute.BrightGreen, "✓ Activity Tracking: enabled")
	} else {
		list.Add(cute.DefaultColor, "○ Activity Tracking: disabled")
	}

	if config.AppConfig.Cache.Audio {
		list.Add(cute.BrightGreen, "✓ Audio Cache: enabled")
		if err := os.MkdirAll("cache/audio", 0755); err != nil {
			cute.Check("Cache Error", err)
		}
	}

	list.Print()

	if video.CheckFFmpeg() {
		cute.Println("✅ FFmpeg", "Video generation enabled")
	} else {
		cute.Println("⚠️  FFmpeg", "Not found, video generation disabled")
	}

	if err := quran.LoadQuran("quran.json"); err != nil {
		cute.Check("Quran Error", err)
	}

	if config.BotToken == "" {
		cute.Check("Config Error", fmt.Errorf("BOT_TOKEN environment variable is required"))
	}

	pref := tele.Settings{
		Token:  config.BotToken,
		Poller: &tele.LongPoller{Timeout: 10 * time.Second},
	}

	b, err := tele.NewBot(pref)
	if err != nil {
		cute.Check("Bot Error", err)
	}

	os.MkdirAll("backgrounds", 0755)

	cfg := config.AppConfig
	queue.Init(cfg.Queue.TextQueueSize, cfg.Queue.ImageQueueSize, cfg.Queue.VideoQueueSize)
	queue.InitResults(cfg.Queue.TextQueueSize + cfg.Queue.ImageQueueSize + cfg.Queue.VideoQueueSize)
	queue.StartWorkers(queue.Results, cfg.Queue.ImageWorkers, cfg.Queue.VideoWorkers, cfg.Queue.TextWorkers)

	logger.Info("Bot is starting...")
	bot.RegisterHandlers(b, "fonts/Nabi.ttf")

	b.Start()
}
