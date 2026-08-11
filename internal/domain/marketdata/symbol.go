package marketdata

type Symbol struct {
	ID            int64
	AssetCode     string
	Name          string
	MarketRank    int
	QuoteAsset    string
	BinanceSymbol string
	IsActive      bool
}
