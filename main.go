package main

import (
	"log/slog"
	"os"
	"time"

	"github.com/zakaria-chahboun/AyatDesingBot/bot"
	"github.com/zakaria-chahboun/AyatDesingBot/config"
	"github.com/zakaria-chahboun/AyatDesingBot/quran"
	tele "gopkg.in/telebot.v3"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	if err := config.Load("config.json"); err != nil {
		logger.Error("Failed to load config", "error", err)
		os.Exit(1)
	}

	if err := quran.LoadQuran("quran.json"); err != nil {
		logger.Error("Failed to load quran", "error", err)
		os.Exit(1)
	}

	if config.AppConfig.BotToken == "" {
		logger.Error("bot_token is required in config.json")
		os.Exit(1)
	}

	pref := tele.Settings{
		Token:  config.AppConfig.BotToken,
		Poller: &tele.LongPoller{Timeout: 10 * time.Second},
	}

	b, err := tele.NewBot(pref)
	if err != nil {
		logger.Error("Failed to create bot", "error", err)
		os.Exit(1)
	}

	os.MkdirAll("backgrounds", 0755)

	logger.Info("Bot is starting...")
	bot.RegisterHandlers(b, "backgrounds", "fonts/Nabi.ttf")

	b.Start()
}
