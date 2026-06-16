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
    local fallback="${2:-}"
    local value

    value="$(printenv "$key" 2>/dev/null || true)"
    if [[ -z "$value" && -n "$fallback" ]]; then
        value="$(printenv "INPUT_$fallback" 2>/dev/null || true)"
    fi
    printf '%s' "$value"
}

if [[ "${GITHUB_ACTIONS:-}" == "true" ]]; then
    args=()
    input_fix="${1:-$(input FIX)}"
    input_paths="${2:-$(input PATHS)}"
    input_ignore="${3:-$(input IGNORE)}"
    input_summary="${4:-$(input SUMMARY)}"
    input_summary_file="${5:-$(input SUMMARY-FILE SUMMARY_FILE)}"
    input_github_step_summary="${6:-$(input GITHUB-STEP-SUMMARY GITHUB_STEP_SUMMARY)}"

    [[ "$input_fix" == "true" ]] || args+=(--nofix)

    if [[ -n "$input_paths" ]]; then
        read -ra _paths <<<"$input_paths"
        for p in "${_paths[@]}"; do args+=(--path "$p"); done
    fi

    if [[ -n "$input_ignore" ]]; then
        read -ra _ignores <<<"$input_ignore"
        for ig in "${_ignores[@]}"; do args+=(--ignore "$ig"); done
    fi

    if [[ -n "$input_summary" ]]; then
        args+=(--summary "$input_summary")
    fi

    if [[ -n "$input_summary_file" ]]; then
        args+=(--summary-file "$input_summary_file")
    fi

    if [[ "$input_github_step_summary" == "true" ]]; then
        args+=(--github-step-summary)
    fi

    exec pedant "${args[@]}"
else
    exec pedant "$@"
fi
