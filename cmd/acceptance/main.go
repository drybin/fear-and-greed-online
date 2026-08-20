package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/drybin/fear-and-greed-online/internal/acceptance"
	"github.com/drybin/fear-and-greed-online/internal/config"
	"github.com/drybin/fear-and-greed-online/internal/infrastructure/binance"
	"github.com/drybin/fear-and-greed-online/internal/services"
	strategysvc "github.com/drybin/fear-and-greed-online/internal/services/strategy"
	"github.com/drybin/fear-and-greed-online/internal/storage/postgres"
	engine "github.com/drybin/fear-and-greed-online/internal/strategy"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("acceptance failed: %v", err)
	}
	log.Println("acceptance passed")
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

	asset := envOrDefault("ACCEPTANCE_ASSET", "BTC")
	acceptancePort := envOrDefault("ACCEPTANCE_PORT", "18081")
	baseURL := fmt.Sprintf("http://127.0.0.1:%s", acceptancePort)

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

	ctx := context.Background()
	symbolRepo := postgres.NewSymbolRepository(db)

	step("verify frozen top-50 universe")
	if err := acceptance.VerifyFrozenUniverse(ctx, db); err != nil {
		return err
	}

	step("verify CLI-queryable active universe")
	if err := acceptance.VerifyCLIQueryableUniverse(ctx, symbolRepo); err != nil {
		return err
	}

	step(fmt.Sprintf("sync candles for %s", asset))
	syncService := services.NewCandleSyncService(
		symbolRepo,
		postgres.NewTimeframeRepository(db),
		postgres.NewCandleRepository(db),
		postgres.NewIngestionJobRepository(db),
		binance.NewClient(cfg.BinanceBaseURL),
		cfg.SyncBackfillDays,
	)
	if err := syncService.Sync(ctx, asset); err != nil {
		return fmt.Errorf("sync candles: %w", err)
	}

	step("run both MVP strategies")
	runner := strategysvc.NewRunner(
		postgres.NewStrategyRepository(db),
		symbolRepo,
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
		nil,
	)
	if err := runner.Run(ctx, "", asset); err != nil {
		return fmt.Errorf("run strategies: %w", err)
	}

	step("start API")
	apiCmd := exec.Command("go", "run", "./cmd/api")
	apiCmd.Dir = repoRoot
	apiCmd.Env = append(os.Environ(),
		"APP_HOST=127.0.0.1",
		fmt.Sprintf("APP_PORT=%s", acceptancePort),
	)
	apiCmd.Stdout = os.Stdout
	apiCmd.Stderr = os.Stderr
	if err := apiCmd.Start(); err != nil {
		return fmt.Errorf("start api: %w", err)
	}
	defer stopProcess(apiCmd.Process)

	client := &http.Client{Timeout: 15 * time.Second}
	step("verify API universe")
	if err := waitForHTTP(client, baseURL+"/health", 45*time.Second); err != nil {
		return fmt.Errorf("api health: %w", err)
	}
	if err := acceptance.VerifyAPIUniverse(client, baseURL); err != nil {
		return err
	}

	step("verify inspectable trend-long-v1 results")
	if err := acceptance.VerifyInspectableStrategyResult(client, baseURL, asset, "1h", "trend-long-v1"); err != nil {
		return err
	}

	step("verify inspectable breakout-retest-v1 results")
	if err := acceptance.VerifyInspectableStrategyResult(client, baseURL, asset, "15m", "breakout-retest-v1"); err != nil {
		return err
	}

	return nil
}

func step(message string) {
	log.Printf("acceptance: %s", message)
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
