package main

import (
	"log"

	"github.com/OpalBolt/broonie/internal/config"
	"github.com/OpalBolt/broonie/internal/db"
	"github.com/OpalBolt/broonie/internal/loop"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Configuration error: %v\n", err)
	}

	database, err := db.Open(cfg.DBPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v\n", err)
	}
	defer database.Close()

	loop.Run(database, cfg)
}
