package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/goeselt/pedant/internal/runner"
)

func TestRenderMarkdownSummaryPass(t *testing.T) {
	out := output{
		Status:          "pass",
		Workspace:       "/work",
		FilesDiscovered: 12,
		TotalFindings:   0,
		Tools:           []runner.Result{},
	}

	got := renderMarkdownSummary(out)

	for _, want := range []string{
		"## Pedant Summary",
		"- Status: `pass`",
		"- Files checked: 12",
		"- Findings: 0",
		"- Tools with findings/errors: 0",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("summary missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "| Tool | Status | Findings |") {
		t.Fatalf("pass summary should not include a tool table:\n%s", got)
	}
}

func TestRenderMarkdownSummaryFindingsAndErrors(t *testing.T) {
	out := output{
		Status:          "error",
		Workspace:       "/work",
		FilesDiscovered: 3,
		TotalFindings:   2,
		Tools: []runner.Result{
			{
				Tool:   "eslint",
				Status: "fail",
				Findings: []runner.Finding{
					{File: "src/app.ts", Line: 12, Col: 5, Rule: "no-unused-vars", Message: "foo is defined but never used"},
					{File: "src/table.ts", Line: 4, Rule: "custom|rule", Message: "message with | pipe"},
				},
			},
			{
				Tool:   "actionlint",
				Status: "error",
				Error:  "could not parse output",
			},
		},
	}

	got := renderMarkdownSummary(out)

	for _, want := range []string{
		"- Status: `error`",
		"- Files checked: 3",
		"- Findings: 2",
		"- Tools with findings/errors: 2",
		"| `eslint` | `fail` | 2 |",
		"| `actionlint` | `error` | 0 |",
		"### eslint",
		"| `src/app.ts:12:5` | `no-unused-vars` | foo is defined but never used |",
		"| `src/table.ts:4` | `custom\\|rule` | message with \\| pipe |",
		"### actionlint",
		"Error: could not parse output",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("summary missing %q:\n%s", want, got)
		}
	}
}

func TestValidateSummaryOptions(t *testing.T) {
	tests := []struct {
		name              string
		format            string
		file              string
		githubStepSummary bool
		wantErr           bool
	}{
		{name: "disabled"},
		{name: "explicit markdown without destination", format: "markdown"},
		{name: "explicit markdown with file", format: "markdown", file: "summary.md"},
		{name: "explicit markdown with github", format: "markdown", githubStepSummary: true},
		{name: "file implies markdown", file: "summary.md"},
		{name: "github implies markdown", githubStepSummary: true},
		{name: "unsupported format", format: "html", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateSummaryOptions(tc.format, tc.file, tc.githubStepSummary)
			if tc.wantErr && err == nil {
				t.Fatal("expected error")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestEmitSummaryWritesFileAndGitHubStepSummary(t *testing.T) {
	dir := t.TempDir()
	summaryPath := filepath.Join(dir, "summary.md")
	stepSummaryPath := filepath.Join(dir, "step-summary.md")
	t.Setenv("GITHUB_STEP_SUMMARY", stepSummaryPath)

	out := output{
		Status:          "pass",
		Workspace:       "/work",
		FilesDiscovered: 1,
		TotalFindings:   0,
		Tools:           []runner.Result{},
	}

	emitOutput(out, false, "", summaryPath, true)

	summary, err := os.ReadFile(summaryPath)
	if err != nil {
		t.Fatalf("read summary file: %v", err)
	}
	stepSummary, err := os.ReadFile(stepSummaryPath)
	if err != nil {
		t.Fatalf("read step summary: %v", err)
	}
	if string(summary) != string(stepSummary) {
		t.Fatalf("summary destinations differ:\nfile:\n%s\nstep:\n%s", summary, stepSummary)
	}
	if !strings.Contains(string(summary), "- Status: `pass`") {
		t.Fatalf("summary missing status:\n%s", summary)
	}
}
