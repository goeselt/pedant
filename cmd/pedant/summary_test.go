package main

import (
	"bytes"
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
		"- Tools run: 0",
		"- Findings: 0",
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
		"- Tools run: 0",
		"- Findings: 2",
		"| `eslint` | `fail` | 2 |",
		"| `actionlint` | `error` | 0 |",
		"### eslint",
		"| `src/app.ts:12:5` | <code>no-unused-vars</code> | foo is defined but never used |",
		"| `src/table.ts:4` | <code>custom&#124;rule</code> | message with \\| pipe |",
		"### actionlint",
		"Error: <code>could not parse output</code>",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("summary missing %q:\n%s", want, got)
		}
	}
}

func TestRenderMarkdownSummaryErrorPipeNotEscaped(t *testing.T) {
	t.Parallel()

	out := output{
		Status:          "error",
		Workspace:       "/work",
		FilesDiscovered: 1,
		TotalFindings:   0,
		Tools: []runner.Result{
			{
				Tool:   "actionlint",
				Status: "error",
				Error:  "failed: pipe | in error",
			},
		},
	}

	got := renderMarkdownSummary(out)

	// Error is wrapped in <code>; | must be HTML-escaped (&#124;), not GFM-escaped (\|).
	if !strings.Contains(got, "Error: <code>failed: pipe &#124; in error</code>") {
		t.Fatalf("error should be in <code> element with HTML-escaped pipe:\n%s", got)
	}
	if strings.Contains(got, `\|`) {
		t.Fatalf("error paragraph should not contain GFM pipe escape:\n%s", got)
	}
}

func TestValidateSummaryOptions(t *testing.T) {
	tests := []struct {
		name              string
		summaryGithubStep bool
		setStepSummaryEnv bool
		wantErr           bool
	}{
		{name: "disabled"},
		{name: "github step with env var", summaryGithubStep: true, setStepSummaryEnv: true},
		{name: "github step without env var", summaryGithubStep: true, wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.setStepSummaryEnv {
				t.Setenv("GITHUB_STEP_SUMMARY", filepath.Join(t.TempDir(), "step-summary.md"))
			} else {
				t.Setenv("GITHUB_STEP_SUMMARY", "")
			}
			err := validateSummaryOptions(tc.summaryGithubStep)
			if tc.wantErr && err == nil {
				t.Fatal("expected error")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestEmitSummaryMarkdownStdout(t *testing.T) {
	// Captures os.Stdout -- must not run in parallel.
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w

	out := output{
		Status:          "pass",
		Workspace:       "/work",
		FilesDiscovered: 5,
		TotalFindings:   0,
		Tools:           []runner.Result{},
	}
	emitOutput(out, false, true, "", false)

	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	os.Stdout = old

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r); err != nil {
		t.Fatal(err)
	}
	got := buf.String()

	for _, want := range []string{
		"## Pedant Summary",
		"- Status: `pass`",
		"- Files checked: 5",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("stdout missing %q:\n%s", want, got)
		}
	}
	// JSON must not appear when --summary-markdown is active.
	if strings.Contains(got, `"status"`) {
		t.Fatalf("stdout must not contain JSON when --summary-markdown is set:\n%s", got)
	}
}

func TestRenderMarkdownSummaryToolCounts(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		run      int
		skipped  int
		wantLine string
	}{
		{
			name:     "no skipped",
			run:      8,
			skipped:  0,
			wantLine: "- Tools run: 8",
		},
		{
			name:     "with skipped",
			run:      8,
			skipped:  4,
			wantLine: "- Tools: 8 run, 4 skipped",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			out := output{
				Status:          "pass",
				Workspace:       "/work",
				FilesDiscovered: 10,
				ToolsRun:        tc.run,
				ToolsSkipped:    tc.skipped,
				TotalFindings:   0,
				Tools:           []runner.Result{},
			}
			got := renderMarkdownSummary(out)
			if !strings.Contains(got, tc.wantLine) {
				t.Fatalf("summary missing %q:\n%s", tc.wantLine, got)
			}
		})
	}
}

func TestEmitOutputJSONStillEmittedWithSummaryFile(t *testing.T) {
	// Captures os.Stdout -- must not run in parallel.
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w

	dir := t.TempDir()
	summaryPath := filepath.Join(dir, "summary.md")

	out := output{
		Status:          "pass",
		Workspace:       "/work",
		FilesDiscovered: 5,
		TotalFindings:   0,
		Tools:           []runner.Result{},
	}
	// --summary-file is set but NOT --summary-markdown: JSON must still appear on stdout.
	emitOutput(out, false, false, summaryPath, false)

	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	os.Stdout = old

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r); err != nil {
		t.Fatal(err)
	}
	got := buf.String()

	if !strings.Contains(got, `"status"`) {
		t.Fatalf("stdout must contain JSON when only --summary-file is set:\n%s", got)
	}
	if strings.Contains(got, "## Pedant Summary") {
		t.Fatalf("stdout must not contain Markdown when only --summary-file is set:\n%s", got)
	}

	// The summary file itself must contain Markdown.
	content, err := os.ReadFile(summaryPath)
	if err != nil {
		t.Fatalf("read summary file: %v", err)
	}
	if !strings.Contains(string(content), "## Pedant Summary") {
		t.Fatalf("summary file missing Markdown header:\n%s", content)
	}
}

func TestWriteGitHubOutputs(t *testing.T) {
	dir := t.TempDir()
	outputPath := filepath.Join(dir, "github_output")
	// Simulate the GitHub Actions runner creating GITHUB_OUTPUT before the job starts.
	if err := os.WriteFile(outputPath, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GITHUB_OUTPUT", outputPath)

	out := output{
		Status:          "fail",
		FilesDiscovered: 24,
		ToolsRun:        10,
		ToolsSkipped:    3,
		TotalFindings:   5,
		Tools:           []runner.Result{},
	}
	writeGitHubOutputs(out)

	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("read GITHUB_OUTPUT file: %v", err)
	}
	got := string(content)

	for _, want := range []string{
		"status=fail",
		"total-findings=5",
		"files-discovered=24",
		"tools-run=10",
		"tools-skipped=3",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("GITHUB_OUTPUT missing %q:\n%s", want, got)
		}
	}

	// Verify multiline summary output.
	var delim string
	for _, line := range strings.Split(got, "\n") {
		if strings.HasPrefix(line, "summary<<") {
			delim = strings.TrimPrefix(line, "summary<<")
			break
		}
	}
	if delim == "" {
		t.Fatalf("GITHUB_OUTPUT missing multiline summary line:\n%s", got)
	}
	if !strings.Contains(got, "## Pedant Summary") {
		t.Fatalf("GITHUB_OUTPUT summary missing Markdown header:\n%s", got)
	}
	// The closing delimiter must appear on its own line after the content.
	if !strings.Contains(got, "\n"+delim+"\n") {
		t.Fatalf("GITHUB_OUTPUT multiline summary missing closing delimiter %q:\n%s", delim, got)
	}
}

func TestWriteGitHubOutputsNoEnvVar(t *testing.T) {
	t.Setenv("GITHUB_OUTPUT", "")
	// Must not panic or error when GITHUB_OUTPUT is not set.
	writeGitHubOutputs(output{Status: "pass"})
}

func TestWriteGitHubOutputsDoesNotCreateMissingFile(t *testing.T) {
	nonExistent := filepath.Join(t.TempDir(), "does_not_exist.txt")
	t.Setenv("GITHUB_OUTPUT", nonExistent)

	writeGitHubOutputs(output{Status: "pass"})

	if _, err := os.Stat(nonExistent); !os.IsNotExist(err) {
		t.Error("writeGitHubOutputs must not create GITHUB_OUTPUT when the file does not exist")
	}
}

func TestRenderMarkdownSummaryWorkspaceConfigs(t *testing.T) {
	t.Parallel()

	out := output{
		Status:          "pass",
		Workspace:       "/work",
		FilesDiscovered: 5,
		TotalFindings:   0,
		Tools:           []runner.Result{},
		WorkspaceConfigs: []configUse{
			{Tool: "eslint", Config: "eslint.config.mjs"},
			{Tool: "prettier", Config: ".prettierrc"},
		},
	}

	got := renderMarkdownSummary(out)

	for _, want := range []string{
		"- Workspace configs: 2",
		"### Workspace Configs",
		"| <code>eslint</code> | <code>eslint.config.mjs</code> |",
		"| <code>prettier</code> | <code>.prettierrc</code> |",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("summary missing %q:\n%s", want, got)
		}
	}
}

func TestRenderMarkdownSummaryNoWorkspaceConfigs(t *testing.T) {
	t.Parallel()

	out := output{
		Status:          "pass",
		Workspace:       "/work",
		FilesDiscovered: 5,
		TotalFindings:   0,
		Tools:           []runner.Result{},
	}

	got := renderMarkdownSummary(out)

	if strings.Contains(got, "Workspace Configs") {
		t.Fatalf("summary should not contain workspace configs section when none present:\n%s", got)
	}
	if strings.Contains(got, "- Workspace configs:") {
		t.Fatalf("summary should not show workspace configs counter when zero:\n%s", got)
	}
}

func TestValidateSummaryFile(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()

	tests := []struct {
		name        string
		summaryFile string
		wantErr     bool
	}{
		{name: "empty allowed", summaryFile: "", wantErr: false},
		{name: "file in workspace", summaryFile: filepath.Join(workspace, "summary.md"), wantErr: false},
		{name: "nested in workspace", summaryFile: filepath.Join(workspace, "sub", "report.md"), wantErr: false},
		{name: "one level up", summaryFile: filepath.Join(workspace, "..", "summary.md"), wantErr: true},
		{name: "absolute outside workspace", summaryFile: filepath.Join(filepath.Dir(workspace), "outside.md"), wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := validateSummaryFile(workspace, tc.summaryFile)
			if tc.wantErr && err == nil {
				t.Fatalf("validateSummaryFile(%q, %q): expected error, got nil", workspace, tc.summaryFile)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("validateSummaryFile(%q, %q): unexpected error: %v", workspace, tc.summaryFile, err)
			}
		})
	}
}

func TestTableTextHTMLEscape(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  string
	}{
		{input: "plain text", want: "plain text"},
		{input: "<script>alert(1)</script>", want: "&lt;script&gt;alert(1)&lt;/script&gt;"},
		{input: "a & b", want: "a &amp; b"},
		{input: "a | b", want: `a \| b`},
		{input: "line1\nline2", want: "line1 line2"},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			t.Parallel()
			if got := tableText(tc.input); got != tc.want {
				t.Errorf("tableText(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestHtmlCodeText(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  string
	}{
		{input: "no-unused-vars", want: "no-unused-vars"},
		{input: "custom|rule", want: "custom&#124;rule"},
		{input: "<b>bold</b>", want: "&lt;b&gt;bold&lt;/b&gt;"},
		{input: "a & b", want: "a &amp; b"},
		{input: "line1\nline2", want: "line1 line2"},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			t.Parallel()
			if got := htmlCodeText(tc.input); got != tc.want {
				t.Errorf("htmlCodeText(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestEmitSummaryWritesFileAndGitHubStepSummary(t *testing.T) {
	dir := t.TempDir()
	summaryPath := filepath.Join(dir, "summary.md")
	stepSummaryPath := filepath.Join(dir, "step-summary.md")
	// Simulate the runner creating GITHUB_STEP_SUMMARY before the job starts.
	if err := os.WriteFile(stepSummaryPath, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GITHUB_STEP_SUMMARY", stepSummaryPath)

	out := output{
		Status:          "pass",
		Workspace:       "/work",
		FilesDiscovered: 1,
		TotalFindings:   0,
		Tools:           []runner.Result{},
	}

	emitOutput(out, false, false, summaryPath, true)

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
