package postgres_test

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/drybin/fear-and-greed-online/internal/domain/marketdata"
	domain "github.com/drybin/fear-and-greed-online/internal/domain/strategy"
	"github.com/drybin/fear-and-greed-online/internal/storage/postgres"
	"github.com/drybin/fear-and-greed-online/internal/testutil"
)

func TestCandleRepositoryUpsertIsIdempotentAndListRangeWorks(t *testing.T) {
	db := testutil.CreateTestDatabase(t)
	ctx := context.Background()

	symbols := postgres.NewSymbolRepository(db)
	timeframes := postgres.NewTimeframeRepository(db)
	candles := postgres.NewCandleRepository(db)

	symbol, err := symbols.FindByAssetCode(ctx, "BTC")
	if err != nil || symbol == nil {
		t.Fatalf("find BTC symbol: %v", err)
	}
	timeframe, err := timeframes.FindByCode(ctx, "1h")
	if err != nil || timeframe == nil {
		t.Fatalf("find 1h timeframe: %v", err)
	}

	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	seed := []marketdata.Candle{
		testCandle(symbol.ID, timeframe.ID, base, "10"),
		testCandle(symbol.ID, timeframe.ID, base.Add(time.Hour), "11"),
	}

	if err := candles.UpsertMany(ctx, seed); err != nil {
		t.Fatalf("insert candles: %v", err)
	}

	updated := []marketdata.Candle{
		testCandle(symbol.ID, timeframe.ID, base, "10.5"),
		testCandle(symbol.ID, timeframe.ID, base.Add(time.Hour), "11.5"),
		testCandle(symbol.ID, timeframe.ID, base.Add(2*time.Hour), "12"),
	}
	if err := candles.UpsertMany(ctx, updated); err != nil {
		t.Fatalf("upsert candles: %v", err)
	}

	items, err := candles.ListRange(ctx, symbol.ID, timeframe.ID, base, base.Add(3*time.Hour))
	if err != nil {
		t.Fatalf("list candles: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("unexpected candle count: got %d want 3", len(items))
	}
	if decimalValue(items[0].Close) != 10.5 {
		t.Fatalf("expected updated close on first candle, got %s", items[0].Close)
	}
	if items[2].OpenTime != base.Add(2*time.Hour) {
		t.Fatalf("unexpected third candle open time: %s", items[2].OpenTime)
	}

	latest, err := candles.LatestOpenTime(ctx, symbol.ID, timeframe.ID)
	if err != nil {
		t.Fatalf("latest candle time: %v", err)
	}
	if latest == nil || !latest.Equal(base.Add(2*time.Hour)) {
		t.Fatalf("unexpected latest candle time: %#v", latest)
	}
}

func TestSignalRepositoryDedupeKeyPreventsDuplicateMarkers(t *testing.T) {
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
		t.Fatalf("find 1h timeframe: %v", err)
	}

	signalTime := time.Date(2026, 1, 2, 10, 0, 0, 0, time.UTC)
	dedupeKey := "trend-long-v1|BTC|1h|entry|" + signalTime.Format(time.RFC3339)

	runID, err := runs.Start(ctx, strategy.ID, symbol.ID, timeframe.ID, strategy.DefaultParams, &signalTime, &signalTime, 10)
	if err != nil {
		t.Fatalf("start run: %v", err)
	}

	first := []domain.Signal{{
		DedupeKey: dedupeKey,
		Time:      signalTime,
		Type:      "entry",
		Side:      "long",
		Price:     100,
		Status:    "confirmed",
		Title:     "Trend entry",
		Details:   "first",
		Meta:      map[string]any{"sma_period": 3},
	}}
	if err := signals.UpsertMany(ctx, runID, strategy.ID, symbol.ID, timeframe.ID, first); err != nil {
		t.Fatalf("upsert first signal: %v", err)
	}

	secondRunID, err := runs.Start(ctx, strategy.ID, symbol.ID, timeframe.ID, strategy.DefaultParams, &signalTime, &signalTime, 10)
	if err != nil {
		t.Fatalf("start second run: %v", err)
	}
	second := []domain.Signal{{
		DedupeKey: dedupeKey,
		Time:      signalTime,
		Type:      "entry",
		Side:      "long",
		Price:     101,
		Status:    "confirmed",
		Title:     "Trend entry",
		Details:   "updated",
		Meta:      map[string]any{"sma_period": 3, "updated": true},
	}}
	if err := signals.UpsertMany(ctx, secondRunID, strategy.ID, symbol.ID, timeframe.ID, second); err != nil {
		t.Fatalf("upsert second signal: %v", err)
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM signals WHERE dedupe_key = $1`, dedupeKey).Scan(&count); err != nil {
		t.Fatalf("count signals: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected one logical signal, got %d", count)
	}

	items, err := signals.ListRange(ctx, strategy.ID, symbol.ID, timeframe.ID, signalTime.Add(-time.Hour), signalTime.Add(time.Hour))
	if err != nil {
		t.Fatalf("list signals: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("unexpected listed signals: got %d want 1", len(items))
	}
	if items[0].Price != 101 {
		t.Fatalf("expected updated signal price, got %v", items[0].Price)
	}
	if items[0].Details != "updated" {
		t.Fatalf("expected updated signal details, got %q", items[0].Details)
	}
}

func TestSignalRepositoryListOnLatestCandles(t *testing.T) {
	db := testutil.CreateTestDatabase(t)
	ctx := context.Background()

	symbols := postgres.NewSymbolRepository(db)
	timeframes := postgres.NewTimeframeRepository(db)
	strategies := postgres.NewStrategyRepository(db)
	candles := postgres.NewCandleRepository(db)
	runs := postgres.NewStrategyRunRepository(db)
	signals := postgres.NewSignalRepository(db)

	btc, err := symbols.FindByAssetCode(ctx, "BTC")
	if err != nil || btc == nil {
		t.Fatalf("find BTC: %v", err)
	}
	eth, err := symbols.FindByAssetCode(ctx, "ETH")
	if err != nil || eth == nil {
		t.Fatalf("find ETH: %v", err)
	}
	hourly, err := timeframes.FindByCode(ctx, "1h")
	if err != nil || hourly == nil {
		t.Fatalf("find 1h: %v", err)
	}
	trend, err := strategies.FindBySlug(ctx, "trend-long-v1")
	if err != nil || trend == nil {
		t.Fatalf("find trend-long-v1: %v", err)
	}
	breakout, err := strategies.FindBySlug(ctx, "prev-day-range-breakout-v1")
	if err != nil || breakout == nil {
		t.Fatalf("find prev-day-range-breakout-v1: %v", err)
	}

	older := time.Date(2026, 8, 17, 8, 0, 0, 0, time.UTC)
	latest := older.Add(time.Hour)
	if err := candles.UpsertMany(ctx, []marketdata.Candle{
		testCandle(btc.ID, hourly.ID, older, "100"),
		testCandle(btc.ID, hourly.ID, latest, "110"),
		testCandle(eth.ID, hourly.ID, older, "200"),
		testCandle(eth.ID, hourly.ID, latest, "210"),
	}); err != nil {
		t.Fatalf("seed candles: %v", err)
	}

	insertTestSignal(t, ctx, runs, signals, trend, btc, hourly, older, "stale-btc", 100)
	insertTestSignal(t, ctx, runs, signals, trend, btc, hourly, latest, "live-btc-trend", 110)
	insertTestSignal(t, ctx, runs, signals, breakout, btc, hourly, latest, "live-btc-breakout", 111)
	insertTestSignal(t, ctx, runs, signals, trend, eth, hourly, latest, "live-eth", 210)

	items, err := signals.ListOnLatestCandles(ctx)
	if err != nil {
		t.Fatalf("list on latest candles: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("expected 3 live signals, got %d: %#v", len(items), items)
	}

	got := map[string]postgres.LiveSignalRecord{}
	for _, item := range items {
		got[item.Title] = item
		if item.SignalTime.Location() != time.UTC {
			t.Fatalf("expected UTC signal time, got %s", item.SignalTime.Location())
		}
	}
	if _, ok := got["stale-btc"]; ok {
		t.Fatal("did not expect a signal from a non-latest candle")
	}
	if got["live-btc-trend"].AssetCode != "BTC" || got["live-btc-trend"].Timeframe != "1h" || got["live-btc-trend"].StrategySlug != "trend-long-v1" {
		t.Fatalf("unexpected BTC trend live signal: %#v", got["live-btc-trend"])
	}
	if got["live-btc-breakout"].StrategySlug != "prev-day-range-breakout-v1" {
		t.Fatalf("unexpected BTC breakout live signal: %#v", got["live-btc-breakout"])
	}
	if got["live-eth"].AssetCode != "ETH" || got["live-eth"].Price != 210 {
		t.Fatalf("unexpected ETH live signal: %#v", got["live-eth"])
	}
}

func insertTestSignal(
	t *testing.T,
	ctx context.Context,
	runs *postgres.StrategyRunRepository,
	signals *postgres.SignalRepository,
	strategy *domain.Definition,
	symbol *marketdata.Symbol,
	timeframe *marketdata.Timeframe,
	at time.Time,
	title string,
	price float64,
) {
	t.Helper()

	runID, err := runs.Start(ctx, strategy.ID, symbol.ID, timeframe.ID, strategy.DefaultParams, &at, &at, 2)
	if err != nil {
		t.Fatalf("start run for %s: %v", title, err)
	}
	if err := signals.UpsertMany(ctx, runID, strategy.ID, symbol.ID, timeframe.ID, []domain.Signal{{
		DedupeKey: fmt.Sprintf("%s|%s|%s|%s|%s", strategy.Slug, symbol.AssetCode, timeframe.Code, title, at.Format(time.RFC3339)),
		Time:      at,
		Type:      "alert",
		Side:      "long",
		Price:     price,
		Status:    "confirmed",
		Title:     title,
		Details:   "test",
	}}); err != nil {
		t.Fatalf("insert signal %s: %v", title, err)
	}
}

func TestIngestionJobRepositoryTracksSuccessfulSync(t *testing.T) {
	db := testutil.CreateTestDatabase(t)
	ctx := context.Background()

	symbols := postgres.NewSymbolRepository(db)
	timeframes := postgres.NewTimeframeRepository(db)
	ingestion := postgres.NewIngestionJobRepository(db)

	symbol, err := symbols.FindByAssetCode(ctx, "BTC")
	if err != nil || symbol == nil {
		t.Fatalf("find BTC symbol: %v", err)
	}
	timeframe, err := timeframes.FindByCode(ctx, "1h")
	if err != nil || timeframe == nil {
		t.Fatalf("find 1h timeframe: %v", err)
	}

	requestedFrom := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	requestedTo := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	jobID, err := ingestion.Start(ctx, symbol.ID, timeframe.ID, requestedFrom, requestedTo)
	if err != nil {
		t.Fatalf("start ingestion job: %v", err)
	}

	loadedFrom := requestedFrom
	loadedTo := requestedTo.Add(-time.Hour)
	if err := ingestion.Complete(ctx, jobID, &loadedFrom, &loadedTo, 24); err != nil {
		t.Fatalf("complete ingestion job: %v", err)
	}

	summary, err := ingestion.LatestSuccessful(ctx, symbol.ID, timeframe.ID)
	if err != nil {
		t.Fatalf("latest successful ingestion job: %v", err)
	}
	if summary == nil {
		t.Fatal("expected successful ingestion summary")
	}
	if summary.CandlesLoaded != 24 {
		t.Fatalf("unexpected candles loaded: got %d want 24", summary.CandlesLoaded)
	}
	if summary.LoadedTo == nil || !summary.LoadedTo.Equal(loadedTo) {
		t.Fatalf("unexpected loaded_to: %#v", summary.LoadedTo)
	}
}

func TestSignalRepositoryListEntryByRunOnlyReturnsEntrySignals(t *testing.T) {
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
		t.Fatalf("find 1h timeframe: %v", err)
	}

	at := time.Date(2026, 8, 18, 5, 0, 0, 0, time.UTC)
	runID, err := runs.Start(ctx, strategy.ID, symbol.ID, timeframe.ID, strategy.DefaultParams, &at, &at, 12)
	if err != nil {
		t.Fatalf("start run: %v", err)
	}
	if err := signals.UpsertMany(ctx, runID, strategy.ID, symbol.ID, timeframe.ID, []domain.Signal{
		{
			DedupeKey: "entry|" + at.Format(time.RFC3339),
			Time:      at,
			Type:      "entry",
			Side:      "long",
			Price:     101.25,
			Status:    "confirmed",
			Title:     "entry signal",
		},
		{
			DedupeKey: "exit|" + at.Add(time.Hour).Format(time.RFC3339),
			Time:      at.Add(time.Hour),
			Type:      "exit",
			Side:      "long",
			Price:     99.5,
			Status:    "confirmed",
			Title:     "exit signal",
		},
	}); err != nil {
		t.Fatalf("insert signals: %v", err)
	}

	items, err := signals.ListEntryByRun(ctx, runID)
	if err != nil {
		t.Fatalf("list entry by run: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected only one entry signal, got %d", len(items))
	}
	if items[0].SignalType != "entry" || items[0].AssetCode != "BTC" || items[0].Timeframe != "1h" {
		t.Fatalf("unexpected entry signal payload: %#v", items[0])
	}
}

func TestSignalNotificationRepositoryClaimAndRelease(t *testing.T) {
	db := testutil.CreateTestDatabase(t)
	ctx := context.Background()
	repo := postgres.NewSignalNotificationRepository(db)
	var signalID *int64

	claimed, err := repo.TryClaim(ctx, "telegram", "trend-long-v1|BTC|1h|entry|2026-08-18T05:00:00Z", signalID)
	if err != nil {
		t.Fatalf("first claim: %v", err)
	}
	if !claimed {
		t.Fatal("expected first claim to succeed")
	}

	claimed, err = repo.TryClaim(ctx, "telegram", "trend-long-v1|BTC|1h|entry|2026-08-18T05:00:00Z", signalID)
	if err != nil {
		t.Fatalf("second claim: %v", err)
	}
	if claimed {
		t.Fatal("expected duplicate claim to be rejected")
	}

	if err := repo.ReleaseClaim(ctx, "telegram", "trend-long-v1|BTC|1h|entry|2026-08-18T05:00:00Z"); err != nil {
		t.Fatalf("release claim: %v", err)
	}
	claimed, err = repo.TryClaim(ctx, "telegram", "trend-long-v1|BTC|1h|entry|2026-08-18T05:00:00Z", signalID)
	if err != nil {
		t.Fatalf("third claim: %v", err)
	}
	if !claimed {
		t.Fatal("expected claim after release to succeed")
	}
	if err := repo.MarkSent(ctx, "telegram", "trend-long-v1|BTC|1h|entry|2026-08-18T05:00:00Z"); err != nil {
		t.Fatalf("mark sent: %v", err)
	}

	var sentCount int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM signal_notifications WHERE channel='telegram' AND sent_at IS NOT NULL`).Scan(&sentCount); err != nil {
		t.Fatalf("count sent notifications: %v", err)
	}
	if sentCount != 1 {
		t.Fatalf("expected one sent notification, got %d", sentCount)
	}
}

func testCandle(symbolID, timeframeID int64, openTime time.Time, close string) marketdata.Candle {
	return marketdata.Candle{
		SymbolID:            symbolID,
		TimeframeID:         timeframeID,
		OpenTime:            openTime,
		CloseTime:           openTime.Add(time.Hour - time.Millisecond),
		Open:                close,
		High:                close,
		Low:                 close,
		Close:               close,
		Volume:              "100",
		QuoteVolume:         "1000",
		Trades:              10,
		TakerBuyBaseVolume:  "50",
		TakerBuyQuoteVolume: "500",
		IsClosed:            true,
		Source:              "test",
	}
}

func decimalValue(raw string) float64 {
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0
	}
	return value
}
