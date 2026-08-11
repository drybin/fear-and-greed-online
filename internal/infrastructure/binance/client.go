package binance

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/drybin/fear-and-greed-online/internal/domain/marketdata"
)

const maxKlinesPerRequest = 1000

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) FetchCandles(ctx context.Context, symbol marketdata.Symbol, timeframe marketdata.Timeframe, from, to time.Time) ([]marketdata.Candle, error) {
	if !from.Before(to) {
		return nil, nil
	}

	var all []marketdata.Candle
	cursor := from.UTC()

	for cursor.Before(to) {
		chunk, err := c.fetchChunk(ctx, symbol, timeframe, cursor, to.UTC())
		if err != nil {
			return nil, err
		}
		if len(chunk) == 0 {
			break
		}

		all = append(all, chunk...)
		last := chunk[len(chunk)-1]
		next := last.OpenTime.Add(timeframe.Duration())
		if !next.After(cursor) {
			break
		}
		cursor = next
	}

	return all, nil
}

func (c *Client) fetchChunk(ctx context.Context, symbol marketdata.Symbol, timeframe marketdata.Timeframe, from, to time.Time) ([]marketdata.Candle, error) {
	endpoint, err := url.Parse(c.baseURL + "/api/v3/klines")
	if err != nil {
		return nil, fmt.Errorf("parse klines endpoint: %w", err)
	}

	query := endpoint.Query()
	query.Set("symbol", symbol.BinanceSymbol)
	query.Set("interval", timeframe.Code)
	query.Set("startTime", strconv.FormatInt(from.UnixMilli(), 10))
	query.Set("endTime", strconv.FormatInt(to.UnixMilli(), 10))
	query.Set("limit", strconv.Itoa(maxKlinesPerRequest))
	endpoint.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create binance request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send binance request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("binance klines status: %s", resp.Status)
	}

	var raw [][]any
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("decode klines payload: %w", err)
	}

	now := time.Now().UTC()
	candles := make([]marketdata.Candle, 0, len(raw))
	for _, row := range raw {
		if len(row) < 11 {
			return nil, fmt.Errorf("unexpected kline row length: %d", len(row))
		}

		openMS, err := anyInt64(row[0])
		if err != nil {
			return nil, fmt.Errorf("parse open time: %w", err)
		}
		closeMS, err := anyInt64(row[6])
		if err != nil {
			return nil, fmt.Errorf("parse close time: %w", err)
		}
		trades, err := anyInt64(row[8])
		if err != nil {
			return nil, fmt.Errorf("parse trades: %w", err)
		}

		openTime := time.UnixMilli(openMS).UTC()
		closeTime := time.UnixMilli(closeMS).UTC()
		candles = append(candles, marketdata.Candle{
			OpenTime:            openTime,
			CloseTime:           closeTime,
			Open:                fmt.Sprint(row[1]),
			High:                fmt.Sprint(row[2]),
			Low:                 fmt.Sprint(row[3]),
			Close:               fmt.Sprint(row[4]),
			Volume:              fmt.Sprint(row[5]),
			QuoteVolume:         fmt.Sprint(row[7]),
			Trades:              trades,
			TakerBuyBaseVolume:  fmt.Sprint(row[9]),
			TakerBuyQuoteVolume: fmt.Sprint(row[10]),
			IsClosed:            now.After(closeTime),
			Source:              "binance_spot",
		})
	}

	return candles, nil
}

func anyInt64(value any) (int64, error) {
	switch v := value.(type) {
	case float64:
		return int64(v), nil
	case int64:
		return v, nil
	case string:
		return strconv.ParseInt(v, 10, 64)
	default:
		return 0, fmt.Errorf("unsupported numeric type %T", value)
	}
}
