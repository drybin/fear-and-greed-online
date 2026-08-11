package marketdata

import "time"

type IngestionJob struct {
	ID            int64
	SymbolID      int64
	TimeframeID   int64
	Status        string
	RequestedFrom time.Time
	RequestedTo   time.Time
	LoadedFrom    *time.Time
	LoadedTo      *time.Time
	CandlesLoaded int
	ErrorMessage  string
	StartedAt     time.Time
	FinishedAt    *time.Time
}
