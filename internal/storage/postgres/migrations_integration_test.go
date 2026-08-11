package postgres_test

import (
	"database/sql"
	"testing"

	"github.com/drybin/fear-and-greed-online/internal/storage/postgres"
	"github.com/drybin/fear-and-greed-online/internal/testutil"
)

func TestApplyMigrationsSeedsMarketAndStrategyReferenceData(t *testing.T) {
	db := testutil.CreateTestDatabase(t)

	assertSeededReferenceData(t, db)
}

func TestRollbackLastMigration(t *testing.T) {
	db := testutil.CreateTestDatabase(t)

	if err := postgres.RollbackLastMigration(db); err != nil {
		t.Fatalf("rollback last migration: %v", err)
	}

	var migrationCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM schema_migrations`).Scan(&migrationCount); err != nil {
		t.Fatalf("count schema migrations: %v", err)
	}
	if migrationCount != 5 {
		t.Fatalf("unexpected migration count after rollback: got %d want 5", migrationCount)
	}

	var indexExists bool
	if err := db.QueryRow(`
		SELECT EXISTS (
			SELECT 1
			FROM pg_indexes
			WHERE indexname = 'signals_strategy_symbol_timeframe_signal_time_idx'
		)
	`).Scan(&indexExists); err != nil {
		t.Fatalf("check rolled back index: %v", err)
	}
	if indexExists {
		t.Fatal("expected mvp query index to be removed after rollback")
	}

	if err := postgres.ApplyMigrations(db); err != nil {
		t.Fatalf("re-apply migrations: %v", err)
	}

	assertSeededReferenceData(t, db)
}

func TestResetMigrations(t *testing.T) {
	db := testutil.CreateTestDatabase(t)

	if _, err := db.Exec(`INSERT INTO candles (
		symbol_id, timeframe_id, open_time, close_time, open, high, low, close,
		volume, quote_volume, trades, taker_buy_base_volume, taker_buy_quote_volume,
		is_closed, source
	) VALUES (
		(SELECT id FROM symbols WHERE asset_code = 'BTC'),
		(SELECT id FROM timeframes WHERE code = '1h'),
		NOW(), NOW() + INTERVAL '1 hour',
		1, 1, 1, 1,
		1, 1, 1, 1, 1,
		TRUE, 'test'
	)`); err != nil {
		t.Fatalf("insert candle before reset: %v", err)
	}

	if err := postgres.ResetMigrations(db); err != nil {
		t.Fatalf("reset migrations: %v", err)
	}

	var candleCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM candles`).Scan(&candleCount); err != nil {
		t.Fatalf("count candles after reset: %v", err)
	}
	if candleCount != 0 {
		t.Fatalf("expected candles to be cleared by reset, got %d", candleCount)
	}

	assertSeededReferenceData(t, db)
}

func assertSeededReferenceData(t *testing.T, db *sql.DB) {
	t.Helper()

	_ = postgres.NewSymbolRepository(db)

	var symbolCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM symbols`).Scan(&symbolCount); err != nil {
		t.Fatalf("count symbols: %v", err)
	}
	if symbolCount != 50 {
		t.Fatalf("unexpected symbol count: got %d want 50", symbolCount)
	}

	var activeSymbols int
	if err := db.QueryRow(`SELECT COUNT(*) FROM symbols WHERE is_active = TRUE`).Scan(&activeSymbols); err != nil {
		t.Fatalf("count active symbols: %v", err)
	}
	if activeSymbols != 38 {
		t.Fatalf("unexpected active symbol count: got %d want 38", activeSymbols)
	}

	var timeframeCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM timeframes`).Scan(&timeframeCount); err != nil {
		t.Fatalf("count timeframes: %v", err)
	}
	if timeframeCount != 3 {
		t.Fatalf("unexpected timeframe count: got %d want 3", timeframeCount)
	}

	var strategyCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM strategies WHERE is_active = TRUE`).Scan(&strategyCount); err != nil {
		t.Fatalf("count strategies: %v", err)
	}
	if strategyCount != 3 {
		t.Fatalf("unexpected strategy count: got %d want 3", strategyCount)
	}
}
