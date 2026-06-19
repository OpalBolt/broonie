package loop

import (
	"context"
	"database/sql"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/OpalBolt/broonie/internal/config"
)

// Run starts the main polling loop.
// It queries repos from the database and polls them on the configured interval.
// Gracefully shuts down on SIGINT/SIGTERM.
func Run(db *sql.DB, cfg config.Config) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Shutting down...")
		cancel()
	}()

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Query number of active repos
		var repoCount int
		err := db.QueryRow("SELECT COUNT(*) FROM repos WHERE active = 1").Scan(&repoCount)
		if err != nil {
			log.Printf("Error counting repos: %v\n", err)
			repoCount = 0
		}

		log.Printf("Watching %d repos\n", repoCount)

		// Sleep for the configured interval
		select {
		case <-ctx.Done():
			return
		case <-time.After(cfg.PollInterval):
		}
	}
}
