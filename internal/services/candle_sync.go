package services

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/drybin/fear-and-greed-online/internal/domain/marketdata"
	"github.com/drybin/fear-and-greed-online/internal/infrastructure/binance"
	"github.com/drybin/fear-and-greed-online/internal/storage/postgres"
)

type CandleSyncService struct {
	symbols      *postgres.SymbolRepository
	timeframes   *postgres.TimeframeRepository
	candles      *postgres.CandleRepository
	ingestion    *postgres.IngestionJobRepository
	client       *binance.Client
	backfillDays int
}

func NewCandleSyncService(
	symbols *postgres.SymbolRepository,
	timeframes *postgres.TimeframeRepository,
	candles *postgres.CandleRepository,
	ingestion *postgres.IngestionJobRepository,
	client *binance.Client,
	backfillDays int,
) *CandleSyncService {
	return &CandleSyncService{
		symbols:      symbols,
		timeframes:   timeframes,
		candles:      candles,
		ingestion:    ingestion,
		client:       client,
		backfillDays: backfillDays,
	}
}

func (s *CandleSyncService) Sync(ctx context.Context, assetFilter string) error {
	assetFilter = strings.ToUpper(strings.TrimSpace(assetFilter))

	symbols, err := s.symbols.ListActive(ctx, assetFilter)
	if err != nil {
		return err
	}
	if len(symbols) == 0 {
		return fmt.Errorf("no active symbols found")
	}

	timeframes, err := s.timeframes.ListActive(ctx)
	if err != nil {
		return err
	}
	if len(timeframes) == 0 {
		return fmt.Errorf("no active timeframes found")
	}

	for _, symbol := range symbols {
		for _, timeframe := range timeframes {
			if err := s.syncPair(ctx, symbol, timeframe); err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *CandleSyncService) syncPair(ctx context.Context, symbol marketdata.Symbol, timeframe marketdata.Timeframe) error {
	requestedFrom, requestedTo, err := s.syncWindow(ctx, symbol.ID, timeframe)
	if err != nil {
		return err
	}

	jobID, err := s.ingestion.Start(ctx, symbol.ID, timeframe.ID, requestedFrom, requestedTo)
	if err != nil {
		return err
	}

	log.Printf("syncing %s %s from %s to %s", symbol.BinanceSymbol, timeframe.Code, requestedFrom.Format(time.RFC3339), requestedTo.Format(time.RFC3339))

	candles, err := s.client.FetchCandles(ctx, symbol, timeframe, requestedFrom, requestedTo)
	if err != nil {
		_ = s.ingestion.Fail(ctx, jobID, err.Error())
		log.Printf("sync failed %s %s job_id=%d: %v", symbol.AssetCode, timeframe.Code, jobID, err)
		return fmt.Errorf("fetch candles for %s %s: %w", symbol.AssetCode, timeframe.Code, err)
	}

	for i := range candles {
		candles[i].SymbolID = symbol.ID
		candles[i].TimeframeID = timeframe.ID
	}

	if err := s.candles.UpsertMany(ctx, candles); err != nil {
		_ = s.ingestion.Fail(ctx, jobID, err.Error())
		return err
	}

	var loadedFrom, loadedTo *time.Time
	if len(candles) > 0 {
		loadedFrom = &candles[0].OpenTime
		loadedTo = &candles[len(candles)-1].OpenTime
	}

	if err := s.ingestion.Complete(ctx, jobID, loadedFrom, loadedTo, len(candles)); err != nil {
		return err
	}

	log.Printf(
		"synced %s %s loaded=%d job_id=%d loaded_from=%s loaded_to=%s",
		symbol.AssetCode,
		timeframe.Code,
		len(candles),
		jobID,
		formatOptionalTime(loadedFrom),
		formatOptionalTime(loadedTo),
	)

	return nil
}

func (s *CandleSyncService) syncWindow(ctx context.Context, symbolID int64, timeframe marketdata.Timeframe) (time.Time, time.Time, error) {
	now := time.Now().UTC()

	latest, err := s.candles.LatestOpenTime(ctx, symbolID, timeframe.ID)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}

	if latest == nil {
		return now.AddDate(0, 0, -s.backfillDays), now, nil
	}

	return latest.UTC(), now, nil
}

func formatOptionalTime(value *time.Time) string {
	if value == nil {
		return "-"
	}
	return value.UTC().Format(time.RFC3339)
}
