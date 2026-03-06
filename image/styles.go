package image

import (
	"os"
	"path/filepath"
	"runtime"

	"gopkg.in/yaml.v3"
)

// Style defines the visual appearance of the generated image
type Style struct {
	ID              string  `yaml:"id"`
	Name            string  `yaml:"name"`
	BackgroundImage string  `yaml:"background_image"`
	BlurValue       float64 `yaml:"blur_value"`
	TextColor       string  `yaml:"text_color"`
}

// StylesConfig holds the configuration for styles
type StylesConfig struct {
	Styles []Style `yaml:"styles"`
}

// PredefinedStyles holds the built-in styles available for users.
var PredefinedStyles []Style

// loadStyles loads styles from the YAML file
func loadStyles() {
	_, filename, _, _ := runtime.Caller(0)
	dir := filepath.Dir(filename)
	stylesPath := filepath.Join(dir, "styles.yaml")
	data, err := os.ReadFile(stylesPath)
	if err != nil {
		panic("Failed to read styles.yaml: " + err.Error())
	}
	var config StylesConfig
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		panic("Failed to unmarshal styles.yaml: " + err.Error())
	}
	PredefinedStyles = config.Styles
}

func init() {
	loadStyles()
}

// GetStyleByID returns a style by its ID or the first style as default if not found
func GetStyleByID(id string) Style {
	for _, s := range PredefinedStyles {
		if s.ID == id {
			return s
		}
	}
	// Fallback to first style
	return PredefinedStyles[0]
}
