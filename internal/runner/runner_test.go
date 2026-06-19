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

func TestLimitedBuffer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		limit     int
		writes    []string
		wantBytes string
		// wantN is the number of bytes reported by the last Write call.
		// Set to -1 to skip the check.
		wantLastN int
	}{
		{
			name:      "within limit",
			limit:     10,
			writes:    []string{"hello"},
			wantBytes: "hello",
			wantLastN: 5,
		},
		{
			name:      "truncates at limit, reports full input length to avoid stalling subprocess",
			limit:     4,
			writes:    []string{"hello world"},
			wantBytes: "hell",
			wantLastN: 11, // full input length, not the truncated 4
		},
		{
			name:      "discards subsequent writes when already full",
			limit:     3,
			writes:    []string{"abc", "extra content"},
			wantBytes: "abc",
			wantLastN: 13, // "extra content" reported as fully consumed
		},
		{
			name:      "zero limit means unbounded",
			limit:     0,
			writes:    []string{strings.Repeat("x", 1000)},
			wantBytes: strings.Repeat("x", 1000),
			wantLastN: 1000,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var buf limitedBuffer
			buf.limit = tc.limit

			var lastN int
			for _, w := range tc.writes {
				n, err := buf.Write([]byte(w))
				if err != nil {
					t.Fatalf("Write(%q) error: %v", w, err)
				}
				lastN = n
			}

			if got := buf.String(); got != tc.wantBytes {
				t.Errorf("content = %q, want %q", got, tc.wantBytes)
			}
			if tc.wantLastN >= 0 && lastN != tc.wantLastN {
				t.Errorf("last Write() returned %d, want %d", lastN, tc.wantLastN)
			}
		})
	}
}

// -- file stat preservation ----------------------------------------------------------

func TestRestoreFileStat(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	path := filepath.Join(tmp, "script.sh")
	if err := os.WriteFile(path, []byte("#!/usr/bin/env bash\necho hi\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	saved := captureFileStat(tmp, []string{"script.sh"})
	if len(saved) != 1 {
		t.Fatalf("captureFileStat: got %d entries, want 1", len(saved))
	}
	st := saved["script.sh"]
	if st.mode&0o111 == 0 {
		t.Fatalf("captureFileStat: mode %v missing execute bit", st.mode)
	}

	// Simulate a tool that drops the execute bit (e.g. via temp+rename).
	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatal(err)
	}

	restoreFileStat(tmp, []string{"script.sh"}, saved)

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode() != st.mode {
		t.Errorf("mode after restore: got %v, want %v", info.Mode(), st.mode)
	}
}

// -- tool timeout --------------------------------------------------------------------

func TestRunWithTimeoutStopsHungTool(t *testing.T) {
	t.Parallel()

	// Use an existing system binary (/bin/sh) to avoid "text file busy" on
	// newly written executables. The tool sleeps for 5 s; the test timeout
	// fires at 25 ms, so the process must be killed well before it exits.
	tool := ToolDef{
		Name:   "slow-tool",
		Binary: "/bin/sh",
		Args: func(bool, string, []string) []string {
			return []string{"-c", "sleep 5"}
		},
		Parse: func(string, string, int, string) ([]Finding, error) {
			return nil, nil
		},
	}

	start := time.Now()
	result := RunWithTimeout(context.Background(), tool, t.TempDir(), false, []string{"x"}, io.Discard, 25*time.Millisecond)
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
