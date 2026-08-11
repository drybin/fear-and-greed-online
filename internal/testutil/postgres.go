package testutil

import (
	"database/sql"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"testing"
	"time"

	_ "github.com/lib/pq"

	"github.com/drybin/fear-and-greed-online/internal/config"
	"github.com/drybin/fear-and-greed-online/internal/storage/postgres"
)

func CreateTestDatabase(t *testing.T) *sql.DB {
	t.Helper()

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	adminCfg := cfg.Postgres
	adminCfg.Database = "postgres"
	adminDB, err := postgres.Open(adminCfg)
	if err != nil {
		t.Fatalf("open admin postgres connection: %v", err)
	}
	t.Cleanup(func() { _ = adminDB.Close() })

	dbName := fmt.Sprintf("fg_online_test_%d_%d", time.Now().UnixNano(), rand.Intn(1000))
	if _, err := adminDB.Exec(`CREATE DATABASE ` + pqQuoteIdentifier(dbName)); err != nil {
		t.Fatalf("create test database: %v", err)
	}

	testCfg := cfg.Postgres
	testCfg.Database = dbName
	testDB, err := postgres.Open(testCfg)
	if err != nil {
		t.Fatalf("open test postgres connection: %v", err)
	}

	if err := postgres.ApplyMigrations(testDB); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}

	t.Cleanup(func() {
		_ = testDB.Close()
		if _, err := adminDB.Exec(`DROP DATABASE IF EXISTS ` + pqQuoteIdentifier(dbName) + ` WITH (FORCE)`); err != nil {
			t.Logf("drop test database %s: %v", dbName, err)
		}
	})

	return testDB
}

func pqQuoteIdentifier(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

func init() {
	if os.Getenv("POSTGRES_PORT") == "" {
		_ = os.Setenv("POSTGRES_PORT", "5433")
	}
}
