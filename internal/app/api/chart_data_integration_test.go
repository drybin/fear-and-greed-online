package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	marketdata "github.com/drybin/fear-and-greed-online/internal/domain/marketdata"
	strategysvc "github.com/drybin/fear-and-greed-online/internal/services/strategy"
	"github.com/drybin/fear-and-greed-online/internal/storage/postgres"
	engine "github.com/drybin/fear-and-greed-online/internal/strategy"
	"github.com/drybin/fear-and-greed-online/internal/testutil"
)

func TestChartDataEndpointReturnsCandlesSignalsTradesAndRuns(t *testing.T) {
	db := testutil.CreateTestDatabase(t)
	ctx := context.Background()

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

	base := time.Now().UTC().Add(-8 * time.Hour).Truncate(time.Hour)
	closes := []string{"10", "10", "10", "9", "11", "12", "11", "9"}
	candles := make([]marketdata.Candle, 0, len(closes))
	for i, closeValue := range closes {
		openTime := base.Add(time.Duration(i) * time.Hour)
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
	)
	if err := runner.Run(ctx, "trend-long-v1", "BTC"); err != nil {
		t.Fatalf("run strategy: %v", err)
	}

	mux := http.NewServeMux()
	registerRoutes(mux, db)

	req := httptest.NewRequest(http.MethodGet, "/chart-data?symbol=BTC&timeframe=1h&strategy=trend-long-v1&from="+base.Format(time.RFC3339)+"&to="+base.Add(8*time.Hour).Format(time.RFC3339), nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: got %d body=%s", rec.Code, rec.Body.String())
	}

	var payload struct {
		Candles    []any          `json:"candles"`
		Signals    []any          `json:"signals"`
		Trades     []any          `json:"trades"`
		RecentRuns []any          `json:"recent_runs"`
		Freshness  map[string]any `json:"freshness"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(payload.Candles) != 8 {
		t.Fatalf("unexpected candles count: got %d want 8", len(payload.Candles))
	}
	if len(payload.Signals) == 0 {
		t.Fatalf("expected non-empty signals")
	}
	if len(payload.Trades) == 0 {
		t.Fatalf("expected non-empty trades")
	}
	if len(payload.RecentRuns) == 0 {
		t.Fatalf("expected non-empty recent_runs")
	}
	if payload.Freshness == nil {
		t.Fatalf("expected freshness payload")
	}
	if _, ok := payload.Freshness["latest_candle_time"]; !ok {
		t.Fatalf("expected latest_candle_time in freshness payload")
	}
	if _, ok := payload.Freshness["last_strategy_run_at"]; !ok {
		t.Fatalf("expected last_strategy_run_at in freshness payload")
	}
}

func TestFreshnessEndpointReturnsOperationalContext(t *testing.T) {
	db := testutil.CreateTestDatabase(t)
	ctx := context.Background()

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

	base := time.Now().UTC().Add(-8 * time.Hour).Truncate(time.Hour)
	closes := []string{"10", "10", "10", "9", "11", "12", "11", "9"}
	candles := make([]marketdata.Candle, 0, len(closes))
	for i, closeValue := range closes {
		openTime := base.Add(time.Duration(i) * time.Hour)
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
	)
	if err := runner.Run(ctx, "trend-long-v1", "BTC"); err != nil {
		t.Fatalf("run strategy: %v", err)
	}

	mux := http.NewServeMux()
	registerRoutes(mux, db)

	req := httptest.NewRequest(http.MethodGet, "/freshness?symbol=BTC&timeframe=1h&strategy=trend-long-v1", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: got %d body=%s", rec.Code, rec.Body.String())
	}

	var payload struct {
		Symbol    map[string]any `json:"symbol"`
		Timeframe map[string]any `json:"timeframe"`
		Strategy  map[string]any `json:"strategy"`
		Freshness map[string]any `json:"freshness"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if payload.Freshness == nil {
		t.Fatalf("expected freshness payload")
	}
	if payload.Freshness["symbol"] != "BTC" {
		t.Fatalf("unexpected freshness symbol: %#v", payload.Freshness["symbol"])
	}
	if payload.Freshness["timeframe"] != "1h" {
		t.Fatalf("unexpected freshness timeframe: %#v", payload.Freshness["timeframe"])
	}
	if _, ok := payload.Freshness["latest_candle_time"]; !ok {
		t.Fatalf("expected latest_candle_time in freshness payload")
	}
	if _, ok := payload.Freshness["last_strategy_run_at"]; !ok {
		t.Fatalf("expected last_strategy_run_at in freshness payload")
	}
}
