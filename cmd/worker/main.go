package main

import (
	"log"
	"os"

	"github.com/drybin/fear-and-greed-online/internal/app/worker"
	"github.com/drybin/fear-and-greed-online/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	if err := worker.Run(cfg, os.Args[1:]); err != nil {
		log.Fatal(err)
	}
}
