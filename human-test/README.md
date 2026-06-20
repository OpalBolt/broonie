# Test helpers for broonie #6

## Setup

```bash
cd helper/test
nix develop

# Encrypt token + insert test repo
export SECRET_KEY=test-key
export GITHUB_TOKEN=ghp_...
python setup.py --owner YOUR_USER --repo YOUR_TEST_REPO

# Optional: create test issues via GitHub API
python setup.py --owner YOUR_USER --repo YOUR_TEST_REPO --create-issues
```

## Run broonie

```bash
cd ../..  # back to repo root
SECRET_KEY=test-key ./broonie
# Ctrl+C after first poll cycle
```

## Verify

```bash
cd helper/test
nix develop
python verify.py
```

## What gets tested

| Check | Expected |
|-------|----------|
| AUTO, no deps | status = pending |
| AUTO, blocked deps | status = blocked |
| HITL | status = blocked |
| Malformed frontmatter | status = blocked |
| Missing frontmatter | status = blocked |
