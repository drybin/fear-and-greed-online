package telegram

import (
	"testing"
)

func TestNewSOCKS5Transport_BadScheme(t *testing.T) {
	_, err := newSOCKS5Transport("http://127.0.0.1:1080")
	if err == nil {
		t.Fatal("expected error for http scheme")
	}
}

func TestNewSOCKS5Transport_EmptyHost(t *testing.T) {
	_, err := newSOCKS5Transport("socks5://")
	if err == nil {
		t.Fatal("expected error for empty host")
	}
}

func TestNewSOCKS5Transport_OK(t *testing.T) {
	transport, err := newSOCKS5Transport("socks5://127.0.0.1:9")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if transport == nil {
		t.Fatal("expected transport")
	}
}

func TestNewSOCKS5Transport_SOCKS5h(t *testing.T) {
	transport, err := newSOCKS5Transport("socks5h://127.0.0.1:9")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if transport == nil {
		t.Fatal("expected transport")
	}
}
