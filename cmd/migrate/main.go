package main

import (
	"log"
	"os"

	"github.com/drybin/fear-and-greed-online/internal/config"
	"github.com/drybin/fear-and-greed-online/internal/storage/postgres"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	command := "up"
	if len(os.Args) > 1 {
		command = os.Args[1]
	}

	db, err := postgres.Open(cfg.Postgres)
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	switch command {
	case "up":
		if err := postgres.ApplyMigrations(db); err != nil {
			log.Fatal(err)
		}
		log.Println("migrations applied")
	case "down":
		if err := postgres.RollbackLastMigration(db); err != nil {
			log.Fatal(err)
		}
		log.Println("migration rolled back")
	case "reset":
		if err := postgres.ResetMigrations(db); err != nil {
			log.Fatal(err)
		}
		log.Println("database reset and migrations applied")
	default:
		log.Fatalf("unsupported migrate command: %s (supported: up, down, reset)", command)
	}
}
