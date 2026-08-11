package strategy

type Definition struct {
	ID                  int64
	Code                string
	Version             string
	Slug                string
	Name                string
	Description         string
	Category            string
	DefaultParams       map[string]any
	SupportedTimeframes []string
	RequiredHistoryBars int
	IsActive            bool
}
