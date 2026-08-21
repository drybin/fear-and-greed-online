package strategy

import (
	"fmt"
	"sort"
	"time"

	domain "github.com/drybin/fear-and-greed-online/internal/domain/strategy"
)

type PrevDayRangeBreakoutV1 struct{}

func NewPrevDayRangeBreakoutV1() *PrevDayRangeBreakoutV1 {
	return &PrevDayRangeBreakoutV1{}
}

func (s *PrevDayRangeBreakoutV1) Slug() string {
	return "prev-day-range-breakout-v1"
}

type dayRange struct {
	high float64
	low  float64
}

func (s *PrevDayRangeBreakoutV1) Run(input RunInput) (RunOutput, error) {
	switch input.Timeframe {
	case "15m", "1h", "4h":
	default:
		return RunOutput{}, nil
	}
	if len(input.Candles) == 0 {
		return RunOutput{}, nil
	}

	msk, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		return RunOutput{}, fmt.Errorf("load Europe/Moscow: %w", err)
	}

	rangesByDay := buildMoscowDayRanges(input.Candles, msk)
	if len(rangesByDay) < 2 {
		return RunOutput{}, nil
	}
	dayKeys := sortedDayKeys(rangesByDay)

	var out RunOutput
	for _, candle := range input.Candles {
		local := candle.Time.In(msk)
		prevKey := moscowDayKey(local.AddDate(0, 0, -1))
		prev, ok := rangesByDay[prevKey]
		if !ok {
			continue
		}

		meta := map[string]any{
			"prev_day": prevKey,
			"day_high": prev.high,
			"day_low":  prev.low,
			"open":     candle.Open,
			"high":     candle.High,
			"low":      candle.Low,
			"close":    candle.Close,
			// Prev PHD High: previous PDH if it is still at/above current PDH;
			// otherwise the signal candle high (fresh breakout above prior highs).
			"prev_pdh_high": resolvePrevPDHHigh(dayKeys, rangesByDay, prevKey, prev.high, candle.High),
		}
		if prevPDL, ok := previousSessionLow(dayKeys, rangesByDay, prevKey); ok {
			meta["prev_pdh_low"] = prevPDL
		}

		if candle.Close > prev.high {
			out.Signals = append(out.Signals, domain.Signal{
				DedupeKey: fmt.Sprintf("%s|%s|%s|alert|up|%s", input.StrategySlug, input.Symbol, input.Timeframe, candle.Time.UTC().Format(time.RFC3339)),
				Time:      candle.Time,
				Type:      "alert",
				Side:      "long",
				Price:     candle.Close,
				Status:    "confirmed",
				Title:     "Prev-day high breakout",
				Details:   "Close closed above previous Moscow day high",
				Meta:      meta,
			})
			continue
		}

		if candle.Close < prev.low && candle.Close > candle.Open {
			out.Signals = append(out.Signals, domain.Signal{
				DedupeKey: fmt.Sprintf("%s|%s|%s|alert|down|%s", input.StrategySlug, input.Symbol, input.Timeframe, candle.Time.UTC().Format(time.RFC3339)),
				Time:      candle.Time,
				Type:      "alert",
				Side:      "short",
				Price:     candle.Close,
				Status:    "confirmed",
				Title:     "Prev-day low breakout",
				Details:   "Green candle closed below previous Moscow day low",
				Meta:      meta,
			})
		}
	}

	return out, nil
}

func buildMoscowDayRanges(candles []domain.Candle, msk *time.Location) map[string]dayRange {
	rangesByDay := make(map[string]dayRange)
	for _, candle := range candles {
		key := moscowDayKey(candle.Time.In(msk))
		existing, ok := rangesByDay[key]
		if !ok {
			rangesByDay[key] = dayRange{high: candle.High, low: candle.Low}
			continue
		}
		if candle.High > existing.high {
			existing.high = candle.High
		}
		if candle.Low < existing.low {
			existing.low = candle.Low
		}
		rangesByDay[key] = existing
	}
	return rangesByDay
}

func sortedDayKeys(rangesByDay map[string]dayRange) []string {
	keys := make([]string, 0, len(rangesByDay))
	for key := range rangesByDay {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func dayIndex(dayKeys []string, key string) int {
	for i, item := range dayKeys {
		if item == key {
			return i
		}
	}
	return -1
}

func previousSessionLow(dayKeys []string, rangesByDay map[string]dayRange, prevKey string) (float64, bool) {
	idx := dayIndex(dayKeys, prevKey)
	if idx <= 0 {
		return 0, false
	}
	return rangesByDay[dayKeys[idx-1]].low, true
}

func resolvePrevPDHHigh(dayKeys []string, rangesByDay map[string]dayRange, prevKey string, pdh, signalHigh float64) float64 {
	idx := dayIndex(dayKeys, prevKey)
	if idx > 0 {
		// Walk back for the nearest prior session high that is still >= current PDH.
		for i := idx - 1; i >= 0; i-- {
			high := rangesByDay[dayKeys[i]].high
			if high >= pdh {
				return high
			}
		}
	}
	return signalHigh
}

func moscowDayKey(t time.Time) string {
	return t.Format("2006-01-02")
}
