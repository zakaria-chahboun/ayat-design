package config

import (
	"encoding/json"
	"os"
)

type Style struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	BackgroundImage string  `json:"background_image"`
	BlurValue       float64 `json:"blur_value"`
	TextColor       string  `json:"text_color"`
	IsNew           bool    `json:"isNew"`
}

type Reciter struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Folder string `json:"folder"`
	IsNew  bool   `json:"isNew"`
}

type Cache struct {
	Audio bool `json:"audio"`
}

type Queue struct {
	TextWorkers    int `json:"text_workers"`
	ImageWorkers   int `json:"image_workers"`
	VideoWorkers   int `json:"video_workers"`
	TextQueueSize  int `json:"text_queue_size"`
	ImageQueueSize int `json:"image_queue_size"`
	VideoQueueSize int `json:"video_queue_size"`
}

// Limits defines the maximum number of verses allowed per output type.
type Limits struct {
	TextVerses  int `json:"text_verses"`
	ImageVerses int `json:"image_verses"`
	VideoVerses int `json:"video_verses"`
}

type Config struct {
	Styles   []Style   `json:"styles"`
	Reciters []Reciter `json:"reciters"`
	Cache    Cache     `json:"cache"`
	Queue    Queue     `json:"queue"`
	Limits   Limits    `json:"limits"`
}

var AppConfig Config
var BotToken string
var PocketBaseURL string
var PocketBaseEmail string
var PocketBasePassword string
var PocketBaseCollection string

func Load(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, &AppConfig); err != nil {
		return err
	}
	BotToken = os.Getenv("BOT_TOKEN")
	PocketBaseURL = os.Getenv("POCKETBASE_URL")
	PocketBaseEmail = os.Getenv("POCKETBASE_EMAIL")
	PocketBasePassword = os.Getenv("POCKETBASE_PASSWORD")
	PocketBaseCollection = os.Getenv("POCKETBASE_COLLECTION")
	if PocketBaseCollection == "" {
		PocketBaseCollection = "ayatbot_activites"
	}

	// Queue defaults
	if AppConfig.Queue.TextWorkers <= 0 {
		AppConfig.Queue.TextWorkers = 5
	}
	if AppConfig.Queue.ImageWorkers <= 0 {
		AppConfig.Queue.ImageWorkers = 3
	}
	if AppConfig.Queue.VideoWorkers <= 0 {
		AppConfig.Queue.VideoWorkers = 1
	}
	if AppConfig.Queue.TextQueueSize <= 0 {
		AppConfig.Queue.TextQueueSize = 200
	}
	if AppConfig.Queue.ImageQueueSize <= 0 {
		AppConfig.Queue.ImageQueueSize = 20
	}
	if AppConfig.Queue.VideoQueueSize <= 0 {
		AppConfig.Queue.VideoQueueSize = 5
	}

	// Limits defaults
	if AppConfig.Limits.TextVerses <= 0 {
		AppConfig.Limits.TextVerses = 30
	}
	if AppConfig.Limits.ImageVerses <= 0 {
		AppConfig.Limits.ImageVerses = 10
	}
	if AppConfig.Limits.VideoVerses <= 0 {
		AppConfig.Limits.VideoVerses = 3
	}

	return nil
}

func GetStyleByID(id string) Style {
	for _, s := range AppConfig.Styles {
		if s.ID == id {
			return s
		}
	}
	if len(AppConfig.Styles) > 0 {
		return AppConfig.Styles[0]
	}
	return Style{}
}

func GetReciterByID(id string) Reciter {
	for _, r := range AppConfig.Reciters {
		if r.ID == id {
			return r
		}
	}
	if len(AppConfig.Reciters) > 0 {
		return AppConfig.Reciters[0]
	}
	return Reciter{}
}

func IsBypassKeyword(word string) bool {
	if word == "" {
		return false
	}
	return os.Getenv("BYPASS_KEYWORD") == word
}
