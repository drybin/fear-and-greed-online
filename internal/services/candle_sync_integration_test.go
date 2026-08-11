package services_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/drybin/fear-and-greed-online/internal/domain/marketdata"
	"github.com/drybin/fear-and-greed-online/internal/infrastructure/binance"
	"github.com/drybin/fear-and-greed-online/internal/services"
	"github.com/drybin/fear-and-greed-online/internal/storage/postgres"
	"github.com/drybin/fear-and-greed-online/internal/testutil"
)

func TestCandleSyncServiceBackfillsCandlesAndRecordsSuccessfulJob(t *testing.T) {
	db := testutil.CreateTestDatabase(t)
	ctx := context.Background()

	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	server := newOneShotBinanceServer(t, map[string][][]any{
		"BTCUSDT|1h": {
			binanceKlineRow(base, "10", time.Hour),
			binanceKlineRow(base.Add(time.Hour), "11", time.Hour),
		},
	})
	t.Cleanup(server.Close)

	service := newCandleSyncService(t, db, server.URL)
	if err := service.Sync(ctx, "BTC"); err != nil {
		t.Fatalf("initial sync: %v", err)
	}

	symbol, timeframe := btcOneHourContext(t, db, ctx)
	count, err := countCandles(ctx, db, symbol.ID, timeframe.ID)
	if err != nil {
		t.Fatalf("count candles: %v", err)
	}
	if count != 2 {
		t.Fatalf("unexpected candle count after initial sync: got %d want 2", count)
	}

	summary, err := postgres.NewIngestionJobRepository(db).LatestSuccessful(ctx, symbol.ID, timeframe.ID)
	if err != nil {
		t.Fatalf("latest successful ingestion job: %v", err)
	}
	if summary == nil || summary.CandlesLoaded != 2 {
		t.Fatalf("unexpected ingestion summary: %#v", summary)
	}
}

func TestCandleSyncServiceIncrementalSyncAddsMissingCandles(t *testing.T) {
	db := testutil.CreateTestDatabase(t)
	ctx := context.Background()

	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	symbol, timeframe := btcOneHourContext(t, db, ctx)
	candles := postgres.NewCandleRepository(db)
	if err := candles.UpsertMany(ctx, []marketdata.Candle{
		testMarketCandle(symbol.ID, timeframe.ID, base, "10"),
		testMarketCandle(symbol.ID, timeframe.ID, base.Add(time.Hour), "11"),
	}); err != nil {
		t.Fatalf("seed candles: %v", err)
	}

	server := newIncrementalBinanceServer(t, base.Add(time.Hour), binanceKlineRow(base.Add(2*time.Hour), "12", time.Hour))
	t.Cleanup(server.Close)

	service := newCandleSyncService(t, db, server.URL)
	if err := service.Sync(ctx, "BTC"); err != nil {
		t.Fatalf("incremental sync: %v", err)
	}

	count, err := countCandles(ctx, db, symbol.ID, timeframe.ID)
	if err != nil {
		t.Fatalf("count candles: %v", err)
	}
	if count != 3 {
		t.Fatalf("unexpected candle count after incremental sync: got %d want 3", count)
	}
}

func TestCandleSyncServiceRecordsFailedIngestionJob(t *testing.T) {
	db := testutil.CreateTestDatabase(t)
	ctx := context.Background()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "upstream unavailable", http.StatusBadGateway)
	}))
	t.Cleanup(server.Close)

	service := newCandleSyncService(t, db, server.URL)
	err := service.Sync(ctx, "BTC")
	if err == nil {
		t.Fatal("expected sync to fail when binance returns an error")
	}

	var failedJobs int
	if err := db.QueryRow(`SELECT COUNT(*) FROM ingestion_jobs WHERE status = 'failed'`).Scan(&failedJobs); err != nil {
		t.Fatalf("count failed ingestion jobs: %v", err)
	}
	if failedJobs == 0 {
		t.Fatal("expected at least one failed ingestion job")
	}
}

func newCandleSyncService(t *testing.T, db *sql.DB, baseURL string) *services.CandleSyncService {
	t.Helper()

	return services.NewCandleSyncService(
		postgres.NewSymbolRepository(db),
		postgres.NewTimeframeRepository(db),
		postgres.NewCandleRepository(db),
		postgres.NewIngestionJobRepository(db),
		binance.NewClient(baseURL),
		30,
	)
}

func btcOneHourContext(t *testing.T, db *sql.DB, ctx context.Context) (*marketdata.Symbol, marketdata.Timeframe) {
	t.Helper()

	symbols := postgres.NewSymbolRepository(db)
	timeframes := postgres.NewTimeframeRepository(db)

	symbol, err := symbols.FindByAssetCode(ctx, "BTC")
	if err != nil || symbol == nil {
		t.Fatalf("find BTC symbol: %v", err)
	}
	timeframe, err := timeframes.FindByCode(ctx, "1h")
	if err != nil || timeframe == nil {
		t.Fatalf("find 1h timeframe: %v", err)
	}

	return symbol, *timeframe
}

func newOneShotBinanceServer(t *testing.T, batches map[string][][]any) *httptest.Server {
	t.Helper()

	var mu sync.Mutex
	served := make(map[string]bool)

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v3/klines" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		symbol := r.URL.Query().Get("symbol")
		interval := r.URL.Query().Get("interval")
		if interval != "1h" {
			writeKlines(t, w, nil)
			return
		}

		key := symbol + "|" + interval
		mu.Lock()
		defer mu.Unlock()
		if served[key] {
			writeKlines(t, w, nil)
			return
		}
		served[key] = true
		writeKlines(t, w, batches[key])
	}))
}

func newIncrementalBinanceServer(t *testing.T, threshold time.Time, rows ...[]any) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v3/klines" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		interval := r.URL.Query().Get("interval")
		if interval != "1h" {
			writeKlines(t, w, nil)
			return
		}

		startMS := r.URL.Query().Get("startTime")
		if startMS == "" {
			writeKlines(t, w, nil)
			return
		}

		parsedMS, err := strconv.ParseInt(startMS, 10, 64)
		if err != nil {
			t.Fatalf("parse startTime: %v", err)
		}
		startTime := time.UnixMilli(parsedMS).UTC()
		if startTime.Before(threshold) {
			writeKlines(t, w, nil)
			return
		}

		writeKlines(t, w, rows)
	}))
}

func writeKlines(t *testing.T, w http.ResponseWriter, payload [][]any) {
	t.Helper()

	if payload == nil {
		payload = [][]any{}
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		t.Fatalf("encode klines payload: %v", err)
	}
}

func binanceKlineRow(openTime time.Time, close string, duration time.Duration) []any {
	closeTime := openTime.Add(duration - time.Millisecond)
	return []any{
		float64(openTime.UnixMilli()),
		close,
		close,
		close,
		close,
		"100",
		float64(closeTime.UnixMilli()),
		"1000",
		float64(10),
		"50",
		"500",
	}
}

func testMarketCandle(symbolID, timeframeID int64, openTime time.Time, close string) marketdata.Candle {
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

func countCandles(ctx context.Context, db *sql.DB, symbolID, timeframeID int64) (int, error) {
	var count int
	err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM candles WHERE symbol_id = $1 AND timeframe_id = $2`, symbolID, timeframeID).Scan(&count)
	return count, err
}
