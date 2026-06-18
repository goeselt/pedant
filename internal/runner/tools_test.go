package runner

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
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

func TestWorkspaceConfigRelFound(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".shellcheckrc"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	got := workspaceConfigRel(dir, ".shellcheckrc")
	if got != ".shellcheckrc" {
		t.Errorf("workspaceConfigRel() = %q, want .shellcheckrc", got)
	}
}

func TestWorkspaceConfigRelNotFound(t *testing.T) {
	t.Parallel()

	got := workspaceConfigRel(t.TempDir(), ".shellcheckrc")
	if got != "" {
		t.Errorf("workspaceConfigRel() = %q, want empty", got)
	}
}

func TestRunLogsInfoForWorkspaceConfig(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	configFile := "my.cfg"
	if err := os.WriteFile(filepath.Join(dir, configFile), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	tool := ToolDef{
		Name:   "probe",
		Binary: "true",
		CanFix: false,
		FindWorkspaceConfig: func(workspace string) string {
			return workspaceConfigRel(workspace, configFile)
		},
		Args:  func(_ bool, _ string, _ []string) []string { return nil },
		Parse: func(_, _ string, _ int, _ string) ([]Finding, error) { return nil, nil },
	}

	var log strings.Builder
	result := Run(context.Background(), tool, dir, false, []string{}, &log)

	if result.WorkspaceConfig != configFile {
		t.Errorf("Result.WorkspaceConfig = %q, want %q", result.WorkspaceConfig, configFile)
	}

	logOut := log.String()
	if !strings.Contains(logOut, "info -- using workspace config "+configFile) {
		t.Errorf("log missing info line:\n%s", logOut)
	}
	if !strings.Contains(logOut, "workspace-controlled configs can execute arbitrary code") {
		t.Errorf("log missing security note:\n%s", logOut)
	}
}

func TestRunNoWorkspaceConfigNoInfoLog(t *testing.T) {
	t.Parallel()

	tool := ToolDef{
		Name:   "probe",
		Binary: "true",
		CanFix: false,
		FindWorkspaceConfig: func(workspace string) string {
			return workspaceConfigRel(workspace, "nonexistent.cfg")
		},
		Args:  func(_ bool, _ string, _ []string) []string { return nil },
		Parse: func(_, _ string, _ int, _ string) ([]Finding, error) { return nil, nil },
	}

	var log strings.Builder
	result := Run(context.Background(), tool, t.TempDir(), false, []string{}, &log)

	if result.WorkspaceConfig != "" {
		t.Errorf("Result.WorkspaceConfig = %q, want empty", result.WorkspaceConfig)
	}
	if strings.Contains(log.String(), "info --") {
		t.Errorf("unexpected info line in log:\n%s", log.String())
	}
}

func TestEditorconfigArgsForceGCCFormat(t *testing.T) {
	t.Parallel()

	args := editorconfigTool.Args(false, t.TempDir(), []string{"Makefile"})
	if !slices.Contains(args, "-format") || !slices.Contains(args, "gcc") {
		t.Fatalf("editorconfig args = %v, want forced -format gcc", args)
	}
}

func TestWorkspaceConfigRelRejectsSymlinks(t *testing.T) {
	t.Parallel()

	// Symlink pointing to a valid target — must be rejected.
	dir := t.TempDir()
	target := filepath.Join(t.TempDir(), "real.cfg")
	if err := os.WriteFile(target, []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(dir, ".shellcheckrc")); err != nil {
		t.Fatal(err)
	}

	got := workspaceConfigRel(dir, ".shellcheckrc")
	if got != "" {
		t.Errorf("workspaceConfigRel() = %q; symlinks must be rejected", got)
	}

	// Regular file in same dir must still be found.
	dir2 := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir2, ".shellcheckrc"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}
	got = workspaceConfigRel(dir2, ".shellcheckrc")
	if got != ".shellcheckrc" {
		t.Errorf("workspaceConfigRel() = %q, want .shellcheckrc", got)
	}
}
