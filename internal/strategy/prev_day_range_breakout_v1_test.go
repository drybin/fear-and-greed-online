package strategy

import (
	"testing"
	"time"

	domain "github.com/drybin/fear-and-greed-online/internal/domain/strategy"
)

func moscowHourCandle(day time.Time, hour int, open, high, low, close float64) domain.Candle {
	msk := mustMoscow()
	openTime := time.Date(day.Year(), day.Month(), day.Day(), hour, 0, 0, 0, msk)
	return domain.Candle{
		Time:   openTime,
		Open:   open,
		High:   high,
		Low:    low,
		Close:  close,
		Volume: 100,
	}
}

func mustMoscow() *time.Location {
	msk, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		panic(err)
	}
	return msk
}

func TestPrevDayRangeBreakoutV1BreakoutUp(t *testing.T) {
	msk := mustMoscow()
	prevDay := time.Date(2026, 7, 8, 0, 0, 0, 0, msk)
	currDay := time.Date(2026, 7, 9, 0, 0, 0, 0, msk)

	candles := []domain.Candle{
		moscowHourCandle(prevDay, 10, 100, 110, 95, 105),
		moscowHourCandle(prevDay, 14, 105, 108, 90, 100),
		moscowHourCandle(currDay, 10, 109, 111, 108, 111), // close > prev high 110
	}

	out, err := NewPrevDayRangeBreakoutV1().Run(RunInput{
		StrategySlug: "prev-day-range-breakout-v1",
		Symbol:       "BTC",
		Timeframe:    "1h",
		Candles:      candles,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Signals) != 1 {
		t.Fatalf("expected 1 signal, got %d: %#v", len(out.Signals), out.Signals)
	}
	sig := out.Signals[0]
	if sig.Type != "alert" || sig.Side != "long" || sig.Title != "Prev-day high breakout" {
		t.Fatalf("unexpected signal: %#v", sig)
	}
	if got := sig.Meta["day_high"]; got != 110.0 {
		t.Fatalf("expected day_high 110, got %#v", got)
	}
	if got := sig.Meta["day_low"]; got != 90.0 {
		t.Fatalf("expected day_low 90, got %#v", got)
	}
	if _, ok := sig.Meta["prev_pdh_high"]; ok {
		t.Fatalf("expected no prev_pdh_high without T-2 history, got %#v", sig.Meta)
	}
	if len(out.Trades) != 0 {
		t.Fatalf("expected no trades, got %#v", out.Trades)
	}
}

func TestPrevDayRangeBreakoutV1BreakoutDown(t *testing.T) {
	msk := mustMoscow()
	prevDay := time.Date(2026, 7, 8, 0, 0, 0, 0, msk)
	currDay := time.Date(2026, 7, 9, 0, 0, 0, 0, msk)

	candles := []domain.Candle{
		moscowHourCandle(prevDay, 10, 100, 110, 95, 105),
		moscowHourCandle(prevDay, 14, 105, 108, 90, 100),
		moscowHourCandle(currDay, 11, 85, 93, 84, 89), // green candle, close < prev low 90
	}

	out, err := NewPrevDayRangeBreakoutV1().Run(RunInput{
		StrategySlug: "prev-day-range-breakout-v1",
		Symbol:       "BTC",
		Timeframe:    "1h",
		Candles:      candles,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Signals) != 1 {
		t.Fatalf("expected 1 signal, got %d: %#v", len(out.Signals), out.Signals)
	}
	sig := out.Signals[0]
	if sig.Type != "alert" || sig.Side != "short" || sig.Title != "Prev-day low breakout" {
		t.Fatalf("unexpected signal: %#v", sig)
	}
}

func TestPrevDayRangeBreakoutV1SkipsRedCandleBelowLow(t *testing.T) {
	msk := mustMoscow()
	prevDay := time.Date(2026, 7, 8, 0, 0, 0, 0, msk)
	currDay := time.Date(2026, 7, 9, 0, 0, 0, 0, msk)

	candles := []domain.Candle{
		moscowHourCandle(prevDay, 10, 100, 110, 95, 105),
		moscowHourCandle(prevDay, 14, 105, 108, 90, 100),
		moscowHourCandle(currDay, 11, 92, 93, 88, 89), // red candle below prev low — skip
	}

	out, err := NewPrevDayRangeBreakoutV1().Run(RunInput{
		StrategySlug: "prev-day-range-breakout-v1",
		Symbol:       "BTC",
		Timeframe:    "1h",
		Candles:      candles,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Signals) != 0 {
		t.Fatalf("expected no signal for red candle below low, got %#v", out.Signals)
	}
}

func TestPrevDayRangeBreakoutV1NoSignalInsideRange(t *testing.T) {
	msk := mustMoscow()
	prevDay := time.Date(2026, 7, 8, 0, 0, 0, 0, msk)
	currDay := time.Date(2026, 7, 9, 0, 0, 0, 0, msk)

	candles := []domain.Candle{
		moscowHourCandle(prevDay, 10, 100, 110, 90, 105),
		moscowHourCandle(currDay, 10, 100, 105, 95, 100), // inside [90, 110]
		moscowHourCandle(currDay, 11, 100, 110, 90, 110), // equal high — not breakout
		moscowHourCandle(currDay, 12, 90, 95, 90, 90),    // equal low — not breakout
	}

	out, err := NewPrevDayRangeBreakoutV1().Run(RunInput{
		StrategySlug: "prev-day-range-breakout-v1",
		Symbol:       "BTC",
		Timeframe:    "1h",
		Candles:      candles,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Signals) != 0 {
		t.Fatalf("expected no signals inside range, got %#v", out.Signals)
	}
}

func TestPrevDayRangeBreakoutV1InsufficientHistory(t *testing.T) {
	msk := mustMoscow()
	day := time.Date(2026, 7, 9, 0, 0, 0, 0, msk)
	candles := []domain.Candle{
		moscowHourCandle(day, 10, 100, 110, 90, 105),
		moscowHourCandle(day, 11, 105, 112, 100, 111),
	}

	out, err := NewPrevDayRangeBreakoutV1().Run(RunInput{
		StrategySlug: "prev-day-range-breakout-v1",
		Symbol:       "BTC",
		Timeframe:    "1h",
		Candles:      candles,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Signals) != 0 || len(out.Trades) != 0 {
		t.Fatalf("expected empty output without previous day, got %#v", out)
	}
}

func TestPrevDayRangeBreakoutV1IncludesPrevPDHLevels(t *testing.T) {
	msk := mustMoscow()
	dayBeforePrev := time.Date(2026, 7, 7, 0, 0, 0, 0, msk)
	prevDay := time.Date(2026, 7, 8, 0, 0, 0, 0, msk)
	currDay := time.Date(2026, 7, 9, 0, 0, 0, 0, msk)

	candles := []domain.Candle{
		moscowHourCandle(dayBeforePrev, 10, 80, 120, 70, 90),
		moscowHourCandle(prevDay, 10, 100, 110, 95, 105),
		moscowHourCandle(prevDay, 14, 105, 108, 90, 100),
		moscowHourCandle(currDay, 10, 109, 111, 108, 111),
	}

	out, err := NewPrevDayRangeBreakoutV1().Run(RunInput{
		StrategySlug: "prev-day-range-breakout-v1",
		Symbol:       "BTC",
		Timeframe:    "1h",
		Candles:      candles,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Signals) != 1 {
		t.Fatalf("expected 1 signal, got %d: %#v", len(out.Signals), out.Signals)
	}
	sig := out.Signals[0]
	// PDH/PDL for 2026-07-09 come from 2026-07-08.
	if got := sig.Meta["day_high"]; got != 110.0 {
		t.Fatalf("expected day_high 110, got %#v", got)
	}
	if got := sig.Meta["day_low"]; got != 90.0 {
		t.Fatalf("expected day_low 90, got %#v", got)
	}
	// Prev PHD* are previous PDH/PDL values (from 2026-07-07), not "calendar-2" framing.
	if got := sig.Meta["prev_pdh_high"]; got != 120.0 {
		t.Fatalf("expected prev_pdh_high 120, got %#v", got)
	}
	if got := sig.Meta["prev_pdh_low"]; got != 70.0 {
		t.Fatalf("expected prev_pdh_low 70, got %#v", got)
	}
}

func TestPrevDayRangeBreakoutV1SkipsUnsupportedTimeframe(t *testing.T) {
	out, err := NewPrevDayRangeBreakoutV1().Run(RunInput{
		StrategySlug: "prev-day-range-breakout-v1",
		Symbol:       "BTC",
		Timeframe:    "5m",
		Candles:      []domain.Candle{{Close: 1, High: 1, Low: 1}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Signals) != 0 {
		t.Fatalf("expected no signals for unsupported timeframe, got %#v", out.Signals)
	}
}
