package strategy

import "time"

type Signal struct {
	ID         int64
	DedupeKey  string
	Time       time.Time
	Type       string
	Side       string
	Price      float64
	Confidence *float64
	Status     string
	Title      string
	Details    string
	Meta       map[string]any
}
