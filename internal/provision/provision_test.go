package provision

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProvision(t *testing.T) {
	// Setup: create temp source .pi/ with blip.md and one agent
	srcDir := t.TempDir()
	worktree := t.TempDir()

	// Create source structure
	promptsDir := filepath.Join(srcDir, "prompts")
	agentsDir := filepath.Join(srcDir, "agents")
	os.MkdirAll(promptsDir, 0755)
	os.MkdirAll(agentsDir, 0755)

	os.WriteFile(filepath.Join(promptsDir, "blip.md"), []byte("# Blip prompt"), 0644)
	os.WriteFile(filepath.Join(agentsDir, "blip-test.agent.md"), []byte("---\nname: test\nmodel: MODEL_TEST\n---\n"), 0644)

	models := map[string]string{
		"MODEL_TEST": "claude-haiku-4.5",
	}

	err := Provision(worktree, srcDir, models)
	if err != nil {
		t.Fatalf("Provision failed: %v", err)
	}

	// Verify blip.md was copied
	copied, err := os.ReadFile(filepath.Join(worktree, ".pi", "prompts", "blip.md"))
	if err != nil {
		t.Fatalf("blip.md not found: %v", err)
	}
	if string(copied) != "# Blip prompt" {
		t.Errorf("blip.md content mismatch: got %q", string(copied))
	}

	// Verify agent was patched
	patched, err := os.ReadFile(filepath.Join(worktree, ".pi", "agents", "blip-test.agent.md"))
	if err != nil {
		t.Fatalf("agent not found: %v", err)
	}
	if strings.Contains(string(patched), "MODEL_") {
		t.Errorf("unpatched MODEL_ placeholder remains in: %s", string(patched))
	}
	if !strings.Contains(string(patched), "model: claude-haiku-4.5") {
		t.Errorf("expected patched model not found in: %s", string(patched))
	}
}

func TestProvisionUnpatchedModel(t *testing.T) {
	srcDir := t.TempDir()
	worktree := t.TempDir()

	agentsDir := filepath.Join(srcDir, "agents")
	promptsDir := filepath.Join(srcDir, "prompts")
	os.MkdirAll(agentsDir, 0755)
	os.MkdirAll(promptsDir, 0755)

	os.WriteFile(filepath.Join(promptsDir, "blip.md"), []byte("# prompt"), 0644)
	os.WriteFile(filepath.Join(agentsDir, "blip-missing.agent.md"), []byte("---\nmodel: MODEL_UNKNOWN\n---\n"), 0644)

	models := map[string]string{
		"MODEL_KNOWN": "some-model",
	}

	err := Provision(worktree, srcDir, models)
	if err == nil {
		t.Fatal("expected error for unpatched MODEL_UNKNOWN, got nil")
	}
	if !strings.Contains(err.Error(), "MODEL_UNKNOWN") {
		t.Errorf("error should mention MODEL_UNKNOWN placeholder: %v", err)
	}
}

func TestProvisionDoesNotCorruptComments(t *testing.T) {
	srcDir := t.TempDir()
	worktree := t.TempDir()

	agentsDir := filepath.Join(srcDir, "agents")
	promptsDir := filepath.Join(srcDir, "prompts")
	os.MkdirAll(agentsDir, 0755)
	os.MkdirAll(promptsDir, 0755)

	os.WriteFile(filepath.Join(promptsDir, "blip.md"), []byte("# prompt"), 0644)
	// Comment containing model: MODEL_FOO must not be touched
	os.WriteFile(filepath.Join(agentsDir, "doc-agent.agent.md"), []byte("---\nname: doc\nmodel: MODEL_DOC\n---\n# This agent uses model: MODEL_DOC\n"), 0644)

	models := map[string]string{"MODEL_DOC": "claude-haiku-4.5"}

	err := Provision(worktree, srcDir, models)
	if err != nil {
		t.Fatalf("Provision failed: %v", err)
	}

	patched, err := os.ReadFile(filepath.Join(worktree, ".pi", "agents", "doc-agent.agent.md"))
	if err != nil {
		t.Fatalf("agent not found: %v", err)
	}

	// The comment line should preserve "model: MODEL_DOC" intact
	if !strings.Contains(string(patched), "# This agent uses model: MODEL_DOC") {
		t.Errorf("comment was corrupted — MODEL_DOC in comment line was replaced: %s", string(patched))
	}
	// But the frontmatter line should be patched
	if !strings.Contains(string(patched), "model: claude-haiku-4.5") {
		t.Errorf("frontmatter model line was not patched: %s", string(patched))
	}
}
