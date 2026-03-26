package main

import (
	"flag"
	"log/slog"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/zakaria-chahboun/AyatDesingBot/bot"
	"github.com/zakaria-chahboun/AyatDesingBot/config"
	"github.com/zakaria-chahboun/AyatDesingBot/queue"
	"github.com/zakaria-chahboun/AyatDesingBot/quran"
	"github.com/zakaria-chahboun/AyatDesingBot/video"
	"github.com/zakaria-chahboun/AyatDesingBot/web"
	tele "gopkg.in/telebot.v3"
)

func main() {
	serveWeb := flag.Bool("web", false, "Serve web landing page")
	flag.Parse()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	if err := godotenv.Load(); err != nil {
		logger.Warn("No .env file found, relying on environment variables")
	}

	if *serveWeb {
		port := os.Getenv("PORT")
		if port == "" {
			port = "8080"
		}
		logger.Info("Starting web server", "port", port)
		if err := web.Run(port); err != nil {
			logger.Error("Web server error", "error", err)
		}
		return
	}

	if err := config.Load("config.json"); err != nil {
		logger.Error("Failed to load config", "error", err)
		os.Exit(1)
	}

	if bypassKeyword := os.Getenv("BYPASS_KEYWORD"); bypassKeyword != "" {
		logger.Info("Bypass keyword is set in the environment")
	}

	if config.AppConfig.Cache.Audio {
		if err := os.MkdirAll("cache/audio", 0755); err != nil {
			logger.Error("Failed to create cache/audio folder", "error", err)
			os.Exit(1)
		}
		logger.Info("Audio caching enabled, cache folder ready")
	}

	if err := quran.LoadQuran("quran.json"); err != nil {
		logger.Error("Failed to load quran", "error", err)
		os.Exit(1)
	}

	if config.BotToken == "" {
		logger.Error("BOT_TOKEN environment variable is required")
		os.Exit(1)
	}

	if video.CheckFFmpeg() {
		logger.Info("FFmpeg is available, video generation enabled")
	} else {
		logger.Warn("FFmpeg not found, video generation is disabled")
	}

	pref := tele.Settings{
		Token:  config.BotToken,
		Poller: &tele.LongPoller{Timeout: 10 * time.Second},
	}

	b, err := tele.NewBot(pref)
	if err != nil {
		logger.Error("Failed to create bot", "error", err)
		os.Exit(1)
	}

	os.MkdirAll("backgrounds", 0755)

	// Init queue channels, result channel, and worker pools.
	cfg := config.AppConfig
	queue.Init(cfg.Queue.TextQueueSize, cfg.Queue.ImageQueueSize, cfg.Queue.VideoQueueSize)
	queue.InitResults(cfg.Queue.TextQueueSize + cfg.Queue.ImageQueueSize + cfg.Queue.VideoQueueSize)
	queue.StartWorkers(queue.Results, cfg.Queue.ImageWorkers, cfg.Queue.VideoWorkers, cfg.Queue.TextWorkers)

	logger.Info("Bot is starting...")
	bot.RegisterHandlers(b, "fonts/Nabi.ttf")

	b.Start()
}
