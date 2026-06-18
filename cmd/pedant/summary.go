package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/goeselt/pedant/internal/runner"
)

// validateSummaryFile returns an error if summaryFile resolves to a path outside
// workspace. An empty summaryFile is accepted.
func validateSummaryFile(workspace, summaryFile string) error {
	if summaryFile == "" {
		return nil
	}
	abs, err := filepath.Abs(summaryFile)
	if err != nil {
		return fmt.Errorf("summary-file: %w", err)
	}
	rel, err := filepath.Rel(workspace, abs)
	if err != nil {
		return fmt.Errorf("summary-file %q: %w", summaryFile, err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("summary-file %q is outside the workspace %q", summaryFile, workspace)
	}
	return nil
}

func validateSummaryOptions(summaryGithubStep bool) error {
	if summaryGithubStep && os.Getenv("GITHUB_STEP_SUMMARY") == "" {
		return fmt.Errorf("--summary-github-step requires GITHUB_STEP_SUMMARY to be set")
	}
	return nil
}

func emitOutput(out output, pretty bool, summaryMarkdown bool, summaryFile string, summaryGithubStep bool) {
	if !summaryMarkdown && summaryFile == "" && !summaryGithubStep {
		emit(out, pretty)
		return
	}

	report := renderMarkdownSummary(out)

	if summaryMarkdown {
		fmt.Print(report)
	}

	if summaryFile != "" {
		if err := os.WriteFile(summaryFile, []byte(report), 0o644); err != nil {
			fatal("write summary file: %v", err)
		}
	}

	if summaryGithubStep {
		// validateSummaryOptions already guarantees GITHUB_STEP_SUMMARY is set.
		path := os.Getenv("GITHUB_STEP_SUMMARY")
		f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			fatal("open GitHub step summary: %v", err)
		}
		if _, err := f.WriteString(report); err != nil {
			_ = f.Close()
			fatal("write GitHub step summary: %v", err)
		}
		if err := f.Close(); err != nil {
			fatal("close GitHub step summary: %v", err)
		}
	}
}

func renderMarkdownSummary(out output) string {
	var b strings.Builder
	fmt.Fprintln(&b, "## Pedant Summary")
	fmt.Fprintln(&b)
	fmt.Fprintf(&b, "- Status: `%s`\n", out.Status)
	fmt.Fprintf(&b, "- Files checked: %d\n", out.FilesDiscovered)
	fmt.Fprintf(&b, "- Findings: %d\n", out.TotalFindings)
	fmt.Fprintf(&b, "- Tools with findings/errors: %d\n", len(out.Tools))

	if len(out.Tools) == 0 {
		return b.String()
	}

	fmt.Fprintln(&b)
	fmt.Fprintln(&b, "| Tool | Status | Findings |")
	fmt.Fprintln(&b, "| --- | --- | ---: |")
	for _, result := range out.Tools {
		fmt.Fprintf(&b, "| `%s` | `%s` | %d |\n", tableText(result.Tool), tableText(result.Status), len(result.Findings))
	}

	for _, result := range out.Tools {
		fmt.Fprintln(&b)
		fmt.Fprintf(&b, "### %s\n", markdownHeading(result.Tool))

		if result.Error != "" {
			fmt.Fprintln(&b)
			fmt.Fprintf(&b, "Error: <code>%s</code>\n", htmlCodeText(result.Error))
		}

		if len(result.Findings) == 0 {
			continue
		}

		fmt.Fprintln(&b)
		fmt.Fprintln(&b, "| Location | Rule | Message |")
		fmt.Fprintln(&b, "| --- | --- | --- |")
		for _, finding := range result.Findings {
			fmt.Fprintf(
				&b,
				"| `%s` | %s | %s |\n",
				tableText(location(finding)),
				ruleCell(finding),
				tableText(finding.Message),
			)
		}
	}

	return b.String()
}

func location(f runner.Finding) string {
	var sb strings.Builder
	sb.WriteString(f.File)
	if f.Line > 0 {
		fmt.Fprintf(&sb, ":%d", f.Line)
		if f.Col > 0 {
			fmt.Fprintf(&sb, ":%d", f.Col)
		}
	}
	return sb.String()
}

func ruleCell(f runner.Finding) string {
	if f.Rule == "" {
		return ""
	}
	return "<code>" + htmlCodeText(f.Rule) + "</code>"
}

func markdownHeading(s string) string {
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	return strings.TrimSpace(s)
}

// tableText prepares s for use as plain text in a GFM table cell.
// HTML special characters are escaped so that tool output cannot inject markup.
// Note: content placed inside backtick code spans is already protected from HTML
// interpretation by the browser, but we escape anyway for defence in depth.
func tableText(s string) string {
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "|", `\|`)
	return strings.TrimSpace(s)
}

// htmlCodeText prepares s for use inside a <code>…</code> HTML element,
// including inside GFM table cells where a bare | would split a column.
func htmlCodeText(s string) string {
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "|", "&#124;")
	return strings.TrimSpace(s)
}
