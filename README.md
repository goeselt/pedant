# pedant

A Docker-based linting and formatting orchestrator for Git repositories. It runs several tools in a single container
pass, reports findings as JSON on stdout, and can autofix what is fixable. No tool installations required -- pull the
image and run.

## Quick Start

```yaml
- uses: goeselt/pedant@v1
```

With options:

```yaml
- uses: goeselt/pedant@v1
  with:
    fix: 'true' # apply auto-fixes (default: false)
    paths: 'src/ docs/' # restrict scan to these paths
    ignore: 'vendor/ dist/' # exclude these paths
```

The action always runs all applicable tools. When `fix: 'true'`, the caller is responsible for committing any changes.

## Tools

| Tool            | Checks                                                                           | Autofix            |
| --------------- | -------------------------------------------------------------------------------- | ------------------ |
| `editorconfig`  | Indentation, charset, end-of-line, trailing whitespace per `.editorconfig` rules | :x:                |
| `prettier`      | Formatting of JSON, YAML, Markdown, HTML, CSS, JS/TS                             | :white_check_mark: |
| `shfmt`         | Shell script formatting                                                          | :white_check_mark: |
| `textlint`      | Prose style and terminology in Markdown                                          | :white_check_mark: |
| `markdownlint`  | Markdown structure and style                                                     | :white_check_mark: |
| `eslint`        | JavaScript / TypeScript lint                                                     | :white_check_mark: |
| `ruff-format`   | Python code formatting                                                           | :white_check_mark: |
| `ruff`          | Python lint (flake8, isort, pycodestyle, and more)                               | :white_check_mark: |
| `hadolint`      | Dockerfile best practices                                                        | :x:                |
| `shellcheck`    | Shell script correctness                                                         | :x:                |
| `yamllint`      | YAML syntax and style                                                            | :x:                |
| `actionlint`    | GitHub Actions workflow correctness                                              | :x:                |
| `golangci-lint` | Go static analysis                                                               | :x:                |
| `plainify`      | Non-ASCII typographic characters, CRLF, invisible and bidi characters            | :white_check_mark: |

Tools only run when matching files are present. A repository without Go files, for example, will skip `golangci-lint`
automatically.

## Configuration

pedant ships a bundled config for every configurable tool. If a workspace config is found in the repository root, pedant
always uses that instead of the bundled default -- per-tool, so you can override only the tools you care about.

The bundled `eslint` config lints JavaScript and TypeScript out of the box, including JSX -- `.js`, `.jsx`, `.mjs`,
`.cjs`, `.ts`, `.tsx`, `.mts` and `.cts`. It applies the non-type-checked `typescript-eslint` recommended rules, so no
`tsconfig.json` is required; cross-file and type-aware checks remain the TypeScript compiler's job (`tsc`). Supply your
own `eslint.config.*` to enable type-aware rules or any other project-specific setup.

| Tool            | Workspace Config File                                                                                                                            |
| --------------- | ------------------------------------------------------------------------------------------------------------------------------------------------ |
| `editorconfig`  | `.editorconfig-checker.json`, `.ecrc`                                                                                                            |
| `prettier`      | `.prettierrc`, `.prettierrc.json`, `.prettierrc.yml`, `prettier.config.js`, ...                                                                  |
| `shfmt`         | -- (no config)                                                                                                                                   |
| `textlint`      | `.textlintrc`, `.textlintrc.json`, `.textlintrc.yaml`, `.textlintrc.yml`                                                                         |
| `markdownlint`  | `.markdownlint-cli2.yaml`, `.markdownlint-cli2.yml`, `.markdownlint-cli2.jsonc`, `.markdownlint.yaml`, `.markdownlint.yml`, `.markdownlint.json` |
| `eslint`        | `eslint.config.js`, `eslint.config.mjs`, `eslint.config.cjs`, `eslint.config.ts`, ...                                                            |
| `hadolint`      | `.hadolint.yaml`, `.hadolint.yml`                                                                                                                |
| `shellcheck`    | `.shellcheckrc`                                                                                                                                  |
| `yamllint`      | `.yamllint.yml`, `.yamllint.yaml`, `.yamllint`                                                                                                   |
| `actionlint`    | `.github/actionlint.yaml`, `.github/actionlint.yml`, `actionlint.yaml`, `actionlint.yml`                                                         |
| `golangci-lint` | `.golangci.yml`, `.golangci.yaml`, `.golangci.toml`, `.golangci.json`                                                                            |
| `ruff`          | `ruff.toml`, `.ruff.toml`, `pyproject.toml`                                                                                                      |
| `plainify`      | -- (no config)                                                                                                                                   |

## Options

| Flag                  | Description                                      |
| --------------------- | ------------------------------------------------ |
| `--nofix`, `--no-fix` | Check only, do not modify files                  |
| `--path <path>`       | Restrict scan to this path or file (repeatable)  |
| `--ignore <path>`     | Exclude this path or file from scan (repeatable) |
| `--pretty`            | Pretty-print JSON output                         |
| `--quiet`, `-q`       | Suppress progress output; JSON only on stdout    |

Pedant always skips generated, dependency, cache, and temporary directories such as `build/`, `dist/`, `node_modules/`,
`public/`, `target/`, `tmp/`, and `vendor/`. If an explicit `--path` selects files under one of those paths, pedant logs
a warning and omits those files from tool runs.

File discovery uses `git ls-files --exclude-standard`, so `.gitignore`, `.git/info/exclude`, and global Git ignore rules
are respected for untracked files, including when `--path` is used. Files already tracked by Git remain discoverable,
which matches Git's normal ignore behavior.

## Output

Progress is written to **stderr**. JSON is written to **stdout**:

```json
{
  "status": "fail",
  "workspace": "/work",
  "files_discovered": 24,
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
  ]
}
```

The top-level `status` is `"pass"` (all tools clean), `"fail"` (one or more tools reported findings), or `"error"` (one
or more tools could not run to completion). Only tools with findings or errors appear in the `tools` array. Tools with
no matching files or a clean result are omitted.

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

Check the current repository without modifying files:

```bash
docker run --rm -v "$(pwd):/work" ghcr.io/goeselt/pedant:latest --nofix
```

Check and autofix in one pass:

```bash
docker run --rm -v "$(pwd):/work" ghcr.io/goeselt/pedant:latest
```

Restrict the scan to specific paths or files:

```bash
docker run --rm -v "$(pwd):/work" ghcr.io/goeselt/pedant:latest --nofix --path src/ --path README.md
```

Exclude paths or files from the scan:

```bash
docker run --rm -v "$(pwd):/work" ghcr.io/goeselt/pedant:latest --nofix --ignore vendor/ --ignore generated/file.go
```

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) and [LICENSE](LICENSE).
