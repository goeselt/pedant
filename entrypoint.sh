#!/usr/bin/env bash
# Docker entrypoint for pedant.
#
# GitHub Actions mode (GITHUB_ACTIONS=true): the Actions runner injects each
# input as an INPUT_<NAME> environment variable automatically. No positional
# arguments are passed from action.yml, so adding a new input only requires
# updating action.yml and reading INPUT_<NEWNAME> here.
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

    [[ "$(input FIX)" == "true" ]] && args+=(--fix)

    if [[ -n "$(input PATHS)" ]]; then
        while IFS= read -r p; do
            [[ -z "$p" ]] && continue
            args+=(--path "$p")
        done <<<"$(input PATHS)"
    fi

    if [[ -n "$(input IGNORE)" ]]; then
        while IFS= read -r ig; do
            [[ -z "$ig" ]] && continue
            args+=(--ignore "$ig")
        done <<<"$(input IGNORE)"
    fi

    [[ "$(input SUMMARY_MARKDOWN)" == "true" ]] && args+=(--summary-markdown)

    if [[ -n "$(input SUMMARY_FILE)" ]]; then
        args+=(--summary-file "$(input SUMMARY_FILE)")
    fi

    [[ "$(input SUMMARY_GITHUB_STEP)" == "true" ]] && args+=(--summary-github-step)

    exec pedant "${args[@]}"
else
    exec pedant "$@"
fi
