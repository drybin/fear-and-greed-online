package strategy

import (
	"fmt"
	"time"

	domain "github.com/drybin/fear-and-greed-online/internal/domain/strategy"
)

type TrendLongV1 struct{}

func NewTrendLongV1() *TrendLongV1 {
	return &TrendLongV1{}
}

func (s *TrendLongV1) Slug() string {
	return "trend-long-v1"
}

func (s *TrendLongV1) Run(input RunInput) (RunOutput, error) {
	period := intParam(input.Params, "sma_period", 50)
	if period < 2 {
		return RunOutput{}, fmt.Errorf("sma_period must be at least 2")
	}
	if len(input.Candles) < period+2 {
		return RunOutput{}, nil
	}

	closes := make([]float64, len(input.Candles))
	for i, candle := range input.Candles {
		closes[i] = candle.Close
	}
	sma := SMA(closes, period)

	var out RunOutput
	inTrade := false
	entryIndex := -1
	entryPrice := 0.0
	side := "long"

	for i := period; i < len(input.Candles); i++ {
		curr := input.Candles[i]
		prev := input.Candles[i-1]
		if sma[i] == 0 || sma[i-1] == 0 {
			continue
		}

		if !inTrade && prev.Close <= sma[i-1] && curr.Close > sma[i] {
			key := fmt.Sprintf("%s|%s|%s|entry|%s", input.StrategySlug, input.Symbol, input.Timeframe, curr.Time.UTC().Format(time.RFC3339))
			out.Signals = append(out.Signals, domain.Signal{
				DedupeKey: key,
				Time:      curr.Time,
				Type:      "entry",
				Side:      side,
				Price:     curr.Close,
				Status:    "confirmed",
				Title:     "Trend entry",
				Details:   "Close crossed above SMA",
				Meta: map[string]any{
					"sma":        sma[i],
					"sma_period": period,
				},
			})
			inTrade = true
			entryIndex = i
			entryPrice = curr.Close
			continue
		}

		if inTrade && curr.Close < sma[i] {
			pnlAbs := curr.Close - entryPrice
			pnlPct := 0.0
			if entryPrice > 0 {
				pnlPct = pnlAbs / entryPrice * 100
			}
			key := fmt.Sprintf("%s|%s|%s|exit|%s", input.StrategySlug, input.Symbol, input.Timeframe, curr.Time.UTC().Format(time.RFC3339))
			out.Signals = append(out.Signals, domain.Signal{
				DedupeKey: key,
				Time:      curr.Time,
				Type:      "exit",
				Side:      side,
				Price:     curr.Close,
				Status:    "confirmed",
				Title:     "Trend exit",
				Details:   "Close fell below SMA",
				Meta: map[string]any{
					"sma":        sma[i],
					"sma_period": period,
				},
			})
			exitTime := curr.Time
			exitPrice := curr.Close
			out.Trades = append(out.Trades, domain.Trade{
				DedupeKey:  fmt.Sprintf("%s|%s|%s|trade|%s", input.StrategySlug, input.Symbol, input.Timeframe, input.Candles[entryIndex].Time.UTC().Format(time.RFC3339)),
				EntryTime:  input.Candles[entryIndex].Time,
				ExitTime:   &exitTime,
				Side:       side,
				EntryPrice: entryPrice,
				ExitPrice:  &exitPrice,
				PnLAbs:     &pnlAbs,
				PnLPct:     &pnlPct,
				Status:     "closed",
				Meta: map[string]any{
					"sma_period": period,
				},
			})
			inTrade = false
			entryIndex = -1
			entryPrice = 0
		}
	}

	if inTrade && entryIndex >= 0 {
		out.Trades = append(out.Trades, domain.Trade{
			DedupeKey:  fmt.Sprintf("%s|%s|%s|trade|%s", input.StrategySlug, input.Symbol, input.Timeframe, input.Candles[entryIndex].Time.UTC().Format(time.RFC3339)),
			EntryTime:  input.Candles[entryIndex].Time,
			Side:       side,
			EntryPrice: entryPrice,
			Status:     "open",
			Meta: map[string]any{
				"sma_period": period,
			},
		})
	}

	return out, nil
}
