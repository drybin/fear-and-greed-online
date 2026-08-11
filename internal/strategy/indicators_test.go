package strategy

import (
	"testing"
	"time"

	domain "github.com/drybin/fear-and-greed-online/internal/domain/strategy"
)

func TestSMA(t *testing.T) {
	values := []float64{1, 2, 3, 4, 5}
	got := SMA(values, 3)

	want := []float64{0, 0, 2, 3, 4}
	if len(got) != len(want) {
		t.Fatalf("unexpected length: got %d want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("unexpected sma[%d]: got %v want %v", i, got[i], want[i])
		}
	}
}

func TestSMAWithInvalidPeriodReturnsZeros(t *testing.T) {
	values := []float64{1, 2, 3}
	got := SMA(values, 0)
	for i, value := range got {
		if value != 0 {
			t.Fatalf("expected zero at index %d, got %v", i, value)
		}
	}
}

func TestSMAWithSingleValuePeriod(t *testing.T) {
	got := SMA([]float64{4, 7, 1}, 1)
	want := []float64{4, 7, 1}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("unexpected sma[%d]: got %v want %v", i, got[i], want[i])
		}
	}
}

func TestHighestHigh(t *testing.T) {
	candles := []domain.Candle{
		{High: 10},
		{High: 12},
		{High: 8},
		{High: 15},
	}

	got := highestHigh(candles, 1, 4)
	if got != 15 {
		t.Fatalf("unexpected highest high: got %v want 15", got)
	}
}

func TestHighestHighHandlesEmptyRange(t *testing.T) {
	if got := highestHigh([]domain.Candle{{High: 10}}, 0, 0); got != 0 {
		t.Fatalf("expected zero for empty range, got %v", got)
	}
}

func TestHighestHighClampsOutOfBoundsRange(t *testing.T) {
	candles := []domain.Candle{{High: 3}, {High: 9}, {High: 5}}
	if got := highestHigh(candles, -5, 10); got != 9 {
		t.Fatalf("unexpected clamped highest high: got %v want 9", got)
	}
}

func TestParamHelpers(t *testing.T) {
	params := map[string]any{
		"int64":   int64(7),
		"float64": 2.5,
	}

	if got := intParam(params, "int64", 0); got != 7 {
		t.Fatalf("unexpected intParam: got %d want 7", got)
	}
	if got := floatParam(params, "float64", 0); got != 2.5 {
		t.Fatalf("unexpected floatParam: got %v want 2.5", got)
	}
	if got := intParam(nil, "missing", 42); got != 42 {
		t.Fatalf("unexpected intParam fallback: got %d want 42", got)
	}
	if got := floatParam(map[string]any{"int": 3}, "int", 0); got != 3 {
		t.Fatalf("unexpected floatParam from int: got %v want 3", got)
	}
	if got := floatParam(map[string]any{"bad": "x"}, "bad", 1.5); got != 1.5 {
		t.Fatalf("unexpected floatParam fallback: got %v want 1.5", got)
	}
}

func candleAt(base time.Time, offset int, close float64) domain.Candle {
	return domain.Candle{
		Time:   base.Add(time.Duration(offset) * time.Hour),
		Open:   close,
		High:   close + 1,
		Low:    close - 1,
		Close:  close,
		Volume: 100,
	}
}
