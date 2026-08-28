package telegram

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/net/proxy"
)

func newSOCKS5Transport(proxyURL string) (*http.Transport, error) {
	u, err := url.Parse(strings.TrimSpace(proxyURL))
	if err != nil {
		return nil, fmt.Errorf("invalid TG_SOCKS5_PROXY: %w", err)
	}
	if u.Scheme != "socks5" && u.Scheme != "socks5h" {
		return nil, fmt.Errorf(`TG_SOCKS5_PROXY must use socks5:// or socks5h:// (got %q)`, u.Scheme)
	}
	host := u.Hostname()
	if host == "" {
		return nil, fmt.Errorf("TG_SOCKS5_PROXY: host is empty")
	}
	port := u.Port()
	if port == "" {
		port = "1080"
	}
	addr := net.JoinHostPort(host, port)

	var auth *proxy.Auth
	if u.User != nil {
		pw, _ := u.User.Password()
		auth = &proxy.Auth{
			User:     u.User.Username(),
			Password: pw,
		}
	}

	dialer, err := proxy.SOCKS5("tcp", addr, auth, proxy.Direct)
	if err != nil {
		return nil, fmt.Errorf("TG_SOCKS5_PROXY: %w", err)
	}

	return &http.Transport{
		Proxy: func(*http.Request) (*url.URL, error) { return nil, nil },
		DialContext: func(_ context.Context, network, address string) (net.Conn, error) {
			return dialer.Dial(network, address)
		},
	}, nil
}
