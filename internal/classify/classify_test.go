package classify

import (
	"testing"
)

func TestMatchGlob(t *testing.T) {
	t.Parallel()

	tests := []struct {
		pattern string
		file    string
		want    bool
	}{
		// basename patterns
		{"*.go", "main.go", true},
		{"*.go", "src/main.go", true},
		{"*.go", "main.sh", false},
		{"*.sh", "deploy.sh", true},
		{"*.sh", "deploy.go", false},
		// Dockerfile patterns
		{"Dockerfile", "Dockerfile", true},
		{"Dockerfile", "src/Dockerfile", true},
		{"Dockerfile", "Dockerfile.dev", false},
		{"Dockerfile.*", "Dockerfile.dev", true},
		{"Dockerfile.*", "Dockerfile", false},
		{"*.dockerfile", "app.dockerfile", true},
		// path patterns
		{".github/workflows/*.yml", ".github/workflows/ci.yml", true},
		{".github/workflows/*.yml", ".github/workflows/ci.yaml", false},
		{".github/workflows/*.yml", "other/ci.yml", false},
		{".github/workflows/*.yaml", ".github/workflows/deploy.yaml", true},
		// all-files (empty globs handled separately, not tested here)
	}

	for _, tt := range tests {
		t.Run(tt.pattern+"/"+tt.file, func(t *testing.T) {
			t.Parallel()
			got := matchGlob(tt.pattern, tt.file)
			if got != tt.want {
				t.Errorf("matchGlob(%q, %q) = %v, want %v", tt.pattern, tt.file, got, tt.want)
			}
		})
	}
}

func TestForTools(t *testing.T) {
	t.Parallel()

	files := []string{
		"main.go",
		"script.sh",
		"README.md",
		"config.yml",
		".github/workflows/ci.yml",
		"Dockerfile",
	}

	assignments := ForTools(files)

	byTool := make(map[string][]string)
	for _, a := range assignments {
		byTool[a.Tool] = a.Files
	}

	// editorconfig gets all files
	if len(byTool["editorconfig"]) != len(files) {
		t.Errorf("editorconfig: got %d files, want %d", len(byTool["editorconfig"]), len(files))
	}
	// shellcheck and shfmt get only .sh
	for _, tool := range []string{"shellcheck", "shfmt"} {
		if len(byTool[tool]) != 1 || byTool[tool][0] != "script.sh" {
			t.Errorf("%s: got %v, want [script.sh]", tool, byTool[tool])
		}
	}
	// actionlint gets only .github/workflows/*.yml
	if len(byTool["actionlint"]) != 1 || byTool["actionlint"][0] != ".github/workflows/ci.yml" {
		t.Errorf("actionlint: got %v, want [.github/workflows/ci.yml]", byTool["actionlint"])
	}
	// hadolint gets Dockerfile
	if len(byTool["hadolint"]) != 1 || byTool["hadolint"][0] != "Dockerfile" {
		t.Errorf("hadolint: got %v, want [Dockerfile]", byTool["hadolint"])
	}
	// main.go: not matched by any text/yaml/shell tool
	for _, tool := range []string{"shellcheck", "shfmt", "yamllint", "markdownlint", "textlint"} {
		for _, f := range byTool[tool] {
			if f == "main.go" {
				t.Errorf("%s unexpectedly matched main.go", tool)
			}
		}
	}
}
