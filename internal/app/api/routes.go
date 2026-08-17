package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	marketdata "github.com/drybin/fear-and-greed-online/internal/domain/marketdata"
	strategydomain "github.com/drybin/fear-and-greed-online/internal/domain/strategy"
	"github.com/drybin/fear-and-greed-online/internal/storage/postgres"
)

func registerRoutes(mux *http.ServeMux, db *sql.DB) {
	symbols := postgres.NewSymbolRepository(db)
	timeframes := postgres.NewTimeframeRepository(db)
	strategies := postgres.NewStrategyRepository(db)
	candles := postgres.NewCandleRepository(db)
	signals := postgres.NewSignalRepository(db)
	trades := postgres.NewTradeRepository(db)
	runs := postgres.NewStrategyRunRepository(db)
	ingestion := postgres.NewIngestionJobRepository(db)

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"status":  "ok",
			"service": "api",
		})
	})

	mux.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		if err := db.PingContext(r.Context()); err != nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]any{
				"status":   "error",
				"service":  "api",
				"database": "unavailable",
			})
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"status":   "ok",
			"service":  "api",
			"database": "ready",
		})
	})

	mux.HandleFunc("/symbols/active", func(w http.ResponseWriter, r *http.Request) {
		items, err := symbols.ListActive(r.Context(), "")
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{
				"status":  "error",
				"service": "api",
				"error":   err.Error(),
			})
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"items":          items,
			"count":          len(items),
			"active_count":   len(items),
			"inactive_count": 0,
		})
	})

	mux.HandleFunc("/symbols/all", func(w http.ResponseWriter, r *http.Request) {
		items, err := symbols.ListAll(r.Context())
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{
				"status":  "error",
				"service": "api",
				"error":   err.Error(),
			})
			return
		}

		activeCount := 0
		for _, item := range items {
			if item.IsActive {
				activeCount++
			}
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"items":          items,
			"count":          len(items),
			"active_count":   activeCount,
			"inactive_count": len(items) - activeCount,
		})
	})

	mux.HandleFunc("/timeframes", func(w http.ResponseWriter, r *http.Request) {
		items, err := timeframes.ListActive(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": items, "count": len(items)})
	})

	mux.HandleFunc("/strategies", func(w http.ResponseWriter, r *http.Request) {
		items, err := strategies.ListActive(r.Context(), strings.TrimSpace(r.URL.Query().Get("slug")))
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": items, "count": len(items)})
	})

	mux.HandleFunc("/candles", func(w http.ResponseWriter, r *http.Request) {
		ctxData, err := resolveChartContext(r, symbols, timeframes, strategies, false)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		items, err := candles.ListRange(r.Context(), ctxData.symbol.ID, ctxData.timeframe.ID, ctxData.from, ctxData.to)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"symbol":    ctxData.symbol.AssetCode,
			"timeframe": ctxData.timeframe.Code,
			"from":      ctxData.from,
			"to":        ctxData.to,
			"items":     items,
			"count":     len(items),
		})
	})

	mux.HandleFunc("/signals", func(w http.ResponseWriter, r *http.Request) {
		ctxData, err := resolveChartContext(r, symbols, timeframes, strategies, true)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		items, err := signals.ListRange(r.Context(), ctxData.strategy.ID, ctxData.symbol.ID, ctxData.timeframe.ID, ctxData.from, ctxData.to)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": items, "count": len(items)})
	})

	mux.HandleFunc("/trades", func(w http.ResponseWriter, r *http.Request) {
		ctxData, err := resolveChartContext(r, symbols, timeframes, strategies, true)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		items, err := trades.ListRange(r.Context(), ctxData.strategy.ID, ctxData.symbol.ID, ctxData.timeframe.ID, ctxData.from, ctxData.to)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": items, "count": len(items)})
	})

	mux.HandleFunc("/strategy-runs", func(w http.ResponseWriter, r *http.Request) {
		ctxData, err := resolveChartContext(r, symbols, timeframes, strategies, true)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		items, err := runs.ListRecent(r.Context(), ctxData.strategy.ID, ctxData.symbol.ID, ctxData.timeframe.ID, limit)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": items, "count": len(items)})
	})

	mux.HandleFunc("/freshness", func(w http.ResponseWriter, r *http.Request) {
		ctxData, err := resolveChartContext(r, symbols, timeframes, strategies, true)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}

		freshness, err := loadFreshness(
			r.Context(),
			ctxData,
			candles,
			ingestion,
			runs,
		)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"symbol":    ctxData.symbol,
			"timeframe": ctxData.timeframe,
			"strategy":  ctxData.strategy,
			"freshness": freshness,
		})
	})

	mux.HandleFunc("/chart-data", func(w http.ResponseWriter, r *http.Request) {
		ctxData, err := resolveChartContext(r, symbols, timeframes, strategies, true)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}

		candleItems, err := candles.ListRange(r.Context(), ctxData.symbol.ID, ctxData.timeframe.ID, ctxData.from, ctxData.to)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		signalItems, err := signals.ListRange(r.Context(), ctxData.strategy.ID, ctxData.symbol.ID, ctxData.timeframe.ID, ctxData.from, ctxData.to)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		tradeItems, err := trades.ListRange(r.Context(), ctxData.strategy.ID, ctxData.symbol.ID, ctxData.timeframe.ID, ctxData.from, ctxData.to)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		runItems, err := runs.ListRecent(r.Context(), ctxData.strategy.ID, ctxData.symbol.ID, ctxData.timeframe.ID, 10)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		freshness, err := loadFreshness(
			r.Context(),
			ctxData,
			candles,
			ingestion,
			runs,
		)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"symbol":      ctxData.symbol,
			"timeframe":   ctxData.timeframe,
			"strategy":    ctxData.strategy,
			"from":        ctxData.from,
			"to":          ctxData.to,
			"candles":     candleItems,
			"signals":     signalItems,
			"trades":      tradeItems,
			"recent_runs": runItems,
			"freshness":   freshness,
		})
	})

	fileServer := http.FileServer(http.Dir("web"))
	mux.Handle("/", noCacheStatic(fileServer))
}

func noCacheStatic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store, max-age=0")
		w.Header().Set("Pragma", "no-cache")
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]any{
		"status": "error",
		"error":  err.Error(),
	})
}

func resolveChartContext(
	r *http.Request,
	symbols *postgres.SymbolRepository,
	timeframes *postgres.TimeframeRepository,
	strategies *postgres.StrategyRepository,
	requireStrategy bool,
) (*resolvedContext, error) {
	q := r.URL.Query()
	symbolCode := strings.ToUpper(strings.TrimSpace(q.Get("symbol")))
	timeframeCode := strings.TrimSpace(q.Get("timeframe"))
	if symbolCode == "" {
		return nil, fmt.Errorf("symbol is required")
	}
	if timeframeCode == "" {
		return nil, fmt.Errorf("timeframe is required")
	}

	symbol, err := symbols.FindByAssetCode(r.Context(), symbolCode)
	if err != nil {
		return nil, err
	}
	if symbol == nil {
		return nil, fmt.Errorf("unknown symbol: %s", symbolCode)
	}

	timeframe, err := timeframes.FindByCode(r.Context(), timeframeCode)
	if err != nil {
		return nil, err
	}
	if timeframe == nil {
		return nil, fmt.Errorf("unknown timeframe: %s", timeframeCode)
	}

	var strategyDef *strategydomain.Definition
	strategySlug := strings.TrimSpace(q.Get("strategy"))
	if requireStrategy {
		if strategySlug == "" {
			return nil, fmt.Errorf("strategy is required")
		}
		strategyRaw, err := strategies.FindBySlug(r.Context(), strategySlug)
		if err != nil {
			return nil, err
		}
		if strategyRaw == nil {
			return nil, fmt.Errorf("unknown strategy: %s", strategySlug)
		}
		if !supportsStrategyTimeframe(strategyRaw.SupportedTimeframes, timeframe.Code) {
			return nil, fmt.Errorf("strategy %s does not support timeframe %s", strategySlug, timeframe.Code)
		}
		strategyDef = strategyRaw
	}

	from, err := parseTime(q.Get("from"), time.Now().UTC().AddDate(0, 0, -30))
	if err != nil {
		return nil, fmt.Errorf("invalid from: %w", err)
	}
	to, err := parseTime(q.Get("to"), time.Now().UTC())
	if err != nil {
		return nil, fmt.Errorf("invalid to: %w", err)
	}
	if !from.Before(to) {
		return nil, fmt.Errorf("from must be before to")
	}

	return &resolvedContext{
		symbol:    symbol,
		timeframe: timeframe,
		strategy:  strategyDef,
		from:      from.UTC(),
		to:        to.UTC(),
	}, nil
}

type resolvedContext struct {
	symbol    *marketdata.Symbol
	timeframe *marketdata.Timeframe
	strategy  *strategydomain.Definition
	from      time.Time
	to        time.Time
}

func parseTime(raw string, fallback time.Time) (time.Time, error) {
	if strings.TrimSpace(raw) == "" {
		return fallback.UTC(), nil
	}
	if parsed, err := time.Parse(time.RFC3339Nano, raw); err == nil {
		return parsed.UTC(), nil
	}
	if parsed, err := time.Parse(time.RFC3339, raw); err == nil {
		return parsed.UTC(), nil
	}
	return time.Time{}, fmt.Errorf("unsupported time format %q", raw)
}

func supportsStrategyTimeframe(supported []string, code string) bool {
	for _, item := range supported {
		if item == code {
			return true
		}
	}
	return false
}

func buildChartFreshness(
	ctx context.Context,
	candles *postgres.CandleRepository,
	ingestion *postgres.IngestionJobRepository,
	runs *postgres.StrategyRunRepository,
	symbolID, timeframeID, strategyID int64,
) (map[string]any, error) {
	freshness := map[string]any{
		"symbol_id":    symbolID,
		"timeframe_id": timeframeID,
		"strategy_id":  strategyID,
	}

	latestCandle, err := candles.LatestOpenTime(ctx, symbolID, timeframeID)
	if err != nil {
		return nil, err
	}
	if latestCandle != nil {
		freshness["latest_candle_time"] = latestCandle.UTC()
	}

	lastSync, err := ingestion.LatestSuccessful(ctx, symbolID, timeframeID)
	if err != nil {
		return nil, err
	}
	if lastSync != nil {
		freshness["last_candle_sync_at"] = lastSync.FinishedAt.UTC()
		freshness["last_candle_sync_loaded_to"] = lastSync.LoadedTo
		freshness["last_candle_sync_count"] = lastSync.CandlesLoaded
	}

	lastRun, err := runs.LatestSuccessful(ctx, strategyID, symbolID, timeframeID)
	if err != nil {
		return nil, err
	}
	if lastRun != nil && lastRun.FinishedAt != nil {
		freshness["last_strategy_run_at"] = lastRun.FinishedAt.UTC()
		freshness["last_strategy_run_signals_count"] = lastRun.SignalsCount
		freshness["last_strategy_run_trades_count"] = lastRun.TradesCount
	}

	return freshness, nil
}

func loadFreshness(
	ctx context.Context,
	ctxData *resolvedContext,
	candles *postgres.CandleRepository,
	ingestion *postgres.IngestionJobRepository,
	runs *postgres.StrategyRunRepository,
) (map[string]any, error) {
	freshness, err := buildChartFreshness(
		ctx,
		candles,
		ingestion,
		runs,
		ctxData.symbol.ID,
		ctxData.timeframe.ID,
		ctxData.strategy.ID,
	)
	if err != nil {
		return nil, err
	}

	freshness["symbol"] = ctxData.symbol.AssetCode
	freshness["timeframe"] = ctxData.timeframe.Code
	freshness["strategy"] = ctxData.strategy.Slug

	return freshness, nil
}
