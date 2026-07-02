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

# require_bool <input-name> <value>: fail with a clear message when a boolean
# input holds anything other than 'true', 'false', or ''. Without this, typos
# like 'True' or 'yes' would silently behave as 'false'.
require_bool() {
    local name="$1" value="$2"
    case "$value" in
    true | false | '') ;;
    *)
        printf "pedant: input '%s' must be 'true' or 'false', got '%s'\n" "$name" "$value" >&2
        exit 2
        ;;
    esac
}

if [[ "${GITHUB_ACTIONS:-}" == "true" ]]; then
    args=()
    fix=$(input FIX)
    paths=$(input PATHS)
    ignore=$(input IGNORE)
    tool_timeout=$(input TOOL_TIMEOUT)
    summary_markdown=$(input SUMMARY_MARKDOWN)
    summary_file=$(input SUMMARY_FILE)
    summary_github_step=$(input SUMMARY_GITHUB_STEP)

    require_bool fix "$fix"
    require_bool summary-markdown "$summary_markdown"
    require_bool summary-github-step "$summary_github_step"

    if [[ "$fix" == "true" ]]; then
        args+=(--fix)
    fi

    if [[ -n "$paths" ]]; then
        while IFS= read -r p; do
            p="${p%$'\r'}" # tolerate CRLF line endings in multiline inputs
            [[ -z "$p" ]] && continue
            args+=(--path "$p")
        done <<<"$paths"
    fi

    if [[ -n "$ignore" ]]; then
        while IFS= read -r ig; do
            ig="${ig%$'\r'}"
            [[ -z "$ig" ]] && continue
            args+=(--ignore "$ig")
        done <<<"$ignore"
    fi

    if [[ -n "$tool_timeout" ]]; then
        args+=(--tool-timeout "$tool_timeout")
    fi

    if [[ "$summary_markdown" == "true" ]]; then
        args+=(--summary-markdown)
    fi

    if [[ -n "$summary_file" ]]; then
        args+=(--summary-file "$summary_file")
    fi

    if [[ "$summary_github_step" == "true" ]]; then
        args+=(--summary-github-step)
    fi

    exec pedant "${args[@]}"
else
    exec pedant "$@"
fi
