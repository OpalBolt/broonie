package gh

import (
	"context"
	"fmt"
	"strings"

	"github.com/OpalBolt/broonie/internal/db"
	"github.com/google/go-github/v62/github"
)

// CreatePR creates a pull request on GitHub and returns the PR URL.
func CreatePR(ctx context.Context, client *github.Client, repo db.Repo, title, body, head, base string) (string, error) {
	pr := &github.NewPullRequest{
		Title: &title,
		Body:  &body,
		Head:  &head,
		Base:  &base,
	}
	created, _, err := client.PullRequests.Create(ctx, repo.Owner, repo.Name, pr)
	if err != nil {
		return "", fmt.Errorf("create PR for %s/%s: %w", repo.Owner, repo.Name, err)
	}
	return created.GetHTMLURL(), nil
}

// LabelIssue adds a label to a GitHub issue.
func LabelIssue(ctx context.Context, client *github.Client, repo db.Repo, issueNum int, label string) error {
	_, _, err := client.Issues.AddLabelsToIssue(ctx, repo.Owner, repo.Name, issueNum, []string{label})
	if err != nil {
		return fmt.Errorf("label %s/%s#%d with %q: %w", repo.Owner, repo.Name, issueNum, label, err)
	}
	return nil
}

// ExtractPRBody reads .loop/pr.md format: first line is title, blank separator, then body.
// Returns error if title is empty (GitHub rejects empty PR titles).
func ExtractPRBody(content string) (title, body string, err error) {
	lines := strings.SplitN(content, "\n", 3)
	if len(lines) >= 1 {
		title = strings.TrimSpace(lines[0])
	}
	if len(lines) >= 3 {
		body = strings.TrimSpace(lines[2])
	} else if len(lines) >= 2 {
		body = strings.TrimSpace(lines[1])
	}
	if title == "" {
		return "", "", fmt.Errorf("pr.md title cannot be empty")
	}
	return title, body, nil
}
