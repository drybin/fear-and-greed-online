package acceptance

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"

	"github.com/drybin/fear-and-greed-online/internal/storage/postgres"
)

func VerifyFrozenUniverse(ctx context.Context, db *sql.DB) error {
	var totalSymbols int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM symbols`).Scan(&totalSymbols); err != nil {
		return fmt.Errorf("count symbols: %w", err)
	}
	if totalSymbols != ExpectedTotalSymbols {
		return fmt.Errorf("expected %d seeded symbols, got %d", ExpectedTotalSymbols, totalSymbols)
	}

	var activeSymbols int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM symbols WHERE is_active = TRUE`).Scan(&activeSymbols); err != nil {
		return fmt.Errorf("count active symbols: %w", err)
	}
	if activeSymbols != ExpectedActiveSymbols {
		return fmt.Errorf("expected %d active symbols, got %d", ExpectedActiveSymbols, activeSymbols)
	}

	var inactiveSymbols int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM symbols WHERE is_active = FALSE`).Scan(&inactiveSymbols); err != nil {
		return fmt.Errorf("count inactive symbols: %w", err)
	}
	if inactiveSymbols != len(InactiveAssetCodes) {
		return fmt.Errorf("expected %d inactive symbols, got %d", len(InactiveAssetCodes), inactiveSymbols)
	}

	for _, code := range InactiveAssetCodes {
		var isActive bool
		if err := db.QueryRowContext(ctx, `SELECT is_active FROM symbols WHERE asset_code = $1`, code).Scan(&isActive); err != nil {
			return fmt.Errorf("lookup inactive symbol %s: %w", code, err)
		}
		if isActive {
			return fmt.Errorf("expected %s to remain inactive", code)
		}
	}

	rows, err := db.QueryContext(ctx, `
		SELECT asset_code
		FROM symbols
		WHERE is_active = TRUE
		ORDER BY market_cap_rank ASC, asset_code ASC
	`)
	if err != nil {
		return fmt.Errorf("list active symbols: %w", err)
	}
	defer func() { _ = rows.Close() }()

	activeCodes := make([]string, 0, ExpectedActiveSymbols)
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			return fmt.Errorf("scan active symbol: %w", err)
		}
		activeCodes = append(activeCodes, code)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if len(activeCodes) != ExpectedActiveSymbols {
		return fmt.Errorf("expected %d active symbol rows, got %d", ExpectedActiveSymbols, len(activeCodes))
	}
	for _, inactive := range InactiveAssetCodes {
		if slices.Contains(activeCodes, inactive) {
			return fmt.Errorf("inactive symbol %s appeared in active universe", inactive)
		}
	}

	var timeframes int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM timeframes WHERE is_active = TRUE`).Scan(&timeframes); err != nil {
		return fmt.Errorf("count timeframes: %w", err)
	}
	if timeframes != ExpectedTimeframes {
		return fmt.Errorf("expected %d active timeframes, got %d", ExpectedTimeframes, timeframes)
	}

	var strategies int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM strategies WHERE is_active = TRUE`).Scan(&strategies); err != nil {
		return fmt.Errorf("count strategies: %w", err)
	}
	if strategies != ExpectedStrategies {
		return fmt.Errorf("expected %d active strategies, got %d", ExpectedStrategies, strategies)
	}

	for _, slug := range StrategySlugs {
		var found bool
		if err := db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM strategies WHERE slug = $1 AND is_active = TRUE)`, slug).Scan(&found); err != nil {
			return fmt.Errorf("lookup strategy %s: %w", slug, err)
		}
		if !found {
			return fmt.Errorf("expected active strategy %s", slug)
		}
	}

	return nil
}

func VerifyCLIQueryableUniverse(ctx context.Context, symbols *postgres.SymbolRepository) error {
	active, err := symbols.ListActive(ctx, "")
	if err != nil {
		return fmt.Errorf("cli list active symbols: %w", err)
	}
	if len(active) != ExpectedActiveSymbols {
		return fmt.Errorf("cli active symbol count: got %d want %d", len(active), ExpectedActiveSymbols)
	}

	for _, item := range active {
		if !item.IsActive {
			return fmt.Errorf("cli returned inactive symbol %s", item.AssetCode)
		}
		if strings.TrimSpace(item.BinanceSymbol) == "" {
			return fmt.Errorf("cli active symbol %s missing binance mapping", item.AssetCode)
		}
		found, err := symbols.FindByAssetCode(ctx, item.AssetCode)
		if err != nil {
			return fmt.Errorf("cli lookup %s: %w", item.AssetCode, err)
		}
		if found == nil {
			return fmt.Errorf("cli could not resolve active symbol %s", item.AssetCode)
		}
	}

	for _, code := range InactiveAssetCodes {
		found, err := symbols.FindByAssetCode(ctx, code)
		if err != nil {
			return fmt.Errorf("cli lookup inactive %s: %w", code, err)
		}
		if found == nil {
			return fmt.Errorf("inactive symbol %s missing from seeded universe", code)
		}
		if found.IsActive {
			return fmt.Errorf("cli resolved inactive symbol %s as active", code)
		}
	}

	return nil
}

func VerifyAPIUniverse(client *http.Client, baseURL string) error {
	var activePayload struct {
		Items []struct {
			AssetCode string `json:"AssetCode"`
			IsActive  bool   `json:"IsActive"`
		} `json:"items"`
		ActiveCount int `json:"active_count"`
	}
	if err := decodeGET(client, baseURL+"/symbols/active", &activePayload); err != nil {
		return err
	}
	if activePayload.ActiveCount != ExpectedActiveSymbols {
		return fmt.Errorf("/symbols/active count: got %d want %d", activePayload.ActiveCount, ExpectedActiveSymbols)
	}
	if len(activePayload.Items) != ExpectedActiveSymbols {
		return fmt.Errorf("/symbols/active items: got %d want %d", len(activePayload.Items), ExpectedActiveSymbols)
	}

	activeSet := make(map[string]struct{}, len(activePayload.Items))
	for _, item := range activePayload.Items {
		if !item.IsActive {
			return fmt.Errorf("/symbols/active returned inactive %s", item.AssetCode)
		}
		activeSet[item.AssetCode] = struct{}{}
	}
	for _, code := range InactiveAssetCodes {
		if _, ok := activeSet[code]; ok {
			return fmt.Errorf("inactive symbol %s exposed by /symbols/active", code)
		}
	}

	var allPayload struct {
		Items []struct {
			AssetCode string `json:"AssetCode"`
			IsActive  bool   `json:"IsActive"`
		} `json:"items"`
		Count int `json:"count"`
	}
	if err := decodeGET(client, baseURL+"/symbols/all", &allPayload); err != nil {
		return err
	}
	if allPayload.Count != ExpectedTotalSymbols {
		return fmt.Errorf("/symbols/all count: got %d want %d", allPayload.Count, ExpectedTotalSymbols)
	}

	allByCode := make(map[string]bool, len(allPayload.Items))
	for _, item := range allPayload.Items {
		allByCode[item.AssetCode] = item.IsActive
	}
	for _, code := range InactiveAssetCodes {
		isActive, ok := allByCode[code]
		if !ok {
			return fmt.Errorf("inactive symbol %s missing from /symbols/all", code)
		}
		if isActive {
			return fmt.Errorf("inactive symbol %s marked active in /symbols/all", code)
		}
	}

	var strategiesPayload struct {
		Items []struct {
			Slug string `json:"Slug"`
		} `json:"items"`
		Count int `json:"count"`
	}
	if err := decodeGET(client, baseURL+"/strategies", &strategiesPayload); err != nil {
		return err
	}
	if strategiesPayload.Count != ExpectedStrategies {
		return fmt.Errorf("/strategies count: got %d want %d", strategiesPayload.Count, ExpectedStrategies)
	}
	strategySet := make(map[string]struct{}, len(strategiesPayload.Items))
	for _, item := range strategiesPayload.Items {
		strategySet[item.Slug] = struct{}{}
	}
	for _, slug := range StrategySlugs {
		if _, ok := strategySet[slug]; !ok {
			return fmt.Errorf("strategy %s missing from /strategies", slug)
		}
	}

	return nil
}

func VerifyInspectableStrategyResult(client *http.Client, baseURL, symbol, timeframe, strategy string) error {
	chartURL := fmt.Sprintf("%s/chart-data?symbol=%s&timeframe=%s&strategy=%s", baseURL, symbol, timeframe, strategy)
	var chartPayload struct {
		Candles    []any `json:"candles"`
		RecentRuns []struct {
			Status string `json:"Status"`
		} `json:"recent_runs"`
	}
	if err := decodeGET(client, chartURL, &chartPayload); err != nil {
		return fmt.Errorf("%s %s: %w", strategy, timeframe, err)
	}
	if len(chartPayload.Candles) == 0 {
		return fmt.Errorf("%s %s: expected candles in chart-data", strategy, timeframe)
	}
	if len(chartPayload.RecentRuns) == 0 {
		return fmt.Errorf("%s %s: expected recent strategy runs", strategy, timeframe)
	}

	signalsURL := fmt.Sprintf("%s/signals?symbol=%s&timeframe=%s&strategy=%s", baseURL, symbol, timeframe, strategy)
	var signalsPayload struct {
		Items []struct {
			Title   string `json:"Title"`
			Details string `json:"Details"`
		} `json:"items"`
	}
	if err := decodeGET(client, signalsURL, &signalsPayload); err != nil {
		return fmt.Errorf("%s %s signals: %w", strategy, timeframe, err)
	}
	if len(signalsPayload.Items) == 0 {
		return fmt.Errorf("%s %s: expected inspectable signals", strategy, timeframe)
	}
	if signalsPayload.Items[0].Title == "" || signalsPayload.Items[0].Details == "" {
		return fmt.Errorf("%s %s: expected signal detail fields", strategy, timeframe)
	}

	return nil
}

func decodeGET(client *http.Client, url string, target any) error {
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("GET %s: %w", url, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("GET %s: status %d body=%s", url, resp.StatusCode, strings.TrimSpace(string(body)))
	}
	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		return fmt.Errorf("decode GET %s: %w", url, err)
	}
	return nil
}
