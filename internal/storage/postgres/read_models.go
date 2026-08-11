package postgres

import "time"

type SignalRecord struct {
	ID          int64
	StrategyID  int64
	SymbolID    int64
	TimeframeID int64
	DedupeKey   string
	SignalTime  time.Time
	SignalType  string
	Side        string
	Price       float64
	Confidence  *float64
	Status      string
	Title       string
	Details     string
	Meta        map[string]any
}

type TradeRecord struct {
	ID          int64
	StrategyID  int64
	SymbolID    int64
	TimeframeID int64
	DedupeKey   string
	EntryTime   time.Time
	ExitTime    *time.Time
	Side        string
	EntryPrice  float64
	ExitPrice   *float64
	PnLAbs      *float64
	PnLPct      *float64
	Status      string
	Meta        map[string]any
}

type StrategyRunRecord struct {
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

type IngestionJobSummary struct {
	FinishedAt    time.Time
	LoadedTo      *time.Time
	CandlesLoaded int
}
