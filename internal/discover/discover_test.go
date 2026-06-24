package discover

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"testing"
)

func TestFiles(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	gitRun(t, dir, "git", "init")
	gitRun(t, dir, "git", "config", "user.email", "test@example.com")
	gitRun(t, dir, "git", "config", "user.name", "Test")

	write(t, dir, "main.go", "package main\n")
	write(t, dir, "script.sh", "#!/bin/bash\n")
	write(t, dir, "ignore.tmp", "tmp\n")
	write(t, dir, ".gitignore", "*.tmp\n")
	gitRun(t, dir, "git", "add", ".")

	got, err := Files(dir, nil, nil)
	if err != nil {
		t.Fatalf("Files: %v", err)
	}

	want := []string{".gitignore", "main.go", "script.sh"}
	slices.Sort(got)
	slices.Sort(want)

	if len(got) != len(want) {
		t.Fatalf("Files() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("Files()[%d] = %q, want %q", i, got[i], want[i])
		}
	}

	if slices.Contains(got, "ignore.tmp") {
		t.Error("Files() includes ignore.tmp, which is in .gitignore")
	}
}

func TestFilesWithPaths(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	gitRun(t, dir, "git", "init")
	gitRun(t, dir, "git", "config", "user.email", "test@example.com")
	gitRun(t, dir, "git", "config", "user.name", "Test")

	write(t, dir, "root.go", "package main\n")
	write(t, dir, "sub/nested.go", "package sub\n")
	write(t, dir, "other/file.go", "package other\n")
	gitRun(t, dir, "git", "add", ".")

	got, err := Files(dir, []string{"sub"}, nil)
	if err != nil {
		t.Fatalf("Files: %v", err)
	}

	if len(got) != 1 || got[0] != "sub/nested.go" {
		t.Errorf("Files(dir, [sub]) = %v, want [sub/nested.go]", got)
	}
}

func TestFilesWithFilePath(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	gitRun(t, dir, "git", "init")
	gitRun(t, dir, "git", "config", "user.email", "test@example.com")
	gitRun(t, dir, "git", "config", "user.name", "Test")

	write(t, dir, "root.go", "package main\n")
	write(t, dir, "sub/nested.go", "package sub\n")
	write(t, dir, "other/file.go", "package other\n")
	gitRun(t, dir, "git", "add", ".")

	got, err := Files(dir, []string{"sub/nested.go"}, nil)
	if err != nil {
		t.Fatalf("Files: %v", err)
	}

	if len(got) != 1 || got[0] != "sub/nested.go" {
		t.Errorf("Files(dir, [sub/nested.go]) = %v, want [sub/nested.go]", got)
	}
}

func TestFilesWithGitignoredPath(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	gitRun(t, dir, "git", "init")
	gitRun(t, dir, "git", "config", "user.email", "test@example.com")
	gitRun(t, dir, "git", "config", "user.name", "Test")

	write(t, dir, ".gitignore", "ignored/\n")
	write(t, dir, "keep.go", "package main\n")
	write(t, dir, "ignored/nested.go", "package ignored\n")
	gitRun(t, dir, "git", "add", ".gitignore", "keep.go")

	got, err := Files(dir, []string{"ignored"}, nil)
	if err != nil {
		t.Fatalf("Files: %v", err)
	}

	if len(got) != 0 {
		t.Errorf("Files(dir, [ignored]) = %v, want []", got)
	}
}

func TestFilesWithGitignoredFilePath(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	gitRun(t, dir, "git", "init")
	gitRun(t, dir, "git", "config", "user.email", "test@example.com")
	gitRun(t, dir, "git", "config", "user.name", "Test")

	write(t, dir, ".gitignore", "ignored.md\n")
	write(t, dir, "keep.md", "# Keep\n")
	write(t, dir, "ignored.md", "# Ignored\n")
	gitRun(t, dir, "git", "add", ".gitignore", "keep.md")

	got, err := Files(dir, []string{"ignored.md"}, nil)
	if err != nil {
		t.Fatalf("Files: %v", err)
	}

	if len(got) != 0 {
		t.Errorf("Files(dir, [ignored.md]) = %v, want []", got)
	}
}

func TestFilesEmpty(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	gitRun(t, dir, "git", "init")
	gitRun(t, dir, "git", "config", "user.email", "test@example.com")
	gitRun(t, dir, "git", "config", "user.name", "Test")

	got, err := Files(dir, nil, nil)
	if err != nil {
		t.Fatalf("Files: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("Files() on empty repo = %v, want []", got)
	}
}

func TestFilesWithIgnore(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	gitRun(t, dir, "git", "init")
	gitRun(t, dir, "git", "config", "user.email", "test@example.com")
	gitRun(t, dir, "git", "config", "user.name", "Test")

	write(t, dir, "root.go", "package main\n")
	write(t, dir, "vendor/lib.go", "package lib\n")
	write(t, dir, "vendor/other.go", "package lib\n")
	gitRun(t, dir, "git", "add", ".")

	got, err := Files(dir, nil, []string{"vendor/"})
	if err != nil {
		t.Fatalf("Files: %v", err)
	}

	if len(got) != 1 || got[0] != "root.go" {
		t.Errorf("Files(dir, nil, [vendor/]) = %v, want [root.go]", got)
	}
}

func TestFilesWithIgnoredFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	gitRun(t, dir, "git", "init")
	gitRun(t, dir, "git", "config", "user.email", "test@example.com")
	gitRun(t, dir, "git", "config", "user.name", "Test")

	write(t, dir, "root.go", "package main\n")
	write(t, dir, "vendor/lib.go", "package lib\n")
	write(t, dir, "vendor/other.go", "package lib\n")
	gitRun(t, dir, "git", "add", ".")

	got, err := Files(dir, nil, []string{"vendor/lib.go"})
	if err != nil {
		t.Fatalf("Files: %v", err)
	}

	want := []string{"root.go", "vendor/other.go"}
	slices.Sort(got)
	slices.Sort(want)

	if len(got) != len(want) {
		t.Fatalf("Files(dir, nil, [vendor/lib.go]) = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("Files()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestFilesRejectsPathspecMagic(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	gitRun(t, dir, "git", "init")
	gitRun(t, dir, "git", "config", "user.email", "test@example.com")
	gitRun(t, dir, "git", "config", "user.name", "Test")

	tests := []struct {
		name   string
		paths  []string
		ignore []string
	}{
		{name: "magic in paths", paths: []string{":(glob)**/*.go"}},
		{name: "magic in ignore", ignore: []string{":(icase)vendor"}},
		{name: "bare magic prefix", ignore: []string{":(exclude)src"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := Files(dir, tc.paths, tc.ignore)
			if err == nil {
				t.Fatalf("Files with pathspec magic %v/%v: expected error, got nil", tc.paths, tc.ignore)
			}
		})
	}
}

func TestFilesSkipsSymlinks(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	gitRun(t, dir, "git", "init")
	gitRun(t, dir, "git", "config", "user.email", "test@example.com")
	gitRun(t, dir, "git", "config", "user.name", "Test")

	// Regular file -- must be included.
	write(t, dir, "main.go", "package main\n")

	// Symlink pointing to a file inside the workspace -- must be excluded.
	if err := os.Symlink(filepath.Join(dir, "main.go"), filepath.Join(dir, "link.go")); err != nil {
		t.Fatalf("Symlink: %v", err)
	}

	// Symlink pointing outside the workspace (e.g. /etc/passwd) -- must be excluded.
	outsideTarget := filepath.Join(t.TempDir(), "sensitive.txt")
	if err := os.WriteFile(outsideTarget, []byte("secret"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outsideTarget, filepath.Join(dir, "outside.go")); err != nil {
		t.Fatalf("Symlink: %v", err)
	}

	gitRun(t, dir, "git", "add", ".")

	got, err := Files(dir, nil, nil)
	if err != nil {
		t.Fatalf("Files: %v", err)
	}

	if len(got) != 1 || got[0] != "main.go" {
		t.Errorf("Files() = %v, want [main.go] (symlinks must be excluded)", got)
	}
}

func gitRun(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.CommandContext(context.Background(), args[0], args[1:]...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%v: %v\n%s", args, err, out)
	}
}

func write(t *testing.T, dir, rel, content string) {
	t.Helper()
	path := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}
