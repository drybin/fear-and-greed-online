package strategy

import "testing"

func TestRegistryReturnsRegisteredStrategy(t *testing.T) {
	registry := NewRegistry(NewTrendLongV1(), NewBreakoutRetestV1(), NewPrevDayRangeBreakoutV1())

	strategy, err := registry.Get("trend-long-v1")
	if err != nil {
		t.Fatalf("get trend-long-v1: %v", err)
	}
	if strategy.Slug() != "trend-long-v1" {
		t.Fatalf("unexpected strategy slug: %s", strategy.Slug())
	}
}

func TestRegistryReturnsErrorForUnknownStrategy(t *testing.T) {
	registry := NewRegistry(NewTrendLongV1())

	_, err := registry.Get("missing-strategy")
	if err == nil {
		t.Fatal("expected unknown strategy error")
	}
}
