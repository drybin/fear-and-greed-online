package strategy

import (
	"testing"
	"time"

	domain "github.com/drybin/fear-and-greed-online/internal/domain/strategy"
)

func fifteenMinuteCandle(base time.Time, offset int, open, high, low, close float64) domain.Candle {
	return domain.Candle{
		Time:   base.Add(time.Duration(offset) * 15 * time.Minute),
		Open:   open,
		High:   high,
		Low:    low,
		Close:  close,
		Volume: 100,
	}
}

func TestBreakoutRetestV1GeneratesTargetExit(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	candles := []domain.Candle{
		fifteenMinuteCandle(base, 0, 100, 101, 99, 100),
		fifteenMinuteCandle(base, 1, 101, 102, 100, 101),
		fifteenMinuteCandle(base, 2, 102, 103, 101, 102),
		fifteenMinuteCandle(base, 3, 103, 104, 102, 103),
		fifteenMinuteCandle(base, 4, 104, 105, 103, 104),
		fifteenMinuteCandle(base, 5, 105, 106, 104, 106),
		fifteenMinuteCandle(base, 6, 106, 107, 104, 106),
		fifteenMinuteCandle(base, 7, 106, 111, 105, 110),
		fifteenMinuteCandle(base, 8, 110, 111, 109, 110),
	}

	out, err := NewBreakoutRetestV1().Run(RunInput{
		StrategySlug: "breakout-retest-v1",
		Symbol:       "BTC",
		Timeframe:    "15m",
		Candles:      candles,
		Params: map[string]any{
			"lookback_bars": 5,
			"retest_bars":   2,
			"risk_reward":   2.0,
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
	if len(out.Trades) != 1 || out.Trades[0].Status != "closed" {
		t.Fatalf("expected one closed trade, got %#v", out.Trades)
	}
}

func TestBreakoutRetestV1SkipsUnsupportedTimeframe(t *testing.T) {
	out, err := NewBreakoutRetestV1().Run(RunInput{
		StrategySlug: "breakout-retest-v1",
		Symbol:       "BTC",
		Timeframe:    "1h",
		Candles:      []domain.Candle{{}},
		Params:       map[string]any{"lookback_bars": 5, "retest_bars": 2, "risk_reward": 2.0},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Signals) != 0 || len(out.Trades) != 0 {
		t.Fatalf("expected no output for unsupported timeframe, got %#v", out)
	}
}

func TestBreakoutRetestV1GeneratesStopExit(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	candles := []domain.Candle{
		fifteenMinuteCandle(base, 0, 100, 101, 99, 100),
		fifteenMinuteCandle(base, 1, 101, 102, 100, 101),
		fifteenMinuteCandle(base, 2, 102, 103, 101, 102),
		fifteenMinuteCandle(base, 3, 103, 104, 102, 103),
		fifteenMinuteCandle(base, 4, 104, 105, 103, 104),
		fifteenMinuteCandle(base, 5, 105, 106, 104, 106),
		fifteenMinuteCandle(base, 6, 106, 107, 104, 106),
		fifteenMinuteCandle(base, 7, 106, 108, 103, 104),
		fifteenMinuteCandle(base, 8, 104, 105, 103, 104),
	}

	out, err := NewBreakoutRetestV1().Run(RunInput{
		StrategySlug: "breakout-retest-v1",
		Symbol:       "BTC",
		Timeframe:    "15m",
		Candles:      candles,
		Params: map[string]any{
			"lookback_bars": 5,
			"retest_bars":   2,
			"risk_reward":   2.0,
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Signals) != 2 || out.Signals[1].Title != "Breakout retest stop" {
		t.Fatalf("expected stop exit signal, got %#v", out.Signals)
	}
	if len(out.Trades) != 1 || out.Trades[0].Meta["reason"] != "stop" {
		t.Fatalf("expected stop trade metadata, got %#v", out.Trades[0])
	}
}

func TestBreakoutRetestV1ReturnsEmptyWhenInsufficientHistory(t *testing.T) {
	out, err := NewBreakoutRetestV1().Run(RunInput{
		StrategySlug: "breakout-retest-v1",
		Symbol:       "BTC",
		Timeframe:    "15m",
		Candles:      []domain.Candle{{}, {}, {}, {}, {}, {}, {}, {}},
		Params: map[string]any{
			"lookback_bars": 5,
			"retest_bars":   2,
			"risk_reward":   2.0,
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Signals) != 0 || len(out.Trades) != 0 {
		t.Fatalf("expected empty output, got %#v", out)
	}
}

func TestBreakoutRetestV1RejectsInvalidParams(t *testing.T) {
	_, err := NewBreakoutRetestV1().Run(RunInput{
		StrategySlug: "breakout-retest-v1",
		Symbol:       "BTC",
		Timeframe:    "15m",
		Candles:      []domain.Candle{{}, {}, {}, {}, {}, {}, {}, {}, {}, {}},
		Params: map[string]any{
			"lookback_bars": 1,
			"retest_bars":   2,
			"risk_reward":   2.0,
		},
	})
	if err == nil {
		t.Fatal("expected invalid params error")
	}
}
