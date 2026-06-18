# Contributing to Pedant

## How It Works

Pedant runs in five sequential phases. Reading this once is enough to orient
yourself in the codebase.

```
1. Discovery       git ls-files  →  discover.Files()
                                     └─ rejects pathspec-magic injection
                                     └─ deduplicates, skips deleted files

2. Filtering       pathignore.Filter()
                                     └─ removes node_modules/, vendor/, dist/, …
                                     └─ emits warnings for explicit --path hits inside those dirs

3. Classification  runner.ForTools()
                                     └─ iterates runner.Registry in order
                                     └─ each ToolDef.Globs selects matching files
                                     └─ tools with zero matches are silently skipped

4. Execution       runner.Run()  ×N  (one per tool with matching files)
                                     └─ optional fix pass (silent), then check pass
                                     └─ arg batching when total path length > 200 KB
                                     └─ logs info line if workspace config is used

5. Output          aggregate()   →   JSON on stdout  (always, unless --summary-markdown)
                                 →   Markdown        (--summary-markdown / --summary-file /
                                                       --summary-github-step)
                                 →   GITHUB_OUTPUT   (action outputs written automatically)
```

## Design

| File                        | Responsibility                                                                                    |
| --------------------------- | ------------------------------------------------------------------------------------------------- |
| `cmd/pedant/main.go`        | CLI entry point: flag parsing, orchestration, JSON output, exit codes.                            |
| `cmd/pedant/summary.go`     | Markdown rendering, `--summary-*` output, `GITHUB_OUTPUT` writer.                                 |
| `internal/discover/`        | File discovery via `git ls-files`; pathspec-magic rejection for `--path` and `--ignore` values.   |
| `internal/pathignore/`      | Default-ignored directory list, filter, and warning generation.                                   |
| `internal/runner/runner.go` | Tool orchestration: subprocess execution, fix+check cycle, arg batching, progress logging.        |
| `internal/runner/tools.go`  | Tool registry: one `ToolDef` per supported linter/formatter, including config detection.          |
| `internal/runner/classify.go` | File-to-tool assignment: glob matching, `ForTools()`.                                           |
| `entrypoint.sh`             | Docker entrypoint: translates `INPUT_*` env vars from Actions into CLI flags.                     |
| `Dockerfile`                | Multi-stage build: Go binary in stage 1, all lint tools installed in stage 2.                    |

## How to Add a New Tool

Adding a tool requires touching exactly two files.

### 1. `internal/runner/tools.go` — define the tool

Copy an existing `ToolDef` as a starting point. Mandatory fields:

```go
var myTool = ToolDef{
    Name:  "mytool",          // unique name; shown in logs and JSON output
    Globs: []string{"*.ext"}, // files this tool receives; nil = all files
    CanFix: false,            // set true only if the tool can rewrite files
    Args: func(fix bool, workspace string, files []string) []string {
        // return the CLI args for a check pass (fix=false) or fix pass (fix=true)
        return append([]string{"--flag"}, files...)
    },
    Parse: func(stdout, stderr string, exitCode int, workspace string) ([]Finding, error) {
        // parse tool output into []Finding; return nil, nil if clean
    },
}
```

Optional fields worth knowing:

| Field                | When to use                                                               |
| -------------------- | ------------------------------------------------------------------------- |
| `Binary`             | Executable name differs from `Name`                                       |
| `NoBatch`            | Tool does not accept explicit file args (e.g. `golangci-lint ./...`)      |
| `Skip`               | Runtime condition that disqualifies the tool (e.g. no `go.mod` present)   |
| `FindWorkspaceConfig`| Use `makeConfigFinder(candidates...)` to log which config file is in use  |

### 2. `internal/runner/tools.go` — add to `Registry`

```go
var Registry = []ToolDef{
    // ...
    myTool,  // ← add here; order determines execution order
}
```

That is everything. No changes to `main.go`, `classify.go`, or `entrypoint.sh` are required.

### Verify

```bash
go test ./...
docker build -t pedant .
docker run --rm -v "$(pwd):/work" pedant --help  # should list the new tool
```

## Development Setup

Go 1.24 or later, plus Docker for building the image and running end-to-end tests.

```bash
go build ./...
docker build -t pedant .
```

## Local Verification

Unit tests:

```bash
go test ./...
```

End-to-end fixture tests against the locally built image:

```bash
PEDANT_IMAGE=pedant .github/workflows/fixtures/run.sh --all
```

Self-check (pedant linting its own repository):

```bash
docker run --rm -v "$(pwd):/work" pedant --ignore .github/workflows/fixtures

# or

docker run --rm \
  -v "$(pwd):/work" \
  -e INPUT_FIX=false \
  -e INPUT_IGNORE=.github/workflows/fixtures \
  -e GITHUB_ACTIONS=true \
  pedant
```

Summary smoke tests. These intentionally do not ignore `.github/workflows/fixtures`, so the repository should produce
findings and visible summary content.

Markdown summary on stdout (replaces JSON):

```bash
docker run --rm -v "$(pwd):/work" pedant --summary-markdown
```

Markdown summary file (JSON still emitted on stdout):

```bash
docker run --rm -v "$(pwd):/work" pedant --summary-file pedant-summary.md
cat pedant-summary.md
```

GitHub step summary output (JSON still emitted on stdout):

```bash
docker run --rm \
  -v "$(pwd):/work" \
  -e GITHUB_STEP_SUMMARY=/work/pedant-step-summary.md \
  pedant --summary-github-step

cat pedant-step-summary.md
```

## Submitting Changes

Commit messages and PR titles must follow [Conventional Commits](https://www.conventionalcommits.org/). The release
pipeline uses the PR title to determine the next version.
