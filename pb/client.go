package pb

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/chrisbrocklesby/pbclient"
	"github.com/zakaria-chahboun/AyatDesingBot/config"
)

var (
	client    *pbclient.Client
	initOnce  sync.Once
	authError error
)

func Init() error {
	initOnce.Do(func() {
		if config.PocketBaseURL == "" {
			slog.Warn("POCKETBASE_URL not set, activity tracking disabled")
			return
		}

		if config.PocketBaseEmail == "" || config.PocketBasePassword == "" {
			slog.Warn("POCKETBASE_EMAIL or POCKETBASE_PASSWORD not set, activity tracking disabled")
			return
		}

		if config.PocketBaseCollection == "" {
			slog.Warn("POCKETBASE_COLLECTION not set, activity tracking disabled")
			return
		}

		var err error
		client, err = pbclient.NewClient(pbclient.Config{
			BaseURL: config.PocketBaseURL,
		})
		if err != nil {
			slog.Warn("Failed to initialize PocketBase client", "error", err)
			authError = err
			return
		}
		pbclient.SetDefault(client)

		if err := authenticate(); err != nil {
			slog.Warn("Failed to authenticate with PocketBase", "error", err)
			authError = err
			return
		}

		slog.Info("PocketBase client initialized")
	})

	return authError
}

func authenticate() error {
	err := pbclient.LoginUser(config.PocketBaseCollection, config.PocketBaseEmail, config.PocketBasePassword)
	if err != nil {
		return fmt.Errorf("login failed: %w", err)
	}
	return nil
}

func IsEnabled() bool {
	return client != nil
}

func WaitReady(timeout time.Duration) bool {
	if !IsEnabled() {
		return false
	}
	time.Sleep(timeout)
	return true
}

func isUnauthorized(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return contains(errStr, "401") || contains(errStr, "unauthorized")
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func doRequestWithRetry(body map[string]any) error {
	const maxRetries = 2

	for attempt := 0; attempt <= maxRetries; attempt++ {
		_, err := pbclient.Collection[map[string]any]("config.PocketBaseCollection", client).Create(body)

		if err != nil {
			if isUnauthorized(err) && attempt < maxRetries {
				slog.Warn("PocketBase unauthorized, re-authenticating...")
				if authErr := pbclient.LoginUser("config.PocketBaseCollection", config.PocketBaseEmail, config.PocketBasePassword); authErr != nil {
					slog.Warn("Re-authentication failed", "error", authErr)
					continue
				}
				continue
			}

			if attempt >= maxRetries {
				return fmt.Errorf("max retries exceeded: %w", err)
			}
			continue
		}

		return nil
	}

	return fmt.Errorf("max retries exceeded")
}

func doPBRequestWithRetry(method, path string, body interface{}) ([]byte, error) {
	const maxRetries = 2
	httpClient := &http.Client{Timeout: 30 * time.Second}

	for attempt := 0; attempt <= maxRetries; attempt++ {
		var req *http.Request
		var err error

		if body != nil {
			jsonBody, _ := json.Marshal(body)
			req, err = http.NewRequest(method, config.PocketBaseURL+path, bytes.NewBuffer(jsonBody))
			if err == nil {
				req.Header.Set("Content-Type", "application/json")
			}
		} else {
			req, err = http.NewRequest(method, config.PocketBaseURL+path, nil)
		}

		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		resp, err := httpClient.Do(req)
		if err != nil {
			if attempt < maxRetries {
				continue
			}
			return nil, fmt.Errorf("request failed: %w", err)
		}
		defer resp.Body.Close()

		respBody, _ := io.ReadAll(resp.Body)

		if resp.StatusCode == http.StatusUnauthorized {
			slog.Warn("PocketBase unauthorized, re-authenticating...")
			if authErr := pbclient.LoginUser("config.PocketBaseCollection", config.PocketBaseEmail, config.PocketBasePassword); authErr != nil {
				slog.Warn("Re-authentication failed", "error", authErr)
				continue
			}
			continue
		}

		if resp.StatusCode >= 400 {
			if attempt < maxRetries {
				continue
			}
			return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(respBody))
		}

		return respBody, nil
	}

	return nil, fmt.Errorf("max retries exceeded")
}
