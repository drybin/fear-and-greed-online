package notifications

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"strings"
	"time"

	"github.com/drybin/fear-and-greed-online/internal/storage/postgres"
)

type telegramSender interface {
	SendMessage(ctx context.Context, chatID, text string) error
}

type TelegramSignalNotifier struct {
	signals       *postgres.SignalRepository
	notifications *postgres.SignalNotificationRepository
	sender        telegramSender
	chatID        string
	chartBaseURL  string
}

func NewTelegramSignalNotifier(
	signals *postgres.SignalRepository,
	notifications *postgres.SignalNotificationRepository,
	sender telegramSender,
	chatID string,
	chartBaseURL string,
) *TelegramSignalNotifier {
	return &TelegramSignalNotifier{
		signals:       signals,
		notifications: notifications,
		sender:        sender,
		chatID:        chatID,
		chartBaseURL:  chartBaseURL,
	}
}

func (n *TelegramSignalNotifier) NotifyEntries(ctx context.Context, runID int64) error {
	items, err := n.signals.ListEntryByRun(ctx, runID)
	if err != nil {
		return err
	}
	for _, item := range items {
		signalID := item.ID
		claimed, err := n.notifications.TryClaim(ctx, "telegram", item.DedupeKey, &signalID)
		if err != nil {
			return err
		}
		if !claimed {
			continue
		}

		if err := n.sender.SendMessage(ctx, n.chatID, n.messageText(item)); err != nil {
			_ = n.notifications.ReleaseClaim(ctx, "telegram", item.DedupeKey)
			return err
		}
		if err := n.notifications.MarkSent(ctx, "telegram", item.DedupeKey); err != nil {
			log.Printf("telegram sent but mark notification failed for dedupe_key=%s: %v", item.DedupeKey, err)
		}
	}
	return nil
}

func (n *TelegramSignalNotifier) messageText(item postgres.EntrySignalRecord) string {
	signalTime := item.SignalTime.UTC()
	return fmt.Sprintf(
		"New ENTRY signal\n\nAlgorithm: %s\nCoin: %s\nTimeframe: %s\nSignal time (UTC): %s\nSignal: %s @ %.2f\n\nOpen chart: %s",
		item.StrategyName,
		item.AssetCode,
		item.Timeframe,
		signalTime.Format(time.RFC3339),
		item.Title,
		item.Price,
		n.signalLink(item),
	)
}

func (n *TelegramSignalNotifier) signalLink(item postgres.EntrySignalRecord) string {
	base := strings.TrimSuffix(n.chartBaseURL, "/")
	values := url.Values{}
	values.Set("symbol", item.AssetCode)
	values.Set("timeframe", item.Timeframe)
	values.Set("strategy", item.StrategySlug)
	values.Set("signalId", fmt.Sprintf("%d", item.ID))
	values.Set("signalKey", item.DedupeKey)
	values.Set("from", item.SignalTime.Add(-24*time.Hour).UTC().Format(time.RFC3339))
	values.Set("to", item.SignalTime.Add(24*time.Hour).UTC().Format(time.RFC3339))
	return base + "/?" + values.Encode()
}
