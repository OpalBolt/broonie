#!/usr/bin/env bash
# validate.sh — validate broonie issue frontmatter and body structure
# Usage: bash validate.sh <path-to-issue.md>
# Exit 0 on valid, exit 1 with message on invalid.

set -euo pipefail

FILE="${1:-}"
RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'

die() { echo -e "${RED}INVALID:${NC} $*" >&2; exit 1; }
ok()  { echo -e "${GREEN}OK:${NC} $*"; }

[ -n "$FILE" ] || die "Usage: validate.sh <path-to-issue.md>"
[ -f "$FILE" ] || die "File not found: $FILE"

# Extract frontmatter between first two --- markers
FM=$(awk 'BEGIN{c=0} /^---$/{c++; if(c==2) exit; next} c==1{print}' "$FILE")
[ -n "$FM" ] || die "No YAML frontmatter found (missing --- block at top of file)"

# Extract body (everything after second ---)
BODY=$(awk 'BEGIN{c=0} /^---$/{c++; next} c>=2{print}' "$FILE")

# --- Validate type ---
TYPE=$(echo "$FM" | { grep '^type:' || true; } | sed 's/^type:\s*//' | tr -d '[:space:]')
[ -n "$TYPE" ] || die "Missing 'type' field in frontmatter"
[[ "$TYPE" == "AUTO" || "$TYPE" == "HITL" ]] || die "type must be AUTO or HITL, got: $TYPE"
ok "type: $TYPE"

# --- Validate depends-on ---
DEPS_LINE=$(echo "$FM" | grep '^depends-on:' || true)
[ -n "$DEPS_LINE" ] || die "Missing 'depends-on' field in frontmatter"

DEPS=$(echo "$DEPS_LINE" | sed 's/^depends-on:\s*//')
# Must be a YAML list: [], or [#N, ...]
if [[ "$DEPS" == "[]" ]]; then
    ok "depends-on: [] (no dependencies)"
else
    # Check it looks like a YAML list of #N references
    if ! echo "$DEPS" | grep -qE '^\s*\[.*\]\s*$'; then
        die "depends-on must be a YAML list in brackets, got: $DEPS"
    fi
    # Strip brackets, split by comma, validate each entry
    INNER=$(echo "$DEPS" | sed 's/^\s*\[\s*//;s/\s*\]\s*$//')
    while IFS=',' read -ra ENTRIES; do
        for entry in "${ENTRIES[@]}"; do
            entry=$(echo "$entry" | tr -d '[:space:]"' | sed "s/^'//;s/'$//")
            if ! echo "$entry" | grep -qE '^#[0-9]+$'; then
                die "depends-on entry must be #N (e.g. #12), got: $entry"
            fi
        done
    done <<< "$INNER"
    ok "depends-on: $DEPS"
fi

# --- Validate body sections ---
echo "$BODY" | grep -q '## What to build' || die "Missing '## What to build' section in body"
ok "body: What to build section present"

echo "$BODY" | grep -q '## Acceptance criteria' || die "Missing '## Acceptance criteria' section in body"
ok "body: Acceptance criteria section present"

# ponytail: grep-based validation, replace with yq/pyyaml if edge cases appear
echo -e "\n${GREEN}VALID${NC} — issue frontmatter and body structure pass all checks."
