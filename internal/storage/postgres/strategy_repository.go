package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	domain "github.com/drybin/fear-and-greed-online/internal/domain/strategy"
)

type StrategyRepository struct {
	db *sql.DB
}

func NewStrategyRepository(db *sql.DB) *StrategyRepository {
	return &StrategyRepository{db: db}
}

func (r *StrategyRepository) ListActive(ctx context.Context, slugFilter string) ([]domain.Definition, error) {
	query := `
		SELECT id, code, version, slug, name, description, category, default_params, supported_timeframes, required_history_bars, is_active
		FROM strategies
		WHERE is_active = TRUE
	`
	args := []any{}
	if slugFilter != "" {
		query += ` AND slug = $1`
		args = append(args, slugFilter)
	}
	query += ` ORDER BY id ASC`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list active strategies: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var items []domain.Definition
	for rows.Next() {
		var item domain.Definition
		var paramsJSON []byte
		var supportedJSON []byte
		if err := rows.Scan(
			&item.ID,
			&item.Code,
			&item.Version,
			&item.Slug,
			&item.Name,
			&item.Description,
			&item.Category,
			&paramsJSON,
			&supportedJSON,
			&item.RequiredHistoryBars,
			&item.IsActive,
		); err != nil {
			return nil, fmt.Errorf("scan strategy definition: %w", err)
		}
		if err := json.Unmarshal(paramsJSON, &item.DefaultParams); err != nil {
			return nil, fmt.Errorf("decode strategy params: %w", err)
		}
		if err := json.Unmarshal(supportedJSON, &item.SupportedTimeframes); err != nil {
			return nil, fmt.Errorf("decode strategy supported_timeframes: %w", err)
		}
		items = append(items, item)
	}

	return items, rows.Err()
}

func (r *StrategyRepository) FindBySlug(ctx context.Context, slug string) (*domain.Definition, error) {
	items, err := r.ListActive(ctx, slug)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, nil
	}
	return &items[0], nil
}
