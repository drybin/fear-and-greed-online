package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	marketdata "github.com/drybin/fear-and-greed-online/internal/domain/marketdata"
	strategydomain "github.com/drybin/fear-and-greed-online/internal/domain/strategy"
	strategysvc "github.com/drybin/fear-and-greed-online/internal/services/strategy"
	"github.com/drybin/fear-and-greed-online/internal/storage/postgres"
	engine "github.com/drybin/fear-and-greed-online/internal/strategy"
	"github.com/drybin/fear-and-greed-online/internal/testutil"
)

func TestDashboardWorkflowEndToEndAgainstSeededData(t *testing.T) {
	db := testutil.CreateTestDatabase(t)
	ctx := context.Background()
	from, to := seedBTCChartWorkflow(t, db, ctx)

	mux := newTestMux(t, db)

	var symbolsPayload struct {
		Items []struct {
			AssetCode string `json:"AssetCode"`
			IsActive  bool   `json:"IsActive"`
		} `json:"items"`
		ActiveCount int `json:"active_count"`
	}
	decodeGET(t, mux, "/symbols/active", &symbolsPayload)
	if symbolsPayload.ActiveCount == 0 {
		t.Fatalf("expected active symbols in seeded universe")
	}
	if !containsAssetCode(symbolsPayload.Items, "BTC") {
		t.Fatalf("expected BTC in active symbols: %#v", symbolsPayload.Items)
	}

	var timeframesPayload struct {
		Items []struct {
			Code string `json:"Code"`
		} `json:"items"`
	}
	decodeGET(t, mux, "/timeframes", &timeframesPayload)
	if !containsTimeframe(timeframesPayload.Items, "1h") {
		t.Fatalf("expected 1h timeframe: %#v", timeframesPayload.Items)
	}

	var strategiesPayload struct {
		Items []struct {
			Slug string `json:"Slug"`
			Name string `json:"Name"`
		} `json:"items"`
	}
	decodeGET(t, mux, "/strategies", &strategiesPayload)
	if !containsStrategy(strategiesPayload.Items, "trend-long-v1") {
		t.Fatalf("expected trend-long-v1 strategy: %#v", strategiesPayload.Items)
	}

	chartURL := "/chart-data?symbol=BTC&timeframe=1h&strategy=trend-long-v1&from=" + from.Format(time.RFC3339) + "&to=" + to.Format(time.RFC3339)
	var chartPayload struct {
		Symbol struct {
			AssetCode string `json:"AssetCode"`
		} `json:"symbol"`
		Timeframe struct {
			Code string `json:"Code"`
		} `json:"timeframe"`
		Strategy struct {
			Slug string `json:"Slug"`
		} `json:"strategy"`
		Candles    []any          `json:"candles"`
		Signals    []any          `json:"signals"`
		Trades     []any          `json:"trades"`
		RecentRuns []any          `json:"recent_runs"`
		Freshness  map[string]any `json:"freshness"`
	}
	decodeGET(t, mux, chartURL, &chartPayload)

	if chartPayload.Symbol.AssetCode != "BTC" || chartPayload.Timeframe.Code != "1h" || chartPayload.Strategy.Slug != "trend-long-v1" {
		t.Fatalf("unexpected chart context: %#v", chartPayload)
	}
	if len(chartPayload.Candles) == 0 {
		t.Fatal("expected chart candles for dashboard rendering")
	}
	if len(chartPayload.Signals) == 0 {
		t.Fatal("expected chart signals for marker rendering")
	}
	if len(chartPayload.Trades) == 0 {
		t.Fatal("expected chart trades for dashboard summary")
	}
	if len(chartPayload.RecentRuns) == 0 {
		t.Fatal("expected recent strategy runs for dashboard sidebar")
	}
	if chartPayload.Freshness == nil || chartPayload.Freshness["latest_candle_time"] == nil {
		t.Fatalf("expected freshness context in chart payload: %#v", chartPayload.Freshness)
	}

	signalsURL := "/signals?symbol=BTC&timeframe=1h&strategy=trend-long-v1&from=" + from.Format(time.RFC3339) + "&to=" + to.Format(time.RFC3339)
	var signalsPayload struct {
		Items []struct {
			Title   string         `json:"Title"`
			Details string         `json:"Details"`
			Meta    map[string]any `json:"Meta"`
		} `json:"items"`
	}
	decodeGET(t, mux, signalsURL, &signalsPayload)
	if len(signalsPayload.Items) == 0 {
		t.Fatal("expected inspectable signals for dashboard detail panel")
	}
	firstSignal := signalsPayload.Items[0]
	if firstSignal.Title == "" || firstSignal.Details == "" {
		t.Fatalf("expected signal detail fields for inspection panel: %#v", firstSignal)
	}

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("dashboard shell status: got %d", rec.Code)
	}

	body := rec.Body.String()
	for _, needle := range []string{
		`id="signalsNow"`,
		`id="symbol"`,
		`id="timeframe"`,
		`id="strategy"`,
		`id="from"`,
		`id="to"`,
		`id="chart"`,
		`id="freshnessSync"`,
		`id="detailContent"`,
		`fetchJSON('/symbols/active')`,
		"`/chart-data?${params.toString()}`",
		"lightweight-charts",
		"Asia/Bangkok",
		"side-long",
		"#2f6fed",
		`id="themeToggle"`,
		"data-theme",
		"Max-Age=31536000",
	} {
		if !strings.Contains(body, needle) {
			t.Fatalf("dashboard shell missing %q", needle)
		}
	}
}

func TestChartEndpointsRejectUnsupportedStrategyTimeframeCombination(t *testing.T) {
	db := testutil.CreateTestDatabase(t)
	mux := newTestMux(t, db)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/chart-data?symbol=BTC&timeframe=4h&strategy=breakout-retest-v1", nil)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for unsupported strategy/timeframe, got %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "does not support timeframe") {
		t.Fatalf("expected unsupported timeframe error, got body=%s", rec.Body.String())
	}
}

func TestSignalsNowEndpointReturnsOnlyLatestCandleHits(t *testing.T) {
	db := testutil.CreateTestDatabase(t)
	ctx := context.Background()

	symbols := postgres.NewSymbolRepository(db)
	timeframes := postgres.NewTimeframeRepository(db)
	strategies := postgres.NewStrategyRepository(db)
	candles := postgres.NewCandleRepository(db)
	runs := postgres.NewStrategyRunRepository(db)
	signalRepo := postgres.NewSignalRepository(db)

	btc, err := symbols.FindByAssetCode(ctx, "BTC")
	if err != nil || btc == nil {
		t.Fatalf("find BTC: %v", err)
	}
	hourly, err := timeframes.FindByCode(ctx, "1h")
	if err != nil || hourly == nil {
		t.Fatalf("find 1h: %v", err)
	}
	strategy, err := strategies.FindBySlug(ctx, "trend-long-v1")
	if err != nil || strategy == nil {
		t.Fatalf("find strategy: %v", err)
	}

	older := time.Date(2026, 8, 17, 8, 0, 0, 0, time.UTC)
	latest := older.Add(time.Hour)
	if err := candles.UpsertMany(ctx, []marketdata.Candle{
		{
			SymbolID: btc.ID, TimeframeID: hourly.ID, OpenTime: older, CloseTime: older.Add(time.Hour - time.Millisecond),
			Open: "100", High: "100", Low: "100", Close: "100", Volume: "1", QuoteVolume: "1", Trades: 1,
			TakerBuyBaseVolume: "1", TakerBuyQuoteVolume: "1", IsClosed: true, Source: "test",
		},
		{
			SymbolID: btc.ID, TimeframeID: hourly.ID, OpenTime: latest, CloseTime: latest.Add(time.Hour - time.Millisecond),
			Open: "110", High: "110", Low: "110", Close: "110", Volume: "1", QuoteVolume: "1", Trades: 1,
			TakerBuyBaseVolume: "1", TakerBuyQuoteVolume: "1", IsClosed: true, Source: "test",
		},
	}); err != nil {
		t.Fatalf("seed candles: %v", err)
	}

	runID, err := runs.Start(ctx, strategy.ID, btc.ID, hourly.ID, strategy.DefaultParams, &older, &latest, 2)
	if err != nil {
		t.Fatalf("start run: %v", err)
	}
	if err := signalRepo.UpsertMany(ctx, runID, strategy.ID, btc.ID, hourly.ID, []strategydomain.Signal{
		{
			DedupeKey: "trend-long-v1|BTC|1h|stale|" + older.Format(time.RFC3339),
			Time:      older,
			Type:      "alert",
			Side:      "long",
			Price:     100,
			Status:    "confirmed",
			Title:     "stale",
			Details:   "old candle",
		},
		{
			DedupeKey: "trend-long-v1|BTC|1h|live|" + latest.Format(time.RFC3339),
			Time:      latest,
			Type:      "alert",
			Side:      "short",
			Price:     110,
			Status:    "confirmed",
			Title:     "live",
			Details:   "latest candle",
		},
	}); err != nil {
		t.Fatalf("seed signals: %v", err)
	}

	mux := newTestMux(t, db)
	var payload struct {
		Items []struct {
			Title        string  `json:"Title"`
			AssetCode    string  `json:"AssetCode"`
			Timeframe    string  `json:"Timeframe"`
			StrategySlug string  `json:"StrategySlug"`
			Side         string  `json:"Side"`
			Price        float64 `json:"Price"`
		} `json:"items"`
		Count int `json:"count"`
	}
	decodeGET(t, mux, "/signals/now", &payload)
	if payload.Count != 1 || len(payload.Items) != 1 {
		t.Fatalf("expected one live signal, got count=%d items=%#v", payload.Count, payload.Items)
	}
	item := payload.Items[0]
	if item.Title != "live" || item.AssetCode != "BTC" || item.Timeframe != "1h" || item.StrategySlug != "trend-long-v1" || item.Side != "short" {
		t.Fatalf("unexpected live signal: %#v", item)
	}
}

func seedBTCChartWorkflow(t *testing.T, db *sql.DB, ctx context.Context) (time.Time, time.Time) {
	t.Helper()

	if _, err := db.Exec(`UPDATE strategies SET required_history_bars = 3 WHERE slug = 'trend-long-v1'`); err != nil {
		t.Fatalf("lower required_history_bars: %v", err)
	}
	if _, err := db.Exec(`UPDATE strategies SET default_params = '{"sma_period": 3}'::jsonb WHERE slug = 'trend-long-v1'`); err != nil {
		t.Fatalf("lower default sma_period: %v", err)
	}

	symbolRepo := postgres.NewSymbolRepository(db)
	timeframeRepo := postgres.NewTimeframeRepository(db)
	candleRepo := postgres.NewCandleRepository(db)

	symbol, err := symbolRepo.FindByAssetCode(ctx, "BTC")
	if err != nil || symbol == nil {
		t.Fatalf("find BTC symbol: %v", err)
	}
	timeframe, err := timeframeRepo.FindByCode(ctx, "1h")
	if err != nil || timeframe == nil {
		t.Fatalf("find 1h timeframe: %v", err)
	}

	from := time.Now().UTC().Add(-8 * time.Hour).Truncate(time.Hour)
	to := from.Add(8 * time.Hour)
	closes := []string{"10", "10", "10", "9", "11", "12", "11", "9"}
	candles := make([]marketdata.Candle, 0, len(closes))
	for i, closeValue := range closes {
		openTime := from.Add(time.Duration(i) * time.Hour)
		candles = append(candles, marketdata.Candle{
			SymbolID:            symbol.ID,
			TimeframeID:         timeframe.ID,
			OpenTime:            openTime,
			CloseTime:           openTime.Add(time.Hour - time.Millisecond),
			Open:                closeValue,
			High:                "12",
			Low:                 "8",
			Close:               closeValue,
			Volume:              "100",
			QuoteVolume:         "1000",
			Trades:              10,
			TakerBuyBaseVolume:  "50",
			TakerBuyQuoteVolume: "500",
			IsClosed:            true,
			Source:              "test",
		})
	}
	if err := candleRepo.UpsertMany(ctx, candles); err != nil {
		t.Fatalf("insert candles: %v", err)
	}

	runner := strategysvc.NewRunner(
		postgres.NewStrategyRepository(db),
		symbolRepo,
		timeframeRepo,
		candleRepo,
		postgres.NewStrategyRunRepository(db),
		postgres.NewSignalRepository(db),
		postgres.NewTradeRepository(db),
		engine.NewRegistry(engine.NewTrendLongV1(), engine.NewBreakoutRetestV1(), engine.NewPrevDayRangeBreakoutV1()),
		30,
		nil,
	)
	if err := runner.Run(ctx, "trend-long-v1", "BTC"); err != nil {
		t.Fatalf("run strategy: %v", err)
	}

	return from, to
}

func newTestMux(t *testing.T, db *sql.DB) *http.ServeMux {
	t.Helper()

	repoRoot := repositoryRoot(t)
	if err := os.Chdir(repoRoot); err != nil {
		t.Fatalf("chdir repo root: %v", err)
	}

	mux := http.NewServeMux()
	registerRoutes(mux, db)
	return mux
}

func repositoryRoot(t *testing.T) string {
	t.Helper()

	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test file path")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(filename), "..", "..", ".."))
}

func decodeGET(t *testing.T, mux http.Handler, path string, target any) {
	t.Helper()

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET %s: got status %d body=%s", path, rec.Code, rec.Body.String())
	}
	if err := json.Unmarshal(rec.Body.Bytes(), target); err != nil {
		t.Fatalf("decode GET %s: %v", path, err)
	}
}

func containsAssetCode(items []struct {
	AssetCode string `json:"AssetCode"`
	IsActive  bool   `json:"IsActive"`
}, code string) bool {
	for _, item := range items {
		if item.AssetCode == code && item.IsActive {
			return true
		}
	}
	return false
}

func containsTimeframe(items []struct {
	Code string `json:"Code"`
}, code string) bool {
	for _, item := range items {
		if item.Code == code {
			return true
		}
	}
	return false
}

func containsStrategy(items []struct {
	Slug string `json:"Slug"`
	Name string `json:"Name"`
}, slug string) bool {
	for _, item := range items {
		if item.Slug == slug {
			return true
		}
	}
	return false
}
