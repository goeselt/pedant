package pathignore

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFilterRemovesDefaultIgnoredDirs(t *testing.T) {
	t.Parallel()

	files := []string{
		"README.md",
		"src/main.go",
		"tmp/CONTRIBUTING.md",
		"vendor/lib.go",
		"public/data.json",
	}

	got := Filter(files)
	want := []string{"README.md", "src/main.go"}

	if strings.Join(got, "\x00") != strings.Join(want, "\x00") {
		t.Fatalf("Filter() = %v, want %v", got, want)
	}
}

func TestWarningsForExplicitIgnoredPath(t *testing.T) {
	t.Parallel()

	warnings := Warnings(
		[]string{"tmp"},
		[]string{"tmp/CONTRIBUTING.md", "README.md"},
	)

	if len(warnings) != 1 {
		t.Fatalf("Warnings() = %v, want one warning", warnings)
	}
	if !strings.Contains(warnings[0], "--path tmp selects 1 file(s)") {
		t.Errorf("warning = %q, want selected file count", warnings[0])
	}
}

func TestBundledConfigsContainDefaultDirs(t *testing.T) {
	t.Parallel()

	root := filepath.Join("..", "..")
	checks := []struct {
		name   string
		path   string
		format func(string) string
	}{
		{
			name:   "prettier",
			path:   filepath.Join(root, "configs/prettier/.prettierignore"),
			format: func(dir string) string { return dir + "/" },
		},
		{
			name:   "markdownlint",
			path:   filepath.Join(root, "configs/markdownlint/.markdownlint-cli2.yaml"),
			format: func(dir string) string { return "- " + dir + "/**" },
		},
		{
			name:   "eslint",
			path:   filepath.Join(root, "configs/eslint/eslint.config.mjs"),
			format: func(dir string) string { return "'" + dir + "/**'" },
		},
		{
			name:   "editorconfig-checker",
			path:   filepath.Join(root, "configs/editorconfig/.editorconfig-checker.json"),
			format: escapeJSONRegexDir,
		},
	}

	for _, check := range checks {
		check := check
		t.Run(check.name, func(t *testing.T) {
			t.Parallel()

			content, err := os.ReadFile(check.path)
			if err != nil {
				t.Fatalf("ReadFile: %v", err)
			}
			text := string(content)
			for _, dir := range DefaultDirs {
				want := check.format(dir)
				if !strings.Contains(text, want) {
					t.Errorf("%s missing default ignored dir %q as %q", check.path, dir, want)
				}
			}
		})
	}
}

func escapeJSONRegexDir(dir string) string {
	dir = strings.ReplaceAll(dir, ".", "\\\\.")
	return dir + "/"
}
