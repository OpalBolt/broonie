package gh

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// ParseFrontmatter extracts type and depends-on from a hidden <details> JSON block.
//
// Expected format:
//
//	<details>
//	<summary>Metadata</summary>
//
//	```json
//	{"type": "AUTO", "depends-on": ["#1"]}
//	```
//
//	</details>
//
// The summary must contain "metadata" (case-insensitive).
// Unknown JSON keys are ignored (forward-compatible).
// Returns ("", nil, error) if the block is missing or malformed.
func ParseFrontmatter(body string) (issueType string, dependsOn []int, err error) {
	start := strings.Index(strings.ToLower(body), "<details>")
	if start == -1 {
		return "", nil, fmt.Errorf("no <details> block found")
	}

	// Find summary, check it contains "metadata"
	summaryStart := strings.Index(strings.ToLower(body[start:]), "<summary>")
	summaryEnd := strings.Index(strings.ToLower(body[start:]), "</summary>")
	if summaryStart == -1 || summaryEnd == -1 || summaryStart >= summaryEnd {
		return "", nil, fmt.Errorf("<details> missing <summary>")
	}
	summary := body[start+summaryStart+9 : start+summaryEnd]
	if !strings.Contains(strings.ToLower(summary), "metadata") {
		return "", nil, fmt.Errorf("summary must contain 'metadata'")
	}

	// Find fenced JSON block
	jsonStart := strings.Index(body[start:], "```json")
	if jsonStart == -1 {
		return "", nil, fmt.Errorf("no ```json block found in <details>")
	}
	jsonStart += start + 7 // skip ```json
	if nl := strings.IndexByte(body[jsonStart:], '\n'); nl != -1 {
		jsonStart += nl + 1
	}

	jsonEnd := strings.Index(body[jsonStart:], "\n```")
	if jsonEnd == -1 {
		return "", nil, fmt.Errorf("unclosed ```json block")
	}
	raw := body[jsonStart : jsonStart+jsonEnd]

	var fm struct {
		Type      string   `json:"type"`
		DependsOn []string `json:"depends-on"`
	}
	if err := json.Unmarshal([]byte(raw), &fm); err != nil {
		return "", nil, fmt.Errorf("invalid JSON: %w", err)
	}

	if fm.Type != "AUTO" && fm.Type != "HITL" {
		return "", nil, fmt.Errorf("type must be AUTO or HITL, got %q", fm.Type)
	}

	for _, dep := range fm.DependsOn {
		if !strings.HasPrefix(dep, "#") {
			return "", nil, fmt.Errorf("depends-on items must be '#N' strings, got %q", dep)
		}
		n, err := strconv.Atoi(dep[1:])
		if err != nil {
			return "", nil, fmt.Errorf("invalid depends-on reference: %s", dep)
		}
		dependsOn = append(dependsOn, n)
	}

	return fm.Type, dependsOn, nil
}
