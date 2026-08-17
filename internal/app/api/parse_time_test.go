package api

import (
	"testing"
	"time"
)

func TestParseTimeAcceptsRFC3339NanoUTC(t *testing.T) {
	got, err := parseTime("2026-08-17T09:10:00.000Z", time.Time{})
	if err != nil {
		t.Fatalf("parseTime: %v", err)
	}

	want := time.Date(2026, 8, 17, 9, 10, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("parseTime = %s, want %s", got.UTC().Format(time.RFC3339Nano), want.Format(time.RFC3339Nano))
	}
}

func TestParseTimeEmptyUsesFallbackUTC(t *testing.T) {
	fallback := time.Date(2026, 8, 17, 8, 0, 0, 0, time.FixedZone("MSK", 3*3600))
	got, err := parseTime("", fallback)
	if err != nil {
		t.Fatalf("parseTime: %v", err)
	}
	if got.Location() != time.UTC {
		t.Fatalf("expected UTC location, got %s", got.Location())
	}
	if !got.Equal(fallback.UTC()) {
		t.Fatalf("parseTime = %s, want %s", got, fallback.UTC())
	}
}
