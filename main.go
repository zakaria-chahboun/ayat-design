package main

import (
	"log"
	"os"
	"time"

	"github.com/zakaria-chahboun/AyatDesingBot/bot"
	"github.com/zakaria-chahboun/AyatDesingBot/config"
	"github.com/zakaria-chahboun/AyatDesingBot/quran"
	tele "gopkg.in/telebot.v3"
)

func main() {
	if err := config.Load("config.json"); err != nil {
		log.Fatalf("Failed to load config.json: %v", err)
	}

	if err := quran.LoadQuran("quran.json"); err != nil {
		log.Fatalf("error loading quran: %v", err)
	}

	if config.AppConfig.BotToken == "" {
		log.Fatal("bot_token is required in config.json")
	}

	pref := tele.Settings{
		Token:  config.AppConfig.BotToken,
		Poller: &tele.LongPoller{Timeout: 10 * time.Second},
	}

	b, err := tele.NewBot(pref)
	if err != nil {
		log.Fatal(err)
		return
	}

	// Make sure backgrounds directory exists
	os.MkdirAll("backgrounds", 0755)

	log.Println("Bot is starting...")
	bot.RegisterHandlers(b, "backgrounds", "fonts/Nabi.ttf")

	b.Start()
}
