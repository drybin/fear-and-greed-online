package acceptance

const (
	ExpectedTotalSymbols  = 50
	ExpectedActiveSymbols = 38
	ExpectedTimeframes    = 3
	ExpectedStrategies    = 3
)

var InactiveAssetCodes = []string{
	"HYPE",
	"LEO",
	"XMR",
	"CC",
	"CRO",
	"OKB",
	"M",
	"MNT",
	"PI",
	"BGB",
	"KCS",
	"KAS",
}

var StrategySlugs = []string{
	"trend-long-v1",
	"breakout-retest-v1",
	"prev-day-range-breakout-v1",
}
