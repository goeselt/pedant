// Package classify maps discovered files to the tools that should process them,
// using the glob patterns declared on each runner.ToolDef.
package classify

import (
	"path/filepath"
	"strings"

	"github.com/goeselt/pedant/internal/runner"
)

// Assignment pairs a tool name with its matched files.
type Assignment struct {
	Tool  string
	Files []string
}

// ForTools returns ordered tool assignments for the given file list.
func ForTools(files []string) []Assignment {
	var result []Assignment
	for _, t := range runner.Registry {
		matched := matchFiles(files, t.Globs)
		if len(matched) > 0 {
			result = append(result, Assignment{Tool: t.Name, Files: matched})
		}
	}
	return result
}

func matchFiles(files, globs []string) []string {
	if len(globs) == 0 {
		out := make([]string, len(files))
		copy(out, files)
		return out
	}
	var matched []string
	for _, f := range files {
		if matchesAny(f, globs) {
			matched = append(matched, f)
		}
	}
	return matched
}

func matchesAny(file string, globs []string) bool {
	for _, g := range globs {
		if matchGlob(g, file) {
			return true
		}
	}
	return false
}

// matchGlob matches a pattern against a file path.
// Patterns without '/' are matched against the basename only.
// Patterns with '/' are matched against every suffix of the path with the same segment count.
func matchGlob(pattern, file string) bool {
	if !strings.Contains(pattern, "/") {
		ok, _ := filepath.Match(pattern, filepath.Base(file))
		return ok
	}
	segs := strings.Split(file, "/")
	pSegs := strings.Split(pattern, "/")
	if len(pSegs) > len(segs) {
		return false
	}
	for i := 0; i <= len(segs)-len(pSegs); i++ {
		suffix := strings.Join(segs[i:i+len(pSegs)], "/")
		if ok, _ := filepath.Match(pattern, suffix); ok {
			return true
		}
	}
	return false
}
