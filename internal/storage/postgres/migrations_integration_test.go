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
	var beforeCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM schema_migrations`).Scan(&beforeCount); err != nil {
		t.Fatalf("count schema migrations before rollback: %v", err)
	}

	if err := postgres.RollbackLastMigration(db); err != nil {
		t.Fatalf("rollback last migration: %v", err)
	}

	var migrationCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM schema_migrations`).Scan(&migrationCount); err != nil {
		t.Fatalf("count schema migrations: %v", err)
	}
	if migrationCount != beforeCount-1 {
		t.Fatalf("unexpected migration count after rollback: got %d want %d", migrationCount, beforeCount-1)
	}

	// Latest migration 000010 adds requested Spot symbols; rollback must remove them.
	var remainingRequested int
	if err := db.QueryRow(`
		SELECT COUNT(*)
		FROM symbols
		WHERE asset_code IN ('1INCH', 'ANKR', 'APT', 'ARB', 'DASH', 'EGLD', 'STRK')
	`).Scan(&remainingRequested); err != nil {
		t.Fatalf("count rolled back symbols: %v", err)
	}
	if remainingRequested != 0 {
		t.Fatalf("expected requested symbols to be removed after rollback, got %d", remainingRequested)
	}

	var symbolCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM symbols`).Scan(&symbolCount); err != nil {
		t.Fatalf("count symbols after rollback: %v", err)
	}
	if symbolCount != 57 {
		t.Fatalf("unexpected symbol count after rollback: got %d want 57", symbolCount)
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
	if symbolCount != 64 {
		t.Fatalf("unexpected symbol count: got %d want 64", symbolCount)
	}

	var activeSymbols int
	if err := db.QueryRow(`SELECT COUNT(*) FROM symbols WHERE is_active = TRUE`).Scan(&activeSymbols); err != nil {
		t.Fatalf("count active symbols: %v", err)
	}
	if activeSymbols != 53 {
		t.Fatalf("unexpected active symbol count: got %d want 53", activeSymbols)
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
