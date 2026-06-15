package runner

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// Registry lists all supported tools in execution order.
var Registry = []ToolDef{
	editorconfigTool,
	plainifyTool,
	prettierTool,
	shfmtTool,
	ruffFormatTool,
	golangciTool,
	ruffTool,
	textlintTool,
	markdownlintTool,
	eslintTool,
	stylelintTool,
	hadolintTool,
	shellcheckTool,
	yamllintTool,
	actionlintTool,
}

// -- Helpers -----------------------------------------------------------------------------------------

// workspaceConfig returns the path of the first candidate that exists in workspace,
// or "" if none are found.
func workspaceConfig(workspace string, candidates ...string) string {
	for _, c := range candidates {
		if _, err := os.Stat(filepath.Join(workspace, c)); err == nil {
			return filepath.Join(workspace, c)
		}
	}
	return ""
}

// bundledConfig returns path if it exists on the filesystem (inside the Docker image at /etc/pedant/),
// or "" if not present (e.g. when running outside the container during development).
func bundledConfig(path string) string {
	if _, err := os.Stat(path); err == nil {
		return path
	}
	return ""
}

// relativize converts an absolute path to a workspace-relative path.
// Returns the original path if it cannot be made relative.
func relativize(workspace, path string) string {
	if !filepath.IsAbs(path) {
		return path
	}
	rel, err := filepath.Rel(workspace, path)
	if err != nil {
		return path
	}
	return filepath.ToSlash(rel)
}

// -- editorconfig (ec) -------------------------------------------------------------------------------

var editorconfigTool = ToolDef{
	Name:   "editorconfig",
	Binary: "ec",
	CanFix: false,
	Globs:  nil,
	Args: func(_ bool, workspace string, files []string) []string {
		args := []string{"-no-color"}
		if workspaceConfig(workspace, ".editorconfig-checker.json", ".ecrc") == "" {
			if cfg := bundledConfig("/etc/pedant/editorconfig/.editorconfig-checker.json"); cfg != "" {
				args = append(args, "-config", cfg)
			}
		}
		return append(args, files...)
	},
	Parse: parseEditorconfig,
}

// ec default output:
//
//	filename.go:
//	  3: Wrong line endings. (expected LF, got CRLF)
var (
	ecFileRE    = regexp.MustCompile(`^([^:]+):$`)
	ecFindingRE = regexp.MustCompile(`^\s+(\d+):\s+(.+)$`)
)

func parseEditorconfig(stdout, stderr string, _ int, _ string) ([]Finding, error) {
	var findings []Finding
	var currentFile string
	for _, line := range strings.Split(stdout+stderr, "\n") {
		if m := ecFileRE.FindStringSubmatch(line); m != nil {
			currentFile = m[1]
		} else if m := ecFindingRE.FindStringSubmatch(line); m != nil && currentFile != "" {
			lineNum, _ := strconv.Atoi(m[1])
			findings = append(findings, Finding{
				File:    currentFile,
				Line:    lineNum,
				Message: strings.TrimSpace(m[2]),
			})
		}
	}
	return findings, nil
}

// -- prettier ----------------------------------------------------------------------------------------

var prettierConfigCandidates = []string{
	".prettierrc", ".prettierrc.json", ".prettierrc.yml", ".prettierrc.yaml",
	".prettierrc.toml", ".prettierrc.json5",
	"prettier.config.js", "prettier.config.mjs", "prettier.config.cjs",
}

var prettierTool = ToolDef{
	Name:   "prettier",
	CanFix: true,
	Globs:  []string{"*.json", "*.yml", "*.yaml", "*.md", "*.html", "*.css", "*.ts", "*.tsx", "*.js", "*.jsx", "*.mjs", "*.cjs"},
	Args: func(fix bool, workspace string, files []string) []string {
		args := []string{"--no-color"}
		if workspaceConfig(workspace, prettierConfigCandidates...) == "" {
			if cfg := bundledConfig("/etc/pedant/prettier/.prettierrc"); cfg != "" {
				args = append(args, "--config", cfg)
			}
			if ign := bundledConfig("/etc/pedant/prettier/.prettierignore"); ign != "" {
				args = append(args, "--ignore-path", ign)
			}
		}
		if fix {
			args = append(args, "--write")
		} else {
			args = append(args, "--check")
		}
		return append(args, files...)
	},
	Parse: parsePrettier,
}

// prettier --check output: [warn] path/to/file.js
func parsePrettier(stdout, stderr string, _ int, _ string) ([]Finding, error) {
	var findings []Finding
	for _, line := range strings.Split(stdout+stderr, "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "[warn] ") {
			continue
		}
		file := strings.TrimPrefix(trimmed, "[warn] ")
		if strings.HasPrefix(file, "Code style issues") || strings.HasPrefix(file, "All matched files") {
			continue
		}
		findings = append(findings, Finding{File: file, Message: "needs formatting"})
	}
	return findings, nil
}

// -- shfmt -------------------------------------------------------------------------------------------

var shfmtTool = ToolDef{
	Name:   "shfmt",
	CanFix: true,
	Globs:  []string{"*.sh"},
	Args: func(fix bool, _ string, files []string) []string {
		if fix {
			return append([]string{"-w"}, files...)
		}
		return append([]string{"-l"}, files...)
	},
	Parse: parseShfmt,
}

// shfmt -l output: one filename per line for files that need reformatting.
func parseShfmt(stdout, _ string, _ int, _ string) ([]Finding, error) {
	var findings []Finding
	for _, line := range strings.Split(strings.TrimSpace(stdout), "\n") {
		if line != "" {
			findings = append(findings, Finding{File: line, Message: "needs formatting"})
		}
	}
	return findings, nil
}

// -- textlint ----------------------------------------------------------------------------------------

var textlintTool = ToolDef{
	Name:   "textlint",
	CanFix: true,
	Globs:  []string{"*.md"},
	Args: func(fix bool, workspace string, files []string) []string {
		args := []string{}
		if workspaceConfig(workspace, ".textlintrc", ".textlintrc.json", ".textlintrc.yaml", ".textlintrc.yml") == "" {
			if cfg := bundledConfig("/etc/pedant/textlint/.textlintrc.json"); cfg != "" {
				args = append(args, "--config", cfg)
			}
		}
		if fix {
			args = append(args, "--fix")
		} else {
			args = append(args, "--format=json")
		}
		return append(args, files...)
	},
	Parse: parseTextlintJSON,
}

type textlintFileResult struct {
	FilePath string `json:"filePath"`
	Messages []struct {
		RuleID  string `json:"ruleId"`
		Message string `json:"message"`
		Line    int    `json:"line"`
		Column  int    `json:"column"`
	} `json:"messages"`
}

func parseTextlintJSON(stdout, _ string, _ int, workspace string) ([]Finding, error) {
	if strings.TrimSpace(stdout) == "" {
		return nil, nil
	}
	var results []textlintFileResult
	if err := json.Unmarshal([]byte(stdout), &results); err != nil {
		return nil, fmt.Errorf("textlint JSON: %w", err)
	}
	var findings []Finding
	for _, r := range results {
		for _, m := range r.Messages {
			findings = append(findings, Finding{
				File:    relativize(workspace, r.FilePath),
				Line:    m.Line,
				Col:     m.Column,
				Rule:    m.RuleID,
				Message: m.Message,
			})
		}
	}
	return findings, nil
}

// -- markdownlint ------------------------------------------------------------------------------------

var markdownlintTool = ToolDef{
	Name:   "markdownlint",
	Binary: "markdownlint-cli2",
	CanFix: true,
	Globs:  []string{"*.md"},
	Args: func(fix bool, workspace string, files []string) []string {
		args := []string{}
		if workspaceConfig(workspace,
			".markdownlint-cli2.yaml", ".markdownlint-cli2.yml", ".markdownlint-cli2.jsonc",
			".markdownlint.yaml", ".markdownlint.yml", ".markdownlint.json",
		) == "" {
			if cfg := bundledConfig("/etc/pedant/markdownlint/.markdownlint-cli2.yaml"); cfg != "" {
				args = append(args, "--config", cfg)
			}
		}
		if fix {
			args = append(args, "--fix")
		}
		return append(args, files...)
	},
	Parse: parseMarkdownlint,
}

// markdownlint-cli2 default output:
//
//	README.md:1 MD041/first-line-heading First line in a file should be a top-level heading
var markdownlintRE = regexp.MustCompile(`^(.+?):(\d+)(?::\d+)?\s+(?:\S+\s+)?(MD\d+/\S+)\s+(.+)$`)

func parseMarkdownlint(stdout, stderr string, _ int, _ string) ([]Finding, error) {
	var findings []Finding
	for _, line := range strings.Split(stdout+stderr, "\n") {
		m := markdownlintRE.FindStringSubmatch(strings.TrimSpace(line))
		if m == nil {
			continue
		}
		lineNum, _ := strconv.Atoi(m[2])
		// Rule string: take first component before '/' as the rule ID
		ruleFull := m[3]
		rule := strings.SplitN(ruleFull, "/", 2)[0]
		findings = append(findings, Finding{
			File:    m[1],
			Line:    lineNum,
			Rule:    rule,
			Message: m[4],
		})
	}
	return findings, nil
}

// -- eslint ------------------------------------------------------------------------------------------

var eslintConfigCandidates = []string{
	"eslint.config.js", "eslint.config.mjs", "eslint.config.cjs",
	"eslint.config.ts", "eslint.config.mts",
	".eslintrc.js", ".eslintrc.cjs", ".eslintrc.yaml", ".eslintrc.yml", ".eslintrc.json",
}

var eslintTool = ToolDef{
	Name:   "eslint",
	CanFix: true,
	Globs:  []string{"*.js", "*.jsx", "*.mjs", "*.cjs", "*.ts", "*.tsx", "*.mts", "*.cts"},
	Args: func(fix bool, workspace string, files []string) []string {
		args := []string{}
		if workspaceConfig(workspace, eslintConfigCandidates...) == "" {
			if cfg := bundledConfig("/etc/pedant/eslint/eslint.config.mjs"); cfg != "" {
				args = append(args, "--config", cfg)
			}
		}
		if fix {
			args = append(args, "--fix")
		} else {
			args = append(args, "--format=json")
		}
		return append(args, files...)
	},
	Parse: parseEslint,
}

type eslintFileResult struct {
	FilePath string `json:"filePath"`
	Messages []struct {
		RuleID   string `json:"ruleId"`
		Severity int    `json:"severity"`
		Message  string `json:"message"`
		Line     int    `json:"line"`
		Column   int    `json:"column"`
	} `json:"messages"`
}

func parseEslint(stdout, _ string, _ int, workspace string) ([]Finding, error) {
	if strings.TrimSpace(stdout) == "" {
		return nil, nil
	}
	var results []eslintFileResult
	if err := json.Unmarshal([]byte(stdout), &results); err != nil {
		return nil, fmt.Errorf("eslint JSON: %w", err)
	}
	var findings []Finding
	for _, r := range results {
		for _, m := range r.Messages {
			level := "warning"
			if m.Severity == 2 {
				level = "error"
			}
			findings = append(findings, Finding{
				File:    relativize(workspace, r.FilePath),
				Line:    m.Line,
				Col:     m.Column,
				Level:   level,
				Rule:    m.RuleID,
				Message: m.Message,
			})
		}
	}
	return findings, nil
}

// -- hadolint ----------------------------------------------------------------------------------------

var hadolintTool = ToolDef{
	Name:   "hadolint",
	CanFix: false,
	Globs:  []string{"Dockerfile", "Dockerfile.*", "*.dockerfile"},
	Args: func(_ bool, workspace string, files []string) []string {
		args := []string{"--format=json"}
		cfgFile := workspaceConfig(workspace, ".hadolint.yaml", ".hadolint.yml")
		if cfgFile == "" {
			cfgFile = bundledConfig("/etc/pedant/hadolint/.hadolint.yaml")
		}
		if cfgFile != "" {
			args = append(args, "--config", cfgFile)
		}
		return append(args, files...)
	},
	Parse: parseHadolint,
}

type hadolintFinding struct {
	File    string `json:"file"`
	Line    int    `json:"line"`
	Col     int    `json:"column"`
	Code    string `json:"code"`
	Message string `json:"message"`
	Level   string `json:"level"`
}

func parseHadolint(stdout, stderr string, _ int, _ string) ([]Finding, error) {
	// hadolint prints config-parse failures to stderr and still exits 0 with an
	// empty "[]" stdout, which would otherwise look like a clean pass. Surface
	// that as an error so a broken .hadolint.yaml does not silently mask findings.
	if msg := strings.TrimSpace(stderr); strings.HasPrefix(msg, "Error parsing your config") {
		return nil, fmt.Errorf("%s", msg)
	}
	if strings.TrimSpace(stdout) == "" {
		return nil, nil
	}
	var raw []hadolintFinding
	if err := json.Unmarshal([]byte(stdout), &raw); err != nil {
		return nil, fmt.Errorf("hadolint JSON: %w", err)
	}
	findings := make([]Finding, 0, len(raw))
	for _, r := range raw {
		findings = append(findings, Finding{
			File:    r.File,
			Line:    r.Line,
			Col:     r.Col,
			Level:   r.Level,
			Rule:    r.Code,
			Message: r.Message,
		})
	}
	return findings, nil
}

// -- shellcheck --------------------------------------------------------------------------------------

var shellcheckTool = ToolDef{
	Name:   "shellcheck",
	CanFix: false,
	Globs:  []string{"*.sh"},
	Args: func(_ bool, workspace string, files []string) []string {
		args := []string{"--format=json"}
		rcfile := workspaceConfig(workspace, ".shellcheckrc")
		if rcfile == "" {
			rcfile = bundledConfig("/etc/pedant/shellcheck/.shellcheckrc")
		}
		if rcfile != "" {
			args = append(args, "--rcfile", rcfile)
		}
		return append(args, files...)
	},
	Parse: parseShellcheck,
}

type shellcheckFinding struct {
	File    string `json:"file"`
	Line    int    `json:"line"`
	Col     int    `json:"column"`
	Level   string `json:"level"`
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func parseShellcheck(stdout, _ string, _ int, _ string) ([]Finding, error) {
	if strings.TrimSpace(stdout) == "" {
		return nil, nil
	}
	var raw []shellcheckFinding
	if err := json.Unmarshal([]byte(stdout), &raw); err != nil {
		return nil, fmt.Errorf("shellcheck JSON: %w", err)
	}
	findings := make([]Finding, 0, len(raw))
	for _, r := range raw {
		findings = append(findings, Finding{
			File:    r.File,
			Line:    r.Line,
			Col:     r.Col,
			Level:   r.Level,
			Rule:    fmt.Sprintf("SC%d", r.Code),
			Message: r.Message,
		})
	}
	return findings, nil
}

// -- yamllint ----------------------------------------------------------------------------------------

var yamllintTool = ToolDef{
	Name:   "yamllint",
	CanFix: false,
	Globs:  []string{"*.yml", "*.yaml"},
	Args: func(_ bool, workspace string, files []string) []string {
		args := []string{"-f", "parsable"}
		cfgFile := workspaceConfig(workspace, ".yamllint.yml", ".yamllint.yaml", ".yamllint")
		if cfgFile == "" {
			cfgFile = bundledConfig("/etc/pedant/yamllint/.yamllint.yml")
		}
		if cfgFile != "" {
			args = append(args, "-c", cfgFile)
		}
		return append(args, files...)
	},
	Parse: parseYamllint,
}

// yamllint -f parsable output:
//
//	file.yml:1:1: [error] wrong indentation (indentation)
var yamllintRE = regexp.MustCompile(`^(.+):(\d+):(\d+): \[(error|warning)\] (.+)$`)

func parseYamllint(stdout, stderr string, _ int, _ string) ([]Finding, error) {
	var findings []Finding
	for _, line := range strings.Split(stdout+stderr, "\n") {
		m := yamllintRE.FindStringSubmatch(strings.TrimSpace(line))
		if m == nil {
			continue
		}
		lineNum, _ := strconv.Atoi(m[2])
		col, _ := strconv.Atoi(m[3])
		msg := m[5]
		// Extract rule from trailing parentheses: "message text (rule-name)"
		rule := ""
		if i := strings.LastIndex(msg, " ("); i >= 0 && strings.HasSuffix(msg, ")") {
			rule = msg[i+2 : len(msg)-1]
			msg = msg[:i]
		}
		findings = append(findings, Finding{
			File:    m[1],
			Line:    lineNum,
			Col:     col,
			Level:   m[4],
			Rule:    rule,
			Message: strings.TrimSpace(msg),
		})
	}
	return findings, nil
}

// -- actionlint --------------------------------------------------------------------------------------

var actionlintTool = ToolDef{
	Name:   "actionlint",
	CanFix: false,
	Globs:  []string{".github/workflows/*.yml", ".github/workflows/*.yaml"},
	Args: func(_ bool, workspace string, files []string) []string {
		args := []string{"-no-color", "-format", "{{json .}}"}
		cfgFile := workspaceConfig(workspace, ".github/actionlint.yaml", ".github/actionlint.yml", "actionlint.yaml", "actionlint.yml")
		if cfgFile == "" {
			cfgFile = bundledConfig("/etc/pedant/actionlint/actionlint.yaml")
		}
		if cfgFile != "" {
			args = append(args, "-config-file", cfgFile)
		}
		return append(args, files...)
	},
	Parse: parseActionlint,
}

type actionlintFinding struct {
	Message  string `json:"message"`
	Filepath string `json:"filepath"`
	Line     int    `json:"line"`
	Column   int    `json:"column"`
	Kind     string `json:"kind"`
}

func parseActionlint(stdout, _ string, _ int, _ string) ([]Finding, error) {
	if strings.TrimSpace(stdout) == "" {
		return nil, nil
	}
	var raw []actionlintFinding
	if err := json.Unmarshal([]byte(stdout), &raw); err != nil {
		return nil, fmt.Errorf("actionlint JSON: %w", err)
	}
	findings := make([]Finding, 0, len(raw))
	for _, r := range raw {
		findings = append(findings, Finding{
			File:    r.Filepath,
			Line:    r.Line,
			Col:     r.Column,
			Rule:    r.Kind,
			Message: r.Message,
		})
	}
	return findings, nil
}

// -- golangci-lint -----------------------------------------------------------------------------------

var golangciTool = ToolDef{
	Name:   "golangci-lint",
	CanFix: false,
	Globs:  []string{"*.go"},
	// golangci-lint operates on Go packages, so a go.mod at the workspace root
	// is required. Multi-module repos must be linted per-module.
	Skip: func(workspace string) bool {
		_, err := os.Stat(filepath.Join(workspace, "go.mod"))
		return err != nil
	},
	Reason:  "no go.mod at workspace root",
	NoBatch: true, // ignores the file list; always runs ./...
	Args: func(_ bool, workspace string, _ []string) []string {
		// golangci-lint operates on packages, not individual files.
		args := []string{"run", "--output.json.path=stdout"}
		cfgFile := workspaceConfig(workspace, ".golangci.yml", ".golangci.yaml", ".golangci.toml", ".golangci.json")
		if cfgFile == "" {
			if cfg := bundledConfig("/etc/pedant/golangci-lint/.golangci.yml"); cfg != "" {
				args = append(args, "--config", cfg)
			}
		}
		return append(args, "./...")
	},
	Parse: parseGolangciLint,
}

type golangciOutput struct {
	Issues []struct {
		FromLinter string `json:"FromLinter"`
		Text       string `json:"Text"`
		Pos        struct {
			Filename string `json:"Filename"`
			Line     int    `json:"Line"`
			Column   int    `json:"Column"`
		} `json:"Pos"`
	} `json:"Issues"`
}

func parseGolangciLint(stdout, _ string, _ int, workspace string) ([]Finding, error) {
	if strings.TrimSpace(stdout) == "" {
		return nil, nil
	}
	// Use Decoder instead of Unmarshal: golangci-lint appends a plain-text
	// summary after the JSON object, which would cause Unmarshal to fail.
	var out golangciOutput
	if err := json.NewDecoder(strings.NewReader(stdout)).Decode(&out); err != nil {
		return nil, fmt.Errorf("golangci-lint JSON: %w", err)
	}
	findings := make([]Finding, 0, len(out.Issues))
	for _, issue := range out.Issues {
		findings = append(findings, Finding{
			File:    resolveGolangciPath(workspace, issue.Pos.Filename),
			Line:    issue.Pos.Line,
			Col:     issue.Pos.Column,
			Rule:    issue.FromLinter,
			Message: issue.Text,
		})
	}
	return findings, nil
}

// resolveGolangciPath normalizes a golangci-lint file path to a workspace-relative form.
// golangci-lint may emit relative paths with ../ segments when GOROOT is outside
// the project root; resolving against workspace produces a clean relative path.
func resolveGolangciPath(workspace, path string) string {
	if !filepath.IsAbs(path) {
		path = filepath.Clean(filepath.Join(workspace, path))
	}
	return relativize(workspace, path)
}

// -- plainify ----------------------------------------------------------------------------------------

var plainifyTool = ToolDef{
	Name:   "plainify",
	CanFix: true,
	Globs:  nil,
	Args: func(fix bool, workspace string, files []string) []string {
		args := []string{"--workspace", workspace}
		if !fix {
			args = append(args, "--nofix")
		}
		return append(args, files...)
	},
	Parse: parsePlainify,
}

type plainifyOutput struct {
	Findings []struct {
		File    string `json:"file"`
		Line    int    `json:"line"`
		Col     int    `json:"col"`
		Message string `json:"message"`
	} `json:"findings"`
}

func parsePlainify(stdout, stderr string, exitCode int, _ string) ([]Finding, error) {
	if exitCode == 2 {
		return nil, fmt.Errorf("%s", strings.TrimSpace(stderr))
	}
	if strings.TrimSpace(stdout) == "" {
		return nil, nil
	}
	var out plainifyOutput
	if err := json.Unmarshal([]byte(stdout), &out); err != nil {
		return nil, fmt.Errorf("plainify JSON: %w", err)
	}
	findings := make([]Finding, 0, len(out.Findings))
	for _, f := range out.Findings {
		findings = append(findings, Finding{
			File:    f.File,
			Line:    f.Line,
			Col:     f.Col,
			Message: f.Message,
		})
	}
	return findings, nil
}

// -- ruff-format -------------------------------------------------------------------------------------

var ruffConfigCandidates = []string{
	"ruff.toml", ".ruff.toml", "pyproject.toml",
}

var ruffFormatTool = ToolDef{
	Name:   "ruff-format",
	Binary: "ruff",
	CanFix: true,
	Globs:  []string{"*.py"},
	Args: func(fix bool, workspace string, files []string) []string {
		args := []string{"format", "--no-cache"}
		if workspaceConfig(workspace, ruffConfigCandidates...) == "" {
			if cfg := bundledConfig("/etc/pedant/ruff/ruff.toml"); cfg != "" {
				args = append(args, "--config", cfg)
			}
		}
		if !fix {
			args = append(args, "--check")
		}
		return append(args, files...)
	},
	Parse: parseRuffFormat,
}

// ruff format --check output: "Would reformat: path/to/file.py"
func parseRuffFormat(stdout, stderr string, _ int, _ string) ([]Finding, error) {
	var findings []Finding
	for _, line := range strings.Split(stderr+stdout, "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "Would reformat: ") {
			continue
		}
		file := strings.TrimPrefix(trimmed, "Would reformat: ")
		findings = append(findings, Finding{File: file, Message: "needs formatting"})
	}
	return findings, nil
}

// -- stylelint ---------------------------------------------------------------------------------------

var stylelintConfigCandidates = []string{
	".stylelintrc",
	".stylelintrc.json",
	".stylelintrc.yaml",
	".stylelintrc.yml",
	".stylelintrc.js",
	".stylelintrc.mjs",
	".stylelintrc.cjs",
	"stylelint.config.js",
	"stylelint.config.mjs",
	"stylelint.config.cjs",
}

var stylelintTool = ToolDef{
	Name:   "stylelint",
	CanFix: true,
	Globs:  []string{"*.css"},
	Args: func(fix bool, workspace string, files []string) []string {
		args := []string{}
		if workspaceConfig(workspace, stylelintConfigCandidates...) == "" {
			if cfg := bundledConfig("/etc/pedant/stylelint/.stylelintrc.json"); cfg != "" {
				args = append(args, "--config", cfg)
			}
		}
		if fix {
			args = append(args, "--fix")
		} else {
			args = append(args, "--formatter", "json")
		}
		return append(args, files...)
	},
	Parse: parseStylelint,
}

type stylelintFileResult struct {
	Source   string `json:"source"`
	Warnings []struct {
		Line     int    `json:"line"`
		Column   int    `json:"column"`
		Rule     string `json:"rule"`
		Severity string `json:"severity"`
		Text     string `json:"text"`
	} `json:"warnings"`
}

func parseStylelint(stdout, stderr string, _ int, workspace string) ([]Finding, error) {
	// stylelint v16+ writes --formatter json to stderr instead of stdout.
	output := stdout
	if strings.TrimSpace(output) == "" {
		output = stderr
	}
	if strings.TrimSpace(output) == "" {
		return nil, nil
	}
	var results []stylelintFileResult
	if err := json.Unmarshal([]byte(output), &results); err != nil {
		return nil, fmt.Errorf("stylelint JSON: %w", err)
	}
	var findings []Finding
	for _, r := range results {
		for _, w := range r.Warnings {
			msg := w.Text
			if suffix := " (" + w.Rule + ")"; strings.HasSuffix(msg, suffix) {
				msg = msg[:len(msg)-len(suffix)]
			}
			findings = append(findings, Finding{
				File:    relativize(workspace, r.Source),
				Line:    w.Line,
				Col:     w.Column,
				Level:   w.Severity,
				Rule:    w.Rule,
				Message: msg,
			})
		}
	}
	return findings, nil
}

// -- ruff (check) ------------------------------------------------------------------------------------

var ruffTool = ToolDef{
	Name:   "ruff",
	Binary: "ruff",
	CanFix: true,
	Globs:  []string{"*.py"},
	Args: func(fix bool, workspace string, files []string) []string {
		args := []string{"check", "--no-cache"}
		if workspaceConfig(workspace, ruffConfigCandidates...) == "" {
			if cfg := bundledConfig("/etc/pedant/ruff/ruff.toml"); cfg != "" {
				args = append(args, "--config", cfg)
			}
		}
		if fix {
			args = append(args, "--fix")
		} else {
			args = append(args, "--output-format=json")
		}
		return append(args, files...)
	},
	Parse: parseRuff,
}

type ruffFinding struct {
	Code     string `json:"code"`
	Message  string `json:"message"`
	Filename string `json:"filename"`
	Location struct {
		Row    int `json:"row"`
		Column int `json:"column"`
	} `json:"location"`
}

func parseRuff(stdout, _ string, _ int, workspace string) ([]Finding, error) {
	if strings.TrimSpace(stdout) == "" {
		return nil, nil
	}
	var raw []ruffFinding
	if err := json.Unmarshal([]byte(stdout), &raw); err != nil {
		return nil, fmt.Errorf("ruff JSON: %w", err)
	}
	findings := make([]Finding, 0, len(raw))
	for _, r := range raw {
		findings = append(findings, Finding{
			File:    relativize(workspace, r.Filename),
			Line:    r.Location.Row,
			Col:     r.Location.Column,
			Rule:    r.Code,
			Message: r.Message,
		})
	}
	return findings, nil
}
