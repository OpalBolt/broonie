package gh

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	"github.com/OpalBolt/broonie/internal/db"
	"github.com/google/go-github/v62/github"
)

const invalidLabel = "invalid-frontmatter"

// Poll fetches open issues for a repo, parses frontmatter, updates the DB,
// and labels malformed issues. Returns the first pending AUTO issue, or nil.
func Poll(ctx context.Context, client *github.Client, repo db.Repo, dbase *sql.DB) (*db.Issue, error) {
	var allIssues []*github.Issue
	opts := &github.IssueListByRepoOptions{State: "open", Sort: "created", Direction: "asc"}
	for page := 1; page > 0; {
		issues, resp, err := client.Issues.ListByRepo(ctx, repo.Owner, repo.Name, opts)
		if err != nil {
			return nil, fmt.Errorf("list issues for %s/%s: %w", repo.Owner, repo.Name, err)
		}
		allIssues = append(allIssues, issues...)
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	// ponytail: skip pull requests (they show in issues list)
	for _, issue := range allIssues {
		if issue.IsPullRequest() {
			continue
		}

		issueType, dependsOn, fmErr := ParseFrontmatter(issue.GetBody())

		var status string
		if fmErr != nil {
			if !hasLabel(issue.Labels, invalidLabel) {
				log.Printf("frontmatter invalid for %s/%s#%d: %v — labeling", repo.Owner, repo.Name, issue.GetNumber(), fmErr)
				if err := applyLabel(ctx, client, repo, issue.GetNumber(), invalidLabel); err != nil {
					log.Printf("failed to label %s/%s#%d: %v", repo.Owner, repo.Name, issue.GetNumber(), err)
				}
			}
			issueType = "AUTO" // store what we can, status blocked
			status = "blocked"
			dependsOn = nil
		} else if issueType == "HITL" {
			status = "blocked" // HITL: human action needed, skip
		} else {
			// AUTO — resolve dependencies
			depsDone, err := db.AreDepsDone(dbase, repo.ID, dependsOn)
			if err != nil {
				return nil, fmt.Errorf("resolve deps for %s/%s#%d: %w", repo.Owner, repo.Name, issue.GetNumber(), err)
			}
			if depsDone {
				status = "pending"
			} else {
				status = "blocked"
			}
		}

		if dependsOn == nil {
			dependsOn = []int{}
		}
		depJSON, _ := json.Marshal(dependsOn)
		dbIssue := db.Issue{
			RepoID:      repo.ID,
			IssueNumber: issue.GetNumber(),
			Title:       issue.GetTitle(),
			Type:        issueType,
			Status:      status,
			DependsOn:   string(depJSON),
		}
		if err := db.UpsertIssue(dbase, dbIssue); err != nil {
			return nil, fmt.Errorf("upsert issue %s/%s#%d: %w", repo.Owner, repo.Name, issue.GetNumber(), err)
		}
	}

	return db.GetPendingIssue(dbase, repo.ID)
}

func applyLabel(ctx context.Context, client *github.Client, repo db.Repo, issueNum int, label string) error {
	_, _, err := client.Issues.AddLabelsToIssue(ctx, repo.Owner, repo.Name, issueNum, []string{label})
	return err
}

func hasLabel(labels []*github.Label, name string) bool {
	for _, l := range labels {
		if l.GetName() == name {
			return true
		}
	}
	return false
}
