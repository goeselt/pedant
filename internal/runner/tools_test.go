package runner

import (
	"path/filepath"
	"slices"
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

func TestEditorconfigArgsForceGCCFormat(t *testing.T) {
	t.Parallel()

	args := editorconfigTool.Args(false, t.TempDir(), []string{"Makefile"})
	if !slices.Contains(args, "-format") || !slices.Contains(args, "gcc") {
		t.Fatalf("editorconfig args = %v, want forced -format gcc", args)
	}
}
