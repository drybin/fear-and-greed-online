package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type IngestionJobRepository struct {
	db *sql.DB
}

func NewIngestionJobRepository(db *sql.DB) *IngestionJobRepository {
	return &IngestionJobRepository{db: db}
}

func (r *IngestionJobRepository) Start(ctx context.Context, symbolID, timeframeID int64, requestedFrom, requestedTo time.Time) (int64, error) {
	var id int64
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO ingestion_jobs (
			symbol_id, timeframe_id, status, requested_from, requested_to, started_at, created_at
		) VALUES ($1, $2, 'running', $3, $4, NOW(), NOW())
		RETURNING id
	`, symbolID, timeframeID, requestedFrom, requestedTo).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("start ingestion job: %w", err)
	}
	return id, nil
}

func (r *IngestionJobRepository) Complete(ctx context.Context, jobID int64, loadedFrom, loadedTo *time.Time, candlesLoaded int) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE ingestion_jobs
		SET status = 'success',
			loaded_from = $2,
			loaded_to = $3,
			candles_loaded = $4,
			finished_at = NOW()
		WHERE id = $1
	`, jobID, loadedFrom, loadedTo, candlesLoaded)
	if err != nil {
		return fmt.Errorf("complete ingestion job: %w", err)
	}
	return nil
}

func (r *IngestionJobRepository) Fail(ctx context.Context, jobID int64, message string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE ingestion_jobs
		SET status = 'failed',
			error_message = $2,
			finished_at = NOW()
		WHERE id = $1
	`, jobID, message)
	if err != nil {
		return fmt.Errorf("fail ingestion job: %w", err)
	}
	return nil
}

func (r *IngestionJobRepository) LatestSuccessful(ctx context.Context, symbolID, timeframeID int64) (*IngestionJobSummary, error) {
	var summary IngestionJobSummary
	err := r.db.QueryRowContext(ctx, `
		SELECT finished_at, loaded_to, candles_loaded
		FROM ingestion_jobs
		WHERE symbol_id = $1 AND timeframe_id = $2 AND status = 'success' AND finished_at IS NOT NULL
		ORDER BY finished_at DESC
		LIMIT 1
	`, symbolID, timeframeID).Scan(&summary.FinishedAt, &summary.LoadedTo, &summary.CandlesLoaded)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("latest successful ingestion job: %w", err)
	}

	return &summary, nil
}
