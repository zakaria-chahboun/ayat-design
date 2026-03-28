package pb

import (
	"log/slog"
	"time"

	"github.com/chrisbrocklesby/pbclient"
	"github.com/zakaria-chahboun/AyatDesingBot/config"
)

var client *pbclient.Client

func Init() error {
	if config.PocketBaseURL == "" {
		slog.Warn("POCKETBASE_URL not set, activity tracking disabled")
		return nil
	}

	var err error
	client, err = pbclient.NewClient(pbclient.Config{
		BaseURL: config.PocketBaseURL,
	})
	if err != nil {
		slog.Warn("Failed to initialize PocketBase client", "error", err)
		return nil
	}

	slog.Info("PocketBase client initialized")
	return nil
}

func IsEnabled() bool {
	return client != nil
}

func WaitReady(timeout time.Duration) bool {
	if client == nil {
		return false
	}
	return client.WaitReady(timeout) == nil
}
