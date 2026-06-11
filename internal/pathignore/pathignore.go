// Package pathignore centralizes workspace paths that pedant should not pass to tools.
package pathignore

import (
	"fmt"
	"path/filepath"
	"strings"
)

// DefaultDirs are generated, dependency, cache, or temporary directories that
// pedant excludes from all bundled tool runs.
var DefaultDirs = []string{
	".cache",
	".git",
	".next",
	".nuxt",
	"build",
	"coverage",
	"dist",
	"node_modules",
	"out",
	"output",
	"public",
	"target",
	"tmp",
	"vendor",
}

// Filter removes files under DefaultDirs.
func Filter(files []string) []string {
	out := make([]string, 0, len(files))
	for _, file := range files {
		if !MatchesDefaultDir(file) {
			out = append(out, file)
		}
	}
	return out
}

// Warnings reports explicit --path selections that resolve only to files under
// DefaultDirs, making the implicit skip visible without turning it into an error.
func Warnings(selectedPaths, files []string) []string {
	if len(selectedPaths) == 0 || len(files) == 0 {
		return nil
	}

	var warnings []string
	for _, selectedPath := range selectedPaths {
		ignoredByDir := make(map[string]int)
		for _, file := range files {
			if pathSelectsFile(selectedPath, file) {
				if dir, ok := matchingDefaultDir(file); ok {
					ignoredByDir[dir]++
				}
			}
		}
		for _, dir := range DefaultDirs {
			count := ignoredByDir[dir]
			if count == 0 {
				continue
			}
			warnings = append(warnings, fmt.Sprintf(
				"--path %s selects %d file(s) ignored by default path ignore %s/",
				selectedPath, count, dir,
			))
		}
	}
	return warnings
}

// MatchesDefaultDir reports whether path is inside any DefaultDirs entry.
func MatchesDefaultDir(path string) bool {
	_, ok := matchingDefaultDir(path)
	return ok
}

func matchingDefaultDir(path string) (string, bool) {
	path = cleanRelativePath(path)
	for _, dir := range DefaultDirs {
		if path == dir || strings.HasPrefix(path, dir+"/") {
			return dir, true
		}
	}
	return "", false
}

func pathSelectsFile(selectedPath, file string) bool {
	selectedPath = cleanRelativePath(selectedPath)
	file = cleanRelativePath(file)
	if selectedPath == "." {
		return true
	}
	return file == selectedPath || strings.HasPrefix(file, strings.TrimSuffix(selectedPath, "/")+"/")
}

func cleanRelativePath(path string) string {
	path = filepath.ToSlash(filepath.Clean(path))
	path = strings.TrimPrefix(path, "./")
	return path
}
