package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/goeselt/pedant/internal/runner"
)

const summaryMarkdown = "markdown"

func validateSummaryOptions(format, file string, githubStepSummary bool) error {
	if format != "" && format != summaryMarkdown {
		return fmt.Errorf("unsupported summary format %q (supported: %s)", format, summaryMarkdown)
	}
	if githubStepSummary && os.Getenv("GITHUB_STEP_SUMMARY") == "" {
		return fmt.Errorf("--github-step-summary requires GITHUB_STEP_SUMMARY to be set")
	}
	return nil
}

func emitOutput(out output, pretty bool, format, file string, githubStepSummary bool) {
	if format == "" && file == "" && !githubStepSummary {
		emit(out, pretty)
		return
	}
	stdoutFormat := format
	if format == "" {
		format = summaryMarkdown
	}

	var report string
	switch format {
	case summaryMarkdown:
		report = renderMarkdownSummary(out)
	default:
		fatal("unsupported summary format %q", format)
	}

	if stdoutFormat != "" {
		fmt.Print(report)
	}

	if file != "" {
		if err := os.WriteFile(file, []byte(report), 0o644); err != nil {
			fatal("write summary file: %v", err)
		}
	}

	if githubStepSummary {
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
			fmt.Fprintf(&b, "Error: %s\n", tableText(result.Error))
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
	return "`" + tableText(f.Rule) + "`"
}

func markdownHeading(s string) string {
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	return strings.TrimSpace(s)
}

func tableText(s string) string {
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "|", `\|`)
	return strings.TrimSpace(s)
}
