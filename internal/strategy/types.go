package strategy

import (
	domain "github.com/drybin/fear-and-greed-online/internal/domain/strategy"
)

type RunInput struct {
	StrategySlug string
	Symbol       string
	Timeframe    string
	Candles      []domain.Candle
	Params       map[string]any
}

type RunOutput struct {
	Signals []domain.Signal
	Trades  []domain.Trade
}

type Strategy interface {
	Slug() string
	Run(input RunInput) (RunOutput, error)
}
