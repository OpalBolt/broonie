package gh

import (
	"testing"
)

func TestParseFrontmatter(t *testing.T) {
	tests := []struct {
		name      string
		body      string
		wantType  string
		wantDeps  []int
		wantError bool
	}{
		{
			name: "valid AUTO no deps",
			body: `<details>
<summary>Metadata</summary>

` + "```json\n" + `{"type": "AUTO"}
` + "```\n" + `
</details>

Fix the login redirect.`,
			wantType: "AUTO",
			wantDeps: nil,
		},
		{
			name: "valid AUTO with deps",
			body: `<details>
<summary>Metadata</summary>

` + "```json\n" + `{"type": "AUTO", "depends-on": ["#3", "#7"]}
` + "```\n" + `
</details>

Add cache.`,
			wantType: "AUTO",
			wantDeps: []int{3, 7},
		},
		{
			name: "valid HITL",
			body: `<details>
<summary>Metadata</summary>

` + "```json\n" + `{"type": "HITL"}
` + "```\n" + `
</details>

Needs human review.`,
			wantType: "HITL",
			wantDeps: nil,
		},
		{
			name:      "missing details block",
			body:      "Just a normal issue body with no metadata.",
			wantError: true,
		},
		{
			name: "summary without metadata keyword",
			body: `<details>
<summary>Notes</summary>

` + "```json\n" + `{"type": "AUTO"}
` + "```\n" + `
</details>`,
			wantError: true,
		},
		{
			name: "invalid type",
			body: `<details>
<summary>Metadata</summary>

` + "```json\n" + `{"type": "INVALID"}
` + "```\n" + `
</details>`,
			wantError: true,
		},
		{
			name: "depends-on not #N",
			body: `<details>
<summary>Metadata</summary>

` + "```json\n" + `{"type": "AUTO", "depends-on": ["3"]}
` + "```\n" + `
</details>`,
			wantError: true,
		},
		{
			name: "unknown keys ignored",
			body: `<details>
<summary>Metadata</summary>

` + "```json\n" + `{"type": "AUTO", "priority": "high"}
` + "```\n" + `
</details>

Extra keys ok.`,
			wantType: "AUTO",
			wantDeps: nil,
		},
		{
			name: "summary case insensitive",
			body: `<details>
<summary>METADATA</summary>

` + "```json\n" + `{"type": "AUTO"}
` + "```\n" + `
</details>`,
			wantType: "AUTO",
			wantDeps: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotType, gotDeps, err := ParseFrontmatter(tt.body)
			if tt.wantError && err == nil {
				t.Fatalf("expected error, got type=%q deps=%v", gotType, gotDeps)
			}
			if !tt.wantError && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if gotType != tt.wantType {
				t.Errorf("type = %q, want %q", gotType, tt.wantType)
			}
			if !tt.wantError {
				if len(gotDeps) != len(tt.wantDeps) {
					t.Fatalf("deps = %v, want %v", gotDeps, tt.wantDeps)
				}
				for i, d := range tt.wantDeps {
					if gotDeps[i] != d {
						t.Errorf("deps[%d] = %d, want %d", i, gotDeps[i], d)
					}
				}
			}
		})
	}
}
