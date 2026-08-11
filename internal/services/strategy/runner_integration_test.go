package strategy_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/drybin/fear-and-greed-online/internal/domain/marketdata"
	strategysvc "github.com/drybin/fear-and-greed-online/internal/services/strategy"
	"github.com/drybin/fear-and-greed-online/internal/storage/postgres"
	engine "github.com/drybin/fear-and-greed-online/internal/strategy"
	"github.com/drybin/fear-and-greed-online/internal/testutil"
)

func TestRunnerPersistsRunsSignalsAndTradesWithDeduplication(t *testing.T) {
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
	strategyRepo := postgres.NewStrategyRepository(db)
	candleRepo := postgres.NewCandleRepository(db)
	runRepo := postgres.NewStrategyRunRepository(db)
	signalRepo := postgres.NewSignalRepository(db)
	tradeRepo := postgres.NewTradeRepository(db)

	symbol, err := symbolRepo.FindByAssetCode(ctx, "BTC")
	if err != nil || symbol == nil {
		t.Fatalf("find BTC symbol: %v", err)
	}
	timeframe, err := timeframeRepo.FindByCode(ctx, "1h")
	if err != nil || timeframe == nil {
		t.Fatalf("find 1h timeframe: %v", err)
	}
	strategy, err := strategyRepo.FindBySlug(ctx, "trend-long-v1")
	if err != nil || strategy == nil {
		t.Fatalf("find strategy: %v", err)
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
		strategyRepo,
		symbolRepo,
		timeframeRepo,
		candleRepo,
		runRepo,
		signalRepo,
		tradeRepo,
		engine.NewRegistry(engine.NewTrendLongV1()),
		30,
	)

	if err := runner.Run(ctx, "trend-long-v1", "BTC"); err != nil {
		t.Fatalf("first strategy run: %v", err)
	}

	successRuns, err := countStrategyRuns(ctx, db, strategy.ID, symbol.ID, timeframe.ID, "success")
	if err != nil {
		t.Fatalf("count successful runs: %v", err)
	}
	if successRuns == 0 {
		t.Fatal("expected at least one successful strategy run")
	}

	signalCount, err := countSignals(ctx, db, strategy.ID, symbol.ID, timeframe.ID)
	if err != nil {
		t.Fatalf("count signals: %v", err)
	}
	if signalCount == 0 {
		t.Fatal("expected persisted signals")
	}

	tradeCount, err := countTrades(ctx, db, strategy.ID, symbol.ID, timeframe.ID)
	if err != nil {
		t.Fatalf("count trades: %v", err)
	}
	if tradeCount == 0 {
		t.Fatal("expected persisted trades")
	}

	latestRun, err := runRepo.LatestSuccessful(ctx, strategy.ID, symbol.ID, timeframe.ID)
	if err != nil {
		t.Fatalf("latest successful run: %v", err)
	}
	if latestRun == nil || latestRun.SignalsCount == 0 || latestRun.TradesCount == 0 {
		t.Fatalf("unexpected latest successful run summary: %#v", latestRun)
	}

	if err := runner.Run(ctx, "trend-long-v1", "BTC"); err != nil {
		t.Fatalf("second strategy run: %v", err)
	}

	successRunsAfter, err := countStrategyRuns(ctx, db, strategy.ID, symbol.ID, timeframe.ID, "success")
	if err != nil {
		t.Fatalf("count successful runs after rerun: %v", err)
	}
	if successRunsAfter <= successRuns {
		t.Fatalf("expected additional successful runs after rerun: before=%d after=%d", successRuns, successRunsAfter)
	}

	signalCountAfter, err := countSignals(ctx, db, strategy.ID, symbol.ID, timeframe.ID)
	if err != nil {
		t.Fatalf("count signals after rerun: %v", err)
	}
	if signalCountAfter != signalCount {
		t.Fatalf("expected deduplicated signal count to remain stable: before=%d after=%d", signalCount, signalCountAfter)
	}
}

func TestRunnerReplacesStaleSignalsAndTradesFromPreviousRun(t *testing.T) {
	db := testutil.CreateTestDatabase(t)
	ctx := context.Background()

	if _, err := db.Exec(`UPDATE strategies SET required_history_bars = 3 WHERE slug = 'trend-long-v1'`); err != nil {
		t.Fatalf("lower required_history_bars: %v", err)
	}
	if _, err := db.Exec(`UPDATE strategies SET default_params = '{"sma_period": 3}'::jsonb WHERE slug = 'trend-long-v1'`); err != nil {
		t.Fatalf("set initial sma_period: %v", err)
	}

	symbolRepo := postgres.NewSymbolRepository(db)
	timeframeRepo := postgres.NewTimeframeRepository(db)
	strategyRepo := postgres.NewStrategyRepository(db)
	candleRepo := postgres.NewCandleRepository(db)
	runRepo := postgres.NewStrategyRunRepository(db)
	signalRepo := postgres.NewSignalRepository(db)
	tradeRepo := postgres.NewTradeRepository(db)

	symbol, err := symbolRepo.FindByAssetCode(ctx, "BTC")
	if err != nil || symbol == nil {
		t.Fatalf("find BTC symbol: %v", err)
	}
	timeframe, err := timeframeRepo.FindByCode(ctx, "1h")
	if err != nil || timeframe == nil {
		t.Fatalf("find 1h timeframe: %v", err)
	}
	strategy, err := strategyRepo.FindBySlug(ctx, "trend-long-v1")
	if err != nil || strategy == nil {
		t.Fatalf("find strategy: %v", err)
	}

	base := time.Now().UTC().Add(-40 * 24 * time.Hour).Truncate(time.Hour)
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
		strategyRepo,
		symbolRepo,
		timeframeRepo,
		candleRepo,
		runRepo,
		signalRepo,
		tradeRepo,
		engine.NewRegistry(engine.NewTrendLongV1()),
		30,
	)

	if err := runner.Run(ctx, "trend-long-v1", "BTC"); err != nil {
		t.Fatalf("first strategy run: %v", err)
	}

	signalCount, err := countSignals(ctx, db, strategy.ID, symbol.ID, timeframe.ID)
	if err != nil {
		t.Fatalf("count signals: %v", err)
	}
	tradeCount, err := countTrades(ctx, db, strategy.ID, symbol.ID, timeframe.ID)
	if err != nil {
		t.Fatalf("count trades: %v", err)
	}
	if signalCount == 0 || tradeCount == 0 {
		t.Fatalf("expected initial persisted results, got signals=%d trades=%d", signalCount, tradeCount)
	}

	if _, err := db.Exec(`UPDATE strategies SET default_params = '{"sma_period": 8}'::jsonb WHERE slug = 'trend-long-v1'`); err != nil {
		t.Fatalf("set larger sma_period: %v", err)
	}

	if err := runner.Run(ctx, "trend-long-v1", "BTC"); err != nil {
		t.Fatalf("second strategy run: %v", err)
	}

	signalCountAfter, err := countSignals(ctx, db, strategy.ID, symbol.ID, timeframe.ID)
	if err != nil {
		t.Fatalf("count signals after rerun: %v", err)
	}
	tradeCountAfter, err := countTrades(ctx, db, strategy.ID, symbol.ID, timeframe.ID)
	if err != nil {
		t.Fatalf("count trades after rerun: %v", err)
	}
	if signalCountAfter != 0 {
		t.Fatalf("expected stale signals to be removed, got %d", signalCountAfter)
	}
	if tradeCountAfter != 0 {
		t.Fatalf("expected stale trades to be removed, got %d", tradeCountAfter)
	}
}

func countStrategyRuns(ctx context.Context, db *sql.DB, strategyID, symbolID, timeframeID int64, status string) (int, error) {
	var count int
	err := db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM strategy_runs
		WHERE strategy_id = $1 AND symbol_id = $2 AND timeframe_id = $3 AND status = $4
	`, strategyID, symbolID, timeframeID, status).Scan(&count)
	return count, err
}

func countSignals(ctx context.Context, db *sql.DB, strategyID, symbolID, timeframeID int64) (int, error) {
	var count int
	err := db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM signals
		WHERE strategy_id = $1 AND symbol_id = $2 AND timeframe_id = $3
	`, strategyID, symbolID, timeframeID).Scan(&count)
	return count, err
}

func countTrades(ctx context.Context, db *sql.DB, strategyID, symbolID, timeframeID int64) (int, error) {
	var count int
	err := db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM trades
		WHERE strategy_id = $1 AND symbol_id = $2 AND timeframe_id = $3
	`, strategyID, symbolID, timeframeID).Scan(&count)
	return count, err
}
