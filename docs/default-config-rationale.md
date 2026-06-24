---
title: 'Default Config Rationale'
type: reference
status: stable
---

# Default Config Rationale

Each bundled config ships a narrow set of settings chosen to be safe across diverse repositories. The rationale below
documents what each setting does and why it was chosen. Override any setting by placing a workspace config in your
repository root -- pedant will use it instead of the bundled default.

## EditorConfig (`.editorconfig`)

**Global (`[*]`)**

| Setting                    | Value   | Rationale                                                                                                                         |
| -------------------------- | ------- | --------------------------------------------------------------------------------------------------------------------------------- |
| `charset`                  | `utf-8` | Avoids cross-platform encoding issues. UTF-8 is the universal standard.                                                           |
| `end_of_line`              | `lf`    | LF is the POSIX standard. CRLF causes problems in shell scripts and on Linux/macOS. All other tools in pedant enforce LF as well. |
| `indent_size`              | `2`     | Web and config files (JSON, YAML, HTML, CSS, JS/TS) conventionally use 2-space indentation.                                       |
| `indent_style`             | `space` | Spaces render identically across all editors without any configuration.                                                           |
| `insert_final_newline`     | `true`  | Required by POSIX; `git diff` and many Unix tools misbehave on files without a trailing newline.                                  |
| `trim_trailing_whitespace` | `true`  | Trailing whitespace is invisible noise. Removing it keeps diffs clean.                                                            |

**`[*.go]`**

| Setting        | Value | Rationale                                                                                |
| -------------- | ----- | ---------------------------------------------------------------------------------------- |
| `indent_style` | `tab` | `gofmt` enforces tabs. Overriding it would put EditorConfig out of sync with Go tooling. |
| `indent_size`  | `tab` | Matches the `indent_style = tab` setting.                                                |

**`[*.{md,mdx}]`**

| Setting                    | Value   | Rationale                                                                                                                   |
| -------------------------- | ------- | --------------------------------------------------------------------------------------------------------------------------- |
| `indent_size`              | `unset` | Markdown uses indentation semantically (list continuation, nested lists). A fixed size would conflict with those semantics. |
| `trim_trailing_whitespace` | `false` | Two trailing spaces are a Markdown hard line break. Trimming them changes the rendered output.                              |

**`[*.py]`**

| Setting       | Value | Rationale                           |
| ------------- | ----- | ----------------------------------- |
| `indent_size` | `4`   | PEP 8 mandates 4-space indentation. |

**`[*.sh]`**

| Setting       | Value | Rationale                                                                    |
| ------------- | ----- | ---------------------------------------------------------------------------- |
| `indent_size` | `4`   | Conventional for shell; also the default `shfmt` indentation used by pedant. |

**C family (`[*.{c,cc,cpp,...}]`)**

| Setting       | Value | Rationale                                                                                                      |
| ------------- | ----- | -------------------------------------------------------------------------------------------------------------- |
| `indent_size` | `4`   | Conventional in C/C++ projects (K&R, LLVM, Google style all use 2 or 4; 4 is the safer cross-project default). |

**`[{Makefile,...}]`**

| Setting        | Value | Rationale                                                      |
| -------------- | ----- | -------------------------------------------------------------- |
| `indent_style` | `tab` | `make` requires tabs for recipe lines. Spaces break the build. |

---

## editorconfig-checker (`.editorconfig-checker.json`)

| Setting   | Value                                  | Rationale                                                                                                                               |
| --------- | -------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------- |
| `Exclude` | build, dist, node_modules, vendor, ... | Mirrors the directories pedant already skips. Generated and vendored files are not owned by the repository and should not be validated. |

---

## prettier (`.prettierrc`)

| Setting                      | Value         | Rationale                                                                                                                                                        |
| ---------------------------- | ------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `arrowParens`                | `"always"`    | Parentheses around a single parameter are required when adding a TypeScript type annotation. Enforcing them universally avoids churn when types are added later. |
| `bracketSameLine`            | `true`        | The closing `>` stays on the last attribute line. Reduces vertical space in JSX without affecting readability.                                                   |
| `bracketSpacing`             | `true`        | `{ foo }` is more readable than `{foo}` in object literals.                                                                                                      |
| `embeddedLanguageFormatting` | `"auto"`      | Formats fenced code blocks in Markdown and embedded scripts in HTML according to their language.                                                                 |
| `endOfLine`                  | `"lf"`        | Consistent with `.editorconfig`. Prevents CRLF from leaking into source files on Windows.                                                                        |
| `htmlWhitespaceSensitivity`  | `"css"`       | Respects the CSS `display` property when deciding whether whitespace between inline elements is significant. Most accurate for real-world HTML.                  |
| `jsxSingleQuote`             | `false`       | JSX attribute strings follow HTML convention (double quotes), which is distinct from JS string convention.                                                       |
| `printWidth`                 | `120`         | 80 columns is too narrow for modern monitors. 120 is a widely adopted modern default that fits most screens without horizontal scroll.                           |
| `proseWrap`                  | `"always"`    | Wraps Markdown prose at `printWidth`. Long lines in Markdown produce noisy diffs; reflowing at a fixed width keeps them manageable.                              |
| `quoteProps`                 | `"as-needed"` | Only quotes object keys that require it. Reduces noise for the common case.                                                                                      |
| `semi`                       | `false`       | Semicolons are optional and ASI is safe in modern JS when combined with `prettier`. Omitting them reduces visual noise.                                          |
| `singleAttributePerLine`     | `false`       | One attribute per line would expand simple JSX components over many lines. Only useful for large attribute sets, which are better handled by a custom override.  |
| `singleQuote`                | `true`        | Single quotes are the dominant convention in modern JS/TS projects.                                                                                              |
| `tabWidth`                   | `2`           | Consistent with `.editorconfig` and the project-wide default.                                                                                                    |
| `trailingComma`              | `"all"`       | Trailing commas in multi-line structures produce cleaner diffs: adding or removing the last item changes only one line. Valid in all modern JS/TS targets.       |
| `useTabs`                    | `false`       | Consistent with `.editorconfig`.                                                                                                                                 |

**YAML override (`printWidth: 10000`)**

Line breaks in YAML are semantic -- a wrapped scalar is not always equivalent to the original. Disabling wrapping by
setting an effectively infinite width prevents prettier from introducing semantics-changing line breaks in YAML files.

**Markdown override (`printWidth: 120, proseWrap: "always"`)**

Explicitly restates the global values so the Markdown behavior is visible and self-documenting.

**`.prettierignore`**

| Pattern                 | Rationale                                                        |
| ----------------------- | ---------------------------------------------------------------- |
| `*.lock`, `go.sum`      | Lockfiles are machine-generated and must not be reformatted.     |
| `*_gen.go`, `*.pb.go`   | Generated Go files are owned by code generators, not developers. |
| `*.min.js`, `*.min.css` | Minified files should remain minified.                           |
| Build / dependency dirs | Mirrors the global pedant skip list.                             |

---

## textlint (`.textlintrc.json`)

| Setting                          | Value            | Rationale                                                                                                                                                                  |
| -------------------------------- | ---------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `filters.comments`               | `true`           | Allows inline comment suppression (`<!-- textlint-disable -->`). Useful for blockquotes, code examples, or third-party text that intentionally violates terminology rules. |
| `rules.terminology.defaultTerms` | `true`           | Applies the canonical term list from `textlint-rule-terminology` (e.g. "JavaScript" not "JavaScript", "GitHub" not "GitHub").                                              |
| `rules.terminology.skip`         | `["Blockquote"]` | Blockquotes often contain verbatim third-party text. Enforcing terminology there would require altering the quote.                                                         |

---

## markdownlint (`.markdownlint-cli2.yaml`)

| Setting                        | Value                                 | Rationale                                                                                                                                                                                                                              |
| ------------------------------ | ------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `default: true`                | --                                    | Start from the full rule set, then adjust individual rules. Opting in selectively risks silently missing new rules.                                                                                                                    |
| `MD004 style: dash`            | `-`                                   | Consistent with the project's Markdown style guide (CLAUDE.md).                                                                                                                                                                        |
| `MD013 line_length: 120`       | --                                    | Matches `printWidth` in prettier.                                                                                                                                                                                                      |
| `MD013 code_blocks: false`     | --                                    | Line length is not meaningful inside code blocks. Wrapping code produces incorrect output.                                                                                                                                             |
| `MD013 tables: false`          | --                                    | Table column widths are determined by content; wrapping breaks the table syntax.                                                                                                                                                       |
| `MD022 lines_above/below: 1`   | --                                    | Exactly one empty line before and after a heading. Zero is visually cramped; two is excessive.                                                                                                                                         |
| `MD024 siblings_only: true`    | --                                    | Allows identical heading text in different sections (e.g., "Overview" under different top-level headings). Duplicate headings within the same parent are still flagged.                                                                |
| `MD025 front_matter_title: ''` | --                                    | Suppresses the "first line must be a heading" check when the title is already declared in YAML front matter.                                                                                                                           |
| `MD026 punctuation: '.,;:!'`   | --                                    | Headings should be noun phrases, not sentences. Trailing `.`, `,`, `;`, `:`, `!` is almost always a mistake. `?` is excluded because question headings (e.g., `## What is pedant?`) are a common and legitimate documentation pattern. |
| `MD029 style: ordered`         | --                                    | Ordered lists must use sequential numbers (1, 2, 3). Using `1.` for all items works in most renderers but hides the intended order.                                                                                                    |
| `MD033 allowed_elements`       | `br, details, kbd, sub, summary, sup` | A minimal set of HTML elements with no Markdown equivalent: `<br>` (hard line break), `<details>`/`<summary>` (collapsible sections), `<kbd>` (keyboard shortcuts), `<sub>`/`<sup>` (subscript/superscript).                           |
| `MD035 style: '---'`           | --                                    | Consistent horizontal rule syntax across all files.                                                                                                                                                                                    |
| `MD041 front_matter_title: ''` | --                                    | Same as MD025: suppress the first-heading check when a title is in front matter.                                                                                                                                                       |
| `MD046 style: fenced`          | --                                    | Fenced code blocks (` ``` `) over indented code blocks. Indented blocks cannot carry a language identifier.                                                                                                                            |
| `MD048 style: backtick`        | --                                    | Backticks (` ``` `) over tildes (`~~~`) for fenced code blocks. Backticks are the overwhelmingly dominant convention.                                                                                                                  |
| `MD049 style: asterisk`        | `*italic*`                            | Asterisk over underscore for emphasis. Consistent with `MD050`.                                                                                                                                                                        |
| `MD050 style: asterisk`        | `**bold**`                            | Asterisk over underscore for strong emphasis. Consistent with `MD049`.                                                                                                                                                                 |

---

## ESLint (`eslint.config.mjs`)

**Base config:**

| Setting                           | Value | Rationale                                                                                                                                                    |
| --------------------------------- | ----- | ------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `js.configs.recommended`          | --    | Standard ESLint rule baseline. Includes `no-unused-vars`, `no-undef`, and other essential correctness rules.                                                 |
| `ecmaVersion: 'latest'`           | --    | No need to target a specific ECMAScript version. Pedant lints modern source files, not transpiled output.                                                    |
| `sourceType: 'module'`            | --    | ESM is the default module system. CommonJS is handled per extension (`.cjs`).                                                                                |
| `globals.node` + `globals.es2025` | --    | Covers both Node.js built-ins (`process`, `Buffer`) and ECMAScript 2025 globals. Prevents false `no-undef` positives in server-side and modern browser code. |

**Rules:**

| Rule                                           | Severity                            | Rationale                                                                                                                                                                                                                |
| ---------------------------------------------- | ----------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `eqeqeq`                                       | error (`null: 'ignore'`)            | Strict equality everywhere. The `null: 'ignore'` exception allows `== null` as an idiomatic check for both `null` and `undefined`.                                                                                       |
| `no-console`                                   | warn                                | `console.*` calls are almost always debug artifacts. A warning rather than error allows intentional logging in server-side scripts without requiring a suppression comment.                                              |
| `no-constant-binary-expression`                | error                               | Expressions such as `a \|\| true` are always the same value. Almost always a logical mistake.                                                                                                                            |
| `no-constructor-return`                        | error                               | Returning a value from a constructor replaces the newly constructed object. Almost never intentional.                                                                                                                    |
| `no-duplicate-imports`                         | error                               | Multiple import declarations from the same module should be merged into one.                                                                                                                                             |
| `no-eval`                                      | error                               | `eval` executes arbitrary strings as code. A security risk and an obstacle to static analysis.                                                                                                                           |
| `no-extend-native`                             | error                               | Modifying built-in prototypes causes unpredictable behavior across libraries.                                                                                                                                            |
| `no-implicit-coercion`                         | error (`allow: ['!!']`)             | Implicit type coercions (e.g. `+str`, `~~num`) are obscure. `!!` is the idiomatic boolean cast and is widely understood.                                                                                                 |
| `no-implied-eval`                              | error                               | `setTimeout("code")` is equivalent to `eval`. Same risks as `no-eval`.                                                                                                                                                   |
| `no-lone-blocks`                               | error                               | Block statements outside control flow serve no purpose and obscure intent.                                                                                                                                               |
| `no-new-wrappers`                              | error                               | `new String("x")` creates an object, not a primitive. Comparison with `===` always fails.                                                                                                                                |
| `no-param-reassign`                            | warn                                | Reassigning function parameters makes data flow harder to follow. Warnings allow the occasional legitimate case without requiring a suppression comment.                                                                 |
| `no-return-assign`                             | error                               | Assignment inside a `return` is almost always a typo for comparison (`=` vs `===`).                                                                                                                                      |
| `no-self-compare`                              | error                               | `x === x` is always `true` and is almost certainly a mistake (or a `NaN` check that should use `Number.isNaN`).                                                                                                          |
| `no-sequences`                                 | error                               | The comma operator evaluates both operands and returns the last. Rarely intended outside minified code.                                                                                                                  |
| `no-shadow`                                    | error                               | A variable declaration that shadows an outer-scope variable makes the outer variable inaccessible and is a frequent source of subtle bugs in callbacks and nested functions.                                             |
| `no-template-curly-in-string`                  | warn                                | `"${x}"` in a regular string literal is probably a missing backtick. A warning covers typos without being noisy.                                                                                                         |
| `no-throw-literal`                             | error                               | Only `Error` objects (or subclasses) should be thrown. Throwing strings or plain objects makes `catch` handling inconsistent.                                                                                            |
| `no-unmodified-loop-condition`                 | error                               | A loop condition that never changes causes an infinite loop or dead code.                                                                                                                                                |
| `no-unneeded-ternary`                          | error                               | `x ? true : false` can always be written as `Boolean(x)` or just `!!x`.                                                                                                                                                  |
| `no-unused-expressions`                        | error                               | An expression with no side effects that is not assigned does nothing. Almost always a mistake.                                                                                                                           |
| `no-useless-call`                              | error                               | `fn.call(undefined, a)` is identical to `fn(a)`. The `.call` adds noise without value.                                                                                                                                   |
| `no-useless-computed-key`                      | error                               | `{ ["foo"]: 1 }` is identical to `{ foo: 1 }`.                                                                                                                                                                           |
| `no-useless-concat`                            | error                               | `"a" + "b"` should be written as `"ab"`.                                                                                                                                                                                 |
| `no-useless-rename`                            | error                               | `import { foo as foo }` is redundant.                                                                                                                                                                                    |
| `no-useless-return`                            | error                               | A `return` at the end of a function does nothing.                                                                                                                                                                        |
| `no-var`                                       | error                               | `let` and `const` have block scope. `var` has function scope and hoisting semantics that cause subtle bugs.                                                                                                              |
| `object-shorthand`                             | error (`'always'`)                  | `{ foo: foo }` should be `{ foo }`. Shorthand is more concise and universally supported.                                                                                                                                 |
| `prefer-arrow-callback`                        | error                               | Arrow functions are shorter and do not rebind `this`.                                                                                                                                                                    |
| `prefer-const`                                 | error                               | A binding that is never reassigned should be `const`. Makes intent explicit and prevents accidental reassignment.                                                                                                        |
| `prefer-destructuring`                         | warn (`object: true, array: false`) | `const { x } = obj` is clearer than `const x = obj.x` for objects. Array destructuring is position-based and can hurt readability; it is not enforced.                                                                   |
| `prefer-promise-reject-errors`                 | error                               | `Promise.reject` must be called with an `Error` object (or subclass). Rejecting with a string or plain object makes `catch` handlers inconsistent -- callers cannot rely on `.message`, `.stack`, or `instanceof Error`. |
| `prefer-rest-params`                           | error                               | `function f(...args)` is explicit. `arguments` is an implicit, array-like object with surprising behavior.                                                                                                               |
| `prefer-spread`                                | error                               | `fn(...args)` is explicit. `fn.apply(ctx, args)` is verbose and requires a context argument.                                                                                                                             |
| `prefer-template`                              | error                               | Template literals are clearer than concatenation for interpolated strings.                                                                                                                                               |
| `radix`                                        | error                               | `parseInt("08")` without a radix is implementation-defined (legacy octal in some environments). Always pass the radix.                                                                                                   |
| `require-await`                                | warn                                | An `async` function that never `await`s adds a needless Promise wrapper. A warning rather than error covers the case where `async` is required by an interface.                                                          |
| `symbol-description`                           | error                               | `Symbol("description")` is easier to debug than `Symbol()`. The description appears in stack traces and `console.log` output.                                                                                            |
| `unicorn/no-for-loop`                          | error                               | `for (let i = 0; i < arr.length; i++)` should be `for (const item of arr)`. The indexed loop is verbose and error-prone.                                                                                                 |
| `unicorn/no-instanceof-array`                  | error                               | `x instanceof Array` fails across iframes and realms. Use `Array.isArray(x)`.                                                                                                                                            |
| `unicorn/no-negated-condition`                 | error                               | `if (!cond) { A } else { B }` should be `if (cond) { B } else { A }`. The positive form is easier to read; the reader does not need to mentally negate the condition to understand the primary branch.                   |
| `unicorn/no-useless-spread`                    | error                               | `[...arr]` when `arr` is already an array adds a copy with no observable effect.                                                                                                                                         |
| `unicorn/no-useless-undefined`                 | error                               | Explicitly passing `undefined` where it is the default is redundant.                                                                                                                                                     |
| `unicorn/prefer-at`                            | error                               | `arr[arr.length - 1]` should be `arr.at(-1)`. Clearer and handles sparse arrays consistently.                                                                                                                            |
| `unicorn/prefer-includes`                      | error                               | `str.indexOf(x) !== -1` should be `str.includes(x)`.                                                                                                                                                                     |
| `unicorn/prefer-logical-operator-over-ternary` | error                               | `x !== null ? x : y` should be `x ?? y`; `x ? x : y` should be `x \|\| y`. The logical operator makes the intent explicit and is shorter.                                                                                |
| `unicorn/prefer-node-protocol`                 | error                               | `import fs from 'node:fs'` is unambiguous. Bare `'fs'` can be shadowed by a local module.                                                                                                                                |
| `unicorn/prefer-number-properties`             | error                               | `Number.isNaN` and `Number.isFinite` are not coercive. The global `isNaN`/`isFinite` coerce their argument first, which is rarely intended.                                                                              |
| `unicorn/prefer-string-slice`                  | error                               | `str.slice(start, end)` is simpler than `str.substring` or `str.substr` (deprecated).                                                                                                                                    |
| `unicorn/throw-new-error`                      | error                               | `throw Error("msg")` should be `throw new Error("msg")`. Explicit `new` is clearer and consistent across Error subclasses.                                                                                               |

**TypeScript section:**

| Setting                                                       | Value | Rationale                                                                                                                                                                                                                                                                                                             |
| ------------------------------------------------------------- | ----- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `tseslint.configs.recommended` scoped to `.ts,.tsx,.mts,.cts` | --    | Enables the TypeScript parser and plugin rules for TypeScript files only. The non-type-checked `recommended` set is used because pedant cannot assume a `tsconfig.json` exists in every repository. Type-aware rules (`recommended-type-checked`) require a project reference and belong in the project's own config. |
| `@typescript-eslint/consistent-type-imports`                  | error | Type-only imports must use `import type { Foo }`. This allows bundlers and the TypeScript compiler to elide the import entirely at emit time, and makes the intent explicit: the import exists solely for the type checker.                                                                                           |

**JSX section (`.jsx`)**

| Setting                  | Value | Rationale                                                                                                                                                  |
| ------------------------ | ----- | ---------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `ecmaFeatures.jsx: true` | --    | The default espree parser only accepts JSX syntax when explicitly enabled. Scoped to `.jsx` only; `.tsx` gets JSX via the TypeScript parser automatically. |

**CommonJS section (`.cjs`)**

| Setting                  | Value | Rationale                                                                                                                |
| ------------------------ | ----- | ------------------------------------------------------------------------------------------------------------------------ |
| `sourceType: 'commonjs'` | --    | `.cjs` files use `require`/`module.exports`. The global `sourceType: 'module'` setting would produce false parse errors. |

---

## stylelint (`.stylelintrc.json`)

| Setting                                     | Value | Rationale                                                                                                                                                                                                                                                                                  |
| ------------------------------------------- | ----- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `extends: stylelint-config-standard`        | --    | The official stylelint standard config. Covers property ordering, shorthand notation, and modern CSS conventions.                                                                                                                                                                          |
| `selector-class-pattern` (kebab-case / BEM) | --    | Kebab-case is the standard CSS naming convention. BEM (`block__element--modifier`) is the dominant methodology for component-based CSS. Both are valid in a default config. Utility-class frameworks (Tailwind, Bootstrap) use different conventions and require a project-level override. |

---

## ruff (`ruff.toml`)

| Setting                         | Value                  | Rationale                                                                                                                                                                        |
| ------------------------------- | ---------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `line-length = 120`             | --                     | Matches `printWidth` in prettier. PEP 8's 79-column limit is too narrow for modern screens. 120 is a widely used alternative that avoids horizontal scroll on standard monitors. |
| `lint.select: E, F, W`          | pycodestyle + pyflakes | Essential correctness rules. `E`/`W` cover style errors and warnings; `F` covers undefined names, unused imports, and other semantic mistakes.                                   |
| `lint.select: I`                | isort                  | Enforces a consistent import order. Reduces diff noise from unsorted imports.                                                                                                    |
| `lint.select: C`                | flake8-comprehensions  | Flags verbose equivalents of comprehensions and built-in calls: `list(x for x in y)` --> `[x for x in y]`, `dict([(k, v)])` --> `{k: v}`, etc. Keeps Python code idiomatic.      |
| `lint.select: UP`               | pyupgrade              | Flags syntax that can be replaced with a modern Python equivalent (e.g., `f-strings` over `.format()`, `X \| Y` over `Union[X, Y]`).                                             |
| `lint.select: B`                | flake8-bugbear         | Opinionated correctness rules: mutable default arguments, `assert` in tests, loop variable capture, etc.                                                                         |
| `lint.select: SIM`              | flake8-simplify        | Suggests simpler equivalents: `if x == True` --> `if x`, nested `with` --> combined `with`, etc.                                                                                 |
| `lint.select: TCH`              | flake8-type-checking   | Moves type-only imports into `TYPE_CHECKING` blocks, reducing runtime import overhead.                                                                                           |
| `lint.select: RUF`              | ruff-specific          | Additional rules specific to ruff, including implicit optional detection and ambiguous Unicode characters.                                                                       |
| `format.quote-style = "double"` | --                     | PEP 8 and Black convention. Single and double quotes are equivalent in Python; picking one eliminates the choice.                                                                |
| `format.indent-style = "space"` | --                     | PEP 8 requirement. Tabs are not valid in Python source under PEP 8.                                                                                                              |
| `format.line-ending = "lf"`     | --                     | Consistent with the global LF convention enforced by `.editorconfig` and prettier.                                                                                               |

---

## hadolint (`.hadolint.yaml`)

| Setting                    | Value | Rationale                                                                                                                                                                                                                                                                       |
| -------------------------- | ----- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `failure-threshold: style` | --    | Report all findings, including `style`-level suggestions. The default threshold (`info`) suppresses style findings. Pedant's intent is to surface all actionable feedback; style findings in Dockerfiles (e.g., `DL3008 Pin versions in apt-get install`) are worth addressing. |

---

## shellcheck (`.shellcheckrc`)

| Setting                             | Value | Rationale                                                                                                                                                                                     |
| ----------------------------------- | ----- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `shell=bash`                        | --    | Default to Bash when no shebang is present. Most scripts in a typical repository target Bash.                                                                                                 |
| `source-path=SCRIPTDIR`             | --    | When a script sources another file, look for it relative to the script's own directory first. Matches how scripts are typically organized.                                                    |
| `external-sources=true`             | --    | Follow `source` directives even when shellcheck cannot statically verify the sourced file exists. Prevents false "not found" errors in scripts that source files conditionally or at runtime. |
| `enable=add-default-case`           | --    | `case` statements without a default branch silently do nothing on unexpected input.                                                                                                           |
| `enable=avoid-nullary-conditions`   | --    | Conditions like `[ "$x" ]` test whether a variable is non-empty, but the intent is unclear. Explicit `[ -n "$x" ]` is unambiguous.                                                            |
| `enable=check-unassigned-uppercase` | --    | Uppercase variables are typically environment variables. Using one without assigning it first is likely a mistake.                                                                            |
| `enable=check-set-e-suppressed`     | --    | `if cmd; then` and `cmd \|\| true` suppress `set -e` for that command. This is often unintentional when `set -euo pipefail` is active.                                                        |
| `enable=deprecate-which`            | --    | `which` is not POSIX. `command -v` is the portable equivalent and is available in all POSIX-compliant shells.                                                                                 |
| `enable=quote-safe-variables`       | --    | Variables used in safe contexts (e.g., arithmetic) do not need quoting, but quoting them improves clarity and prevents surprises if the context changes.                                      |
| `enable=require-double-brackets`    | --    | `[[ ... ]]` is bash-specific but avoids word splitting, globbing, and operator ambiguities that affect `[ ... ]`. Consistent with the `shell=bash` default.                                   |

---

## yamllint (`.yamllint.yml`)

| Setting                                    | Value | Rationale                                                                                                                                                                                                            |
| ------------------------------------------ | ----- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `extends: default`                         | --    | Start from yamllint's default ruleset, then adjust.                                                                                                                                                                  |
| `braces.max-spaces-inside: 1`              | --    | Allows `{ key: val }` flow mapping style with a single space inside braces. The default (`0`) would disallow it.                                                                                                     |
| `brackets.max-spaces-inside: 1`            | --    | Same as `braces`, applied to flow sequences: `[a, b]` is valid; `[ a, b ]` is also allowed.                                                                                                                          |
| `comments.min-spaces-from-content: 1`      | --    | Allows `key: value # comment` (one space before `#`). The default requires two spaces, which is stricter than necessary.                                                                                             |
| `empty-lines.max: 1`                       | --    | More than one consecutive empty line adds no structural information.                                                                                                                                                 |
| `indentation.indent-sequences: consistent` | --    | Allows either `-` directly under the key or `-` indented relative to the key, as long as the choice is consistent within the file. Enforcing a single style would break compatibility with many existing YAML files. |
| `line-length: disable`                     | --    | Line length in YAML is enforced by prettier (which sets a high `printWidth` to avoid semantic line breaks). A second line-length check from yamllint would produce duplicate, potentially conflicting findings.      |
| `truthy.allowed-values: ['true', 'false']` | --    | Only lowercase `true`/`false` are allowed. YAML 1.1 also treated `yes`, `no`, `on`, `off` as booleans; YAML 1.2 does not. Restricting to `true`/`false` avoids ambiguity.                                            |
| `truthy.check-keys: false`                 | --    | Common key names like `enabled:` or `disabled:` would be falsely flagged. Only values need to follow the truthy convention, not keys.                                                                                |

---

## taplo (`taplo.toml`)

| Setting                    | Value  | Rationale                                                                                                |
| -------------------------- | ------ | -------------------------------------------------------------------------------------------------------- |
| `formatting.column_width`  | `120`  | Matches `printWidth` in prettier and `line-length` in ruff. Consistent line width across all formatters. |
| `formatting.indent_string` | `"  "` | Two-space indent matches `.editorconfig` default and the project-wide convention.                        |

**Workspace config** (`.taplo.toml` or `taplo.toml`): taplo reads the first of these it finds at the repository root.
Supply one to override any formatting option, or to add schema associations for known TOML file types (Cargo.toml,
pyproject.toml, etc.).

---

## golangci-lint (`.golangci.yml`)

**Linter selection:**

| Setting         | Value | Rationale                                                                                                                                                                                                  |
| --------------- | ----- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `default: none` | --    | Opt-in to specific linters rather than starting from a preset. Presets change across golangci-lint versions, which can introduce unexpected failures in CI.                                                |
| `bodyclose`     | --    | HTTP response bodies must be closed after reading. Not closing them leaks goroutines and file descriptors.                                                                                                 |
| `errcheck`      | --    | Every error return must be checked. Ignoring errors silently masks failures.                                                                                                                               |
| `errorlint`     | --    | Enforces correct error wrapping and unwrapping patterns (`%w`, `errors.Is`, `errors.As`). Incorrect wrapping loses the error chain.                                                                        |
| `gocritic`      | --    | Code quality and correctness checks not covered by `govet` or `staticcheck`.                                                                                                                               |
| `govet`         | --    | Equivalent to `go vet`. Catches suspicious constructs (e.g., wrong `Printf` format verbs, unreachable code).                                                                                               |
| `ineffassign`   | --    | Assignments whose value is never read before the variable is overwritten or goes out of scope are dead code.                                                                                               |
| `misspell`      | --    | Catches common spelling errors in comments and string literals.                                                                                                                                            |
| `dupword`       | --    | Catches duplicate consecutive words in comments ("the the"). Trivially fixable and a reliable signal of a copypaste or editing mistake.                                                                    |
| `nilerr`        | --    | Catches the pattern `if err != nil { return nil }` where a non-nil error is swallowed by returning `nil` instead of `err`. The caller receives a clean nil and has no way to detect the failure.           |
| `noctx`         | --    | Flags `http.Get`, `http.Post`, and similar functions that do not accept a `context.Context`. HTTP requests without a context cannot be cancelled or time-boxed, which leads to goroutine leaks under load. |
| `nolintlint`    | --    | Enforces that `//nolint` directives are specific and documented. Bare `//nolint` silently suppresses all linters.                                                                                          |
| `prealloc`      | --    | Suggests pre-allocating slices when the size is known at initialization, avoiding repeated re-allocations.                                                                                                 |
| `revive`        | --    | Opinionated Go style checks, used as a maintained replacement for the deprecated `golint`.                                                                                                                 |
| `staticcheck`   | --    | Deep static analysis: unused code, incorrect API usage, suspicious patterns. Catches issues that `govet` misses.                                                                                           |
| `unused`        | --    | Unexported identifiers that are never referenced are dead code.                                                                                                                                            |

**nolintlint settings:**

| Setting                     | Value | Rationale                                                                                                                                                         |
| --------------------------- | ----- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `require-explanation: true` | --    | Every `//nolint` directive must include a comment explaining why the finding is suppressed. Bare suppression hides technical debt.                                |
| `require-specific: true`    | --    | Every `//nolint` must name the specific linter(s) being suppressed (e.g., `//nolint:errcheck`). Blanket `//nolint` suppresses all linters, including future ones. |

**revive rules:**

The revive rules are a curated subset focused on three areas:

- **Naming** (`var-naming`, `receiver-naming`, `error-naming`): Enforces Go naming conventions (e.g., `err` for errors,
  short receiver names, no underscores in exported names).
- **Error handling** (`error-return`): The error return should be the last return value. `errors.Is`/`errors.As`
  patterns are covered by `errorlint`.
- **Code clarity** (`blank-imports`, `dot-imports`, `if-return`, `increment-decrement`, `range`, `var-declaration`,
  `context-as-argument`, `exported`): Removes redundant constructs and enforces idiomatic patterns.

**Issue exclusions:**

| Setting                      | Value | Rationale                                                                                                                          |
| ---------------------------- | ----- | ---------------------------------------------------------------------------------------------------------------------------------- |
| `exclude-dirs: vendor, _gen` | --    | Vendored dependencies and generated code are not owned by the repository. Linting them produces noise without actionable findings. |
