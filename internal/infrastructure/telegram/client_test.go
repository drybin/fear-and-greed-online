package telegram

import (
	"testing"
	"time"
)

func TestRetryAfterFromParameters(t *testing.T) {
	got := retryAfter(apiResponse{
		Parameters: &struct {
			RetryAfter int `json:"retry_after"`
		}{RetryAfter: 38},
	}, "")
	if got != 38*time.Second {
		t.Fatalf("expected 38s, got %v", got)
	}
}

func TestRetryAfterFromDescription(t *testing.T) {
	got := retryAfter(apiResponse{
		Description: "Too Many Requests: retry after 12",
	}, "")
	if got != 12*time.Second {
		t.Fatalf("expected 12s, got %v", got)
	}
}

func TestRetryAfterFallback(t *testing.T) {
	got := retryAfter(apiResponse{}, "")
	if got != time.Second {
		t.Fatalf("expected 1s fallback, got %v", got)
	}
}
