#!/usr/bin/env python3
"""Set up a test repo in broonie.db for manual testing of #6.

Encrypts a GitHub token with the same AES-GCM format the Go app uses,
inserts a repo row, and optionally creates test issues on GitHub.

Usage:
  nix develop
  export SECRET_KEY=test-key
  export GITHUB_TOKEN=ghp_...
  python setup.py [--owner octocat] [--repo hello-world] [--create-issues] [--db ../broonie.db]
"""

import argparse
import hashlib
import os
import sqlite3
import sys
from pathlib import Path

from cryptography.hazmat.primitives.ciphers.aead import AESGCM


def encrypt_token(plaintext: bytes, secret_key: str) -> bytes:
    """Encrypt a token using the same AES-GCM format as broonie's crypto.Encrypt.

    Key derivation: SHA-256(SECRET_KEY) -> 32 bytes
    Format: 12-byte random nonce || AES-GCM ciphertext+tag
    """
    key = hashlib.sha256(secret_key.encode()).digest()
    nonce = os.urandom(12)
    aesgcm = AESGCM(key)
    ct = aesgcm.encrypt(nonce, plaintext, None)
    return nonce + ct


def insert_repo(db_path: str, owner: str, name: str, token_enc: bytes):
    """Insert or replace a repo row with the encrypted token."""
    conn = sqlite3.connect(db_path)
    conn.execute("""
        INSERT INTO repos (owner, name, token_enc, active)
        VALUES (?, ?, ?, 1)
        ON CONFLICT(owner, name) DO UPDATE SET token_enc = excluded.token_enc, active = 1
    """, (owner, name, token_enc))
    conn.commit()
    conn.close()


def create_test_issues(owner: str, repo: str, token: str):
    """Create 6 test issues on GitHub covering all frontmatter cases."""
    import requests

    headers = {
        "Authorization": f"Bearer {token}",
        "Accept": "application/vnd.github+json",
    }
    url = f"https://api.github.com/repos/{owner}/{repo}/issues"

    issues = [
        ("AUTO, no deps", '<details>\n<summary>Metadata</summary>\n\n```json\n{"type": "AUTO"}\n```\n\n</details>\n\nFix the login redirect.'),
        ("AUTO, depends on #1", '<details>\n<summary>Metadata</summary>\n\n```json\n{"type": "AUTO", "depends-on": ["#1"]}\n```\n\n</details>\n\nAdd session persistence.'),
        ("HITL", '<details>\n<summary>Metadata</summary>\n\n```json\n{"type": "HITL"}\n```\n\n</details>\n\nReview the database schema.'),
        ("malformed: bad type", '<details>\n<summary>Metadata</summary>\n\n```json\n{"type": "BANANA"}\n```\n\n</details>\n\nThis should get labeled.'),
        ("malformed: deps missing #", '<details>\n<summary>Metadata</summary>\n\n```json\n{"type": "AUTO", "depends-on": ["1"]}\n```\n\n</details>\n\nBad syntax.'),
        ("missing metadata block", "Just a normal issue, no metadata at all."),
    ]

    for title, body in issues:
        r = requests.post(url, headers=headers, json={"title": title, "body": body})
        if r.status_code == 201:
            print(f"  created: #{r.json()['number']} — {title}")
        else:
            print(f"  FAILED ({r.status_code}): {title} — {r.json().get('message', '')}")


def main():
    parser = argparse.ArgumentParser(description="Set up test repo for broonie #6 testing")
    parser.add_argument("--owner", default=os.environ.get("GITHUB_OWNER", ""))
    parser.add_argument("--repo", default=os.environ.get("GITHUB_REPO", ""))
    parser.add_argument("--create-issues", action="store_true")
    parser.add_argument("--db", default=str(Path(__file__).resolve().parent.parent / "broonie.db"))
    args = parser.parse_args()

    secret_key = os.environ.get("SECRET_KEY")
    github_token = os.environ.get("GITHUB_TOKEN")

    if not secret_key:
        print("ERROR: SECRET_KEY env var is required", file=sys.stderr)
        sys.exit(1)
    if not github_token:
        print("ERROR: GITHUB_TOKEN env var is required", file=sys.stderr)
        sys.exit(1)
    if not args.owner or not args.repo:
        print("ERROR: --owner and --repo are required (or set GITHUB_OWNER/GITHUB_REPO)", file=sys.stderr)
        sys.exit(1)
    print(f"Encrypting token with SECRET_KEY={secret_key}")
    token_enc = encrypt_token(github_token.encode(), secret_key)

    db_path = args.db
    print(f"Inserting repo {args.owner}/{args.repo} into {db_path}")
    insert_repo(db_path, args.owner, args.repo, token_enc)

    print("Done. Repo ready for polling.")

    if args.create_issues:
        print(f"Creating test issues in {args.owner}/{args.repo}...")
        create_test_issues(args.owner, args.repo, github_token)
        print("Issues created.")


if __name__ == "__main__":
    main()
