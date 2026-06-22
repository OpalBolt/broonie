package provision

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Provision copies .pi/prompts/blip.md and model-patched agent files
// from sourcePiDir into worktreePath/.pi/. Each MODEL_* placeholder
// in agent files is replaced with the value from models map.
// Returns error if any MODEL_ placeholder remains unpatched.
func Provision(worktreePath, sourcePiDir string, models map[string]string) error {
	// ponytail: trust boundary — validate paths
	if worktreePath == "" || sourcePiDir == "" {
		return fmt.Errorf("worktreePath and sourcePiDir must not be empty")
	}
	if _, err := os.Stat(sourcePiDir); err != nil {
		return fmt.Errorf("sourcePiDir %s does not exist: %w", sourcePiDir, err)
	}

	// 1. Create target directories
	promptsDir := filepath.Join(worktreePath, ".pi", "prompts")
	agentsDir := filepath.Join(worktreePath, ".pi", "agents")
	for _, dir := range []string{promptsDir, agentsDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("create dir %s: %w", dir, err)
		}
	}

	// 2. Copy blip.md prompt
	srcPrompt := filepath.Join(sourcePiDir, "prompts", "blip.md")
	dstPrompt := filepath.Join(promptsDir, "blip.md")
	data, err := os.ReadFile(srcPrompt)
	if err != nil {
		return fmt.Errorf("read %s: %w", srcPrompt, err)
	}
	if err := os.WriteFile(dstPrompt, data, 0644); err != nil {
		return fmt.Errorf("write %s: %w", dstPrompt, err)
	}

	// 3. Patch and copy agent files — replace model: MODEL_* only on lines starting with "model:"
	srcAgentsDir := filepath.Join(sourcePiDir, "agents")
	entries, err := os.ReadDir(srcAgentsDir)
	if err != nil {
		return fmt.Errorf("read agents dir %s: %w", srcAgentsDir, err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".agent.md") {
			continue
		}

		srcPath := filepath.Join(srcAgentsDir, entry.Name())
		content, err := os.ReadFile(srcPath)
		if err != nil {
			return fmt.Errorf("read %s: %w", srcPath, err)
		}

		lines := strings.Split(string(content), "\n")
		for i, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "model:") {
				// Extract placeholder: "model: MODEL_FOO" -> "MODEL_FOO"
				placeholder := strings.TrimSpace(strings.TrimPrefix(trimmed, "model:"))
				if strings.HasPrefix(placeholder, "MODEL_") {
					realModel, ok := models[placeholder]
					if !ok {
						return fmt.Errorf("%s: no model mapping for placeholder %q", entry.Name(), placeholder)
					}
					lines[i] = strings.Replace(line, placeholder, realModel, 1)
				}
			}
		}
		patched := strings.Join(lines, "\n")

		dstPath := filepath.Join(agentsDir, entry.Name())
		if err := os.WriteFile(dstPath, []byte(patched), 0644); err != nil {
			return fmt.Errorf("write %s: %w", dstPath, err)
		}
	}

	return nil
}
