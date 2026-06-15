# Contributing to Pedant

## Design

| File                        | Responsibility                                                                           |
| --------------------------- | ---------------------------------------------------------------------------------------- |
| `cmd/pedant/main.go`        | CLI entry point: flag parsing, JSON output, exit codes.                                  |
| `internal/discover/`        | File discovery via `git ls-files`; path filtering for `--path` and `--ignore` flags.     |
| `internal/classify/`        | File type detection: maps extensions and filenames to tool eligibility.                  |
| `internal/runner/runner.go` | Tool orchestration: runs each tool as a subprocess, collects and aggregates results.     |
| `internal/runner/tools.go`  | Tool registry: one `ToolDef` per supported linter/formatter, including config detection. |
| `entrypoint.sh`             | Docker entrypoint: translates `INPUT_*` env vars from Actions into CLI flags.            |
| `Dockerfile`                | Multi-stage build: Go binary in stage 1, all lint tools installed in stage 2.            |

Tool selection is purely runtime: `internal/classify` determines which tools apply to which files based on detected file
types. No tool is hardcoded to always run.

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
docker run --rm -v "$(pwd):/work" pedant --nofix --ignore .github/workflows/fixtures

# or

docker run --rm \
  -v "$(pwd):/work" \
  -e INPUT_FIX=false \
  -e INPUT_IGNORE=.github/workflows/fixtures \
  -e GITHUB_ACTIONS=true \
  pedant
```

## Submitting Changes

Commit messages and PR titles must follow [Conventional Commits](https://www.conventionalcommits.org/). The release
pipeline uses the PR title to determine the next version.
