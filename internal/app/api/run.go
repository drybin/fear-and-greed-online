package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/drybin/fear-and-greed-online/internal/config"
	"github.com/drybin/fear-and-greed-online/internal/storage/postgres"
)

const shutdownTimeout = 5 * time.Second

func Run(cfg *config.Config) error {
	db, err := postgres.Open(cfg.Postgres)
	if err != nil {
		return fmt.Errorf("api database connection failed: %w", err)
	}
	defer func() { _ = db.Close() }()

	if err := postgres.ApplyMigrations(db); err != nil {
		return fmt.Errorf("api migration check failed: %w", err)
	}

	log.Printf("postgres connection is ready")

	mux := http.NewServeMux()
	registerRoutes(mux, db)

	addr := fmt.Sprintf("%s:%s", cfg.AppHost, cfg.AppPort)
	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	log.Printf("api listening on http://%s", addr)
	log.Printf("environment=%s", cfg.AppEnv)

	err = server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("listen and serve: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	return server.Shutdown(ctx)
}
