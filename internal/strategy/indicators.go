package strategy

import (
	domain "github.com/drybin/fear-and-greed-online/internal/domain/strategy"
)

func SMA(values []float64, period int) []float64 {
	out := make([]float64, len(values))
	if period <= 0 {
		return out
	}

	sum := 0.0
	for i, v := range values {
		sum += v
		if i >= period {
			sum -= values[i-period]
		}
		if i >= period-1 {
			out[i] = sum / float64(period)
		}
	}
	return out
}

func highestHigh(candles []domain.Candle, from, to int) float64 {
	if from < 0 {
		from = 0
	}
	if to > len(candles) {
		to = len(candles)
	}
	high := 0.0
	for i := from; i < to; i++ {
		if i == from || candles[i].High > high {
			high = candles[i].High
		}
	}
	return high
}

func intParam(params map[string]any, key string, fallback int) int {
	if params == nil {
		return fallback
	}
	value, ok := params[key]
	if !ok {
		return fallback
	}
	switch v := value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	default:
		return fallback
	}
}

func floatParam(params map[string]any, key string, fallback float64) float64 {
	if params == nil {
		return fallback
	}
	value, ok := params[key]
	if !ok {
		return fallback
	}
	switch v := value.(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case int64:
		return float64(v)
	default:
		return fallback
	}
}
