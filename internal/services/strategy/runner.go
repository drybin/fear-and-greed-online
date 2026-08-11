package strategy

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	market "github.com/drybin/fear-and-greed-online/internal/domain/marketdata"
	domain "github.com/drybin/fear-and-greed-online/internal/domain/strategy"
	"github.com/drybin/fear-and-greed-online/internal/storage/postgres"
	engine "github.com/drybin/fear-and-greed-online/internal/strategy"
)

type Runner struct {
	strategies   *postgres.StrategyRepository
	symbols      *postgres.SymbolRepository
	timeframes   *postgres.TimeframeRepository
	candles      *postgres.CandleRepository
	runs         *postgres.StrategyRunRepository
	signals      *postgres.SignalRepository
	trades       *postgres.TradeRepository
	registry     *engine.Registry
	lookbackDays int
}

func NewRunner(
	strategies *postgres.StrategyRepository,
	symbols *postgres.SymbolRepository,
	timeframes *postgres.TimeframeRepository,
	candles *postgres.CandleRepository,
	runs *postgres.StrategyRunRepository,
	signals *postgres.SignalRepository,
	trades *postgres.TradeRepository,
	registry *engine.Registry,
	lookbackDays int,
) *Runner {
	return &Runner{
		strategies:   strategies,
		symbols:      symbols,
		timeframes:   timeframes,
		candles:      candles,
		runs:         runs,
		signals:      signals,
		trades:       trades,
		registry:     registry,
		lookbackDays: lookbackDays,
	}
}

func (r *Runner) Run(ctx context.Context, strategyFilter, assetFilter string) error {
	defs, err := r.strategies.ListActive(ctx, strings.TrimSpace(strategyFilter))
	if err != nil {
		return err
	}
	if len(defs) == 0 {
		return fmt.Errorf("no active strategies found")
	}

	symbols, err := r.symbols.ListActive(ctx, strings.ToUpper(strings.TrimSpace(assetFilter)))
	if err != nil {
		return err
	}
	if len(symbols) == 0 {
		return fmt.Errorf("no active symbols found")
	}

	timeframes, err := r.timeframes.ListActive(ctx)
	if err != nil {
		return err
	}
	if len(timeframes) == 0 {
		return fmt.Errorf("no active timeframes found")
	}

	for _, def := range defs {
		strategyImpl, err := r.registry.Get(def.Slug)
		if err != nil {
			return err
		}

		for _, symbol := range symbols {
			for _, timeframe := range timeframes {
				if !supportsTimeframe(def.SupportedTimeframes, timeframe.Code) {
					continue
				}
				if err := r.runOne(ctx, def, strategyImpl, symbol, timeframe); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (r *Runner) runOne(ctx context.Context, def domain.Definition, impl engine.Strategy, symbol market.Symbol, timeframe market.Timeframe) error {
	earliest, err := r.candles.EarliestOpenTime(ctx, symbol.ID, timeframe.ID)
	if err != nil {
		return err
	}
	if earliest == nil {
		log.Printf("strategy run skipped %s %s %s no candles available", def.Slug, symbol.AssetCode, timeframe.Code)
		return nil
	}

	from := earliest.UTC()
	to := time.Now().UTC()

	rows, err := r.candles.ListRange(ctx, symbol.ID, timeframe.ID, from, to)
	if err != nil {
		return err
	}
	if len(rows) < def.RequiredHistoryBars {
		log.Printf(
			"strategy run skipped %s %s %s insufficient history: have=%d need=%d",
			def.Slug,
			symbol.AssetCode,
			timeframe.Code,
			len(rows),
			def.RequiredHistoryBars,
		)
		return nil
	}

	inputFrom := rows[0].OpenTime
	inputTo := rows[len(rows)-1].OpenTime
	runID, err := r.runs.Start(ctx, def.ID, symbol.ID, timeframe.ID, def.DefaultParams, &inputFrom, &inputTo, len(rows))
	if err != nil {
		return err
	}

	log.Printf(
		"running strategy %s on %s %s run_id=%d candles=%d window=%s..%s",
		def.Slug,
		symbol.AssetCode,
		timeframe.Code,
		runID,
		len(rows),
		inputFrom.UTC().Format(time.RFC3339),
		inputTo.UTC().Format(time.RFC3339),
	)

	output, err := impl.Run(engine.RunInput{
		StrategySlug: def.Slug,
		Symbol:       symbol.AssetCode,
		Timeframe:    timeframe.Code,
		Candles:      toStrategyCandles(rows),
		Params:       def.DefaultParams,
	})
	if err != nil {
		_ = r.runs.Fail(ctx, runID, err.Error())
		log.Printf(
			"strategy run failed %s %s %s run_id=%d: %v",
			def.Slug,
			symbol.AssetCode,
			timeframe.Code,
			runID,
			err,
		)
		return fmt.Errorf("run strategy %s on %s %s: %w", def.Slug, symbol.AssetCode, timeframe.Code, err)
	}

	if err := r.signals.ReplaceRange(ctx, runID, def.ID, symbol.ID, timeframe.ID, inputFrom, inputTo, output.Signals); err != nil {
		_ = r.runs.Fail(ctx, runID, err.Error())
		return err
	}
	if err := r.trades.ReplaceRange(ctx, runID, def.ID, symbol.ID, timeframe.ID, inputFrom, inputTo, output.Trades); err != nil {
		_ = r.runs.Fail(ctx, runID, err.Error())
		return err
	}

	if err := r.runs.Complete(ctx, runID, len(output.Signals), len(output.Trades)); err != nil {
		return err
	}

	log.Printf(
		"strategy run complete %s %s %s run_id=%d signals=%d trades=%d",
		def.Slug,
		symbol.AssetCode,
		timeframe.Code,
		runID,
		len(output.Signals),
		len(output.Trades),
	)

	return nil
}

func supportsTimeframe(supported []string, code string) bool {
	for _, item := range supported {
		if item == code {
			return true
		}
	}
	return false
}

func toStrategyCandles(rows []market.Candle) []domain.Candle {
	out := make([]domain.Candle, 0, len(rows))
	for _, row := range rows {
		open, _ := strconv.ParseFloat(row.Open, 64)
		high, _ := strconv.ParseFloat(row.High, 64)
		low, _ := strconv.ParseFloat(row.Low, 64)
		closeValue, _ := strconv.ParseFloat(row.Close, 64)
		volume, _ := strconv.ParseFloat(row.Volume, 64)
		out = append(out, domain.Candle{
			Time:   row.OpenTime,
			Open:   open,
			High:   high,
			Low:    low,
			Close:  closeValue,
			Volume: volume,
		})
	}
	return out
}
