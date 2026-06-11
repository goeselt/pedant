package runner

import (
	"path/filepath"
	"testing"
)

func TestRelativize(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	path := filepath.Join(workspace, "src", "main.go")

	got := relativize(workspace, path)
	if got != "src/main.go" {
		t.Fatalf("relativize() = %q, want src/main.go", got)
	}
}
