package main

import (
	"log"

	"github.com/drybin/fear-and-greed-online/internal/app/api"
	"github.com/drybin/fear-and-greed-online/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	if err := api.Run(cfg); err != nil {
		log.Fatal(err)
	}
}
