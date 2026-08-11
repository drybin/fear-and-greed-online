package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/drybin/fear-and-greed-online/internal/domain/marketdata"
)

type TimeframeRepository struct {
	db *sql.DB
}

func NewTimeframeRepository(db *sql.DB) *TimeframeRepository {
	return &TimeframeRepository{db: db}
}

func (r *TimeframeRepository) ListActive(ctx context.Context) ([]marketdata.Timeframe, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, code, duration_sec, is_active
		FROM timeframes
		WHERE is_active = TRUE
		ORDER BY duration_sec ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("list timeframes: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var items []marketdata.Timeframe
	for rows.Next() {
		var item marketdata.Timeframe
		if err := rows.Scan(&item.ID, &item.Code, &item.DurationSec, &item.IsActive); err != nil {
			return nil, fmt.Errorf("scan timeframe: %w", err)
		}
		items = append(items, item)
	}

	return items, rows.Err()
}

func (r *TimeframeRepository) FindByCode(ctx context.Context, code string) (*marketdata.Timeframe, error) {
	var item marketdata.Timeframe
	err := r.db.QueryRowContext(ctx, `
		SELECT id, code, duration_sec, is_active
		FROM timeframes
		WHERE code = $1
	`, code).Scan(&item.ID, &item.Code, &item.DurationSec, &item.IsActive)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("find timeframe by code: %w", err)
	}
	return &item, nil
}
