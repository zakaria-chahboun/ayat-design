package pb

import (
	"encoding/json"
	"log/slog"

	"github.com/zakaria-chahboun/AyatDesingBot/config"
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

type CollectionResponse struct {
	ID     string        `json:"id"`
	Name   string        `json:"name"`
	Schema []FieldSchema `json:"schema"`
}

type FieldPatchRequest struct {
	Schema []FieldSchema `json:"schema"`
}

func RunMigrations() error {
	if !IsEnabled() {
		slog.Info("PocketBase not enabled, skipping migrations")
		return nil
	}

	resp, err := doPBRequestWithRetry("GET", "/api/collections/"+config.PocketBaseCollection, nil)
	if err != nil {
		slog.Warn("Failed to get collection info", "error", err)
		return err
	}

	var collection CollectionResponse
	if err := json.Unmarshal(resp, &collection); err != nil {
		slog.Warn("Failed to parse collection response", "error", err)
		return err
	}

	existingFields := make(map[string]bool)
	for _, field := range collection.Schema {
		existingFields[field.Name] = true
	}

	fieldsToAdd := []FieldSchema{
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
	}

	var newFields []FieldSchema
	for _, field := range fieldsToAdd {
		if !existingFields[field.Name] {
			newFields = append(newFields, field)
			slog.Info("Adding field to collection", "field", field.Name)
		}
	}

	if len(newFields) == 0 {
		slog.Info("All fields already exist, no migration needed")
		return nil
	}

	allFields := append(collection.Schema, newFields...)
	patch := FieldPatchRequest{Schema: allFields}

	_, err = doPBRequestWithRetry("PATCH", "/api/collections/"+config.PocketBaseCollection, patch)
	if err != nil {
		slog.Warn("Failed to update collection schema", "error", err)
		return err
	}

	slog.Info("Migration completed successfully", "fields_added", len(newFields))
	return nil
}
