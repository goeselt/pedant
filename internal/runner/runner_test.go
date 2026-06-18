package runner

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// -- filterEnv / isSensitiveEnvVar ----------------------------------------------------

func TestIsSensitiveEnvVar(t *testing.T) {
	t.Parallel()

	sensitive := []string{
		"GITHUB_TOKEN",
		"ACTIONS_RUNTIME_TOKEN",
		"ACTIONS_CACHE_URL",
		"ACTIONS_ID_TOKEN_REQUEST_TOKEN",
		"INPUT_FIX",
		"INPUT_PATHS",
		"MY_API_SECRET",
		"DB_PASSWORD",
		"DEPLOY_KEY",
		"SIGNING_CREDENTIAL",
		"github_token",    // case-insensitive
		"actions_runtime", // prefix match
	}
	safe := []string{
		"PATH",
		"HOME",
		"GOPATH",
		"GOMODCACHE",
		"NODE_ENV",
		"LANG",
		"LC_ALL",
		"PYTHONPATH",
		"GITHUB_WORKSPACE",
		"GITHUB_ACTIONS",
		"GITHUB_SHA",
		"GITHUB_REF",
		"GITHUB_REPOSITORY",
		"PKG_CONFIG_PATH",
		"SSH_AUTH_SOCK",
	}

	for _, name := range sensitive {
		if !isSensitiveEnvVar(name) {
			t.Errorf("isSensitiveEnvVar(%q) = false, want true", name)
		}
	}
	for _, name := range safe {
		if isSensitiveEnvVar(name) {
			t.Errorf("isSensitiveEnvVar(%q) = true, want false", name)
		}
	}
}

func TestFilterEnv(t *testing.T) {
	t.Parallel()

	env := []string{
		"PATH=/usr/bin:/usr/local/bin",
		"HOME=/root",
		"GOPATH=/go",
		"GITHUB_WORKSPACE=/work",
		"GITHUB_ACTIONS=true",
		"GITHUB_TOKEN=secret123",
		"ACTIONS_RUNTIME_TOKEN=token456",
		"ACTIONS_CACHE_URL=https://cache.example.com",
		"INPUT_FIX=false",
		"INPUT_PATHS=src/",
		"MY_SECRET=topSecret",
		"DB_PASSWORD=pass123",
		"API_KEY=key123",
	}

	got := filterEnv(env)

	byName := make(map[string]bool, len(got))
	for _, kv := range got {
		name, _, _ := strings.Cut(kv, "=")
		byName[name] = true
	}

	// Sensitive vars must be removed.
	for _, name := range []string{
		"GITHUB_TOKEN", "ACTIONS_RUNTIME_TOKEN", "ACTIONS_CACHE_URL",
		"INPUT_FIX", "INPUT_PATHS", "MY_SECRET", "DB_PASSWORD", "API_KEY",
	} {
		if byName[name] {
			t.Errorf("filterEnv: %q should be removed but is present", name)
		}
	}

	// Safe vars must be kept.
	for _, name := range []string{
		"PATH", "HOME", "GOPATH", "GITHUB_WORKSPACE", "GITHUB_ACTIONS",
	} {
		if !byName[name] {
			t.Errorf("filterEnv: %q should be kept but is absent", name)
		}
	}
}

func TestFilterEnvEmpty(t *testing.T) {
	t.Parallel()

	if got := filterEnv(nil); got == nil || len(got) != 0 {
		t.Errorf("filterEnv(nil) = %v, want empty non-nil slice", got)
	}
}

// -- limitedBuffer --------------------------------------------------------------------

func TestLimitedBufferWithinLimit(t *testing.T) {
	t.Parallel()

	var buf limitedBuffer
	buf.limit = 10

	n, err := buf.Write([]byte("hello"))
	if err != nil || n != 5 {
		t.Fatalf("Write() = %d, %v; want 5, nil", n, err)
	}
	if got := buf.String(); got != "hello" {
		t.Errorf("content = %q, want %q", got, "hello")
	}
}

func TestLimitedBufferTruncatesAtLimit(t *testing.T) {
	t.Parallel()

	var buf limitedBuffer
	buf.limit = 4

	n, err := buf.Write([]byte("hello world"))
	if err != nil {
		t.Fatalf("Write() error: %v", err)
	}
	// Returns the full input length so the writing process is not stalled.
	if n != 11 {
		t.Errorf("Write() = %d, want 11", n)
	}
	if got := buf.String(); got != "hell" {
		t.Errorf("content = %q, want %q", got, "hell")
	}
}

func TestLimitedBufferDiscardsWhenFull(t *testing.T) {
	t.Parallel()

	var buf limitedBuffer
	buf.limit = 3

	if _, err := buf.Write([]byte("abc")); err != nil {
		t.Fatalf("Write() error: %v", err)
	}
	if _, err := buf.Write([]byte("extra content")); err != nil {
		t.Fatalf("Write() error: %v", err)
	}

	if got := buf.String(); got != "abc" {
		t.Errorf("content = %q, want %q", got, "abc")
	}
}

func TestLimitedBufferZeroLimitUnbounded(t *testing.T) {
	t.Parallel()

	var buf limitedBuffer
	// limit == 0 means no limit.
	data := strings.Repeat("x", 1000)
	if _, err := buf.Write([]byte(data)); err != nil {
		t.Fatalf("Write() error: %v", err)
	}
	if got := len(buf.String()); got != 1000 {
		t.Errorf("len(content) = %d, want 1000", got)
	}
}

// -- tool timeout --------------------------------------------------------------------

func TestRunWithTimeoutStopsHungTool(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	toolPath := filepath.Join(dir, "slow-tool")
	if err := os.WriteFile(toolPath, []byte("#!/bin/sh\nsleep 5\n"), 0o755); err != nil {
		t.Fatalf("write fake tool: %v", err)
	}

	tool := ToolDef{
		Name:   "slow-tool",
		Binary: toolPath,
		Args: func(bool, string, []string) []string {
			return nil
		},
		Parse: func(string, string, int, string) ([]Finding, error) {
			return nil, nil
		},
	}

	start := time.Now()
	result := RunWithTimeout(context.Background(), tool, dir, false, []string{"x"}, io.Discard, 25*time.Millisecond)
	elapsed := time.Since(start)

	if result.Status != "error" {
		t.Fatalf("Status = %q, want error", result.Status)
	}
	if !strings.Contains(result.Error, "timed out after 25ms") {
		t.Fatalf("Error = %q, want timeout message", result.Error)
	}
	if elapsed > time.Second {
		t.Fatalf("timeout took %s, want under 1s", elapsed)
	}
}
