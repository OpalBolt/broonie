# broonie

A self-hosted server that watches your GitHub repos and autonomously implements issues overnight. You jam a feature into well-structured issues, go to bed, and wake up to pull requests.

## The name

A *brownie* is a creature from Scottish and English folklore — a household spirit that does your chores while you sleep. Leave out a bowl of milk, wake up to a clean house. broonie is that, for your issue tracker.

## How it works

- **Go app** owns the boring, deterministic lifecycle: polling, scheduling, git worktrees, branches, PRs, the web GUI.
- **AI agent** (running inside [pi](https://github.com/always-further/pi-coding-agent), inside [nono](https://github.com/always-further/nono) sandboxes) owns only the coding: implements the issue, verifies its work, writes the PR description.

The two halves talk through one channel — a `.loop/` directory at the worktree root. The agent drops a `pr.md` and a `done` file when it's finished; the Go app watches for those files, then pushes and opens the PR.

## Status

Early. See [design.md](design.md) for the full picture.

## Stack

- Go (`net/http`, `html/template`, HTMX, Pico CSS)
- SQLite for repo config and queue state
- nono via the `nono-go` SDK for sandboxing
- pi for agent harnessing
- Single Docker container
