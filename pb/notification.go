package pb

import (
	"fmt"
	"log/slog"

	"github.com/chrisbrocklesby/pbclient"
	"github.com/zakaria-chahboun/AyatDesingBot/config"
)

type AyatNotification struct {
	ID               string `json:"id"`
	MarkdownMessage  string `json:"markdown_message"`
	IncludeWelcoming bool   `json:"include_welcoming"`
	DeleteAfterSec   int    `json:"delete_after_sec"`
	Done             bool   `json:"done"`
}

func GetPendingNotification() (*AyatNotification, error) {
	if !IsEnabled() {
		return nil, nil
	}

	result, err := pbclient.Collection[AyatNotification](config.PocketBaseCollectionNotifications).
		List("filter=(done=false)&perPage=1")
	if err != nil {
		return nil, err
	}

	if len(result.Items) == 0 {
		return nil, nil
	}

	notif := result.Items[0]

	if notif.MarkdownMessage == "" {
		return nil, nil
	}

	slog.Info("Found pending notification", "id", notif.ID)
	return &notif, nil
}

func MarkNotificationDone(id string) error {
	if !IsEnabled() {
		return nil
	}

	_, err := pbclient.Collection[AyatNotification](config.PocketBaseCollectionNotifications).
		Update(id, map[string]any{"done": true})
	return err
}

func GetDistinctUserIDs() ([]int64, error) {
	if !IsEnabled() {
		return nil, nil
	}

	result, err := pbclient.Collection[AyatActivity](config.PocketBaseCollectionActivities).
		List("perPage=1000")
	if err != nil {
		return nil, err
	}

	seen := make(map[int64]bool)
	var userIDs []int64
	for _, activity := range result.Items {
		if !seen[activity.UserID] {
			seen[activity.UserID] = true
			userIDs = append(userIDs, activity.UserID)
		}
	}

	return userIDs, nil
}

func GetUserFullName(userID int64) (string, error) {
	if !IsEnabled() {
		return "", nil
	}

	result, err := pbclient.Collection[AyatActivity](config.PocketBaseCollectionActivities).
		List(fmt.Sprintf("filter=(user_id=%d)&sort=-created&perPage=1", userID))
	if err != nil {
		return "", err
	}

	if len(result.Items) == 0 {
		return "", nil
	}

	return result.Items[0].FullName, nil
}

func GetAllPendingNotifications(limit int) ([]AyatNotification, error) {
	if !IsEnabled() {
		return nil, nil
	}

	result, err := pbclient.Collection[AyatNotification](config.PocketBaseCollectionNotifications).
		List(fmt.Sprintf("filter=(done=false)&perPage=%d&sort=created", limit))
	if err != nil {
		return nil, err
	}

	var notifications []AyatNotification
	for _, notif := range result.Items {
		if notif.MarkdownMessage != "" {
			notifications = append(notifications, notif)
		}
	}

	return notifications, nil
}

func GetNotificationByID(id string) (*AyatNotification, error) {
	if !IsEnabled() {
		return nil, nil
	}

	result, err := pbclient.Collection[AyatNotification](config.PocketBaseCollectionNotifications).
		List(fmt.Sprintf("filter=(id='%s')&perPage=1", id))
	if err != nil {
		return nil, err
	}

	if len(result.Items) == 0 {
		return nil, nil
	}

	return &result.Items[0], nil
}
