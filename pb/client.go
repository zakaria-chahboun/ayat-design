package pb

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"reflect"
	"strings"

	"github.com/chrisbrocklesby/pbclient"
	"github.com/zakaria-chahboun/AyatDesingBot/config"
)

func Init() error {
	if config.PocketBaseURL == "" || config.PocketBaseEmail == "" || config.PocketBasePassword == "" {
		var missing []string
		if config.PocketBaseURL == "" {
			missing = append(missing, "POCKETBASE_URL")
		}
		if config.PocketBaseEmail == "" {
			missing = append(missing, "POCKETBASE_EMAIL")
		}
		if config.PocketBasePassword == "" {
			missing = append(missing, "POCKETBASE_PASSWORD")
		}
		slog.Info("Activity tracking disabled", "missing", missing)
		return nil
	}

	client, err := pbclient.NewClient(pbclient.Config{
		BaseURL: config.PocketBaseURL,
	})
	if err != nil {
		return err
	}
	pbclient.SetDefault(client)

	if err := pbclient.LoginUser("users", config.PocketBaseEmail, config.PocketBasePassword); err != nil {
		return err
	}

	checkFields()

	slog.Info("PocketBase initialized")
	return nil
}

func IsEnabled() bool {
	return config.PocketBaseURL != "" && config.PocketBaseEmail != "" && config.PocketBasePassword != ""
}

type collectionSchema struct {
	Schema []struct {
		Name string `json:"name"`
	} `json:"schema"`
}

func checkFields() {
	url := config.PocketBaseURL + "/api/collections/" + config.PocketBaseCollection

	resp, err := http.Get(url)
	if err != nil {
		slog.Warn("Failed to get collection schema", "error", err)
		return
	}
	defer resp.Body.Close()

	var schema collectionSchema
	if err := json.NewDecoder(resp.Body).Decode(&schema); err != nil {
		slog.Warn("Failed to parse collection schema", "error", err)
		return
	}

	existingFields := make(map[string]bool)
	for _, field := range schema.Schema {
		existingFields[field.Name] = true
	}

	var missing []string
	for _, field := range getActivityFields() {
		if !existingFields[field] {
			missing = append(missing, field)
		}
	}

	if len(missing) > 0 {
		slog.Warn("Missing fields in collection", "fields", missing)
	} else {
		slog.Info("All required fields exist in collection")
	}
}

func getActivityFields() []string {
	t := reflect.TypeOf(AyatActivity{})
	var fields []string
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if jsonTag := field.Tag.Get("json"); jsonTag != "" && jsonTag != "-" {
			if idx := strings.Index(jsonTag, ","); idx > 0 {
				fields = append(fields, jsonTag[:idx])
			} else {
				fields = append(fields, jsonTag)
			}
		}
	}
	return fields
}
