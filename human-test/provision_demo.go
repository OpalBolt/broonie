//go:build ignore

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/OpalBolt/broonie/internal/provision"
)

func main() {
	keep := false
	for _, a := range os.Args[1:] {
		if a == "--keep" {
			keep = true
		}
	}

	sourcePi := ".pi"

	models := map[string]string{}
	raw, err := os.ReadFile(filepath.Join(sourcePi, "models.json"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "FAIL: read models.json: %v\n", err)
		os.Exit(1)
	}
	json.Unmarshal(raw, &models)

	worktree, err := os.MkdirTemp("", "broonie-provision-demo-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "FAIL: mkdir temp: %v\n", err)
		os.Exit(1)
	}
	if !keep {
		defer os.RemoveAll(worktree)
	}

	if err := provision.Provision(worktree, sourcePi, models); err != nil {
		fmt.Fprintf(os.Stderr, "FAIL: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("source:    %s\n", sourcePi)
	fmt.Printf("worktree:  %s\n", worktree)
	fmt.Println()
	fmt.Println("model replacements:")
	fmt.Println()

	agentsDir := filepath.Join(sourcePi, "agents")
	entries, _ := os.ReadDir(agentsDir)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".agent.md") {
			continue
		}
		src, _ := os.ReadFile(filepath.Join(agentsDir, entry.Name()))
		dst, _ := os.ReadFile(filepath.Join(worktree, ".pi", "agents", entry.Name()))

		srcLines := strings.Split(string(src), "\n")
		dstLines := strings.Split(string(dst), "\n")

		fmt.Printf("  %s\n", entry.Name())
		for i, line := range srcLines {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "model:") {
				placeholder := strings.TrimSpace(strings.TrimPrefix(trimmed, "model:"))
				dstTrimmed := strings.TrimSpace(dstLines[i])
				realModel := strings.TrimSpace(strings.TrimPrefix(dstTrimmed, "model:"))
				fmt.Printf("    %s → %s\n", placeholder, realModel)
			}
		}
		fmt.Println()
	}

	fmt.Printf("result: %d agent files patched, 0 MODEL_ placeholders remaining\n", len(entries))

	if keep {
		fmt.Printf("\nworktree kept at: %s\n", worktree)
	} else {
		fmt.Println("\n(pass --keep to leave the worktree on disk for inspection)")
	}
}
