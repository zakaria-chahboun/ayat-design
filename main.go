package main

import (
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/zakaria-chahboun/AyatDesingBot/bot"
	"github.com/zakaria-chahboun/AyatDesingBot/quran"
	tele "gopkg.in/telebot.v3"
)

func main() {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found or error reading it, falling back to system environment variables")
	}

	if err := quran.LoadQuran("quran.json"); err != nil {
		log.Fatalf("error loading quran: %v", err)
	}

	token := os.Getenv("BOT_TOKEN")
	if token == "" {
		log.Fatal("BOT_TOKEN environment variable is required")
	}

	pref := tele.Settings{
		Token:  token,
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
