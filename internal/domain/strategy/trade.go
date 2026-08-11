package strategy

import "time"

type Trade struct {
	ID         int64
	DedupeKey  string
	EntryTime  time.Time
	ExitTime   *time.Time
	Side       string
	EntryPrice float64
	ExitPrice  *float64
	PnLAbs     *float64
	PnLPct     *float64
	Status     string
	Meta       map[string]any
}
