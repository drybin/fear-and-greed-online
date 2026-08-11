package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/drybin/fear-and-greed-online/internal/domain/marketdata"
)

type SymbolRepository struct {
	db *sql.DB
}

func NewSymbolRepository(db *sql.DB) *SymbolRepository {
	return &SymbolRepository{db: db}
}

func (r *SymbolRepository) ListActive(ctx context.Context, assetFilter string) ([]marketdata.Symbol, error) {
	query := `
		SELECT id, asset_code, name, market_cap_rank, quote_asset, binance_symbol, is_active
		FROM symbols
		WHERE is_active = TRUE
	`
	args := []any{}
	if assetFilter != "" {
		query += ` AND asset_code = $1`
		args = append(args, assetFilter)
	}
	query += ` ORDER BY market_cap_rank ASC, asset_code ASC`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list active symbols: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var symbols []marketdata.Symbol
	for rows.Next() {
		var item marketdata.Symbol
		if err := rows.Scan(
			&item.ID,
			&item.AssetCode,
			&item.Name,
			&item.MarketRank,
			&item.QuoteAsset,
			&item.BinanceSymbol,
			&item.IsActive,
		); err != nil {
			return nil, fmt.Errorf("scan symbol: %w", err)
		}
		symbols = append(symbols, item)
	}

	return symbols, rows.Err()
}

func (r *SymbolRepository) ListAll(ctx context.Context) ([]marketdata.Symbol, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, asset_code, name, market_cap_rank, quote_asset, binance_symbol, is_active
		FROM symbols
		ORDER BY market_cap_rank ASC, asset_code ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("list all symbols: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var symbols []marketdata.Symbol
	for rows.Next() {
		var item marketdata.Symbol
		if err := rows.Scan(
			&item.ID,
			&item.AssetCode,
			&item.Name,
			&item.MarketRank,
			&item.QuoteAsset,
			&item.BinanceSymbol,
			&item.IsActive,
		); err != nil {
			return nil, fmt.Errorf("scan symbol: %w", err)
		}
		symbols = append(symbols, item)
	}

	return symbols, rows.Err()
}

func (r *SymbolRepository) FindByAssetCode(ctx context.Context, assetCode string) (*marketdata.Symbol, error) {
	var item marketdata.Symbol
	err := r.db.QueryRowContext(ctx, `
		SELECT id, asset_code, name, market_cap_rank, quote_asset, binance_symbol, is_active
		FROM symbols
		WHERE asset_code = $1
	`, assetCode).Scan(
		&item.ID,
		&item.AssetCode,
		&item.Name,
		&item.MarketRank,
		&item.QuoteAsset,
		&item.BinanceSymbol,
		&item.IsActive,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("find symbol by asset_code: %w", err)
	}

	return &item, nil
}
