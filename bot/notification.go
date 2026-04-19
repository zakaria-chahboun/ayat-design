package bot

import (
	"bytes"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/zakaria-chahboun/AyatDesingBot/config"
	"github.com/zakaria-chahboun/AyatDesingBot/pb"
	tele "gopkg.in/telebot.v3"
)

func sendLogFile(b *tele.Bot, c tele.Context, filename, content string) error {
	doc := &tele.Document{
		FileName: filename,
		File:     tele.FromReader(strings.NewReader(content)),
	}
	_, err := b.Send(c.Chat(), doc)
	return err
}

func buildErrorLog(adminFullName, adminUsername string, errors []string) string {
	var buf bytes.Buffer
	buf.WriteString("=== Notification Error Log ===\n")
	buf.WriteString(fmt.Sprintf("Timestamp: %s\n", time.Now().Format("2006-01-02 15:04:05")))
	buf.WriteString(fmt.Sprintf("Admin: fullname=%s username=%s\n\n", adminFullName, adminUsername))

	if len(errors) > 0 {
		buf.WriteString("=== Errors ===\n")
		buf.WriteString(strings.Join(errors, "\n"))
	} else {
		buf.WriteString("No errors")
	}

	return buf.String()
}

func truncateText(text string, maxLen int) string {
	runes := []rune(text)
	if len(runes) > maxLen {
		return string(runes[:maxLen]) + "..."
	}
	return text
}

func buildNotificationButtonLabel(notif *pb.AyatNotification) string {
	text := truncateText(notif.MarkdownMessage, 50)
	if notif.DeleteAfterSec > 0 {
		return text + ".. ⏳"
	}
	return text
}

func ShowPendingNotificationsList(b *tele.Bot, c tele.Context) error {
	sender := c.Sender()
	fullName := sender.FirstName
	if sender.LastName != "" {
		fullName += " " + sender.LastName
	}
	username := sender.Username

	isAdmin := username == config.TelegramAdminUsername
	if !isAdmin {
		return nil
	}

	notifications, err := pb.GetAllPendingNotifications(10)
	if err != nil {
		slog.Error("Failed to get pending notifications", "error", err, "fullname", fullName, "username", username)
		return c.Send("⚠️ Error loading notifications")
	}

	if len(notifications) == 0 {
		return c.Send("📭 No pending notifications found")
	}

	menu := &tele.ReplyMarkup{}
	var rows []tele.Row

	for _, notif := range notifications {
		btn := menu.Data(buildNotificationButtonLabel(&notif), "notify_send", notif.ID)
		rows = append(rows, menu.Row(btn))
	}

	menu.Inline(rows...)

	return c.Send("📋 Select a notification to send:", menu)
}

func SendSpecificNotification(b *tele.Bot, c tele.Context, notificationID string, loadingMsg *tele.Message) (string, error) {
	sender := c.Sender()
	fullName := sender.FirstName
	if sender.LastName != "" {
		fullName += " " + sender.LastName
	}
	username := sender.Username

	isAdmin := username == config.TelegramAdminUsername
	adminInfo := fmt.Sprintf("fullname=%s username=%s", fullName, username)

	notif, err := pb.GetNotificationByID(notificationID)
	if err != nil {
		errLog := buildErrorLog(fullName, username, []string{fmt.Sprintf("get_notification: %v", err)})
		slog.Error("Failed to get notification", "error", err, "notification_id", notificationID, "fullname", fullName, "username", username)
		if isAdmin {
			if err := sendLogFile(b, c, "error_get_notification.txt", errLog); err != nil {
				return "", err
			}
			return "⚠️ Error - see attached file", nil
		}
		return "", nil
	}

	if notif == nil {
		return "❌ Notification not found", nil
	}

	deleteAfterSec := notif.DeleteAfterSec
	if deleteAfterSec > 0 {
		if deleteAfterSec < 5 {
			deleteAfterSec = 5
		}
		if deleteAfterSec > 300 {
			deleteAfterSec = 300
		}
	}

	userIDs, err := pb.GetDistinctUserIDs()
	if err != nil {
		errLog := buildErrorLog(fullName, username, []string{fmt.Sprintf("get_users: %v", err)})
		slog.Error("Failed to get users", "error", err, "fullname", fullName, "username", username)
		if isAdmin {
			if err := sendLogFile(b, c, "error_get_users.txt", errLog); err != nil {
				return "", err
			}
			return "⚠️ Error - see attached file", nil
		}
		return "", nil
	}

	if len(userIDs) == 0 {
		return "👥 No users found to send to", nil
	}

	var sentCount, failCount int
	var errors []string

	for i, userID := range userIDs {
		targetFullName, err := pb.GetUserFullName(userID)
		if err != nil {
			errLog := fmt.Sprintf("user_id=%d: get_fullname: %v", userID, err)
			errors = append(errors, errLog)
			slog.Warn("Failed to get user fullname", "user_id", userID, "error", err)
			targetFullName = ""
		}

		var message string
		if notif.IncludeWelcoming && targetFullName != "" {
			message = fmt.Sprintf("السلام عليكم %s\n%s", targetFullName, notif.MarkdownMessage)
		} else {
			message = notif.MarkdownMessage
		}

		chat := &tele.Chat{ID: userID}
		msg, err := b.Send(chat, message, tele.ModeMarkdownV2)
		if err != nil {
			errLog := fmt.Sprintf("user_id=%d: send: %v", userID, err)
			errors = append(errors, errLog)
			slog.Warn("Failed to send notification", "user_id", userID, "error", err)
			failCount++
		} else {
			sentCount++
			if deleteAfterSec > 0 && msg != nil {
				go func(chatID int64, messageID int) {
					time.Sleep(time.Duration(deleteAfterSec) * time.Second)
					if err := b.Delete(&tele.Message{ID: messageID, Chat: &tele.Chat{ID: chatID}}); err != nil {
						slog.Warn("Failed to delete notification message", "chat_id", chatID, "message_id", messageID, "error", err)
					}
				}(chat.ID, msg.ID)
			}
		}

		if loadingMsg != nil {
			displayName := targetFullName
			if displayName == "" {
				displayName = "user"
			}
			_, _ = b.Edit(loadingMsg, fmt.Sprintf("📤 Sending to %s (%d/%d)...", displayName, i+1, len(userIDs)))
		}
	}

	if err := pb.MarkNotificationDone(notif.ID); err != nil {
		errLog := buildErrorLog(fullName, username, []string{fmt.Sprintf("mark_done: %v", err)})
		slog.Error("Failed to mark notification as done", "error", err, "fullname", fullName, "username", username)
		if isAdmin {
			if err := sendLogFile(b, c, "error_mark_done.txt", errLog); err != nil {
				return "", err
			}
			return "⚠️ Error - see attached file", nil
		}
		return "", nil
	}

	slog.Info("Notification sent",
		"notification_id", notif.ID,
		"sent_count", sentCount,
		"failed_count", failCount,
		"delete_after_sec", deleteAfterSec,
		"admin_info", adminInfo,
	)

	response := fmt.Sprintf("✅ Sent to %d users", sentCount)
	if failCount > 0 {
		response += fmt.Sprintf("\n❌ Failed to send to %d users", failCount)
	}
	if isAdmin && len(errors) > 0 {
		errLogFile := "notification_errors.txt"
		errLog := buildErrorLog(fullName, username, errors)
		if err := sendLogFile(b, c, errLogFile, errLog); err != nil {
			return response + "\n\n⚠️ Failed to send error log file", nil
		}
		return response + "\n\n📎 Errors - see attached file", nil
	}

	return response, nil
}

func SendNotification(b *tele.Bot, c tele.Context) error {
	sender := c.Sender()
	fullName := sender.FirstName
	if sender.LastName != "" {
		fullName += " " + sender.LastName
	}
	username := sender.Username

	isAdmin := username == config.TelegramAdminUsername
	adminInfo := fmt.Sprintf("fullname=%s username=%s", fullName, username)

	notif, err := pb.GetPendingNotification()
	if err != nil {
		errLog := buildErrorLog(fullName, username, []string{fmt.Sprintf("get_pending_notification: %v", err)})
		slog.Error("Failed to get pending notification", "error", err, "fullname", fullName, "username", username)
		if isAdmin {
			if err := sendLogFile(b, c, "error_get_notification.txt", errLog); err != nil {
				return c.Send("⚠️ Failed to send error log")
			}
			return c.Send("⚠️ Error - see attached file")
		}
		return nil
	}

	if notif == nil {
		return c.Send("📭 No pending notifications found")
	}

	deleteAfterSec := notif.DeleteAfterSec
	if deleteAfterSec > 0 {
		if deleteAfterSec < 5 {
			deleteAfterSec = 5
		}
		if deleteAfterSec > 300 {
			deleteAfterSec = 300
		}
	}

	userIDs, err := pb.GetDistinctUserIDs()
	if err != nil {
		errLog := buildErrorLog(fullName, username, []string{fmt.Sprintf("get_users: %v", err)})
		slog.Error("Failed to get users", "error", err, "fullname", fullName, "username", username)
		if isAdmin {
			if err := sendLogFile(b, c, "error_get_users.txt", errLog); err != nil {
				return c.Send("⚠️ Failed to send error log")
			}
			return c.Send("⚠️ Error - see attached file")
		}
		return nil
	}

	if len(userIDs) == 0 {
		return c.Send("👥 No users found to send to")
	}

	var sentCount, failCount int
	var errors []string
	for _, userID := range userIDs {
		targetFullName, err := pb.GetUserFullName(userID)
		if err != nil {
			errLog := fmt.Sprintf("user_id=%d: get_fullname: %v", userID, err)
			errors = append(errors, errLog)
			slog.Warn("Failed to get user fullname", "user_id", userID, "error", err)
			targetFullName = ""
		}

		var message string
		if notif.IncludeWelcoming && targetFullName != "" {
			message = fmt.Sprintf("السلام عليكم %s\n%s", targetFullName, notif.MarkdownMessage)
		} else {
			message = notif.MarkdownMessage
		}

		chat := &tele.Chat{ID: userID}
		msg, err := b.Send(chat, message, tele.ModeMarkdownV2)
		if err != nil {
			errLog := fmt.Sprintf("user_id=%d: send: %v", userID, err)
			errors = append(errors, errLog)
			slog.Warn("Failed to send notification", "user_id", userID, "error", err)
			failCount++
			continue
		}

		if deleteAfterSec > 0 && msg != nil {
			go func(chatID int64, messageID int) {
				time.Sleep(time.Duration(deleteAfterSec) * time.Second)
				if err := b.Delete(&tele.Message{ID: messageID, Chat: &tele.Chat{ID: chatID}}); err != nil {
					slog.Warn("Failed to delete notification message", "chat_id", chatID, "message_id", messageID, "error", err)
				}
			}(chat.ID, msg.ID)
		}

		sentCount++
	}

	if err := pb.MarkNotificationDone(notif.ID); err != nil {
		errLog := buildErrorLog(fullName, username, []string{fmt.Sprintf("mark_done: %v", err)})
		slog.Error("Failed to mark notification as done", "error", err, "fullname", fullName, "username", username)
		if isAdmin {
			if err := sendLogFile(b, c, "error_mark_done.txt", errLog); err != nil {
				return c.Send("⚠️ Failed to send error log")
			}
			return c.Send("⚠️ Error - see attached file")
		}
		return nil
	}

	slog.Info("Notification sent",
		"notification_id", notif.ID,
		"sent_count", sentCount,
		"failed_count", failCount,
		"delete_after_sec", deleteAfterSec,
		"admin_info", adminInfo,
	)

	response := fmt.Sprintf("✅ Sent to %d users\n❌ Failed to send to %d users", sentCount, failCount)
	if isAdmin && len(errors) > 0 {
		errLogFile := "notification_errors.txt"
		errLog := buildErrorLog(fullName, username, errors)
		if err := sendLogFile(b, c, errLogFile, errLog); err != nil {
			return c.Send(response + "\n\n⚠️ Failed to send error log file")
		}
		return c.Send(response + "\n\n📎 Errors - see attached file")
	}

	return c.Send(response, tele.ModeMarkdownV2)
}
