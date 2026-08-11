package strategy

import (
	"fmt"
	"time"

	domain "github.com/drybin/fear-and-greed-online/internal/domain/strategy"
)

type BreakoutRetestV1 struct{}

func NewBreakoutRetestV1() *BreakoutRetestV1 {
	return &BreakoutRetestV1{}
}

func (s *BreakoutRetestV1) Slug() string {
	return "breakout-retest-v1"
}

func (s *BreakoutRetestV1) Run(input RunInput) (RunOutput, error) {
	lookback := intParam(input.Params, "lookback_bars", 20)
	retestBars := intParam(input.Params, "retest_bars", 6)
	riskReward := floatParam(input.Params, "risk_reward", 2.0)

	if input.Timeframe != "15m" {
		return RunOutput{}, nil
	}
	if lookback < 2 || retestBars < 1 || riskReward <= 0 {
		return RunOutput{}, fmt.Errorf("invalid breakout-retest params")
	}
	if len(input.Candles) < lookback+retestBars+2 {
		return RunOutput{}, nil
	}

	var out RunOutput
	inTrade := false
	breakoutLevel := 0.0
	stopLevel := 0.0
	entryPrice := 0.0
	entryIndex := -1

	for i := lookback; i < len(input.Candles); i++ {
		curr := input.Candles[i]

		if !inTrade {
			level := highestHigh(input.Candles, i-lookback, i)
			if curr.Close <= level {
				continue
			}
			for j := i + 1; j < len(input.Candles) && j <= i+retestBars; j++ {
				next := input.Candles[j]
				if next.Low <= level && next.Close > level {
					breakoutLevel = level
					stopLevel = next.Low
					entryPrice = next.Close
					entryIndex = j
					inTrade = true
					out.Signals = append(out.Signals, domain.Signal{
						DedupeKey: fmt.Sprintf("%s|%s|%s|entry|%s", input.StrategySlug, input.Symbol, input.Timeframe, next.Time.UTC().Format(time.RFC3339)),
						Time:      next.Time,
						Type:      "entry",
						Side:      "long",
						Price:     entryPrice,
						Status:    "confirmed",
						Title:     "Breakout retest entry",
						Details:   "Retest held above breakout level",
						Meta: map[string]any{
							"breakout_level": breakoutLevel,
							"stop_level":     stopLevel,
							"lookback_bars":  lookback,
							"retest_bars":    retestBars,
						},
					})
					i = j
					break
				}
			}
			continue
		}

		risk := entryPrice - stopLevel
		if risk <= 0 {
			inTrade = false
			continue
		}
		target := entryPrice + risk*riskReward

		if curr.Low <= stopLevel {
			exitTime := curr.Time
			exitPrice := stopLevel
			pnlAbs := exitPrice - entryPrice
			pnlPct := pnlAbs / entryPrice * 100
			out.Signals = append(out.Signals, domain.Signal{
				DedupeKey: fmt.Sprintf("%s|%s|%s|exit|%s", input.StrategySlug, input.Symbol, input.Timeframe, curr.Time.UTC().Format(time.RFC3339)),
				Time:      curr.Time,
				Type:      "exit",
				Side:      "long",
				Price:     exitPrice,
				Status:    "confirmed",
				Title:     "Breakout retest stop",
				Details:   "Price hit stop level",
				Meta: map[string]any{
					"breakout_level": breakoutLevel,
					"stop_level":     stopLevel,
				},
			})
			out.Trades = append(out.Trades, domain.Trade{
				DedupeKey:  fmt.Sprintf("%s|%s|%s|trade|%s", input.StrategySlug, input.Symbol, input.Timeframe, input.Candles[entryIndex].Time.UTC().Format(time.RFC3339)),
				EntryTime:  input.Candles[entryIndex].Time,
				ExitTime:   &exitTime,
				Side:       "long",
				EntryPrice: entryPrice,
				ExitPrice:  &exitPrice,
				PnLAbs:     &pnlAbs,
				PnLPct:     &pnlPct,
				Status:     "closed",
				Meta: map[string]any{
					"reason":         "stop",
					"breakout_level": breakoutLevel,
					"risk_reward":    riskReward,
				},
			})
			inTrade = false
			continue
		}

		if curr.High >= target {
			exitTime := curr.Time
			exitPrice := target
			pnlAbs := exitPrice - entryPrice
			pnlPct := pnlAbs / entryPrice * 100
			out.Signals = append(out.Signals, domain.Signal{
				DedupeKey: fmt.Sprintf("%s|%s|%s|exit|%s", input.StrategySlug, input.Symbol, input.Timeframe, curr.Time.UTC().Format(time.RFC3339)),
				Time:      curr.Time,
				Type:      "exit",
				Side:      "long",
				Price:     exitPrice,
				Status:    "confirmed",
				Title:     "Breakout retest target",
				Details:   "Price hit profit target",
				Meta: map[string]any{
					"breakout_level": breakoutLevel,
					"target":         target,
				},
			})
			out.Trades = append(out.Trades, domain.Trade{
				DedupeKey:  fmt.Sprintf("%s|%s|%s|trade|%s", input.StrategySlug, input.Symbol, input.Timeframe, input.Candles[entryIndex].Time.UTC().Format(time.RFC3339)),
				EntryTime:  input.Candles[entryIndex].Time,
				ExitTime:   &exitTime,
				Side:       "long",
				EntryPrice: entryPrice,
				ExitPrice:  &exitPrice,
				PnLAbs:     &pnlAbs,
				PnLPct:     &pnlPct,
				Status:     "closed",
				Meta: map[string]any{
					"reason":         "target",
					"breakout_level": breakoutLevel,
					"risk_reward":    riskReward,
				},
			})
			inTrade = false
		}
	}

	if inTrade && entryIndex >= 0 {
		out.Trades = append(out.Trades, domain.Trade{
			DedupeKey:  fmt.Sprintf("%s|%s|%s|trade|%s", input.StrategySlug, input.Symbol, input.Timeframe, input.Candles[entryIndex].Time.UTC().Format(time.RFC3339)),
			EntryTime:  input.Candles[entryIndex].Time,
			Side:       "long",
			EntryPrice: entryPrice,
			Status:     "open",
			Meta: map[string]any{
				"breakout_level": breakoutLevel,
				"stop_level":     stopLevel,
				"risk_reward":    riskReward,
			},
		})
	}

	return out, nil
}
