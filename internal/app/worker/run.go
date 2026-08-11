package worker

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/drybin/fear-and-greed-online/internal/config"
	"github.com/drybin/fear-and-greed-online/internal/infrastructure/binance"
	"github.com/drybin/fear-and-greed-online/internal/services"
	strategysvc "github.com/drybin/fear-and-greed-online/internal/services/strategy"
	"github.com/drybin/fear-and-greed-online/internal/storage/postgres"
	engine "github.com/drybin/fear-and-greed-online/internal/strategy"
)

func Run(cfg *config.Config, args []string) error {
	db, err := postgres.Open(cfg.Postgres)
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()

	if err := postgres.ApplyMigrations(db); err != nil {
		return err
	}

	log.Printf("worker started in %s", cfg.AppEnv)
	log.Printf("postgres connection is ready")

	if err := db.Ping(); err != nil {
		return fmt.Errorf("worker database check failed: %w", err)
	}

	if len(args) == 0 {
		log.Printf("worker bootstrap complete")
		return nil
	}

	switch args[0] {
	case "sync-candles":
		return runSyncCandles(context.Background(), cfg, db, args[1:])
	case "list-active-symbols":
		return runListActiveSymbols(context.Background(), db)
	case "run-strategies":
		return runStrategies(context.Background(), cfg, db, args[1:])
	default:
		return fmt.Errorf("unsupported worker command: %s", args[0])
	}
}

func runSyncCandles(ctx context.Context, cfg *config.Config, db *sql.DB, args []string) error {
	assetFilter := ""
	if len(args) >= 2 && args[0] == "--asset" {
		assetFilter = strings.TrimSpace(args[1])
	}

	service := services.NewCandleSyncService(
		postgres.NewSymbolRepository(db),
		postgres.NewTimeframeRepository(db),
		postgres.NewCandleRepository(db),
		postgres.NewIngestionJobRepository(db),
		binance.NewClient(cfg.BinanceBaseURL),
		cfg.SyncBackfillDays,
	)

	return service.Sync(ctx, assetFilter)
}

func Bootstrap(cfg *config.Config) error {
	return Run(cfg, nil)
}

func runStrategies(ctx context.Context, cfg *config.Config, db *sql.DB, args []string) error {
	strategyFilter := ""
	assetFilter := ""
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--strategy":
			if i+1 < len(args) {
				strategyFilter = strings.TrimSpace(args[i+1])
				i++
			}
		case "--asset":
			if i+1 < len(args) {
				assetFilter = strings.TrimSpace(args[i+1])
				i++
			}
		}
	}

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

	return runner.Run(ctx, strategyFilter, assetFilter)
}

func runListActiveSymbols(ctx context.Context, db *sql.DB) error {
	items, err := postgres.NewSymbolRepository(db).ListActive(ctx, "")
	if err != nil {
		return err
	}

	writer := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	if _, err := fmt.Fprintln(writer, "RANK\tASSET\tBINANCE_SYMBOL\tQUOTE\tACTIVE"); err != nil {
		return err
	}
	for _, item := range items {
		if _, err := fmt.Fprintf(writer, "%d\t%s\t%s\t%s\t%t\n", item.MarketRank, item.AssetCode, item.BinanceSymbol, item.QuoteAsset, item.IsActive); err != nil {
			return err
		}
	}
	return writer.Flush()
}
