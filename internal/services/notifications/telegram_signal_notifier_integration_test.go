package notifications_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	domain "github.com/drybin/fear-and-greed-online/internal/domain/strategy"
	"github.com/drybin/fear-and-greed-online/internal/services/notifications"
	"github.com/drybin/fear-and-greed-online/internal/storage/postgres"
	"github.com/drybin/fear-and-greed-online/internal/testutil"
)

type fakeSender struct {
	fail bool
	sent []string
}

func (f *fakeSender) SendMessage(_ context.Context, _ string, text string) error {
	if f.fail {
		return fmt.Errorf("send failed")
	}
	f.sent = append(f.sent, text)
	return nil
}

func TestTelegramSignalNotifierSendsOnlyEntryOnce(t *testing.T) {
	db := testutil.CreateTestDatabase(t)
	ctx := context.Background()

	strategies := postgres.NewStrategyRepository(db)
	symbols := postgres.NewSymbolRepository(db)
	timeframes := postgres.NewTimeframeRepository(db)
	runs := postgres.NewStrategyRunRepository(db)
	signals := postgres.NewSignalRepository(db)

	strategy, err := strategies.FindBySlug(ctx, "trend-long-v1")
	if err != nil || strategy == nil {
		t.Fatalf("find strategy: %v", err)
	}
	symbol, err := symbols.FindByAssetCode(ctx, "BTC")
	if err != nil || symbol == nil {
		t.Fatalf("find BTC symbol: %v", err)
	}
	timeframe, err := timeframes.FindByCode(ctx, "1h")
	if err != nil || timeframe == nil {
		t.Fatalf("find timeframe: %v", err)
	}

	base := time.Date(2026, 8, 18, 6, 0, 0, 0, time.UTC)
	runID, err := runs.Start(ctx, strategy.ID, symbol.ID, timeframe.ID, strategy.DefaultParams, &base, &base, 10)
	if err != nil {
		t.Fatalf("start run: %v", err)
	}
	if err := signals.UpsertMany(ctx, runID, strategy.ID, symbol.ID, timeframe.ID, []domain.Signal{
		{
			DedupeKey: "trend-long-v1|BTC|1h|entry|" + base.Format(time.RFC3339),
			Time:      base,
			Type:      "entry",
			Side:      "long",
			Price:     100,
			Status:    "confirmed",
			Title:     "Trend entry",
		},
		{
			DedupeKey: "trend-long-v1|BTC|1h|exit|" + base.Add(time.Hour).Format(time.RFC3339),
			Time:      base.Add(time.Hour),
			Type:      "exit",
			Side:      "long",
			Price:     90,
			Status:    "confirmed",
			Title:     "Trend exit",
		},
	}); err != nil {
		t.Fatalf("insert signals: %v", err)
	}

	sender := &fakeSender{}
	notifier := notifications.NewTelegramSignalNotifier(
		signals,
		postgres.NewSignalNotificationRepository(db),
		sender,
		"-100123456789",
		"http://dr54.ru/",
	)
	if err := notifier.NotifyEntries(ctx, runID); err != nil {
		t.Fatalf("notify entries first: %v", err)
	}
	if err := notifier.NotifyEntries(ctx, runID); err != nil {
		t.Fatalf("notify entries second: %v", err)
	}

	if len(sender.sent) != 1 {
		t.Fatalf("expected one outgoing telegram message, got %d", len(sender.sent))
	}
	message := sender.sent[0]
	for _, mustContain := range []string{
		"Algorithm: Trend Long v1",
		"Coin: BTC",
		"Open chart: http://dr54.ru/?",
		"signalId=",
		"signalKey=",
	} {
		if !strings.Contains(message, mustContain) {
			t.Fatalf("message missing %q: %s", mustContain, message)
		}
	}
}

func TestTelegramSignalNotifierReleasesClaimOnSendFailure(t *testing.T) {
	db := testutil.CreateTestDatabase(t)
	ctx := context.Background()

	strategies := postgres.NewStrategyRepository(db)
	symbols := postgres.NewSymbolRepository(db)
	timeframes := postgres.NewTimeframeRepository(db)
	runs := postgres.NewStrategyRunRepository(db)
	signals := postgres.NewSignalRepository(db)

	strategy, _ := strategies.FindBySlug(ctx, "trend-long-v1")
	symbol, _ := symbols.FindByAssetCode(ctx, "BTC")
	timeframe, _ := timeframes.FindByCode(ctx, "1h")
	at := time.Date(2026, 8, 18, 8, 0, 0, 0, time.UTC)
	runID, err := runs.Start(ctx, strategy.ID, symbol.ID, timeframe.ID, strategy.DefaultParams, &at, &at, 10)
	if err != nil {
		t.Fatalf("start run: %v", err)
	}
	if err := signals.UpsertMany(ctx, runID, strategy.ID, symbol.ID, timeframe.ID, []domain.Signal{{
		DedupeKey: "trend-long-v1|BTC|1h|entry|" + at.Format(time.RFC3339),
		Time:      at,
		Type:      "entry",
		Side:      "long",
		Price:     101,
		Status:    "confirmed",
		Title:     "Trend entry",
	}}); err != nil {
		t.Fatalf("insert signal: %v", err)
	}

	sender := &fakeSender{fail: true}
	notifier := notifications.NewTelegramSignalNotifier(
		signals,
		postgres.NewSignalNotificationRepository(db),
		sender,
		"-100123456789",
		"http://dr54.ru/",
	)
	if err := notifier.NotifyEntries(ctx, runID); err == nil {
		t.Fatal("expected notify to fail")
	}

	// After failed send, claim must be released so retry can claim again.
	sender.fail = false
	if err := notifier.NotifyEntries(ctx, runID); err != nil {
		t.Fatalf("retry notify: %v", err)
	}
	if len(sender.sent) != 1 {
		t.Fatalf("expected successful retry to send one message, got %d", len(sender.sent))
	}
}
