package acceptance_test

import (
	"context"
	"database/sql"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/drybin/fear-and-greed-online/internal/acceptance"
	"github.com/drybin/fear-and-greed-online/internal/app/api"
	marketdata "github.com/drybin/fear-and-greed-online/internal/domain/marketdata"
	strategysvc "github.com/drybin/fear-and-greed-online/internal/services/strategy"
	"github.com/drybin/fear-and-greed-online/internal/storage/postgres"
	engine "github.com/drybin/fear-and-greed-online/internal/strategy"
	"github.com/drybin/fear-and-greed-online/internal/testutil"
)

func TestMVPFrozenUniverseAcceptance(t *testing.T) {
	db := testutil.CreateTestDatabase(t)
	ctx := context.Background()

	if err := acceptance.VerifyFrozenUniverse(ctx, db); err != nil {
		t.Fatalf("verify frozen universe: %v", err)
	}
	if err := acceptance.VerifyCLIQueryableUniverse(ctx, postgres.NewSymbolRepository(db)); err != nil {
		t.Fatalf("verify cli universe: %v", err)
	}

	seedStrategyFixtures(t, db, ctx)

	chdirRepoRoot(t)
	server := httptest.NewServer(api.NewServeMux(db))
	t.Cleanup(server.Close)

	client := server.Client()
	if err := acceptance.VerifyAPIUniverse(client, server.URL); err != nil {
		t.Fatalf("verify api universe: %v", err)
	}
	if err := acceptance.VerifyInspectableStrategyResult(client, server.URL, "BTC", "1h", "trend-long-v1"); err != nil {
		t.Fatalf("verify trend-long-v1: %v", err)
	}
	if err := acceptance.VerifyInspectableStrategyResult(client, server.URL, "BTC", "15m", "breakout-retest-v1"); err != nil {
		t.Fatalf("verify breakout-retest-v1: %v", err)
	}
}

func seedStrategyFixtures(t *testing.T, db *sql.DB, ctx context.Context) {
	t.Helper()

	if _, err := db.Exec(`UPDATE strategies SET required_history_bars = 3 WHERE slug = 'trend-long-v1'`); err != nil {
		t.Fatalf("lower trend-long history: %v", err)
	}
	if _, err := db.Exec(`UPDATE strategies SET default_params = '{"sma_period": 3}'::jsonb WHERE slug = 'trend-long-v1'`); err != nil {
		t.Fatalf("lower trend-long sma: %v", err)
	}
	if _, err := db.Exec(`UPDATE strategies SET required_history_bars = 9, default_params = '{"lookback_bars": 5, "retest_bars": 2, "risk_reward": 2.0}'::jsonb WHERE slug = 'breakout-retest-v1'`); err != nil {
		t.Fatalf("lower breakout params: %v", err)
	}

	symbolRepo := postgres.NewSymbolRepository(db)
	timeframeRepo := postgres.NewTimeframeRepository(db)
	candleRepo := postgres.NewCandleRepository(db)

	symbol, err := symbolRepo.FindByAssetCode(ctx, "BTC")
	if err != nil || symbol == nil {
		t.Fatalf("find BTC symbol: %v", err)
	}

	if err := upsertTrendFixtureCandles(ctx, candleRepo, timeframeRepo, symbol, "1h"); err != nil {
		t.Fatalf("seed trend candles: %v", err)
	}
	if err := upsertBreakoutFixtureCandles(ctx, candleRepo, timeframeRepo, symbol, "15m"); err != nil {
		t.Fatalf("seed breakout candles: %v", err)
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
		t.Fatalf("run trend-long-v1: %v", err)
	}
	if err := runner.Run(ctx, "breakout-retest-v1", "BTC"); err != nil {
		t.Fatalf("run breakout-retest-v1: %v", err)
	}
}

func upsertTrendFixtureCandles(ctx context.Context, candleRepo *postgres.CandleRepository, timeframeRepo *postgres.TimeframeRepository, symbol *marketdata.Symbol, timeframeCode string) error {
	timeframe, err := timeframeRepo.FindByCode(ctx, timeframeCode)
	if err != nil || timeframe == nil {
		return err
	}

	from := time.Now().UTC().Add(-8 * time.Hour).Truncate(time.Hour)
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
			Source:              "acceptance-test",
		})
	}
	return candleRepo.UpsertMany(ctx, candles)
}

func upsertBreakoutFixtureCandles(ctx context.Context, candleRepo *postgres.CandleRepository, timeframeRepo *postgres.TimeframeRepository, symbol *marketdata.Symbol, timeframeCode string) error {
	timeframe, err := timeframeRepo.FindByCode(ctx, timeframeCode)
	if err != nil || timeframe == nil {
		return err
	}

	base := time.Now().UTC().Add(-3 * time.Hour).Truncate(15 * time.Minute)
	series := []struct {
		open, high, low, close string
	}{
		{"100", "101", "99", "100"},
		{"101", "102", "100", "101"},
		{"102", "103", "101", "102"},
		{"103", "104", "102", "103"},
		{"104", "105", "103", "104"},
		{"105", "106", "104", "106"},
		{"106", "107", "104", "106"},
		{"106", "111", "105", "110"},
		{"110", "111", "109", "110"},
	}
	candles := make([]marketdata.Candle, 0, len(series))
	for i, bar := range series {
		openTime := base.Add(time.Duration(i) * 15 * time.Minute)
		candles = append(candles, marketdata.Candle{
			SymbolID:            symbol.ID,
			TimeframeID:         timeframe.ID,
			OpenTime:            openTime,
			CloseTime:           openTime.Add(15*time.Minute - time.Millisecond),
			Open:                bar.open,
			High:                bar.high,
			Low:                 bar.low,
			Close:               bar.close,
			Volume:              "100",
			QuoteVolume:         "1000",
			Trades:              10,
			TakerBuyBaseVolume:  "50",
			TakerBuyQuoteVolume: "500",
			IsClosed:            true,
			Source:              "acceptance-test",
		})
	}
	return candleRepo.UpsertMany(ctx, candles)
}

func chdirRepoRoot(t *testing.T) {
	t.Helper()

	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test file path")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(filename), "..", ".."))
	if err := os.Chdir(repoRoot); err != nil {
		t.Fatalf("chdir repo root: %v", err)
	}
}
