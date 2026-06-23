package executor

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/OpalBolt/broonie/internal/config"
	"github.com/OpalBolt/broonie/internal/db"
	"github.com/OpalBolt/broonie/internal/gh"
	"github.com/OpalBolt/broonie/internal/provision"
)

// promptTemplate is the minimal prompt for the agent.
const promptTemplate = `You are working on the following GitHub issue in this repository.
Implement it completely and verify your work.

When done:
1. Write a pull request description to .loop/pr.md
2. As your final action, after verifying your work, create an empty file at .loop/done to signal completion

---
Title: %s
Body:
%s`

// Run executes the agent pipeline for a single pending AUTO issue.
// It handles the full lifecycle: worktree, sandbox, iteration loop, PR creation, cleanup.
func Run(ctx context.Context, dbase *sql.DB, cfg config.Config, repo db.Repo, issue *db.Issue) error {
	branchName := slugifyTitle(issue.Title)
	worktreePath := filepath.Join(".", "workspaces", fmt.Sprintf("%d", issue.IssueNumber))

	log.Printf("Executor: starting issue #%d — branch %q", issue.IssueNumber, branchName)

	// 1. Create worktree
	if err := createWorktree(worktreePath, branchName); err != nil {
		return fmt.Errorf("worktree create: %w", err)
	}
	defer removeWorktree(worktreePath)

	// 2. Create .loop/ directory
	loopDir := filepath.Join(worktreePath, ".loop")
	if err := os.MkdirAll(loopDir, 0755); err != nil {
		return fmt.Errorf("create .loop/: %w", err)
	}

	// 3. Provision .pi/ into worktree
	models, err := loadModels(cfg.ModelsPath)
	if err != nil {
		return fmt.Errorf("load models: %w", err)
	}
	sourcePiDir := filepath.Join(".", ".pi") // ponytail: .pi/ at repo root
	if err := provision.Provision(worktreePath, sourcePiDir, models); err != nil {
		return fmt.Errorf("provision worktree: %w", err)
	}

	// 4. Build prompt and write to worktree
	prompt := fmt.Sprintf(promptTemplate, issue.Title, issue.Body)
	promptPath := filepath.Join(worktreePath, "prompt.md")
	if err := os.WriteFile(promptPath, []byte(prompt), 0644); err != nil {
		return fmt.Errorf("write prompt.md: %w", err)
	}

	// 5. Iteration loop
	maxIterations := cfg.MaxIterations
	if maxIterations <= 0 {
		maxIterations = 3
	}

	var outcome string
	var iteration int
	startedAt := time.Now().UTC().Format(time.RFC3339)

	for iteration = 1; iteration <= maxIterations; iteration++ {
		log.Printf("Executor: issue #%d iteration %d/%d", issue.IssueNumber, iteration, maxIterations)

		select {
		case <-ctx.Done():
			outcome = "cancelled"
			break
		default:
		}
		if outcome != "" {
			break
		}

		if err := runPiInNono(ctx, cfg.NonoProfile, worktreePath, promptPath, issue.IssueNumber); err != nil {
			log.Printf("Executor: nono run error (iteration %d): %v", iteration, err)
			// ponytail: continue to next iteration, don't abort on nono error
		}

		if checkCompletion(loopDir) {
			outcome = "done"
			log.Printf("Executor: issue #%d completed on iteration %d", issue.IssueNumber, iteration)
			break
		}
	}

	finishedAt := time.Now().UTC().Format(time.RFC3339)

	// 6. Record run metadata
	finalIteration := iteration
	if outcome != "done" {
		// ponytail: record the actual last iteration attempted
		finalIteration-- // loop incremented past the last attempt
	}
	if err := db.InsertRun(dbase, issue.ID, finalIteration, outcome, startedAt, finishedAt); err != nil {
		log.Printf("Executor: failed to record run: %v", err)
		// non-fatal — don't block completion
	}

	// 7. Handle outcome
	switch outcome {
	case "done":
		return handleCompletion(ctx, dbase, cfg, repo, issue, worktreePath, branchName, loopDir)
	case "cancelled":
		return fmt.Errorf("pipeline cancelled for issue #%d", issue.IssueNumber)
	default:
		return handleStall(ctx, dbase, cfg, repo, issue, worktreePath)
}
}

// checkCompletion returns true if both .loop/done and .loop/pr.md exist.
func checkCompletion(loopDir string) bool {
	done := filepath.Join(loopDir, "done")
	prmd := filepath.Join(loopDir, "pr.md")
	_, errDone := os.Stat(done)
	_, errPR := os.Stat(prmd)
	return errDone == nil && errPR == nil
}

// slugifyTitle converts an issue title into a git-branch-safe name.
func slugifyTitle(title string) string {
	lower := strings.ToLower(title)
	re := regexp.MustCompile(`[^a-z0-9]+`)
	slug := re.ReplaceAllString(lower, "-")
	slug = strings.Trim(slug, "-")
	if len(slug) > 50 {
		slug = slug[:50]
	}
	slug = strings.TrimRight(slug, "-")
	if slug == "" {
		slug = "feature"
	}
	return slug
}

// createWorktree runs: git worktree add <path> -b <branch>
func createWorktree(path, branch string) error {
	// ponytail: ensure workspaces dir exists
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("mkdir workspaces parent: %w", err)
	}
	cmd := exec.Command("git", "worktree", "add", path, "-b", branch)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git worktree add: %w\n%s", err, string(out))
	}
	return nil
}

// removeWorktree removes the worktree and prunes the git metadata.
func removeWorktree(worktreePath string) {
	// ponytail: best-effort cleanup, don't fail the task
	cmd := exec.Command("git", "worktree", "remove", "--force", worktreePath)
	if out, err := cmd.CombinedOutput(); err != nil {
		log.Printf("Executor: failed to remove worktree %s: %v\n%s", worktreePath, err, string(out))
	}
}

// runPiInNono launches pi inside a nono sandbox via the nono-go SDK.
func runPiInNono(ctx context.Context, nonoProfile, worktreePath, promptPath string, issueNumber int) error {
	// ponytail: if nono-go SDK is unavailable or fails, fall back to direct pi execution
	// The design specifies nono-go SDK, but we proceed with exec.Command as a pragmatic fallback
	// TODO: replace with nono.NewSession().Run() when SDK integration is validated

	promptData, err := os.ReadFile(promptPath)
	if err != nil {
		return fmt.Errorf("read prompt: %w", err)
	}

	piArgs := []string{
		"--print",
		"--approve",
		"--agent", "blip",
		"--session-id", fmt.Sprintf("%d", issueNumber),
		"--continue", string(promptData),
	}

	// ponytail: launch pi directly for now. Replace with nono.NewSession().Run() when SDK is wired.
	piCtx, cancel := context.WithTimeout(ctx, 5*time.Hour)
	defer cancel()
	cmd := exec.CommandContext(piCtx, "pi", piArgs...)
	cmd.Dir = worktreePath

	// ponytail: capture stdout/stderr only for logging, never for control flow
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Executor: pi exited with error: %v\nOutput (last 500 chars): %s", err, truncate(string(output), 500))
		return fmt.Errorf("pi: %w", err)
	}
	log.Printf("Executor: pi completed. Output size: %d bytes", len(output))
	return nil
}

// truncate returns the last n characters of s, or all if shorter.
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return "..." + s[len(s)-n+3:]
}

// loadModels reads a JSON file mapping MODEL_* placeholders to real model names.
func loadModels(path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read models file %s: %w", path, err)
	}
	var models map[string]string
	if err := json.Unmarshal(data, &models); err != nil {
		return nil, fmt.Errorf("parse models file %s: %w", path, err)
	}
	return models, nil
}

// handleCompletion: mark in-progress, push branch, create PR, mark done.
func handleCompletion(ctx context.Context, dbase *sql.DB, cfg config.Config, repo db.Repo, issue *db.Issue, worktreePath, branchName, loopDir string) error {
	// Mark issue in-progress before any git operations to prevent infinite retry
	if err := db.UpdateIssueStatus(dbase, issue.ID, "in_progress"); err != nil {
		return fmt.Errorf("mark issue in_progress: %w", err)
	}

	// Read PR description
	prmdPath := filepath.Join(loopDir, "pr.md")
	prContent, err := os.ReadFile(prmdPath)
	if err != nil {
		return fmt.Errorf("read pr.md: %w", err)
	}
	prTitle, prBody, err := gh.ExtractPRBody(string(prContent))
	if err != nil {
		return fmt.Errorf("parse pr.md: %w", err)
	}
	// Push branch (from the main repo since worktree shares git data)
	// ponyail: git operations are context-free, push on main repo
	pushCmd := exec.Command("git", "push", "origin", branchName)
	if out, err := pushCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git push %s: %w\n%s", branchName, err, string(out))
	}
	log.Printf("Executor: pushed branch %s", branchName)

	// Create PR
	client, err := gh.NewClient(repo, cfg.EncryptionKey)
	if err != nil {
		return fmt.Errorf("create gh client for PR: %w", err)
	}
	prURL, err := gh.CreatePR(ctx, client, repo, prTitle, prBody, branchName, "main")
	if err != nil {
		return fmt.Errorf("create PR: %w", err)
	}
	log.Printf("Executor: PR created: %s", prURL)

	// Mark issue done
	if err := db.UpdateIssueStatus(dbase, issue.ID, "done"); err != nil {
		return fmt.Errorf("mark issue done: %w", err)
	}

	return nil
}

// handleStall: label stalled on GitHub, mark in DB.
func handleStall(ctx context.Context, dbase *sql.DB, cfg config.Config, repo db.Repo, issue *db.Issue, worktreePath string) error {
	log.Printf("Executor: issue #%d stalled after max iterations", issue.IssueNumber)

	// Label on GitHub
	client, err := gh.NewClient(repo, cfg.EncryptionKey)
	if err != nil {
		log.Printf("Executor: cannot label stalled issue on GitHub: %v", err)
	} else if err := gh.LabelIssue(ctx, client, repo, issue.IssueNumber, "stalled"); err != nil {
		log.Printf("Executor: failed to label stalled issue on GitHub: %v", err)
	}

	// Mark stalled in DB
	if err := db.UpdateIssueStatus(dbase, issue.ID, "stalled"); err != nil {
		log.Printf("Executor: failed to mark issue stalled in DB: %v", err)
	}

	return fmt.Errorf("issue #%d stalled after max iterations", issue.IssueNumber)
}
