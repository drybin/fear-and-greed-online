package strategy

import "time"

type Run struct {
	ID           int64
	StrategyID   int64
	SymbolID     int64
	TimeframeID  int64
	Status       string
	StartedAt    time.Time
	FinishedAt   *time.Time
	Params       map[string]any
	InputFrom    *time.Time
	InputTo      *time.Time
	CandlesCount int
	SignalsCount int
	TradesCount  int
	ErrorMessage string
}
