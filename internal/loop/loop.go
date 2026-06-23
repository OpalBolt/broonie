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
	"github.com/OpalBolt/broonie/internal/db"
	"github.com/OpalBolt/broonie/internal/executor"
	"github.com/OpalBolt/broonie/internal/gh"
)

// Run starts the main polling loop.
// It queries repos from the database and polls them on the configured interval.
// Gracefully shuts down on SIGINT/SIGTERM.
func Run(database *sql.DB, cfg config.Config) {
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

		pollRepos(ctx, database, cfg)

		// Sleep for the configured interval
		select {
		case <-ctx.Done():
			return
		case <-time.After(cfg.PollInterval):
		}
	}
}

// pollRepos iterates active repos, polls GitHub, and logs the next pending issue.
func pollRepos(ctx context.Context, database *sql.DB, cfg config.Config) {
	repos, err := db.ListActiveRepos(database)
	if err != nil {
		log.Printf("Error listing repos: %v", err)
		return
	}

	log.Printf("Watching %d repos", len(repos))

	for _, repo := range repos {
		select {
		case <-ctx.Done():
			return
		default:
		}

		client, err := gh.NewClient(repo, cfg.EncryptionKey)
		if err != nil {
			log.Printf("Skipping %s/%s: %v", repo.Owner, repo.Name, err)
			continue
		}

		issue, err := gh.Poll(ctx, client, repo, database)
		if err != nil {
			log.Printf("Poll error for %s/%s: %v", repo.Owner, repo.Name, err)
			continue
		}

		if issue != nil {
			log.Printf("Selected %s/%s#%d: %s", repo.Owner, repo.Name, issue.IssueNumber, issue.Title)
			// ponytail: serial execution — process one issue per poll cycle
			if err := executor.Run(ctx, database, cfg, repo, issue); err != nil {
				log.Printf("Executor error for %s/%s#%d: %v", repo.Owner, repo.Name, issue.IssueNumber, err)
			}
			// ponytail: after executing one issue, return to enforce serial execution;
			// next poll cycle will pick up next pending issue
			return
		}
	}
}
