package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const maxSendAttempts = 4

var retryAfterPattern = regexp.MustCompile(`retry after (\d+)`)

type Client struct {
	botToken   string
	httpClient *http.Client
}

func NewClient(botToken, socks5Proxy string) (*Client, error) {
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}
	if strings.TrimSpace(socks5Proxy) != "" {
		transport, err := newSOCKS5Transport(socks5Proxy)
		if err != nil {
			return nil, err
		}
		httpClient.Transport = transport
	}
	return &Client{
		botToken:   botToken,
		httpClient: httpClient,
	}, nil
}

type sendMessageRequest struct {
	ChatID                string `json:"chat_id"`
	Text                  string `json:"text"`
	DisableWebPagePreview bool   `json:"disable_web_page_preview"`
}

type apiResponse struct {
	OK          bool   `json:"ok"`
	Description string `json:"description"`
	Parameters  *struct {
		RetryAfter int `json:"retry_after"`
	} `json:"parameters"`
}

func (c *Client) SendMessage(ctx context.Context, chatID, text string) error {
	body, err := json.Marshal(sendMessageRequest{
		ChatID:                chatID,
		Text:                  text,
		DisableWebPagePreview: false,
	})
	if err != nil {
		return fmt.Errorf("marshal telegram payload: %w", err)
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", c.botToken)
	var lastErr error
	for attempt := 1; attempt <= maxSendAttempts; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
		if err != nil {
			return fmt.Errorf("create telegram request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("send telegram request: %w", err)
		}

		parsed, readErr := decodeAPIResponse(resp.Body)
		_ = resp.Body.Close()
		if readErr != nil {
			return readErr
		}

		if resp.StatusCode == http.StatusOK && parsed.OK {
			return nil
		}

		lastErr = fmt.Errorf("telegram send failed: status=%s description=%s", resp.Status, parsed.Description)
		if resp.StatusCode != http.StatusTooManyRequests || attempt == maxSendAttempts {
			return lastErr
		}

		wait := retryAfter(parsed, resp.Header.Get("Retry-After"))
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
	return lastErr
}

func decodeAPIResponse(body io.Reader) (apiResponse, error) {
	var parsed apiResponse
	if err := json.NewDecoder(body).Decode(&parsed); err != nil {
		return apiResponse{}, fmt.Errorf("decode telegram response: %w", err)
	}
	return parsed, nil
}

func retryAfter(parsed apiResponse, header string) time.Duration {
	if parsed.Parameters != nil && parsed.Parameters.RetryAfter > 0 {
		return time.Duration(parsed.Parameters.RetryAfter) * time.Second
	}
	if header != "" {
		if seconds, err := strconv.Atoi(strings.TrimSpace(header)); err == nil && seconds > 0 {
			return time.Duration(seconds) * time.Second
		}
	}
	if match := retryAfterPattern.FindStringSubmatch(parsed.Description); len(match) == 2 {
		if seconds, err := strconv.Atoi(match[1]); err == nil && seconds > 0 {
			return time.Duration(seconds) * time.Second
		}
	}
	return time.Second
}
