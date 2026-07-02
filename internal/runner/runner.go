// Package runner executes individual lint/format tools as subprocesses and aggregates their results.
package runner

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

// Finding represents a single issue found by a tool.
type Finding struct {
	File    string `json:"file"`
	Line    int    `json:"line,omitempty"`
	Col     int    `json:"col,omitempty"`
	Level   string `json:"level,omitempty"`
	Rule    string `json:"rule,omitempty"`
	Message string `json:"message,omitempty"`
}

// Result holds the outcome of running one tool.
type Result struct {
	Tool            string    `json:"name"`
	Status          string    `json:"status"` // "pass", "fail", "error", "skip"
	Findings        []Finding `json:"findings"`
	Error           string    `json:"error,omitempty"`
	WorkspaceConfig string    `json:"workspace_config,omitempty"`
}

// ToolDef describes how to invoke a tool and interpret its output.
type ToolDef struct {
	Name   string
	Binary string // executable name; empty defaults to Name
	CanFix bool
	// Globs selects which discovered files this tool receives.
	// Patterns without '/' match against the basename; patterns with '/' match
	// against any path suffix with the same segment count. Empty/nil means all files.
	Globs []string
	// NoBatch disables automatic file-argument batching for tools that do not
	// accept an explicit file list (e.g. golangci-lint which uses ./... instead).
	NoBatch bool
	// Skip returns true when the tool should not run for this workspace.
	// A nil Skip means never skip.
	Skip   func(workspace string) bool
	Reason string // optional explanation appended to skip log when Skip returns true
	// FindWorkspaceConfig returns the workspace-relative path of a
	// workspace-supplied configuration file if one is present, or "" if the
	// bundled default will be used. When non-empty, Run logs an info line and
	// stores the path in Result.WorkspaceConfig.
	FindWorkspaceConfig func(workspace string) string
	Args                func(fix bool, workspace string, files []string) []string
	Parse               func(stdout, stderr string, exitCode int, workspace string) ([]Finding, error)
}

// -- Constants ------------------------------------------------------------------------

// maxFileArgBytes is the maximum total byte-length of file-path arguments per tool invocation.
// Kept well below Linux ARG_MAX (~2 MB) to leave headroom for the binary path, flags, and environment variables.
const maxFileArgBytes = 200 * 1024

// maxOutputBytes caps the in-memory buffer used to collect tool stdout/stderr.
// Output beyond this limit is silently discarded; the resulting parse error
// surfaces as a tool error result rather than an OOM crash.
const maxOutputBytes = 50 * 1024 * 1024 // 50 MB

// maxErrorLen caps the length of error messages stored in Result.Error.
// Keeps the JSON output and Markdown summary bounded when a tool writes large
// diagnostic text to stderr (e.g. after reading a symlinked file as config).
const maxErrorLen = 4000

// DefaultToolTimeout is the maximum wall-clock duration allowed for a single tool
// (across all its batches). Tools that exceed this limit are killed by the
// OS and reported as errors, preventing a deadlocked or pathologically slow
// linter from stalling the pipeline indefinitely.
const DefaultToolTimeout = 10 * time.Minute

// -- Environment filtering ------------------------------------------------------------

// isSensitiveEnvVar reports whether an environment variable name matches a
// pattern associated with secrets. Tool subprocesses (which may load
// workspace-controlled config files that execute arbitrary code) receive a
// filtered copy of the process environment with these variables removed.
func isSensitiveEnvVar(name string) bool {
	upper := strings.ToUpper(name)
	// Block GitHub Actions file-command paths: code run by a tool subprocess
	// (e.g. via a workspace-controlled config) could otherwise append to these
	// files and inject environment variables, PATH entries, or step outputs
	// into the surrounding workflow. The file names contain unguessable IDs,
	// so removing the variables meaningfully raises the bar.
	switch upper {
	case "GITHUB_ENV", "GITHUB_OUTPUT", "GITHUB_PATH", "GITHUB_STATE", "GITHUB_STEP_SUMMARY":
		return true
	}
	// Block well-known CI secret namespaces by prefix.
	for _, p := range []string{"INPUT_", "ACTIONS_"} {
		if strings.HasPrefix(upper, p) {
			return true
		}
	}
	// Block common secret naming patterns by suffix.
	for _, s := range []string{"_TOKEN", "_SECRET", "_KEY", "_PASSWORD", "_CREDENTIAL"} {
		if strings.HasSuffix(upper, s) {
			return true
		}
	}
	return false
}

// filterEnv returns a copy of env with likely-secret variables removed.
// Retained: PATH, HOME, GOPATH, GITHUB_WORKSPACE, and all other operational vars.
// Removed: *_TOKEN, *_SECRET, *_KEY, *_PASSWORD, *_CREDENTIAL, INPUT_*, ACTIONS_*,
// and the GitHub Actions file-command paths (GITHUB_ENV, GITHUB_OUTPUT, ...).
func filterEnv(env []string) []string {
	out := make([]string, 0, len(env))
	for _, kv := range env {
		name, _, _ := strings.Cut(kv, "=")
		if !isSensitiveEnvVar(name) {
			out = append(out, kv)
		}
	}
	return out
}

// -- Output buffering -----------------------------------------------------------------

// limitedBuffer is a bytes.Buffer that stops accepting writes once limit bytes
// have been accumulated. Excess data is silently discarded. limit == 0 means
// no limit (behaves identically to bytes.Buffer).
type limitedBuffer struct {
	bytes.Buffer
	limit int
}

func (b *limitedBuffer) Write(p []byte) (int, error) {
	full := len(p)
	if b.limit > 0 {
		available := b.limit - b.Len()
		if available <= 0 {
			return full, nil // discard; report full len so the process is not stalled
		}
		if len(p) > available {
			p = p[:available]
		}
	}
	_, err := b.Buffer.Write(p)
	return full, err // always report full len; a short-write would stall the subprocess
}

// -- Core execution -------------------------------------------------------------------

// splitBatches partitions files into slices whose total path length stays within maxFileArgBytes.
// A single-element result means no splitting was needed.
func splitBatches(files []string) [][]string {
	if len(files) == 0 {
		return [][]string{files}
	}
	var batches [][]string
	var cur []string
	var curBytes int
	for _, f := range files {
		need := len(f) + 1 // +1 for the argv NUL separator
		if curBytes+need > maxFileArgBytes && len(cur) > 0 {
			batches = append(batches, cur)
			cur = nil
			curBytes = 0
		}
		cur = append(cur, f)
		curBytes += need
	}
	if len(cur) > 0 {
		batches = append(batches, cur)
	}
	return batches
}

// Run executes a single tool with the default timeout and writes human-readable progress to log.
func Run(ctx context.Context, def ToolDef, workspace string, fix bool, files []string, log io.Writer) Result {
	return RunWithTimeout(ctx, def, workspace, fix, files, log, DefaultToolTimeout)
}

// RunWithTimeout executes a single tool and writes human-readable progress to log.
// In fix mode, fixable tools run a silent fix pass followed by a check pass;
// only what could not be fixed is reported. Non-fixable tools always run check-only.
//
// When the total size of file-path arguments would exceed maxFileArgBytes, the file list is split into batches
// and the tool is invoked once per batch.
// Findings from all batches are merged.
// Tools that do not accept an explicit file list (NoBatch: true) are always invoked once.
func RunWithTimeout(ctx context.Context, def ToolDef, workspace string, fix bool, files []string, log io.Writer, timeout time.Duration) Result {
	if def.Skip != nil && def.Skip(workspace) {
		reason := def.Reason
		if reason == "" {
			reason = "not applicable"
		}
		_, _ = fmt.Fprintf(log, "[%s] skip -- %s\n", def.Name, reason)
		return Result{Tool: def.Name, Status: "skip"}
	}

	var wsConfig string
	if def.FindWorkspaceConfig != nil {
		wsConfig = def.FindWorkspaceConfig(workspace)
		if wsConfig != "" {
			_, _ = fmt.Fprintf(log, "[%s] info -- using workspace config %s; workspace-controlled configs can execute arbitrary code\n", def.Name, wsConfig)
		}
	}

	_, _ = fmt.Fprintf(log, "[%s] checking %d files...\n", def.Name, len(files))

	binary := def.Binary
	if binary == "" {
		binary = def.Name
	}

	batches := [][]string{files}
	if !def.NoBatch {
		batches = splitBatches(files)
	}

	// Compute a filtered environment once per Run call; all batches share it.
	// Secrets (tokens, keys, passwords, action inputs) are removed so that
	// workspace-controlled config files cannot read them.
	env := filterEnv(os.Environ())

	toolCtx := ctx
	cancel := func() {}
	if timeout > 0 {
		// Cap per-tool wall-clock time so a deadlocked or pathologically slow linter
		// cannot stall the pipeline indefinitely.
		toolCtx, cancel = context.WithTimeout(ctx, timeout)
	}
	defer cancel()

	var allFindings []Finding
	for _, batch := range batches {
		findings, errMsg := invokeBatch(toolCtx, def, binary, workspace, fix, batch, log, env)
		if errors.Is(toolCtx.Err(), context.DeadlineExceeded) {
			errMsg = fmt.Sprintf("timed out after %s", timeout)
		}
		if errMsg != "" {
			if len(errMsg) > maxErrorLen {
				errMsg = errMsg[:maxErrorLen] + " [truncated]"
			}
			_, _ = fmt.Fprintf(log, "[%s] error -- %s\n", def.Name, errMsg)
			return Result{Tool: def.Name, Status: "error", Error: errMsg, WorkspaceConfig: wsConfig}
		}
		allFindings = append(allFindings, findings...)
	}

	// NoBatch tools scan the workspace themselves (e.g. golangci-lint ./...)
	// and can report findings in files outside the discovered selection
	// (--path/--ignore, gitignored, or default-ignored files). Drop those so
	// every tool honors the same file selection.
	if def.NoBatch {
		allFindings = keepAssignedFiles(allFindings, files)
	}

	if len(allFindings) == 0 {
		_, _ = fmt.Fprintf(log, "[%s] pass\n", def.Name)
		return Result{Tool: def.Name, Status: "pass", WorkspaceConfig: wsConfig}
	}

	_, _ = fmt.Fprintf(log, "[%s] %d finding(s)\n", def.Name, len(allFindings))
	for _, f := range allFindings {
		printFinding(log, f)
	}
	return Result{
		Tool:            def.Name,
		Status:          "fail",
		Findings:        allFindings,
		WorkspaceConfig: wsConfig,
	}
}

// keepAssignedFiles returns the findings whose File is in files.
// Finding paths and assigned files are both workspace-relative with forward
// slashes, so exact string comparison is sufficient.
func keepAssignedFiles(findings []Finding, files []string) []Finding {
	assigned := make(map[string]bool, len(files))
	for _, f := range files {
		assigned[f] = true
	}
	kept := findings[:0]
	for _, f := range findings {
		if assigned[f.File] {
			kept = append(kept, f)
		}
	}
	return kept
}

// newToolCommand creates a subprocess in its own process group (Setpgid: true)
// and replaces exec.Cmd's default Cancel with a SIGKILL to the entire process
// group (-pgid). This matters because several linters (golangci-lint, Node.js
// tools) spawn child processes of their own: killing only the parent would
// leave those children running as orphans, leaking resources and potentially
// keeping the workspace lock held after the timeout fires.
// Passing -pgid to syscall.Kill is the idiomatic Linux approach for this; the
// container is always Linux so there is no portability concern.
func newToolCommand(ctx context.Context, binary string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return nil
		}
		if err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL); err != nil && !errors.Is(err, syscall.ESRCH) {
			return err
		}
		return nil
	}
	return cmd
}

// runFixBatch executes the fix pass for one batch and restores file ownership
// afterwards.  Errors from the tool itself are logged as warnings and do not
// abort the subsequent check pass.
func runFixBatch(ctx context.Context, def ToolDef, binary, workspace string, batch []string, log io.Writer, env []string) {
	// Capture ownership before fix: some tools (e.g. shfmt) write via a
	// temp file + rename, creating a new root-owned file inside the container.
	savedStats := captureFileStat(workspace, batch)

	fixCmd := newToolCommand(ctx, binary, def.Args(true, workspace, batch)...)
	fixCmd.Dir = workspace
	fixCmd.Env = env
	var fixStderr limitedBuffer
	fixStderr.limit = maxOutputBytes
	fixCmd.Stderr = &fixStderr
	if err := fixCmd.Run(); err != nil {
		if !errors.Is(ctx.Err(), context.DeadlineExceeded) {
			var exitErr *exec.ExitError
			if !errors.As(err, &exitErr) {
				// Non-exit error (e.g. binary not found): log because the fixer could not run at all.
				_, _ = fmt.Fprintf(log, "[%s] warning -- fix pass: %v\n", def.Name, err)
			}
			// ExitError: many fixers exit non-zero when they encounter issues even
			// in fix mode (e.g. markdownlint reports fixed issues on stderr and exits 1).
			// Suppress the stderr log here; the check pass reports what actually remains.
		}
	}

	restoreFileStat(workspace, batch, savedStats)
}

// RunFixWithTimeout runs only the fix pass for def (no check pass, no findings).
// Called during the first phase of a two-phase --fix run so that every fixer
// has modified files before any tool's check pass reports findings.
func RunFixWithTimeout(ctx context.Context, def ToolDef, workspace string, files []string, log io.Writer, timeout time.Duration) {
	if !def.CanFix {
		return
	}
	if def.Skip != nil && def.Skip(workspace) {
		return
	}

	binary := def.Binary
	if binary == "" {
		binary = def.Name
	}

	batches := [][]string{files}
	if !def.NoBatch {
		batches = splitBatches(files)
	}

	env := filterEnv(os.Environ())

	toolCtx := ctx
	cancel := func() {}
	if timeout > 0 {
		toolCtx, cancel = context.WithTimeout(ctx, timeout)
	}
	defer cancel()

	_, _ = fmt.Fprintf(log, "[%s] fixing %d file(s)...\n", def.Name, len(files))

	for _, batch := range batches {
		if toolCtx.Err() != nil {
			break
		}
		runFixBatch(toolCtx, def, binary, workspace, batch, log, env)
	}
}

// invokeBatch runs the check pass for a single batch of files.
// When fix is true and def.CanFix, it also runs a fix pass first.
// In the two-phase --fix flow this is called with fix=false so only the check
// pass executes (the fix pass has already been done by RunFixWithTimeout).
// Returns (findings, errMsg); errMsg != "" is a hard error that aborts batching.
func invokeBatch(ctx context.Context, def ToolDef, binary, workspace string, fix bool, files []string, log io.Writer, env []string) ([]Finding, string) {
	if fix && def.CanFix {
		runFixBatch(ctx, def, binary, workspace, files, log, env)
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return nil, "timed out"
		}
	}

	// Check pass -- always runs; reports what remains (or all findings in check-only mode).
	cmd := newToolCommand(ctx, binary, def.Args(false, workspace, files)...)
	cmd.Dir = workspace
	cmd.Env = env

	var stdout, stderr limitedBuffer
	stdout.limit = maxOutputBytes
	stderr.limit = maxOutputBytes
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	runErr := cmd.Run()
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return nil, "timed out"
	}

	exitCode := 0
	if runErr != nil {
		var exitErr *exec.ExitError
		if errors.As(runErr, &exitErr) {
			exitCode = exitErr.ExitCode()
		} else {
			return nil, runErr.Error()
		}
	}

	findings, parseErr := def.Parse(stdout.String(), stderr.String(), exitCode, workspace)
	if parseErr != nil {
		return nil, "could not parse output: " + parseErr.Error()
	}

	if len(findings) == 0 && exitCode != 0 {
		// Non-zero exit with no parseable findings means the tool itself failed (e.g. missing input file, bad config).
		// Most tools report diagnostics on stderr, but some write them to stdout.
		errMsg := strings.TrimSpace(stderr.String())
		if errMsg == "" {
			errMsg = strings.TrimSpace(stdout.String())
		}
		if errMsg == "" {
			errMsg = fmt.Sprintf("exited with code %d", exitCode)
		}
		return nil, errMsg
	}

	return findings, ""
}

// fileStat holds the ownership and permission bits of a file as captured
// before a fix pass, so they can be restored afterwards.
type fileStat struct {
	uid  uint32
	gid  uint32
	mode os.FileMode
}

// captureFileStat records the UID, GID, and mode of each file in files
// (resolved relative to workspace) before a fix pass runs.
func captureFileStat(workspace string, files []string) map[string]fileStat {
	stats := make(map[string]fileStat, len(files))
	for _, f := range files {
		info, err := os.Lstat(filepath.Join(workspace, f))
		if err != nil {
			continue
		}
		st, ok := info.Sys().(*syscall.Stat_t)
		if !ok {
			continue
		}
		stats[f] = fileStat{uid: st.Uid, gid: st.Gid, mode: info.Mode()}
	}
	return stats
}

// restoreFileStat applies the ownership and mode recorded in saved to each
// file that was modified by a fix pass.  Errors are silently ignored: a
// failed restore is better than aborting the check pass entirely.
func restoreFileStat(workspace string, files []string, saved map[string]fileStat) {
	for _, f := range files {
		orig, ok := saved[f]
		if !ok {
			continue
		}
		absPath := filepath.Join(workspace, f)
		info, err := os.Lstat(absPath)
		if err != nil {
			continue
		}
		st, ok := info.Sys().(*syscall.Stat_t)
		if !ok {
			continue
		}
		if st.Uid != orig.uid || st.Gid != orig.gid {
			_ = syscall.Chown(absPath, int(orig.uid), int(orig.gid))
		}
		if info.Mode() != orig.mode {
			_ = os.Chmod(absPath, orig.mode)
		}
	}
}

func printFinding(w io.Writer, f Finding) {
	var sb strings.Builder
	sb.WriteString("  ")
	sb.WriteString(f.File)
	if f.Line > 0 {
		fmt.Fprintf(&sb, ":%d", f.Line)
		if f.Col > 0 {
			fmt.Fprintf(&sb, ":%d", f.Col)
		}
	}
	if f.Level != "" {
		fmt.Fprintf(&sb, " %s", f.Level)
	}
	if f.Rule != "" {
		fmt.Fprintf(&sb, " [%s]", f.Rule)
	}
	if f.Message != "" {
		sb.WriteString(" ")
		sb.WriteString(f.Message)
	}
	_, _ = fmt.Fprintln(w, sb.String())
}
