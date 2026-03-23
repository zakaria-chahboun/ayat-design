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
}

type Reciter struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Folder string `json:"folder"`
}

type Cache struct {
	Audio bool `json:"audio"`
}

type Config struct {
	Styles   []Style   `json:"styles"`
	Reciters []Reciter `json:"reciters"`
	Cache    Cache     `json:"cache"`
}

var AppConfig Config
var BotToken string

func Load(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, &AppConfig); err != nil {
		return err
	}
	BotToken = os.Getenv("BOT_TOKEN")
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
