package executor

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSlugifyTitle(t *testing.T) {
	tests := []struct {
		input, expected string
	}{
		{"Add user authentication", "add-user-authentication"},
		{"Fix: null pointer in /api/users", "fix-null-pointer-in-api-users"},
		{"  Multiple   spaces  ", "multiple-spaces"},
		{"!!! special chars !!!", "special-chars"},
		{"", "feature"},                       // empty falls back
		{"a very long title that exceeds fifty characters total length", "a-very-long-title-that-exceeds-fifty-characters-to"},
	}
	for _, tt := range tests {
		got := slugifyTitle(tt.input)
		if got != tt.expected {
			t.Errorf("slugifyTitle(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestCheckCompletion(t *testing.T) {
	dir := t.TempDir()

	// Neither file exists
	if checkCompletion(dir) {
		t.Error("expected false when no files exist")
	}

	// Only done exists
	os.WriteFile(filepath.Join(dir, "done"), []byte{}, 0644)
	if checkCompletion(dir) {
		t.Error("expected false when only done exists")
	}

	// Both exist
	os.WriteFile(filepath.Join(dir, "pr.md"), []byte{}, 0644)
	if !checkCompletion(dir) {
		t.Error("expected true when both done and pr.md exist")
	}
}
