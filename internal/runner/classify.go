package runner

import (
	"path/filepath"
	"strings"
)

// Assignment pairs a ToolDef with the files it should process.
type Assignment struct {
	Def   ToolDef
	Files []string
}

// ForTools returns ordered tool assignments for the given file list.
// Tools without matching files are omitted; their order follows Registry.
func ForTools(files []string) []Assignment {
	var result []Assignment
	for _, t := range Registry {
		matched := matchFiles(files, t.Globs)
		if len(matched) > 0 {
			result = append(result, Assignment{Def: t, Files: matched})
		}
	}
	return result
}

// -- Glob matching ------------------------------------------------------------------

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
// Patterns without '/' match against the basename only.
// Patterns with '/' match against every suffix of the path with the same segment count,
// so ".github/workflows/*.yml" matches at any depth.
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
