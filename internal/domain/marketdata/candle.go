package marketdata

import "time"

type Candle struct {
	SymbolID            int64
	TimeframeID         int64
	OpenTime            time.Time
	CloseTime           time.Time
	Open                string
	High                string
	Low                 string
	Close               string
	Volume              string
	QuoteVolume         string
	Trades              int64
	TakerBuyBaseVolume  string
	TakerBuyQuoteVolume string
	IsClosed            bool
	Source              string
}
