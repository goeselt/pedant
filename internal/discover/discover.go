// Package discover finds lintable files in a Git workspace using git ls-files.
package discover

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// rejectPathspecMagic returns an error if entry starts with ":", the git pathspec magic signature.
// Both the long form ":(...)" and the short forms (":!", ":^", ":/") are rejected because they would inject pathspec
// magic beyond the :(exclude) prefix that Files itself adds.
func rejectPathspecMagic(kind, entry string) error {
	if strings.HasPrefix(entry, ":") {
		return fmt.Errorf("%s %q: pathspec magic is not allowed", kind, entry)
	}
	return nil
}

// Files returns workspace-relative paths of all lintable files.
// It delegates to git ls-files so that .gitignore rules are respected hierarchically.
// If paths is non-empty, the scan is restricted to those subdirectories or files.
// If ignore is non-empty, those paths are excluded using git's :(exclude) pathspec magic.
// Values starting with ":" are rejected because they would inject additional pathspec magic and could cause unexpected
// file selection or exclusion.
func Files(workspace string, paths, ignore []string) ([]string, error) {
	for _, p := range paths {
		if err := rejectPathspecMagic("path", p); err != nil {
			return nil, err
		}
	}
	for _, ig := range ignore {
		if err := rejectPathspecMagic("ignore", ig); err != nil {
			return nil, err
		}
	}

	// -z: NUL-terminated output so filenames containing newlines or quotes round-trip correctly.
	// Without it, git would also quote unusual filenames, which we would then have to unquote.
	args := []string{"ls-files", "-z", "--cached", "--others", "--exclude-standard"}
	if len(paths) > 0 || len(ignore) > 0 {
		args = append(args, "--")
		args = append(args, paths...)
		for _, ig := range ignore {
			args = append(args, ":(exclude)"+ig)
		}
	}

	cmd := exec.CommandContext(context.Background(), "git", args...)
	cmd.Dir = workspace

	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git ls-files: %w", err)
	}

	raw := strings.TrimRight(string(out), "\x00")
	if raw == "" {
		return nil, nil
	}

	entries := strings.Split(raw, "\x00")
	files := make([]string, 0, len(entries))
	seen := make(map[string]bool, len(entries))
	for _, l := range entries {
		if l == "" || seen[l] {
			continue
		}
		// Skip files that are absent or are symlinks.
		// Symlinks can point outside the workspace and would cause linters to read unintended files (e.g. /etc/passwd)
		// whose content might surface in tool output or the step summary.
		info, err := os.Lstat(filepath.Join(workspace, l))
		if err != nil || info.Mode()&os.ModeSymlink != 0 {
			continue
		}
		seen[l] = true
		files = append(files, l)
	}
	return files, nil
}
