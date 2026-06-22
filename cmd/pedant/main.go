// pedant -- unified linting and formatting orchestrator.
// Usage: pedant [options] [workspace]
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/goeselt/pedant/internal/discover"
	"github.com/goeselt/pedant/internal/pathignore"
	"github.com/goeselt/pedant/internal/runner"
)

const appName = "pedant"

// multiFlag is a repeatable string flag (--path a --path b).
type multiFlag []string

func (f *multiFlag) String() string     { return strings.Join(*f, ",") }
func (f *multiFlag) Set(v string) error { *f = append(*f, v); return nil }

func main() {
	// --- Flags -------------------------------------------------------------------

	fs := flag.NewFlagSet(appName, flag.ExitOnError)

	var (
		fix               bool
		pretty            bool
		quiet             bool
		summaryMarkdown   bool
		summaryGithubStep bool
		summaryFile       string
		toolTimeout       time.Duration
		pathList          multiFlag
		ignList           multiFlag
	)

	fs.BoolVar(&fix, "fix", false, "Apply auto-fixes in-place; check-only by default")
	fs.BoolVar(&pretty, "pretty", false, "Pretty-print JSON output")
	fs.BoolVar(&quiet, "quiet", false, "Suppress progress output (JSON only on stdout)")
	fs.BoolVar(&quiet, "q", false, "Alias for --quiet")
	fs.BoolVar(&summaryMarkdown, "summary-markdown", false, "Write a Markdown summary to stdout instead of JSON")
	fs.StringVar(&summaryFile, "summary-file", "", "Write the summary to this file; JSON still emitted on stdout")
	fs.BoolVar(&summaryGithubStep, "summary-github-step", false, "Append the summary to $GITHUB_STEP_SUMMARY; JSON still emitted on stdout")
	fs.DurationVar(&toolTimeout, "tool-timeout", runner.DefaultToolTimeout, "Maximum wall-clock duration for one tool, e.g. 30s, 5m, 1h")
	fs.Var(&pathList, "path", "Restrict scan to this path or file (repeatable)")
	fs.Var(&ignList, "ignore", "Exclude this path or file from scan (repeatable)")

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr,
			"Usage: %s [options] [workspace]\n\n"+
				"Discover, classify, and lint/format files in a Git repository.\n"+
				"Default mode is check-only; use --fix to apply auto-fixes.\n\n"+
				"Arguments:\n"+
				"  [workspace]          Repository root to lint (default: current directory)\n\n"+
				"Options:\n",
			appName)
		fs.PrintDefaults()
		fmt.Fprintf(os.Stderr,
			"\nProgress output goes to stderr. JSON result goes to stdout unless --summary-markdown is set.\n"+
				"--summary-file and --summary-github-step write Markdown independently; JSON is still emitted.\n"+
				"Exit codes: 0 = pass, 1 = findings, 2 = error.\n\n"+
				"Tools (in execution order):\n")
		for _, t := range runner.Registry {
			fix := "check only"
			if t.CanFix {
				fix = "check + fix"
			}
			fmt.Fprintf(os.Stderr, "  %-15s %s\n", t.Name, fix)
		}
	}

	if err := fs.Parse(os.Args[1:]); err != nil {
		os.Exit(2)
	}

	// --- Validation --------------------------------------------------------------

	if err := validateSummaryOptions(summaryGithubStep); err != nil {
		fatal("%v", err)
	}
	if toolTimeout <= 0 {
		fatal("--tool-timeout must be greater than zero")
	}

	workspace := "."
	if args := fs.Args(); len(args) > 0 {
		workspace = args[0]
	}

	absWorkspace, err := resolveWorkspace(workspace)
	if err != nil {
		fatal("%v", err)
	}

	if err := validateSummaryFile(absWorkspace, summaryFile); err != nil {
		fatal("%v", err)
	}

	// --- Discovery ---------------------------------------------------------------

	var log io.Writer = os.Stderr
	if quiet {
		log = io.Discard
	}

	logf := func(format string, a ...any) {
		_, _ = fmt.Fprintf(log, format, a...)
	}

	logf("[%s] discovering files in %s...\n", appName, absWorkspace)
	files, err := discover.Files(absWorkspace, pathList, ignList)
	if err != nil {
		fatal("discover: %v", err)
	}
	logf("[%s] %d file(s) found\n", appName, len(files))

	for _, warning := range pathignore.Warnings(pathList, files) {
		logf("[%s] warning -- %s\n", appName, warning)
	}
	files = pathignore.Filter(files)

	if len(files) == 0 {
		out := aggregate(absWorkspace, files, nil, nil, 0, 0, "pass")
		emitOutput(out, pretty, summaryMarkdown, summaryFile, summaryGithubStep)
		os.Exit(0)
	}

	// --- Tool execution ----------------------------------------------------------

	ctx := context.Background()
	assignments := runner.ForTools(files)

	// Fix phase: run every fixer silently so that all files are in their final
	// fixed state before any tool's check pass reports findings.
	// Without this, a checker that runs between two fixers would see a
	// mid-fix snapshot and report transient findings that vanish on the next run.
	if fix {
		for _, a := range assignments {
			runner.RunFixWithTimeout(ctx, a.Def, absWorkspace, a.Files, log, toolTimeout)
		}
	}

	// Check phase: run every tool in check-only mode and collect findings.
	var results []runner.Result
	var wsConfigs []configUse
	toolsRun, toolsSkipped := 0, 0
	anyFail := false
	anyError := false

	for _, a := range assignments {
		result := runner.RunWithTimeout(ctx, a.Def, absWorkspace, false, a.Files, log, toolTimeout)
		switch result.Status {
		case "fail":
			anyFail = true
			toolsRun++
		case "error":
			anyError = true
			toolsRun++
		case "pass":
			toolsRun++
		case "skip":
			toolsSkipped++
		}
		if result.Status != "pass" && result.Status != "skip" {
			results = append(results, result)
		}
		if result.WorkspaceConfig != "" {
			wsConfigs = append(wsConfigs, configUse{Tool: result.Tool, Config: result.WorkspaceConfig})
		}
	}

	logf("[%s] done -- %d finding(s)\n", appName, totalFindings(results))

	// --- Output ------------------------------------------------------------------

	outStatus := "pass"
	if anyError {
		outStatus = "error"
	} else if anyFail {
		outStatus = "fail"
	}

	out := aggregate(absWorkspace, files, results, wsConfigs, toolsRun, toolsSkipped, outStatus)
	emitOutput(out, pretty, summaryMarkdown, summaryFile, summaryGithubStep)

	// Exit codes: 0 = pass, 1 = findings, 2 = tool execution error.
	// A tool error takes precedence over findings: a clean lint result could not
	// be produced, which is more severe than ordinary findings.
	if anyError {
		os.Exit(2)
	}
	if anyFail {
		os.Exit(1)
	}
}

// --- Helpers -------------------------------------------------------------------------

func resolveWorkspace(dir string) (string, error) {
	info, err := os.Stat(dir)
	if err != nil {
		return "", fmt.Errorf("workspace %q: %w", dir, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("workspace %q is not a directory", dir)
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", fmt.Errorf("workspace %q: %w", dir, err)
	}
	return abs, nil
}

func totalFindings(results []runner.Result) int {
	total := 0
	for _, r := range results {
		total += len(r.Findings)
	}
	return total
}

// configUse records a tool that used a workspace-supplied configuration file.
type configUse struct {
	Tool   string `json:"tool"`
	Config string `json:"config"`
}

type output struct {
	Status           string          `json:"status"`
	Workspace        string          `json:"workspace"`
	FilesDiscovered  int             `json:"files_discovered"`
	ToolsRun         int             `json:"tools_run"`
	ToolsSkipped     int             `json:"tools_skipped"`
	TotalFindings    int             `json:"total_findings"`
	Tools            []runner.Result `json:"tools"`
	WorkspaceConfigs []configUse     `json:"workspace_configs,omitempty"`
}

func aggregate(workspace string, files []string, results []runner.Result, wsConfigs []configUse, toolsRun, toolsSkipped int, status string) output {
	if results == nil {
		results = []runner.Result{}
	}
	return output{
		Status:           status,
		Workspace:        workspace,
		FilesDiscovered:  len(files),
		ToolsRun:         toolsRun,
		ToolsSkipped:     toolsSkipped,
		TotalFindings:    totalFindings(results),
		Tools:            results,
		WorkspaceConfigs: wsConfigs,
	}
}

func emit(out output, pretty bool) {
	var b []byte
	var err error
	if pretty {
		b, err = json.MarshalIndent(out, "", "  ")
	} else {
		b, err = json.Marshal(out)
	}
	if err != nil {
		fatal("marshal output: %v", err)
	}
	fmt.Println(string(b))
}

func fatal(format string, a ...any) {
	fmt.Fprintf(os.Stderr, "[%s] error: %s\n", appName, fmt.Sprintf(format, a...))
	os.Exit(2)
}
