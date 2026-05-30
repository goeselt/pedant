// Package discover finds lintable files in a Git workspace using git ls-files.
package discover

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Files returns workspace-relative paths of all lintable files.
// It delegates to git ls-files so that .gitignore rules are respected hierarchically.
// If paths is non-empty, the scan is restricted to those subdirectories or files.
// If ignore is non-empty, those paths are excluded using git's :(exclude) pathspec magic.
func Files(workspace string, paths, ignore []string) ([]string, error) {
	// -z: NUL-terminated output so filenames containing newlines or quotes
	// round-trip correctly. Without it, git would also quote unusual filenames,
	// which we would then have to unquote.
	args := []string{"ls-files", "-z", "--cached", "--others", "--exclude-standard"}
	if len(paths) > 0 || len(ignore) > 0 {
		args = append(args, "--")
		args = append(args, paths...)
		for _, ig := range ignore {
			args = append(args, ":(exclude)"+ig)
		}
	}

	cmd := exec.Command("git", args...)
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
		// Skip files tracked in the index but deleted from the working tree.
		if _, err := os.Stat(filepath.Join(workspace, l)); err != nil {
			continue
		}
		seen[l] = true
		files = append(files, l)
	}
	return files, nil
}
