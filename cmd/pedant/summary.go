package main

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/goeselt/pedant/internal/runner"
)

// resolveSummaryFile resolves summaryFile against workspace and returns the
// absolute path to write to. Relative paths are workspace-relative (matching
// the action input contract). Symlinks in the parent directory are resolved
// before the containment check so that a symlinked directory planted inside
// the workspace cannot redirect the write outside of it.
// An empty summaryFile returns "".
func resolveSummaryFile(workspace, summaryFile string) (string, error) {
	if summaryFile == "" {
		return "", nil
	}
	abs := summaryFile
	if !filepath.IsAbs(abs) {
		abs = filepath.Join(workspace, abs)
	}
	abs = filepath.Clean(abs)

	realWorkspace, err := filepath.EvalSymlinks(workspace)
	if err != nil {
		return "", fmt.Errorf("summary-file: resolve workspace: %w", err)
	}
	// The summary file itself may not exist yet, but its parent directory must
	// exist for the write to succeed; resolving it up front also fails early
	// instead of after all tools have run.
	realDir, err := filepath.EvalSymlinks(filepath.Dir(abs))
	if err != nil {
		return "", fmt.Errorf("summary-file %q: %w", summaryFile, err)
	}
	resolved := filepath.Join(realDir, filepath.Base(abs))

	rel, err := filepath.Rel(realWorkspace, resolved)
	if err != nil {
		return "", fmt.Errorf("summary-file %q: %w", summaryFile, err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("summary-file %q is outside the workspace %q", summaryFile, workspace)
	}
	// Fail fast when the target itself is a symlink instead of after all tools
	// have run. writeSummaryFile enforces the same rule with O_NOFOLLOW, which
	// also covers a link created between this check and the write.
	if info, err := os.Lstat(resolved); err == nil && info.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("summary-file %q is a symlink; refusing to follow it", summaryFile)
	}
	return resolved, nil
}

// writeSummaryFile writes report to path without following a symlink at path.
// The workspace content is untrusted: a symlink planted at the configured
// summary path could otherwise redirect the write to an arbitrary file, such
// as the GitHub Actions file-command files mounted into the container.
func writeSummaryFile(path, report string) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC|syscall.O_NOFOLLOW, 0o644)
	if err != nil {
		if errors.Is(err, syscall.ELOOP) {
			return fmt.Errorf("summary-file %q is a symlink; refusing to follow it", path)
		}
		return err
	}
	if _, err := f.WriteString(report); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}

func validateSummaryOptions(summaryGithubStep bool) error {
	if summaryGithubStep && os.Getenv("GITHUB_STEP_SUMMARY") == "" {
		return fmt.Errorf("--summary-github-step requires GITHUB_STEP_SUMMARY to be set")
	}
	return nil
}

// writeGitHubOutputs writes action output variables to $GITHUB_OUTPUT when set.
// Non-fatal: if the file cannot be opened, outputs are silently skipped.
func writeGitHubOutputs(out output) {
	path := os.Getenv("GITHUB_OUTPUT")
	if path == "" {
		return
	}
	// O_CREATE is intentionally omitted: the GitHub Actions runner creates
	// GITHUB_OUTPUT before the job starts. If the file does not exist (e.g.
	// because GITHUB_OUTPUT was overwritten by a prior step), we skip silently
	// rather than writing to an arbitrary path.
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer func() { _ = f.Close() }()
	_, _ = fmt.Fprintf(f, "status=%s\ntotal-findings=%d\nfiles-discovered=%d\ntools-run=%d\ntools-skipped=%d\n",
		out.Status, out.TotalFindings, out.FilesDiscovered, out.ToolsRun, out.ToolsSkipped)
	// Multiline output format: required for values that contain newlines.
	// A random delimiter prevents injection if the Markdown content ever
	// contains the delimiter string.
	delim := outputDelimiter()
	_, _ = fmt.Fprintf(f, "summary<<%s\n%s%s\n", delim, renderMarkdownSummary(out), delim)
}

// outputDelimiter returns a random string suitable for use as a GITHUB_OUTPUT
// multiline delimiter.
func outputDelimiter() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "pedant_eof"
	}
	return "pedant_" + hex.EncodeToString(b)
}

func emitOutput(out output, pretty bool, summaryMarkdown bool, summaryFile string, summaryGithubStep bool) {
	writeGitHubOutputs(out)

	// --summary-file and --summary-github-step write Markdown to their own
	// destinations independently; they do not suppress JSON on stdout.
	// Only --summary-markdown suppresses JSON because both compete for stdout.
	if summaryFile != "" || summaryGithubStep || summaryMarkdown {
		report := renderMarkdownSummary(out)

		if summaryMarkdown {
			fmt.Print(report)
		}

		if summaryFile != "" {
			if err := writeSummaryFile(summaryFile, report); err != nil {
				fatal("write summary file: %v", err)
			}
		}

		if summaryGithubStep {
			// validateSummaryOptions already guarantees GITHUB_STEP_SUMMARY is set.
			// O_CREATE is intentionally omitted: the runner creates GITHUB_STEP_SUMMARY
			// before the job starts. If the path was overwritten by a prior step we
			// skip silently rather than creating a file at an arbitrary location.
			path := os.Getenv("GITHUB_STEP_SUMMARY")
			f, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0o644)
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

	if !summaryMarkdown {
		emit(out, pretty)
	}
}

func renderMarkdownSummary(out output) string {
	var b strings.Builder
	fmt.Fprintln(&b, "## Pedant Summary")
	fmt.Fprintln(&b)
	fmt.Fprintf(&b, "- Status: `%s`\n", out.Status)
	fmt.Fprintf(&b, "- Files checked: %d\n", out.FilesDiscovered)
	if out.ToolsSkipped > 0 {
		fmt.Fprintf(&b, "- Tools: %d run, %d skipped\n", out.ToolsRun, out.ToolsSkipped)
	} else {
		fmt.Fprintf(&b, "- Tools run: %d\n", out.ToolsRun)
	}
	fmt.Fprintf(&b, "- Findings: %d\n", out.TotalFindings)
	if len(out.WorkspaceConfigs) > 0 {
		fmt.Fprintf(&b, "- Workspace configs: %d\n", len(out.WorkspaceConfigs))
	}

	if len(out.Tools) == 0 && len(out.WorkspaceConfigs) == 0 {
		return b.String()
	}

	fmt.Fprintln(&b)
	fmt.Fprintln(&b, "| Tool | Status | Findings |")
	fmt.Fprintln(&b, "| --- | --- | ---: |")
	for _, result := range out.Tools {
		fmt.Fprintf(&b, "| <code>%s</code> | <code>%s</code> | %d |\n", htmlCodeText(result.Tool), htmlCodeText(result.Status), len(result.Findings))
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
				"| <code>%s</code> | %s | %s |\n",
				htmlCodeText(location(finding)),
				ruleCell(finding),
				tableText(finding.Message),
			)
		}
	}

	if len(out.WorkspaceConfigs) > 0 {
		fmt.Fprintln(&b)
		fmt.Fprintln(&b, "### Workspace Configs")
		fmt.Fprintln(&b)
		fmt.Fprintln(&b, "| Tool | Config |")
		fmt.Fprintln(&b, "| --- | --- |")
		for _, wc := range out.WorkspaceConfigs {
			fmt.Fprintf(&b, "| <code>%s</code> | <code>%s</code> |\n", htmlCodeText(wc.Tool), htmlCodeText(wc.Config))
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
// Markdown delimiters (backticks, brackets) are escaped as numeric character
// references: entity references render as literal text and cannot open code
// spans, links, or images, so attacker-influenced tool output (file names,
// lint messages) cannot inject links into PR comments or step summaries.
func tableText(s string) string {
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "`", "&#96;")
	s = strings.ReplaceAll(s, "[", "&#91;")
	s = strings.ReplaceAll(s, "]", "&#93;")
	s = strings.ReplaceAll(s, "|", `\|`)
	return strings.TrimSpace(s)
}

// htmlCodeText prepares s for use inside a <code>...</code> HTML element,
// including inside GFM table cells where a bare | would split a column.
// Markdown inline parsing still applies between inline HTML tags, so Markdown
// delimiters are escaped the same way as in tableText.
func htmlCodeText(s string) string {
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "`", "&#96;")
	s = strings.ReplaceAll(s, "[", "&#91;")
	s = strings.ReplaceAll(s, "]", "&#93;")
	s = strings.ReplaceAll(s, "|", "&#124;")
	return strings.TrimSpace(s)
}
