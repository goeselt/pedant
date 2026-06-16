// Package runner executes individual lint/format tools as subprocesses and aggregates their results.
package runner

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
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
	Tool     string    `json:"name"`
	Status   string    `json:"status"` // "pass", "fail", "error", "skip"
	Findings []Finding `json:"findings"`
	Error    string    `json:"error,omitempty"`
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
	Args   func(fix bool, workspace string, files []string) []string
	Parse  func(stdout, stderr string, exitCode int, workspace string) ([]Finding, error)
}

// maxFileArgBytes is the maximum total byte-length of file-path arguments per tool invocation.
// Kept well below Linux ARG_MAX (~2 MB) to leave headroom for the binary path, flags, and environment variables.
const maxFileArgBytes = 200 * 1024

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

// Run executes a single tool and writes human-readable progress to log.
// In fix mode, fixable tools run a silent fix pass followed by a check pass;
// only what could not be fixed is reported. Non-fixable tools always run check-only.
//
// When the total size of file-path arguments would exceed maxFileArgBytes, the file list is split into batches
// and the tool is invoked once per batch.
// Findings from all batches are merged.
// Tools that do not accept an explicit file list (NoBatch: true) are always invoked once.
func Run(ctx context.Context, def ToolDef, workspace string, fix bool, files []string, log io.Writer) Result {
	if def.Skip != nil && def.Skip(workspace) {
		reason := def.Reason
		if reason == "" {
			reason = "not applicable"
		}
		_, _ = fmt.Fprintf(log, "[%s] skip -- %s\n", def.Name, reason)
		return Result{Tool: def.Name, Status: "skip"}
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

	var allFindings []Finding
	for _, batch := range batches {
		findings, errMsg := invokeBatch(ctx, def, binary, workspace, fix, batch, log)
		if errMsg != "" {
			_, _ = fmt.Fprintf(log, "[%s] error -- %s\n", def.Name, errMsg)
			return Result{Tool: def.Name, Status: "error", Error: errMsg}
		}
		allFindings = append(allFindings, findings...)
	}

	if len(allFindings) == 0 {
		_, _ = fmt.Fprintf(log, "[%s] pass\n", def.Name)
		return Result{Tool: def.Name, Status: "pass"}
	}

	_, _ = fmt.Fprintf(log, "[%s] %d finding(s)\n", def.Name, len(allFindings))
	for _, f := range allFindings {
		printFinding(log, f)
	}
	return Result{
		Tool:     def.Name,
		Status:   "fail",
		Findings: allFindings,
	}
}

// invokeBatch runs one fix+check cycle for a single batch of files.
// Returns (findings, errMsg); errMsg != "" is a hard error that aborts batching.
func invokeBatch(ctx context.Context, def ToolDef, binary, workspace string, fix bool, files []string, log io.Writer) ([]Finding, string) {
	if fix && def.CanFix {
		fixCmd := exec.CommandContext(ctx, binary, def.Args(true, workspace, files)...)
		fixCmd.Dir = workspace
		var fixStderr bytes.Buffer
		fixCmd.Stderr = &fixStderr
		if err := fixCmd.Run(); err != nil {
			var exitErr *exec.ExitError
			if !errors.As(err, &exitErr) {
				_, _ = fmt.Fprintf(log, "[%s] warning -- fix pass: %v\n", def.Name, err)
			} else if msg := strings.TrimSpace(fixStderr.String()); msg != "" {
				_, _ = fmt.Fprintf(log, "[%s] warning -- fix pass stderr: %s\n", def.Name, msg)
			}
		}
	}

	// Check pass -- always runs; reports what remains (or all findings in check-only mode).
	cmd := exec.CommandContext(ctx, binary, def.Args(false, workspace, files)...)
	cmd.Dir = workspace

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	runErr := cmd.Run()

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
