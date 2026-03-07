package image

import (
	"github.com/zakaria-chahboun/AyatDesingBot/config"
)

type Style struct {
	ID              string
	Name            string
	BackgroundImage string
	BlurValue       float64
	TextColor       string
}

var PredefinedStyles []Style

func loadStyles() {
	for _, s := range config.AppConfig.Styles {
		PredefinedStyles = append(PredefinedStyles, Style{
			ID:              s.ID,
			Name:            s.Name,
			BackgroundImage: s.BackgroundImage,
			BlurValue:       s.BlurValue,
			TextColor:       s.TextColor,
		})
	}
}

func init() {
	loadStyles()
}

func GetStyleByID(id string) Style {
	for _, s := range PredefinedStyles {
		if s.ID == id {
			return s
		}
	}
	return PredefinedStyles[0]
}
