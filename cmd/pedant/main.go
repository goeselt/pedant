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

	"github.com/goeselt/pedant/internal/classify"
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
	fs := flag.NewFlagSet(appName, flag.ExitOnError)

	var (
		nofix    bool
		pretty   bool
		quiet    bool
		pathList multiFlag
		ignList  multiFlag
	)

	fs.BoolVar(&nofix, "nofix", false, "Check only, do not modify files")
	fs.BoolVar(&nofix, "no-fix", false, "Alias for --nofix")
	fs.BoolVar(&pretty, "pretty", false, "Pretty-print JSON output")
	fs.BoolVar(&quiet, "quiet", false, "Suppress progress output (JSON only on stdout)")
	fs.BoolVar(&quiet, "q", false, "Alias for --quiet")
	fs.Var(&pathList, "path", "Restrict scan to this path or file (repeatable)")
	fs.Var(&ignList, "ignore", "Exclude this path or file from scan (repeatable)")

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr,
			"Usage: %s [options] [workspace]\n\n"+
				"Discover, classify, and lint/format files in a Git repository.\n\n"+
				"Arguments:\n"+
				"  [workspace]          Repository root to lint (default: current directory)\n\n"+
				"Options:\n",
			appName)
		fs.PrintDefaults()
		fmt.Fprintf(os.Stderr,
			"\nProgress output goes to stderr. JSON result goes to stdout.\n"+
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

	workspace := "."
	if args := fs.Args(); len(args) > 0 {
		workspace = args[0]
	}

	absWorkspace, err := resolveWorkspace(workspace)
	if err != nil {
		fatal("%v", err)
	}

	fix := !nofix

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
	filteredFiles := pathignore.Filter(files)
	if ignored := len(files) - len(filteredFiles); ignored > 0 {
		logf("[%s] %d file(s) ignored by default path ignores\n", appName, ignored)
	}
	files = filteredFiles

	if len(files) == 0 {
		out := aggregate(absWorkspace, files, nil, true)
		emit(out, pretty)
		os.Exit(0)
	}

	assignments := classify.ForTools(files)

	ctx := context.Background()
	var results []runner.Result
	anyFail := false
	anyError := false

	for _, a := range assignments {
		def, ok := toolByName(a.Tool)
		if !ok {
			continue
		}
		result := runner.Run(ctx, def, absWorkspace, fix, a.Files, log)
		switch result.Status {
		case "fail":
			anyFail = true
		case "error":
			anyError = true
		}
		if result.Status != "pass" && result.Status != "skip" {
			results = append(results, result)
		}
	}

	logf("[%s] done -- %d finding(s)\n", appName, totalFindings(results))

	out := aggregate(absWorkspace, files, results, !anyFail && !anyError)
	emit(out, pretty)

	// Exit codes: 0 = pass, 1 = findings, 2 = tool execution error.
	// A tool error takes precedence over findings: it means a clean lint
	// result could not be produced, which is more severe than ordinary findings.
	if anyError {
		os.Exit(2)
	}
	if anyFail {
		os.Exit(1)
	}
}

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

func toolByName(name string) (runner.ToolDef, bool) {
	for _, t := range runner.Registry {
		if t.Name == name {
			return t, true
		}
	}
	return runner.ToolDef{}, false
}

func totalFindings(results []runner.Result) int {
	total := 0
	for _, r := range results {
		total += len(r.Findings)
	}
	return total
}

type output struct {
	Status          string          `json:"status"`
	Workspace       string          `json:"workspace"`
	FilesDiscovered int             `json:"files_discovered"`
	TotalFindings   int             `json:"total_findings"`
	Tools           []runner.Result `json:"tools"`
}

func aggregate(workspace string, files []string, results []runner.Result, pass bool) output {
	status := "pass"
	if !pass {
		status = "fail"
	}
	if results == nil {
		results = []runner.Result{}
	}
	return output{
		Status:          status,
		Workspace:       workspace,
		FilesDiscovered: len(files),
		TotalFindings:   totalFindings(results),
		Tools:           results,
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
