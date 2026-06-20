#!/usr/bin/env python3
"""Verify broonie #6 poller behavior by cross-referencing DB state with GitHub labels.

Issues with the 'invalid-frontmatter' label on GitHub should be blocked.
Valid AUTO issues with no deps should be pending.
Valid AUTO issues with unsatisfied deps should be blocked.
HITL issues should be blocked.

Usage:
  nix develop
  export GITHUB_TOKEN=ghp_...
  python verify.py --owner YOU --repo TEST_REPO [--db ../broonie.db]
"""

import argparse
import os
import sqlite3
import sys
from pathlib import Path

import requests


def get_invalid_issue_numbers(owner: str, repo: str, token: str) -> set[int]:
    """Fetch issue numbers that have the invalid-frontmatter label."""
    headers = {
        "Authorization": f"Bearer {token}",
        "Accept": "application/vnd.github+json",
    }
    url = f"https://api.github.com/repos/{owner}/{repo}/issues"
    params = {"labels": "invalid-frontmatter", "state": "open", "per_page": 100}
    r = requests.get(url, headers=headers, params=params)
    r.raise_for_status()
    return {issue["number"] for issue in r.json()}


def check(db_path: str, owner: str, repo: str, token: str) -> int:
    conn = sqlite3.connect(db_path)
    conn.row_factory = sqlite3.Row
    failures = 0

    def expect(cond, msg):
        nonlocal failures
        if not cond:
            print(f"  FAIL: {msg}")
            failures += 1
        else:
            print(f"  ok: {msg}")

    rows = conn.execute("SELECT COUNT(*) as n FROM issues").fetchone()
    expect(rows["n"] > 0, f"issues table has {rows['n']} rows (expected > 0)")
    if rows["n"] == 0:
        conn.close()
        return 1

    print("Fetching invalid-frontmatter labels from GitHub...")
    invalid_nums = get_invalid_issue_numbers(owner, repo, token)
    print(f"  {len(invalid_nums)} issue(s) with invalid-frontmatter label: {sorted(invalid_nums)}")

    issues = conn.execute(
        "SELECT issue_number, type, status, depends_on FROM issues ORDER BY issue_number"
    ).fetchall()

    for i in issues:
        num = i["issue_number"]
        typ = i["type"]
        status = i["status"]
        deps = i["depends_on"]

        if num in invalid_nums:
            expect(status == "blocked", f"#{num} (invalid FM) status=blocked (got {status})")
        elif typ == "HITL":
            expect(status == "blocked", f"#{num} (HITL) status=blocked (got {status})")
        elif typ == "AUTO" and deps in ("[]", "null"):
            expect(status in ("pending", "done"), f"#{num} (AUTO, no deps) status=pending|done (got {status})")
        elif typ == "AUTO" and deps not in ("[]", "null"):
            expect(status in ("blocked", "pending", "done"), f"#{num} (AUTO, has deps) status=blocked|pending|done (got {status})")
        else:
            expect(False, f"#{num} unexpected type/status: type={typ} status={status} deps={deps}")

    conn.close()
    return failures


def main():
    parser = argparse.ArgumentParser(description="Verify broonie #6 poller DB state")
    parser.add_argument("--owner", default=os.environ.get("GITHUB_OWNER"))
    parser.add_argument("--repo", default=os.environ.get("GITHUB_REPO"))
    parser.add_argument("--db", default=str(Path(__file__).resolve().parent.parent / "broonie.db"))
    args = parser.parse_args()

    token = os.environ.get("GITHUB_TOKEN")
    if not token:
        print("ERROR: GITHUB_TOKEN env var is required", file=sys.stderr)
        sys.exit(1)
    if not args.owner or not args.repo:
        print("ERROR: --owner and --repo are required (or set GITHUB_OWNER/GITHUB_REPO)", file=sys.stderr)
        sys.exit(1)

    failures = check(args.db, args.owner, args.repo, token)
    if failures:
        print(f"\n{failures} check(s) failed.")
        sys.exit(1)
    else:
        print("\nAll checks passed.")
        sys.exit(0)


if __name__ == "__main__":
    main()
