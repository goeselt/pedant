# pedant

One GitHub Action. Sixteen linters and formatters. Zero per-repository setup.

Pedant runs formatting, idiomatic lint, and style checks for Go, Python, JS/TS, CSS, Shell, Markdown, YAML, TOML,
Dockerfile, and GitHub Actions in a single container pass. It occupies a different position than tools like
[mega-linter](https://github.com/oxsecurity/megalinter) or
[super-linter](https://github.com/super-linter/super-linter):

- **Narrow scope by design.** A curated set of sixteen tools with pre-tuned defaults rather than a catalogue of
  hundreds. Security scanning, dependency analysis, and IaC misconfiguration belong in dedicated tools with dedicated
  workflows -- pedant stays focused on the checks worth running on every commit.
- **Zero required configuration.** Bundled defaults work for most repositories without adding any config files. Drop a
  tool-specific file in the repo root to override exactly one tool; everything else keeps using the bundled default.
- **No tool installation in CI.** One `docker pull` covers all sixteen tools at pinned versions. No `setup-*` action per
  tool, no per-repository version pinning, no drift between repositories over time.
- **Autofix included.** `fix: 'true'` applies all fixable findings in the same pass. Fixers run before checkers, so the
  check phase always sees the post-fix state.

## Quick Start

```yaml
- uses: goeselt/pedant@v1
```

With options:

```yaml
- uses: goeselt/pedant@v1
  with:
    fix: 'true' # apply auto-fixes (default: false -- check only)
    paths: | # restrict scan to these paths (one per line)
      src/
      docs/
    ignore: | # exclude these paths (one per line)
      vendor/
      dist/
    tool-timeout: 10m # fail a single stuck tool instead of hanging the job
    summary-github-step: 'true' # append Markdown summary to the GitHub step summary
    summary-file: 'pedant-summary.md' # also write the summary to a file
```

The action always runs all applicable tools. When `fix: 'true'`, the caller is responsible for committing any changes.

### Action Outputs

| Output             | Description                                                  |
| ------------------ | ------------------------------------------------------------ |
| `status`           | `pass`, `fail`, or `error`                                   |
| `total-findings`   | Total number of findings across all tools                    |
| `files-discovered` | Number of files discovered and checked                       |
| `tools-run`        | Number of tools that executed (not skipped)                  |
| `tools-skipped`    | Number of tools skipped (no matching files or condition met) |

Use outputs to drive downstream steps:

```yaml
- uses: goeselt/pedant@v1
  id: lint
- if: steps.lint.outputs.status == 'fail'
  run: echo "Lint failed with ${{ steps.lint.outputs.total-findings }} finding(s)"
```

## Tools

| Tool            | Checks                                                                           | Autofix            |
| --------------- | -------------------------------------------------------------------------------- | ------------------ |
| `plainify`      | Non-ASCII typographic characters, CRLF, invisible and bidi characters            | :white_check_mark: |
| `shfmt`         | Shell script formatting                                                          | :white_check_mark: |
| `taplo`         | TOML formatting                                                                  | :white_check_mark: |
| `ruff-format`   | Python code formatting                                                           | :white_check_mark: |
| `ruff`          | Python lint (flake8, isort, pycodestyle, and more)                               | :white_check_mark: |
| `textlint`      | Prose style and terminology in Markdown                                          | :white_check_mark: |
| `markdownlint`  | Markdown structure and style                                                     | :white_check_mark: |
| `eslint`        | JavaScript / TypeScript lint                                                     | :white_check_mark: |
| `stylelint`     | CSS lint                                                                         | :white_check_mark: |
| `prettier`      | Formatting of JSON, YAML, Markdown, HTML, CSS, JS/TS                             | :white_check_mark: |
| `editorconfig`  | Indentation, charset, end-of-line, trailing whitespace per `.editorconfig` rules |                    |
| `golangci-lint` | Go static analysis                                                               |                    |
| `hadolint`      | Dockerfile best practices                                                        |                    |
| `shellcheck`    | Shell script correctness                                                         |                    |
| `yamllint`      | YAML syntax and style                                                            |                    |
| `actionlint`    | GitHub Actions workflow correctness                                              |                    |

Tools only run when matching files are present. A repository without Go files, for example, will skip `golangci-lint`
automatically.

## Out of Scope

pedant covers code style, formatting, and language-specific lint. The following categories are intentionally excluded:

- **SAST and secret scanning** -- GitHub Advanced Security (CodeQL, secret scanning) covers these natively for GitHub
  repositories; running a second SAST pass in pedant would duplicate effort and add noise.
- **Dependency and container vulnerability scanning** -- Trivy, Grype, and `npm audit` match CVEs against dependency
  trees and container images; Dependabot surfaces fixes as pull requests. These tools operate on build artifacts, not
  source files.
- **IaC misconfiguration** -- Checkov and tfsec analyze Terraform, Kubernetes, and CloudFormation manifests for security
  misconfigurations. Their findings carry severity ratings and remediation workflows that belong in a dedicated review
  gate, not alongside formatting errors.

These tools require different permissions, longer runtimes, and produce findings with severity semantics that do not fit
a per-commit lint pass.

## Configuration

pedant ships a bundled config for every configurable tool. If a workspace config is found in the repository root, pedant
always uses that instead of the bundled default -- per-tool, so you can override only the tools you care about.

The bundled `eslint` config covers all common JS/TS extensions (`.js`, `.jsx`, `.mjs`, `.cjs`, `.ts`, `.tsx`, `.mts`,
`.cts`) without requiring a `tsconfig.json`. Type-aware and cross-file checks are out of scope for pedant -- those
belong in the TypeScript compiler (`tsc`). Supply your own `eslint.config.*` to enable them or any other
project-specific rules.

| Tool            | Workspace Config File                                                                                                                            |
| --------------- | ------------------------------------------------------------------------------------------------------------------------------------------------ |
| `editorconfig`  | `.editorconfig-checker.json`, `.ecrc`                                                                                                            |
| `prettier`      | `.prettierrc`, `.prettierrc.json`, `.prettierrc.yml`, `prettier.config.js`, ...                                                                  |
| `shfmt`         | (no config)                                                                                                                                      |
| `taplo`         | `.taplo.toml`, `taplo.toml`                                                                                                                      |
| `textlint`      | `.textlintrc`, `.textlintrc.json`, `.textlintrc.yaml`, `.textlintrc.yml`                                                                         |
| `markdownlint`  | `.markdownlint-cli2.yaml`, `.markdownlint-cli2.yml`, `.markdownlint-cli2.jsonc`, `.markdownlint.yaml`, `.markdownlint.yml`, `.markdownlint.json` |
| `eslint`        | `eslint.config.js`, `eslint.config.mjs`, `eslint.config.cjs`, `eslint.config.ts`, ...                                                            |
| `stylelint`     | `.stylelintrc`, `.stylelintrc.json`, `.stylelintrc.yaml`, `.stylelintrc.yml`, `stylelint.config.js`, ...                                         |
| `hadolint`      | `.hadolint.yaml`, `.hadolint.yml`                                                                                                                |
| `shellcheck`    | `.shellcheckrc`                                                                                                                                  |
| `yamllint`      | `.yamllint.yml`, `.yamllint.yaml`, `.yamllint`                                                                                                   |
| `actionlint`    | `.github/actionlint.yaml`, `.github/actionlint.yml`, `actionlint.yaml`, `actionlint.yml`                                                         |
| `golangci-lint` | `.golangci.yml`, `.golangci.yaml`, `.golangci.toml`, `.golangci.json`                                                                            |
| `ruff`          | `ruff.toml`, `.ruff.toml`, `pyproject.toml`                                                                                                      |
| `plainify`      | (no config)                                                                                                                                      |

## Options

| Flag                    | Description                                                                   |
| ----------------------- | ----------------------------------------------------------------------------- |
| `--fix`                 | Apply auto-fixes in-place; check-only by default                              |
| `--path <path>`         | Restrict scan to this path or file (repeatable)                               |
| `--ignore <path>`       | Exclude this path or file from scan (repeatable)                              |
| `--tool-timeout <dur>`  | Maximum wall-clock duration for one tool, e.g. `30s`, `5m`, or `1h`           |
| `--pretty`              | Pretty-print JSON output                                                      |
| `--quiet`, `-q`         | Suppress progress output; JSON only on stdout                                 |
| `--summary-markdown`    | Write a Markdown summary to stdout instead of JSON                            |
| `--summary-file <path>` | Write the summary to this file; JSON is still emitted on stdout               |
| `--summary-github-step` | Append the summary to `$GITHUB_STEP_SUMMARY`; JSON is still emitted on stdout |

### File Discovery

File discovery uses `git ls-files --exclude-standard`, so `.gitignore`, `.git/info/exclude`, and global Git ignore rules
are respected for untracked files, including when `--path` is used. Files already tracked by Git remain discoverable,
which matches Git's normal ignore behavior.

Pedant always skips generated, dependency, cache, and temporary directories such as `build/`, `dist/`, `node_modules/`,
`public/`, `target/`, `tmp/`, and `vendor/`. If an explicit `--path` selects files under one of those paths, pedant logs
a warning and omits those files from tool runs.

## Output

Progress is written to **stderr**. By default, JSON is written to **stdout**:

```json
{
  "status": "fail",
  "workspace": "/work",
  "files_discovered": 24,
  "tools_run": 12,
  "tools_skipped": 2,
  "total_findings": 2,
  "tools": [
    {
      "name": "shellcheck",
      "status": "fail",
      "findings": [
        {
          "file": "deploy.sh",
          "line": 12,
          "col": 5,
          "rule": "SC2006",
          "message": "Use $(...) notation instead of legacy backticks."
        }
      ]
    }
  ],
  "workspace_configs": [{ "tool": "shellcheck", "config": ".shellcheckrc" }]
}
```

The top-level `status` is `"pass"` (all tools clean), `"fail"` (one or more tools reported findings), or `"error"` (one
or more tools could not run to completion). Only tools with findings or errors appear in the `tools` array. Tools with
no matching files or a clean result are omitted; `tools_skipped` counts how many were not applicable.

`workspace_configs` lists tools that used a workspace-supplied configuration file rather than the bundled default. It is
omitted from the output when empty.

### Markdown Summary

Use `--summary-markdown`, `--summary-file <path>`, and/or `--summary-github-step` to produce a concise human-readable
report.

- `--summary-markdown` writes Markdown to **stdout** instead of JSON.
- `--summary-file` and `--summary-github-step` write Markdown to their respective destinations; **JSON is still emitted
  on stdout** so downstream steps can parse it.

The summary always includes the overall status, checked file count, tool counts, and finding count. Detailed sections
include only tools with findings or errors, which keeps CI output focused on actionable information.

Write a Markdown summary to stdout (replaces JSON):

```bash
docker run --rm -v "$(pwd):/work" ghcr.io/goeselt/pedant:latest --summary-markdown
```

Write a local Markdown summary file and still emit JSON on stdout:

```bash
docker run --rm -v "$(pwd):/work" ghcr.io/goeselt/pedant:latest --summary-file pedant-summary.md
```

### Exit Codes

| Code | Meaning                                                                                             |
| ---- | --------------------------------------------------------------------------------------------------- |
| `0`  | All tools passed                                                                                    |
| `1`  | One or more tools reported findings                                                                 |
| `2`  | Runtime error (bad arguments, Git not found, etc.) or one or more tools could not run to completion |

## Local Usage

```bash
docker pull ghcr.io/goeselt/pedant:latest
```

Check the current repository without modifying files (default):

```bash
docker run --rm -v "$(pwd):/work" ghcr.io/goeselt/pedant:latest
```

Check and autofix in one pass:

```bash
docker run --rm -v "$(pwd):/work" ghcr.io/goeselt/pedant:latest --fix
```

Restrict the scan to specific paths or files:

```bash
docker run --rm -v "$(pwd):/work" ghcr.io/goeselt/pedant:latest --path src/ --path README.md
```

Exclude paths or files from the scan:

```bash
docker run --rm -v "$(pwd):/work" ghcr.io/goeselt/pedant:latest --ignore vendor/ --ignore generated/file.go
```

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) and [LICENSE](LICENSE).
