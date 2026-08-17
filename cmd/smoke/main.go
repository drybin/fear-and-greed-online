package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/drybin/fear-and-greed-online/internal/config"
	"github.com/drybin/fear-and-greed-online/internal/infrastructure/binance"
	"github.com/drybin/fear-and-greed-online/internal/services"
	strategysvc "github.com/drybin/fear-and-greed-online/internal/services/strategy"
	"github.com/drybin/fear-and-greed-online/internal/storage/postgres"
	engine "github.com/drybin/fear-and-greed-online/internal/strategy"
)

const (
	expectedActiveSymbols = 38
	expectedTimeframes    = 3
	expectedStrategies    = 3
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("smoke failed: %v", err)
	}
	log.Println("smoke passed")
}

func run() error {
	repoRoot, err := findRepoRoot()
	if err != nil {
		return err
	}
	if err := os.Chdir(repoRoot); err != nil {
		return fmt.Errorf("chdir repo root: %w", err)
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	smokeAsset := envOrDefault("SMOKE_ASSET", "BTC")
	smokeStrategy := envOrDefault("SMOKE_STRATEGY", "trend-long-v1")
	smokePort := envOrDefault("SMOKE_PORT", "18080")
	baseURL := fmt.Sprintf("http://127.0.0.1:%s", smokePort)

	step("start postgres")
	if err := runCommand(repoRoot, "docker", "compose", "up", "-d", "postgres"); err != nil {
		return fmt.Errorf("postgres up: %w", err)
	}

	step("wait for postgres")
	db, err := waitForPostgres(cfg, 45*time.Second)
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()

	step("apply migrations")
	if err := postgres.ApplyMigrations(db); err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}

	step("verify seeded reference data")
	if err := verifySeed(context.Background(), db); err != nil {
		return err
	}

	step(fmt.Sprintf("sync candles for %s", smokeAsset))
	syncService := services.NewCandleSyncService(
		postgres.NewSymbolRepository(db),
		postgres.NewTimeframeRepository(db),
		postgres.NewCandleRepository(db),
		postgres.NewIngestionJobRepository(db),
		binance.NewClient(cfg.BinanceBaseURL),
		cfg.SyncBackfillDays,
	)
	if err := syncService.Sync(context.Background(), smokeAsset); err != nil {
		return fmt.Errorf("sync candles: %w", err)
	}

	step(fmt.Sprintf("run strategy %s for %s", smokeStrategy, smokeAsset))
	runner := strategysvc.NewRunner(
		postgres.NewStrategyRepository(db),
		postgres.NewSymbolRepository(db),
		postgres.NewTimeframeRepository(db),
		postgres.NewCandleRepository(db),
		postgres.NewStrategyRunRepository(db),
		postgres.NewSignalRepository(db),
		postgres.NewTradeRepository(db),
		engine.NewRegistry(
			engine.NewTrendLongV1(),
			engine.NewBreakoutRetestV1(),
			engine.NewPrevDayRangeBreakoutV1(),
		),
		cfg.SyncBackfillDays,
	)
	if err := runner.Run(context.Background(), smokeStrategy, smokeAsset); err != nil {
		return fmt.Errorf("run strategies: %w", err)
	}

	step("start API")
	apiCmd := exec.Command("go", "run", "./cmd/api")
	apiCmd.Dir = repoRoot
	apiCmd.Env = append(os.Environ(),
		"APP_HOST=127.0.0.1",
		fmt.Sprintf("APP_PORT=%s", smokePort),
	)
	apiCmd.Stdout = os.Stdout
	apiCmd.Stderr = os.Stderr
	if err := apiCmd.Start(); err != nil {
		return fmt.Errorf("start api: %w", err)
	}
	defer stopProcess(apiCmd.Process)

	step("verify API endpoints")
	client := &http.Client{Timeout: 10 * time.Second}
	if err := waitForHTTP(client, baseURL+"/health", 45*time.Second); err != nil {
		return fmt.Errorf("api health: %w", err)
	}

	var readyPayload struct {
		Status   string `json:"status"`
		Database string `json:"database"`
	}
	if err := decodeGET(client, baseURL+"/ready", &readyPayload); err != nil {
		return err
	}
	if readyPayload.Status != "ok" || readyPayload.Database != "ready" {
		return fmt.Errorf("unexpected /ready payload: %#v", readyPayload)
	}

	var symbolsPayload struct {
		Items []struct {
			AssetCode string `json:"AssetCode"`
			IsActive  bool   `json:"IsActive"`
		} `json:"items"`
		ActiveCount int `json:"active_count"`
	}
	if err := decodeGET(client, baseURL+"/symbols/active", &symbolsPayload); err != nil {
		return err
	}
	if symbolsPayload.ActiveCount == 0 {
		return fmt.Errorf("expected active symbols in /symbols/active")
	}
	if !containsAsset(symbolsPayload.Items, smokeAsset) {
		return fmt.Errorf("%s not found in /symbols/active", smokeAsset)
	}

	var strategiesPayload struct {
		Items []struct {
			Slug string `json:"Slug"`
		} `json:"items"`
	}
	if err := decodeGET(client, baseURL+"/strategies", &strategiesPayload); err != nil {
		return err
	}
	if !containsStrategy(strategiesPayload.Items, smokeStrategy) {
		return fmt.Errorf("%s not found in /strategies", smokeStrategy)
	}

	chartURL := fmt.Sprintf("%s/chart-data?symbol=%s&timeframe=1h&strategy=%s", baseURL, smokeAsset, smokeStrategy)
	var chartPayload struct {
		Candles    []any          `json:"candles"`
		RecentRuns []any          `json:"recent_runs"`
		Freshness  map[string]any `json:"freshness"`
	}
	if err := decodeGET(client, chartURL, &chartPayload); err != nil {
		return err
	}
	if len(chartPayload.Candles) == 0 {
		return fmt.Errorf("expected candles in /chart-data")
	}
	if len(chartPayload.RecentRuns) == 0 {
		return fmt.Errorf("expected recent strategy runs in /chart-data")
	}
	if chartPayload.Freshness == nil || chartPayload.Freshness["latest_candle_time"] == nil {
		return fmt.Errorf("expected freshness.latest_candle_time in /chart-data")
	}

	resp, err := client.Get(baseURL + "/")
	if err != nil {
		return fmt.Errorf("GET /: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read dashboard shell: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("dashboard shell status: got %d", resp.StatusCode)
	}
	for _, needle := range []string{`id="chart"`, `id="symbol"`, `id="signalsNow"`, "lightweight-charts"} {
		if !strings.Contains(string(body), needle) {
			return fmt.Errorf("dashboard shell missing %q", needle)
		}
	}

	return nil
}

func step(message string) {
	log.Printf("smoke: %s", message)
}

func verifySeed(ctx context.Context, db *sql.DB) error {
	var activeSymbols int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM symbols WHERE is_active = TRUE`).Scan(&activeSymbols); err != nil {
		return fmt.Errorf("count active symbols: %w", err)
	}
	if activeSymbols != expectedActiveSymbols {
		return fmt.Errorf("expected %d active symbols, got %d", expectedActiveSymbols, activeSymbols)
	}

	var inactiveSymbols int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM symbols WHERE is_active = FALSE`).Scan(&inactiveSymbols); err != nil {
		return fmt.Errorf("count inactive symbols: %w", err)
	}
	if inactiveSymbols == 0 {
		return fmt.Errorf("expected inactive symbols in seeded universe")
	}

	var timeframes int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM timeframes WHERE is_active = TRUE`).Scan(&timeframes); err != nil {
		return fmt.Errorf("count timeframes: %w", err)
	}
	if timeframes != expectedTimeframes {
		return fmt.Errorf("expected %d active timeframes, got %d", expectedTimeframes, timeframes)
	}

	var strategies int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM strategies WHERE is_active = TRUE`).Scan(&strategies); err != nil {
		return fmt.Errorf("count strategies: %w", err)
	}
	if strategies != expectedStrategies {
		return fmt.Errorf("expected %d active strategies, got %d", expectedStrategies, strategies)
	}

	var btcActive bool
	if err := db.QueryRowContext(ctx, `SELECT is_active FROM symbols WHERE asset_code = 'BTC'`).Scan(&btcActive); err != nil {
		return fmt.Errorf("lookup BTC seed: %w", err)
	}
	if !btcActive {
		return fmt.Errorf("expected BTC to be active in seed data")
	}

	return nil
}

func waitForPostgres(cfg *config.Config, timeout time.Duration) (*sql.DB, error) {
	deadline := time.Now().Add(timeout)
	var lastErr error
	for time.Now().Before(deadline) {
		db, err := postgres.Open(cfg.Postgres)
		if err != nil {
			lastErr = err
			time.Sleep(time.Second)
			continue
		}
		if err := db.Ping(); err != nil {
			lastErr = err
			_ = db.Close()
			time.Sleep(time.Second)
			continue
		}
		return db, nil
	}
	return nil, fmt.Errorf("postgres not ready: %v", lastErr)
}

func waitForHTTP(client *http.Client, url string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	var lastErr error
	for time.Now().Before(deadline) {
		resp, err := client.Get(url)
		if err != nil {
			lastErr = err
			time.Sleep(time.Second)
			continue
		}
		_ = resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			return nil
		}
		lastErr = fmt.Errorf("status %d", resp.StatusCode)
		time.Sleep(time.Second)
	}
	return fmt.Errorf("endpoint %s not ready: %v", url, lastErr)
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

func runCommand(dir, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func stopProcess(process *os.Process) {
	if process == nil {
		return
	}
	_ = process.Signal(syscall.SIGTERM)
	done := make(chan struct{})
	go func() {
		_, _ = process.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		_ = process.Kill()
	}
}

func findRepoRoot() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	dir := cwd
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("repository root not found from %s", cwd)
		}
		dir = parent
	}
}

func envOrDefault(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func containsAsset(items []struct {
	AssetCode string `json:"AssetCode"`
	IsActive  bool   `json:"IsActive"`
}, code string) bool {
	for _, item := range items {
		if item.AssetCode == code && item.IsActive {
			return true
		}
	}
	return false
}

func containsStrategy(items []struct {
	Slug string `json:"Slug"`
}, slug string) bool {
	for _, item := range items {
		if item.Slug == slug {
			return true
		}
	}
	return false
}
