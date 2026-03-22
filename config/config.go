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

type Config struct {
	BotToken string    `json:"bot_token"`
	Styles   []Style   `json:"styles"`
	Reciters []Reciter `json:"reciters"`
}

var AppConfig Config

func Load(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &AppConfig)
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
