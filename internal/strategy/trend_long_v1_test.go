package strategy

import (
	"testing"
	"time"

	domain "github.com/drybin/fear-and-greed-online/internal/domain/strategy"
)

func TestTrendLongV1GeneratesEntryExitAndClosedTrade(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	candles := make([]domain.Candle, 0, 8)
	closes := []float64{10, 10, 10, 9, 11, 12, 11, 9}
	for i, closeValue := range closes {
		candles = append(candles, candleAt(base, i, closeValue))
	}

	out, err := NewTrendLongV1().Run(RunInput{
		StrategySlug: "trend-long-v1",
		Symbol:       "BTC",
		Timeframe:    "1h",
		Candles:      candles,
		Params: map[string]any{
			"sma_period": 3,
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(out.Signals) != 2 {
		t.Fatalf("unexpected signals count: got %d want 2", len(out.Signals))
	}
	if out.Signals[0].Type != "entry" || out.Signals[1].Type != "exit" {
		t.Fatalf("unexpected signal sequence: %#v", out.Signals)
	}
	if len(out.Trades) != 1 {
		t.Fatalf("unexpected trades count: got %d want 1", len(out.Trades))
	}
	if out.Trades[0].Status != "closed" {
		t.Fatalf("expected closed trade, got %s", out.Trades[0].Status)
	}
}

func TestTrendLongV1ReturnsOpenTradeWhenNoExit(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	candles := make([]domain.Candle, 0, 7)
	closes := []float64{10, 10, 10, 9, 11, 12, 13}
	for i, closeValue := range closes {
		candles = append(candles, candleAt(base, i, closeValue))
	}

	out, err := NewTrendLongV1().Run(RunInput{
		StrategySlug: "trend-long-v1",
		Symbol:       "BTC",
		Timeframe:    "1h",
		Candles:      candles,
		Params: map[string]any{
			"sma_period": 3,
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(out.Signals) != 1 {
		t.Fatalf("unexpected signals count: got %d want 1", len(out.Signals))
	}
	if len(out.Trades) != 1 || out.Trades[0].Status != "open" {
		t.Fatalf("expected one open trade, got %#v", out.Trades)
	}
}

func TestTrendLongV1RejectsInvalidPeriod(t *testing.T) {
	_, err := NewTrendLongV1().Run(RunInput{
		StrategySlug: "trend-long-v1",
		Symbol:       "BTC",
		Timeframe:    "1h",
		Candles:      []domain.Candle{{Close: 10}, {Close: 11}},
		Params:       map[string]any{"sma_period": 1},
	})
	if err == nil {
		t.Fatal("expected invalid sma_period error")
	}
}

func TestTrendLongV1ReturnsEmptyWhenInsufficientHistory(t *testing.T) {
	out, err := NewTrendLongV1().Run(RunInput{
		StrategySlug: "trend-long-v1",
		Symbol:       "BTC",
		Timeframe:    "1h",
		Candles:      []domain.Candle{{Close: 10}, {Close: 11}, {Close: 12}},
		Params:       map[string]any{"sma_period": 5},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Signals) != 0 || len(out.Trades) != 0 {
		t.Fatalf("expected empty output, got %#v", out)
	}
}

func TestTrendLongV1UsesStableDedupeKeys(t *testing.T) {
	base := time.Date(2026, 1, 1, 4, 0, 0, 0, time.UTC)
	candles := make([]domain.Candle, 0, 8)
	closes := []float64{10, 10, 10, 9, 11, 12, 11, 9}
	for i, closeValue := range closes {
		candles = append(candles, candleAt(base, i, closeValue))
	}

	out, err := NewTrendLongV1().Run(RunInput{
		StrategySlug: "trend-long-v1",
		Symbol:       "BTC",
		Timeframe:    "1h",
		Candles:      candles,
		Params:       map[string]any{"sma_period": 3},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	wantEntryKey := "trend-long-v1|BTC|1h|entry|" + candles[4].Time.UTC().Format(time.RFC3339)
	wantExitKey := "trend-long-v1|BTC|1h|exit|" + candles[6].Time.UTC().Format(time.RFC3339)
	if out.Signals[0].DedupeKey != wantEntryKey {
		t.Fatalf("unexpected entry dedupe key: got %q want %q", out.Signals[0].DedupeKey, wantEntryKey)
	}
	if out.Signals[1].DedupeKey != wantExitKey {
		t.Fatalf("unexpected exit dedupe key: got %q want %q", out.Signals[1].DedupeKey, wantExitKey)
	}
	if out.Signals[0].Meta["sma_period"] != 3 {
		t.Fatalf("expected sma_period metadata, got %#v", out.Signals[0].Meta)
	}
}
