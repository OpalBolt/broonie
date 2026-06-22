#!/usr/bin/env python3
"""Verify broonie #10 provision package by running Go tests + integration smoke test.

Usage:
  nix develop
  python human-test/provision_verify.py
"""
import json
import os
import subprocess
import sys
import tempfile
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parent.parent
FAILURES = 0


def run(cmd: list[str], cwd=None, env=None) -> subprocess.CompletedProcess:
    return subprocess.run(cmd, capture_output=True, text=True, cwd=cwd, env=env)


def check(label: str, ok: bool, detail: str = ""):
    global FAILURES
    if ok:
        print(f"  ok: {label}")
    else:
        print(f"  FAIL: {label}" + (f" — {detail}" if detail else ""))
        FAILURES += 1


def run_unit_tests() -> bool:
    env = os.environ.copy()
    env["GOCACHE"] = tempfile.mkdtemp()
    env["GOPATH"] = tempfile.mkdtemp()
    result = run(
        ["go", "test", "./internal/provision/", "-v", "-count=1"],
        cwd=REPO_ROOT,
        env=env,
    )
    check("go test ./internal/provision/", result.returncode == 0,
          result.stderr.split("\n")[-2] if result.stderr else "unknown error")
    if result.stdout:
        for line in result.stdout.strip().split("\n"):
            if line.startswith("---"):
                print(f"      {line}")
    return result.returncode == 0


def integration_smoke() -> bool:
    env = os.environ.copy()
    env["GOCACHE"] = tempfile.mkdtemp()
    env["GOPATH"] = tempfile.mkdtemp()

    smoke_file = REPO_ROOT / "human-test" / "provision_smoke_main.go"
    smoke_src = '''//go:build ignore

package main

import (
\t"encoding/json"
\t"fmt"
\t"os"
\t"path/filepath"
\t"strings"

\t"github.com/OpalBolt/broonie/internal/provision"
)

func main() {
\tworktree, _ := os.MkdirTemp("", "broonie-smoke-worktree-*")
\tdefer os.RemoveAll(worktree)

\tsourcePi := ".pi"
\tmodels := map[string]string{}
\tmodelsPath := filepath.Join(sourcePi, "models.json")
\traw, _ := os.ReadFile(modelsPath)
\tjson.Unmarshal(raw, &models)

\terr := provision.Provision(worktree, sourcePi, models)
\tif err != nil {
\t\tfmt.Printf("FAIL: Provision error: %v\\n", err)
\t\tos.Exit(1)
\t}

\tsrcBlip, _ := os.ReadFile(filepath.Join(sourcePi, "prompts", "blip.md"))
\tdstBlip, _ := os.ReadFile(filepath.Join(worktree, ".pi", "prompts", "blip.md"))
\tif string(srcBlip) != string(dstBlip) {
\t\tfmt.Println("FAIL: blip.md content mismatch")
\t\tos.Exit(1)
\t}
\tfmt.Println("ok: blip.md copied verbatim")

\tentries, _ := os.ReadDir(filepath.Join(worktree, ".pi", "agents"))
\tcount := 0
\tfor _, e := range entries {
\t\tif strings.HasSuffix(e.Name(), ".agent.md") {
\t\t\tcount++
\t\t\tdata, _ := os.ReadFile(filepath.Join(worktree, ".pi", "agents", e.Name()))
\t\t\tif strings.Contains(string(data), "model: MODEL_") {
\t\t\t\tfmt.Printf("FAIL: MODEL_ placeholder remains in %s\\n", e.Name())
\t\t\t\tos.Exit(1)
\t\t\t}
\t\t}
\t}
\tif count != 7 {
\t\tfmt.Printf("FAIL: expected 7 agent files, got %d\\n", count)
\t\tos.Exit(1)
\t}
\tfmt.Printf("ok: %d agents patched, no MODEL_ placeholders\\n", count)

\tfmt.Println("PASS")
}
'''
    smoke_file.write_text(smoke_src)

    result = run(
        ["go", "run", "human-test/provision_smoke_main.go"],
        cwd=REPO_ROOT,
        env=env,
    )
    smoke_file.unlink()

    ok = result.returncode == 0
    check("integration smoke test (real .pi/ tree)", ok,
          result.stderr.strip().split("\n")[-1] if result.stderr else "")
    if result.stdout:
        for line in result.stdout.strip().split("\n"):
            print(f"      {line}")
    return ok


def main():
    print("=== Unit tests ===")
    ut_ok = run_unit_tests()

    print("\n=== Integration smoke ===")
    is_ok = integration_smoke()

    print()
    if FAILURES:
        print(f"{FAILURES} check(s) failed.")
        sys.exit(1)
    else:
        print("All checks passed.")
        sys.exit(0)


if __name__ == "__main__":
    main()
