package acceptance

const (
	ExpectedTotalSymbols  = 64
	ExpectedActiveSymbols = 53
	ExpectedTimeframes    = 3
	ExpectedStrategies    = 3
)

var InactiveAssetCodes = []string{
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
