package runner

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"
)

func TestParsePrettier(t *testing.T) {
	t.Parallel()

	stdout := `[warn] src/foo.js
[warn] src/bar.ts
[warn] Code style issues found in 2 files. Forgot to run Prettier?
`
	findings, err := parsePrettier(stdout, "", 1, "")
	if err != nil {
		t.Fatalf("parsePrettier: %v", err)
	}
	if len(findings) != 2 {
		t.Fatalf("got %d findings, want 2: %v", len(findings), findings)
	}
	if findings[0].File != "src/foo.js" {
		t.Errorf("findings[0].File = %q, want src/foo.js", findings[0].File)
	}
	if findings[1].File != "src/bar.ts" {
		t.Errorf("findings[1].File = %q, want src/bar.ts", findings[1].File)
	}
}

func TestParseShfmt(t *testing.T) {
	t.Parallel()

	stdout := "deploy.sh\nsetup.sh\n"
	findings, err := parseShfmt(stdout, "", 1, "")
	if err != nil {
		t.Fatalf("parseShfmt: %v", err)
	}
	if len(findings) != 2 {
		t.Fatalf("got %d findings, want 2", len(findings))
	}
	if findings[0].File != "deploy.sh" {
		t.Errorf("findings[0].File = %q, want deploy.sh", findings[0].File)
	}
}

func TestParseShellcheck(t *testing.T) {
	t.Parallel()

	stdout := `[{"file":"script.sh","line":5,"endLine":5,"column":3,"endColumn":10,"level":"error","code":2034,"message":"foo appears unused."}]`
	findings, err := parseShellcheck(stdout, "", 1, "")
	if err != nil {
		t.Fatalf("parseShellcheck: %v", err)
	}
	if len(findings) != 1 {
		t.Fatalf("got %d findings, want 1", len(findings))
	}
	f := findings[0]
	if f.File != "script.sh" {
		t.Errorf("File = %q, want script.sh", f.File)
	}
	if f.Line != 5 {
		t.Errorf("Line = %d, want 5", f.Line)
	}
	if f.Rule != "SC2034" {
		t.Errorf("Rule = %q, want SC2034", f.Rule)
	}
	if f.Level != "error" {
		t.Errorf("Level = %q, want error", f.Level)
	}
}

func TestParseYamllint(t *testing.T) {
	t.Parallel()

	stdout := `.github/workflows/ci.yml:1:1: [error] wrong indentation: expected 2 but found 0 (indentation)
.github/workflows/ci.yml:5:1: [warning] missing document start "---" (document-start)
`
	findings, err := parseYamllint(stdout, "", 1, "")
	if err != nil {
		t.Fatalf("parseYamllint: %v", err)
	}
	if len(findings) != 2 {
		t.Fatalf("got %d findings, want 2: %v", len(findings), findings)
	}
	if findings[0].Rule != "indentation" {
		t.Errorf("findings[0].Rule = %q, want indentation", findings[0].Rule)
	}
	if findings[0].Level != "error" {
		t.Errorf("findings[0].Level = %q, want error", findings[0].Level)
	}
	if findings[1].Level != "warning" {
		t.Errorf("findings[1].Level = %q, want warning", findings[1].Level)
	}
}

func TestParseHadolint(t *testing.T) {
	t.Parallel()

	stdout := `[{"line":1,"column":1,"code":"DL3006","message":"Always tag the version.","file":"Dockerfile","level":"warning"}]`
	findings, err := parseHadolint(stdout, "", 1, "")
	if err != nil {
		t.Fatalf("parseHadolint: %v", err)
	}
	if len(findings) != 1 {
		t.Fatalf("got %d findings, want 1", len(findings))
	}
	if findings[0].Rule != "DL3006" {
		t.Errorf("Rule = %q, want DL3006", findings[0].Rule)
	}
}

func TestParseHadolintSurfacesConfigParseError(t *testing.T) {
	t.Parallel()

	// hadolint quirk: a malformed .hadolint.yaml prints an "Error parsing your
	// config file" message to stderr and still exits 0 with "[]" on stdout.
	// Without the stderr inspection the run would be reported as a clean pass.
	stderr := `Error parsing your config file in '/work/.hadolint.yaml':
expected labeltype instead of !!str
`
	findings, err := parseHadolint("[]", stderr, 0, "")
	if err == nil {
		t.Fatalf("parseHadolint with config-parse error: want error, got findings=%v", findings)
	}
	if findings != nil {
		t.Errorf("parseHadolint with config-parse error: got findings=%v, want nil", findings)
	}
	if !strings.Contains(err.Error(), "Error parsing your config") {
		t.Errorf("error %q does not mention the config-parse message", err.Error())
	}
}

func TestParseMarkdownlint(t *testing.T) {
	t.Parallel()

	stdout := `README.md:1 MD041/first-line-heading/first-line-h1 First line should be a top-level heading
docs/guide.md:10 error MD013/line-length Line length [Expected: 120; Actual: 145]
`
	findings, err := parseMarkdownlint(stdout, "", 1, "")
	if err != nil {
		t.Fatalf("parseMarkdownlint: %v", err)
	}
	if len(findings) != 2 {
		t.Fatalf("got %d findings, want 2: %v", len(findings), findings)
	}
	if findings[0].File != "README.md" {
		t.Errorf("findings[0].File = %q, want README.md", findings[0].File)
	}
	if findings[0].Rule != "MD041" {
		t.Errorf("findings[0].Rule = %q, want MD041", findings[0].Rule)
	}
	if findings[0].Line != 1 {
		t.Errorf("findings[0].Line = %d, want 1", findings[0].Line)
	}
}

func TestParseEditorconfig(t *testing.T) {
	t.Parallel()

	output := "main.go:\n" +
		"  3: Wrong line endings. (expected LF, got CRLF)\n" +
		"  7: Trailing whitespace found.\n" +
		"script.sh:\n" +
		"  1: Wrong indentation style found.\n"
	findings, err := parseEditorconfig(output, "", 0, "")
	if err != nil {
		t.Fatalf("parseEditorconfig: %v", err)
	}
	if len(findings) != 3 {
		t.Fatalf("got %d findings, want 3: %v", len(findings), findings)
	}
	if findings[0].File != "main.go" || findings[0].Line != 3 {
		t.Errorf("findings[0] = %+v, want {File:main.go Line:3}", findings[0])
	}
	if findings[2].File != "script.sh" || findings[2].Line != 1 {
		t.Errorf("findings[2] = %+v, want {File:script.sh Line:1}", findings[2])
	}
}

func TestParseActionlint(t *testing.T) {
	t.Parallel()

	stdout := `[{"message":"unexpected key \"environment\"","filepath":".github/workflows/ci.yml","line":5,"column":3,"kind":"syntax-check"}]`
	findings, err := parseActionlint(stdout, "", 1, "")
	if err != nil {
		t.Fatalf("parseActionlint: %v", err)
	}
	if len(findings) != 1 {
		t.Fatalf("got %d findings, want 1", len(findings))
	}
	if findings[0].File != ".github/workflows/ci.yml" {
		t.Errorf("File = %q", findings[0].File)
	}
	if findings[0].Line != 5 {
		t.Errorf("Line = %d, want 5", findings[0].Line)
	}
}

func TestParseGolangciLint(t *testing.T) {
	t.Parallel()

	stdout := `{"Issues":[{"FromLinter":"errcheck","Text":"Error return value of ` + "`" + `f.Close` + "`" + ` is not checked","Pos":{"Filename":"main.go","Offset":0,"Line":12,"Column":3}}],"Report":{}}`
	findings, err := parseGolangciLint(stdout, "", 1, "")
	if err != nil {
		t.Fatalf("parseGolangciLint: %v", err)
	}
	if len(findings) != 1 {
		t.Fatalf("got %d findings, want 1", len(findings))
	}
	if findings[0].File != "main.go" {
		t.Errorf("File = %q, want main.go", findings[0].File)
	}
	if findings[0].Line != 12 {
		t.Errorf("Line = %d, want 12", findings[0].Line)
	}
	if findings[0].Rule != "errcheck" {
		t.Errorf("Rule = %q, want errcheck", findings[0].Rule)
	}
}

func TestParseRuffFormat(t *testing.T) {
	t.Parallel()

	stderr := "Would reformat: src/main.py\nWould reformat: src/utils.py\n2 files would be reformatted\n"
	findings, err := parseRuffFormat("", stderr, 1, "")
	if err != nil {
		t.Fatalf("parseRuffFormat: %v", err)
	}
	if len(findings) != 2 {
		t.Fatalf("got %d findings, want 2: %v", len(findings), findings)
	}
	if findings[0].File != "src/main.py" {
		t.Errorf("findings[0].File = %q, want src/main.py", findings[0].File)
	}
	if findings[1].File != "src/utils.py" {
		t.Errorf("findings[1].File = %q, want src/utils.py", findings[1].File)
	}
}

func TestParseRuff(t *testing.T) {
	t.Parallel()

	stdout := `[{"code":"F401","message":"os imported but unused","filename":"src/main.py","location":{"row":1,"column":8},"end_location":{"row":1,"column":10},"fix":null,"noqa_row":1,"url":"https://docs.astral.sh/ruff/rules/unused-import"}]`
	findings, err := parseRuff(stdout, "", 1, "")
	if err != nil {
		t.Fatalf("parseRuff: %v", err)
	}
	if len(findings) != 1 {
		t.Fatalf("got %d findings, want 1", len(findings))
	}
	if findings[0].File != "src/main.py" {
		t.Errorf("File = %q, want src/main.py", findings[0].File)
	}
	if findings[0].Line != 1 {
		t.Errorf("Line = %d, want 1", findings[0].Line)
	}
	if findings[0].Rule != "F401" {
		t.Errorf("Rule = %q, want F401", findings[0].Rule)
	}
	if findings[0].Message != "os imported but unused" {
		t.Errorf("Message = %q", findings[0].Message)
	}
}

func TestParseEmptyOutputPass(t *testing.T) {
	t.Parallel()

	for _, parse := range []func(string, string, int, string) ([]Finding, error){
		parsePrettier, parseShfmt, parseShellcheck, parseYamllint,
		parseHadolint, parseMarkdownlint, parseEditorconfig, parseActionlint,
		parseGolangciLint, parseRuff, parseRuffFormat,
	} {
		findings, err := parse("", "", 0, "")
		if err != nil {
			t.Errorf("parse with empty output returned error: %v", err)
		}
		if len(findings) != 0 {
			t.Errorf("parse with empty output returned %d findings, want 0", len(findings))
		}
	}
}

// -- splitBatches -------------------------------------------------------------------------

func TestSplitBatchesUnderLimit(t *testing.T) {
	t.Parallel()

	files := []string{"a.go", "b.go", "c.go"}
	batches := splitBatches(files)
	if len(batches) != 1 {
		t.Fatalf("expected 1 batch for small input, got %d", len(batches))
	}
	if len(batches[0]) != 3 {
		t.Errorf("batch[0] = %v, want 3 files", batches[0])
	}
}

func TestSplitBatchesEmpty(t *testing.T) {
	t.Parallel()

	batches := splitBatches(nil)
	if len(batches) != 1 || len(batches[0]) != 0 {
		t.Fatalf("splitBatches(nil) = %v, want [[]]", batches)
	}
}

func TestSplitBatchesSplitsAtLimit(t *testing.T) {
	t.Parallel()

	// Build a list where each file name is exactly half of maxFileArgBytes, so
	// two files exceed the limit and must land in separate batches.
	half := maxFileArgBytes/2 - 1
	name := func(ch byte) string {
		s := make([]byte, half)
		for i := range s {
			s[i] = ch
		}
		return string(s)
	}
	files := []string{name('a'), name('b'), name('c')}

	batches := splitBatches(files)
	if len(batches) < 2 {
		t.Fatalf("expected >=2 batches for large input, got %d", len(batches))
	}

	// Every original file must appear exactly once across all batches.
	seen := make(map[string]int)
	for _, b := range batches {
		for _, f := range b {
			seen[f]++
		}
	}
	for _, f := range files {
		if seen[f] != 1 {
			t.Errorf("file %q appears %d times across batches, want 1", f[:8]+"...", seen[f])
		}
	}
}

// -- batched invocation -------------------------------------------------------------------

func TestRunBatchedMergesFindings(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	// Produce 3 long path strings (ASCII, no NUL bytes) that together exceed
	// maxFileArgBytes so splitBatches partitions them into >=2 batches.
	// These paths do not need to exist on disk: fakeTool.Args ignores them
	// and passes nil to the binary instead.
	half := maxFileArgBytes/2 - 1
	longFiles := make([]string, 3)
	for i := range longFiles {
		longFiles[i] = strings.Repeat(fmt.Sprintf("%d", i+1), half)
	}

	batches := splitBatches(longFiles)
	if len(batches) < 2 {
		t.Skip("batching did not split as expected -- adjust test")
	}

	// fakeTool returns one synthetic finding per invocation regardless of input,
	// so the total findings count equals the number of batches after merging.
	fakeTool := ToolDef{
		Name:   "fake",
		Binary: "true", // exits 0, no output needed
		CanFix: false,
		Args:   func(_ bool, _ string, _ []string) []string { return nil },
		Parse: func(_, _ string, _ int, _ string) ([]Finding, error) {
			return []Finding{{File: "x", Message: "synthetic"}}, nil
		},
	}

	result := Run(context.Background(), fakeTool, dir, false, longFiles, io.Discard)
	if result.Status != "fail" {
		t.Fatalf("status = %q, want fail", result.Status)
	}
	if len(result.Findings) != len(batches) {
		t.Errorf("findings = %d, want %d (one per batch)", len(result.Findings), len(batches))
	}
}
