#!/usr/bin/env bash
# Docker entrypoint for pedant.
#
# GitHub Actions mode (GITHUB_ACTIONS=true): translates INPUT_* env vars that
# the Actions runner injects from action.yml inputs into pedant CLI flags.
#
# Direct invocation mode: passes all arguments through to pedant unchanged so
# that local `docker run --rm -v $(pwd):/work pedant [options]` keeps working.
set -euo pipefail

if [[ "${GITHUB_ACTIONS:-}" == "true" ]]; then
    args=()
    [[ "${INPUT_FIX:-false}" == "true" ]] || args+=(--nofix)

    if [[ -n "${INPUT_PATHS:-}" ]]; then
        read -ra _paths <<<"${INPUT_PATHS}"
        for p in "${_paths[@]}"; do args+=(--path "$p"); done
    fi

    if [[ -n "${INPUT_IGNORE:-}" ]]; then
        read -ra _ignores <<<"${INPUT_IGNORE}"
        for ig in "${_ignores[@]}"; do args+=(--ignore "$ig"); done
    fi

    exec pedant "${args[@]}"
else
    exec pedant "$@"
fi
