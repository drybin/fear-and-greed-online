package postgres

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

func ApplyMigrations(db *sql.DB) error {
	migrationsDir, err := resolveMigrationsDir()
	if err != nil {
		return err
	}

	if err := ensureSchemaMigrationsTable(db); err != nil {
		return err
	}

	names, err := listUpMigrations(migrationsDir)
	if err != nil {
		return err
	}

	for _, name := range names {
		applied, err := migrationApplied(db, name)
		if err != nil {
			return err
		}
		if applied {
			continue
		}

		body, err := os.ReadFile(filepath.Join(migrationsDir, name))
		if err != nil {
			return fmt.Errorf("read migration %s: %w", name, err)
		}

		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("begin migration %s: %w", name, err)
		}

		if _, err := tx.Exec(string(body)); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("execute migration %s: %w", name, err)
		}
		if _, err := tx.Exec(`INSERT INTO schema_migrations(version) VALUES ($1)`, name); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("track migration %s: %w", name, err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %s: %w", name, err)
		}
	}

	return nil
}

func RollbackLastMigration(db *sql.DB) error {
	migrationsDir, err := resolveMigrationsDir()
	if err != nil {
		return err
	}

	if err := ensureSchemaMigrationsTable(db); err != nil {
		return err
	}

	version, err := latestAppliedMigration(db)
	if err != nil {
		return err
	}
	if version == "" {
		return fmt.Errorf("no applied migrations to roll back")
	}

	downName := strings.Replace(version, ".up.sql", ".down.sql", 1)
	downPath := filepath.Join(migrationsDir, downName)
	body, err := os.ReadFile(downPath)
	if err != nil {
		return fmt.Errorf("read migration %s: %w", downName, err)
	}

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin rollback %s: %w", downName, err)
	}

	if _, err := tx.Exec(string(body)); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("execute rollback %s: %w", downName, err)
	}
	if _, err := tx.Exec(`DELETE FROM schema_migrations WHERE version = $1`, version); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("untrack migration %s: %w", version, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit rollback %s: %w", downName, err)
	}

	return nil
}

func ResetMigrations(db *sql.DB) error {
	if _, err := db.Exec(`
		DROP TABLE IF EXISTS trades;
		DROP TABLE IF EXISTS signal_notifications;
		DROP TABLE IF EXISTS signals;
		DROP TABLE IF EXISTS strategy_runs;
		DROP TABLE IF EXISTS strategies;
		DROP TABLE IF EXISTS ingestion_jobs;
		DROP TABLE IF EXISTS candles;
		DROP TABLE IF EXISTS timeframes;
		DROP TABLE IF EXISTS symbols;
		DROP TABLE IF EXISTS schema_migrations;
	`); err != nil {
		return fmt.Errorf("reset database schema: %w", err)
	}

	return ApplyMigrations(db)
}

func ensureSchemaMigrationsTable(db *sql.DB) error {
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}
	return nil
}

func listUpMigrations(migrationsDir string) ([]string, error) {
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return nil, fmt.Errorf("read migrations: %w", err)
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".up.sql") {
			continue
		}
		names = append(names, entry.Name())
	}
	sort.Strings(names)
	return names, nil
}

func latestAppliedMigration(db *sql.DB) (string, error) {
	var version string
	err := db.QueryRow(`SELECT version FROM schema_migrations ORDER BY version DESC LIMIT 1`).Scan(&version)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", fmt.Errorf("latest applied migration: %w", err)
	}
	return version, nil
}

func migrationApplied(db *sql.DB, name string) (bool, error) {
	var exists bool
	if err := db.QueryRow(`SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)`, name).Scan(&exists); err != nil {
		return false, fmt.Errorf("check migration %s: %w", name, err)
	}
	return exists, nil
}

func resolveMigrationsDir() (string, error) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("resolve migrations dir: runtime caller unavailable")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", "..", "..", "migrations")), nil
}
