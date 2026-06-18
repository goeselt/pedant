#!/usr/bin/env bash
# Docker entrypoint for pedant.
#
# GitHub Actions mode (GITHUB_ACTIONS=true): translates INPUT_* env vars that
# the Actions runner injects from action.yml inputs into pedant CLI flags.
#
# Direct invocation mode: passes all arguments through to pedant unchanged so
# that local `docker run --rm -v $(pwd):/work pedant [options]` keeps working.
set -euo pipefail

input() {
    local key="INPUT_$1"
    local value
    value="$(printenv "$key" 2>/dev/null || true)"
    printf '%s' "$value"
}

if [[ "${GITHUB_ACTIONS:-}" == "true" ]]; then
    args=()
    input_fix="${1:-$(input FIX)}"
    input_paths="${2:-$(input PATHS)}"
    input_ignore="${3:-$(input IGNORE)}"
    input_summary_markdown="${4:-$(input SUMMARY_MARKDOWN)}"
    input_summary_file="${5:-$(input SUMMARY_FILE)}"
    input_summary_github_step="${6:-$(input SUMMARY_GITHUB_STEP)}"

    [[ "$input_fix" == "true" ]] || args+=(--nofix)

    if [[ -n "$input_paths" ]]; then
        read -ra _paths <<<"$input_paths"
        for p in "${_paths[@]}"; do args+=(--path "$p"); done
    fi

    if [[ -n "$input_ignore" ]]; then
        read -ra _ignores <<<"$input_ignore"
        for ig in "${_ignores[@]}"; do args+=(--ignore "$ig"); done
    fi

    if [[ "$input_summary_markdown" == "true" ]]; then
        args+=(--summary-markdown)
    fi

    if [[ -n "$input_summary_file" ]]; then
        args+=(--summary-file "$input_summary_file")
    fi

    if [[ "$input_summary_github_step" == "true" ]]; then
        args+=(--summary-github-step)
    fi

    exec pedant "${args[@]}"
else
    exec pedant "$@"
fi
