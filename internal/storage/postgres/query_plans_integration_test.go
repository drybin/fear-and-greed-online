package postgres_test

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	marketdata "github.com/drybin/fear-and-greed-online/internal/domain/marketdata"
	domain "github.com/drybin/fear-and-greed-online/internal/domain/strategy"
	"github.com/drybin/fear-and-greed-online/internal/storage/postgres"
	"github.com/drybin/fear-and-greed-online/internal/testutil"
)

func TestCriticalQueriesUseMVPIndexes(t *testing.T) {
	db := testutil.CreateTestDatabase(t)
	ctx := context.Background()

	symbols := postgres.NewSymbolRepository(db)
	timeframes := postgres.NewTimeframeRepository(db)
	strategies := postgres.NewStrategyRepository(db)
	candles := postgres.NewCandleRepository(db)
	signals := postgres.NewSignalRepository(db)
	trades := postgres.NewTradeRepository(db)
	ingestion := postgres.NewIngestionJobRepository(db)
	runs := postgres.NewStrategyRunRepository(db)

	symbol, err := symbols.FindByAssetCode(ctx, "BTC")
	if err != nil || symbol == nil {
		t.Fatalf("find BTC symbol: %v", err)
	}
	timeframe, err := timeframes.FindByCode(ctx, "1h")
	if err != nil || timeframe == nil {
		t.Fatalf("find 1h timeframe: %v", err)
	}
	strategy, err := strategies.FindBySlug(ctx, "trend-long-v1")
	if err != nil || strategy == nil {
		t.Fatalf("find strategy: %v", err)
	}

	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	seed := make([]marketdata.Candle, 0, 720)
	for i := 0; i < 720; i++ {
		openTime := base.Add(time.Duration(i) * time.Hour)
		seed = append(seed, testCandle(symbol.ID, timeframe.ID, openTime, "10"))
	}
	if err := candles.UpsertMany(ctx, seed); err != nil {
		t.Fatalf("seed candles: %v", err)
	}

	inputTo := base.Add(24 * time.Hour)
	runID, err := runs.Start(ctx, strategy.ID, symbol.ID, timeframe.ID, strategy.DefaultParams, &base, &inputTo, 720)
	if err != nil {
		t.Fatalf("start strategy run: %v", err)
	}
	signalTime := base.Add(12 * time.Hour)
	if err := signals.UpsertMany(ctx, runID, strategy.ID, symbol.ID, timeframe.ID, []domain.Signal{{
		DedupeKey: "trend-long-v1|BTC|1h|entry|" + signalTime.Format(time.RFC3339),
		Time:      signalTime,
		Type:      "entry",
		Side:      "long",
		Price:     100,
		Status:    "confirmed",
		Title:     "Trend entry",
		Details:   "test",
	}}); err != nil {
		t.Fatalf("seed signal: %v", err)
	}
	if err := trades.UpsertMany(ctx, runID, strategy.ID, symbol.ID, timeframe.ID, []domain.Trade{{
		DedupeKey:  "trend-long-v1|BTC|1h|trade|" + signalTime.Format(time.RFC3339),
		EntryTime:  signalTime,
		Side:       "long",
		EntryPrice: 100,
		Status:     "open",
	}}); err != nil {
		t.Fatalf("seed trade: %v", err)
	}
	if err := runs.Complete(ctx, runID, 1, 1); err != nil {
		t.Fatalf("complete strategy run: %v", err)
	}

	jobID, err := ingestion.Start(ctx, symbol.ID, timeframe.ID, base, base.Add(30*24*time.Hour))
	if err != nil {
		t.Fatalf("start ingestion job: %v", err)
	}
	loadedTo := base.Add(24 * time.Hour)
	if err := ingestion.Complete(ctx, jobID, &base, &loadedTo, 720); err != nil {
		t.Fatalf("complete ingestion job: %v", err)
	}

	from := base
	to := base.Add(30 * 24 * time.Hour)
	cases := []struct {
		name      string
		query     string
		args      []any
		wantIndex string
	}{
		{
			name: "candles range",
			query: `
				SELECT open_time
				FROM candles
				WHERE symbol_id = $1 AND timeframe_id = $2 AND open_time >= $3 AND open_time <= $4
				ORDER BY open_time ASC`,
			args:      []any{symbol.ID, timeframe.ID, from, to},
			wantIndex: "candles_symbol_timeframe_open_time_idx",
		},
		{
			name: "signals range",
			query: `
				SELECT signal_time
				FROM signals
				WHERE strategy_id = $1 AND symbol_id = $2 AND timeframe_id = $3
					AND signal_time >= $4 AND signal_time <= $5
				ORDER BY signal_time ASC`,
			args:      []any{strategy.ID, symbol.ID, timeframe.ID, from, to},
			wantIndex: "signals_strategy_symbol_timeframe_signal_time_idx",
		},
		{
			name: "trades range",
			query: `
				SELECT entry_time
				FROM trades
				WHERE strategy_id = $1 AND symbol_id = $2 AND timeframe_id = $3
					AND entry_time >= $4 AND entry_time <= $5
				ORDER BY entry_time ASC`,
			args:      []any{strategy.ID, symbol.ID, timeframe.ID, from, to},
			wantIndex: "trades_strategy_symbol_timeframe_entry_time_idx",
		},
		{
			name: "latest successful ingestion",
			query: `
				SELECT finished_at
				FROM ingestion_jobs
				WHERE symbol_id = $1 AND timeframe_id = $2 AND status = 'success' AND finished_at IS NOT NULL
				ORDER BY finished_at DESC
				LIMIT 1`,
			args:      []any{symbol.ID, timeframe.ID},
			wantIndex: "ingestion_jobs_success_finished_at_idx",
		},
		{
			name: "recent strategy runs",
			query: `
				SELECT started_at
				FROM strategy_runs
				WHERE strategy_id = $1 AND symbol_id = $2 AND timeframe_id = $3
				ORDER BY started_at DESC
				LIMIT 10`,
			args:      []any{strategy.ID, symbol.ID, timeframe.ID},
			wantIndex: "strategy_runs_strategy_symbol_timeframe_started_at_idx",
		},
		{
			name: "latest successful strategy run",
			query: `
				SELECT finished_at
				FROM strategy_runs
				WHERE strategy_id = $1 AND symbol_id = $2 AND timeframe_id = $3 AND status = 'success' AND finished_at IS NOT NULL
				ORDER BY finished_at DESC
				LIMIT 1`,
			args:      []any{strategy.ID, symbol.ID, timeframe.ID},
			wantIndex: "strategy_runs_success_finished_at_idx",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			plan := explainPlan(t, db, tc.query, tc.args...)
			if !strings.Contains(plan, tc.wantIndex) {
				t.Fatalf("expected index %q in plan:\n%s", tc.wantIndex, plan)
			}
		})
	}
}

func explainPlan(t *testing.T, db *sql.DB, query string, args ...any) string {
	t.Helper()

	rows, err := db.QueryContext(context.Background(), "EXPLAIN "+query, args...)
	if err != nil {
		t.Fatalf("explain query: %v", err)
	}
	defer func() { _ = rows.Close() }()

	var lines []string
	for rows.Next() {
		var line string
		if err := rows.Scan(&line); err != nil {
			t.Fatalf("scan explain line: %v", err)
		}
		lines = append(lines, line)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("explain rows: %v", err)
	}

	return strings.Join(lines, "\n")
}
