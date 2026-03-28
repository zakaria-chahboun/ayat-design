package pb

import (
	"log/slog"
	"os"

	"github.com/chrisbrocklesby/pbclient"
)

type FieldSchema struct {
	Name     string      `json:"name"`
	Type     string      `json:"type"`
	Required bool        `json:"required"`
	Options  interface{} `json:"options,omitempty"`
}

type SelectOptions struct {
	MaxSelect int      `json:"maxSelect"`
	Values    []string `json:"values"`
}

type CollectionRequest struct {
	Name       string        `json:"name"`
	Type       string        `json:"type"`
	Schema     []FieldSchema `json:"schema"`
	Indexes    []string      `json:"indexes,omitempty"`
	ListRule   string        `json:"listRule,omitempty"`
	ViewRule   string        `json:"viewRule,omitempty"`
	CreateRule string        `json:"createRule,omitempty"`
	UpdateRule string        `json:"updateRule,omitempty"`
	DeleteRule string        `json:"deleteRule,omitempty"`
}

func RunMigrations() error {
	url := os.Getenv("POCKETBASE_URL")
	if url == "" {
		slog.Info("POCKETBASE_URL not set, skipping migrations")
		return nil
	}

	c, err := pbclient.NewClient(pbclient.Config{
		BaseURL: url,
	})
	if err != nil {
		return err
	}

	collection := CollectionRequest{
		Name: "ayat_activities",
		Type: "base",
		Schema: []FieldSchema{
			{
				Name:     "user_id",
				Type:     "number",
				Required: true,
			},
			{
				Name: "username",
				Type: "text",
			},
			{
				Name:     "fullname",
				Type:     "text",
				Required: true,
			},
			{
				Name: "action",
				Type: "select",
				Options: SelectOptions{
					MaxSelect: 1,
					Values:    []string{"start", "text", "image", "video"},
				},
			},
			{
				Name: "status",
				Type: "select",
				Options: SelectOptions{
					MaxSelect: 1,
					Values:    []string{"success", "failed", "error"},
				},
			},
			{
				Name: "error_message",
				Type: "text",
			},
			{
				Name: "surah_name",
				Type: "text",
			},
			{
				Name: "ayah_range",
				Type: "text",
			},
			{
				Name: "duration_ms",
				Type: "number",
			},
		},
		CreateRule: "",
	}

	_, err = pbclient.Collection[map[string]any]("collections", c).Create(collection)
	if err != nil {
		slog.Warn("Collection might already exist or migration failed", "error", err)
		return err
	}

	slog.Info("ayat_activities collection created successfully")
	return nil
}
