---
name: create-issue
description: >
  Create well-formed GitHub issues for the broonie autonomous pipeline. Produces issues
  with valid YAML frontmatter (type, depends-on) and structured bodies that downstream
  Blip can consume. Use when the user wants to create a broonie issue, author a task for
  the autonomous agent, or convert a rough idea into a pipeline-ready issue. Also trigger
  on phrases like "make an issue for...", "create a task for broonie...", "write this up
  as an issue...", or any time an idea needs to become a trackable, auto-implementable
  issue.
---

# Create Issue (broonie)

Produce a GitHub issue ready for the broonie autonomous pipeline — valid YAML frontmatter,
well-structured body, right-sized scope.

## Workflow

### 1. Detect the repo

Derive `owner/repo` from `git remote get-url origin`. If ambiguous, ask once.

### 2. Understand the rough idea

Read what the user gave you. Identify gaps silently: motivation, scope, dependencies,
what "done" looks like.

### 3. Iterative questioning

Ask 2–3 focused questions per round until you have: clear motivation, defined scope,
verifiable acceptance criteria, and dependencies for `depends-on`.

### 4. Draft the issue

Produce Markdown with YAML frontmatter and structured body:

```markdown
---
type: AUTO
depends-on: ["#12", "#15"]
---

## What to build
[High-level approach, end-to-end behavior — not layer-by-layer instructions.]

## Acceptance criteria
- [ ] Specific, verifiable condition

## Implementation notes
[Key decisions or constraints. Omit if nothing meaningful.]

## Blocked by
#12, #15 — or "None — can start immediately."
```

**Frontmatter rules** (server-side validator will reject violations):
- `type` (required): `AUTO` (pipeline picks it up) or `HITL` (pipeline skips until human acts)
- `depends-on` (required): YAML list of `#N` strings, e.g. `["#12", "#15"]` or `[]` for none

**Splitting rules**: vertical slices only. Each issue is a narrow but complete path through
all layers, independently verifiable. Prefer many thin issues over few thick ones.

### 5. Validate

Write the issue to a temp file, then run the validator:

```bash
cat > /tmp/broonie-issue-<slug>.md << 'ISSUE_EOF'
[full issue markdown]
ISSUE_EOF

bash skills/create-issue/scripts/validate.sh /tmp/broonie-issue-<slug>.md
```

The validator checks: frontmatter parses, `type` valid, `depends-on` well-formed,
required body sections present. Fix and re-validate until clean.

### 6. Create

```bash
gh issue create \
  --repo <owner>/<repo> \
  --title "<title>" \
  --body-file /tmp/broonie-issue-<slug>.md
```

Share the issue URL. Clean up the temp file.
