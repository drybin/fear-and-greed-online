package marketdata

import "time"

type Timeframe struct {
	ID          int64
	Code        string
	DurationSec int
	IsActive    bool
}

func (t Timeframe) Duration() time.Duration {
	return time.Duration(t.DurationSec) * time.Second
}
