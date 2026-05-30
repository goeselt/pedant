#!/usr/bin/env bash
# Pedant e2e behavioural test runner.
set -euo pipefail

# Each scenario directory under bundled/, workspace/ or clean/ contains a set
# of fixture files plus expected.json. expected.json is a compact behavioural
# spec, NOT the full pedant output:
#
#   {
#     "status": "pass" | "fail",
#     "tools_failing": ["<sorted tool names>"],
#     "tools_errored": ["<sorted tool names>"]
#   }
#
# The runner copies the scenario into a temporary git workspace (without
# expected.json), runs pedant --nofix --pretty, projects the JSON output down
# to the same compact shape, and diffs the two. Exact rule codes, line numbers
# and tool messages are deliberately not asserted -- those are tool concerns
# and are covered by the unit tests in internal/runner/parse_test.go.
#
# Usage:
#   ./run.sh --all                 # diff every scenario
#   ./run.sh bundled/hadolint      # diff one scenario
#   ./run.sh --update --all        # regenerate every expected.json
#   ./run.sh --update bundled/hadolint
#
# Environment:
#   PEDANT_IMAGE  Docker image tag (default: pedant:latest)

Script_Dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
readonly Script_Dir
readonly Image=${PEDANT_IMAGE:-pedant:latest}

# Projection jq program: full pedant output -> compact behavioural spec.
readonly Project_Jq='
{
  status: .status,
  tools_failing: ([.tools[]? | select(.status == "fail") | .name] | sort),
  tools_errored: ([.tools[]? | select(.status == "error") | .name] | sort)
}
'

usage() {
    cat >&2 <<EOF
Usage: $(basename "$0") [--update] (--all | scenario)

Examples:
  $(basename "$0") --all                 # diff every scenario
  $(basename "$0") bundled/hadolint      # diff a single scenario
  $(basename "$0") --update --all        # regenerate every expected.json
  $(basename "$0") --update workspace/eslint
EOF
}

# capture_actual <scenario_dir>: print the compact projection of pedant's
# --nofix output for a fresh git workspace cloned from scenario_dir minus its
# expected.json.
capture_actual() {
    local scenario_dir=$1
    local tmp
    tmp=$(mktemp -d)
    cp -a "$scenario_dir/." "$tmp/"
    rm -f "$tmp/expected.json"
    (
        cd "$tmp"
        git init -q
        git config user.email "e2e@pedant.local"
        git config user.name "pedant-e2e"
        git add -A
        git commit -qm init
    ) >/dev/null 2>&1
    local raw
    raw=$(docker run --rm -v "$tmp":/work "$Image" --nofix --pretty 2>/dev/null || true)
    rm -rf "$tmp"
    printf '%s\n' "$raw" | jq -S "$Project_Jq"
}

# run_one <scenario_rel> <update_flag>: diff or, with update_flag=1, overwrite expected.json.
run_one() {
    local scenario_rel=$1
    local update=$2
    local scenario_dir=$Script_Dir/$scenario_rel
    local expected=$scenario_dir/expected.json

    if [[ ! -d $scenario_dir ]]; then
        printf 'ERROR: scenario not found: %s\n' "$scenario_rel" >&2
        return 2
    fi

    local actual
    if ! actual=$(capture_actual "$scenario_dir"); then
        printf 'ERROR: %s -- pedant invocation or projection failed\n' "$scenario_rel" >&2
        return 2
    fi

    if [[ $update -eq 1 ]]; then
        printf '%s\n' "$actual" >"$expected"
        printf 'UPDATED: %s\n' "$scenario_rel"
        return 0
    fi

    if [[ ! -f $expected ]]; then
        printf 'ERROR: %s -- expected.json missing (use --update to bootstrap)\n' "$scenario_rel" >&2
        return 2
    fi

    local expected_norm
    expected_norm=$(jq -S '.' "$expected")

    if [[ "$expected_norm" == "$actual" ]]; then
        printf 'PASS: %s\n' "$scenario_rel"
        return 0
    fi

    printf 'FAIL: %s\n' "$scenario_rel" >&2
    diff -u <(printf '%s\n' "$expected_norm") <(printf '%s\n' "$actual") >&2 || true
    return 1
}

# list_scenarios: print every scenario path relative to Script_Dir. Anything
# below Script_Dir that contains an expected.json counts as a scenario.
list_scenarios() {
    local f d
    while IFS= read -r f; do
        d=${f%/expected.json}
        printf '%s\n' "${d#"$Script_Dir"/}"
    done < <(find "$Script_Dir" -mindepth 2 -name expected.json | sort)
}

# run_all <update_flag>: run every scenario and print a summary.
run_all() {
    local update=$1
    local fail=0 count=0 s
    while IFS= read -r s; do
        count=$((count + 1))
        run_one "$s" "$update" || fail=$((fail + 1))
    done < <(list_scenarios)

    printf '\n%d/%d passed\n' "$((count - fail))" "$count" >&2
    [[ $fail -eq 0 ]]
}

main() {
    if ! command -v jq >/dev/null 2>&1; then
        printf 'ERROR: jq is required on the host\n' >&2
        exit 2
    fi

    local update=0 all=0
    while [[ $# -gt 0 ]]; do
        case $1 in
        -h | --help)
            usage
            exit 0
            ;;
        --update)
            update=1
            shift
            ;;
        --all)
            all=1
            shift
            ;;
        -*)
            printf 'ERROR: unknown flag: %s\n' "$1" >&2
            usage
            exit 2
            ;;
        *)
            break
            ;;
        esac
    done

    if [[ $all -eq 1 && $# -gt 0 ]]; then
        printf 'ERROR: --all and a scenario argument are mutually exclusive\n' >&2
        exit 2
    fi

    if [[ $# -gt 1 ]]; then
        usage
        exit 2
    fi

    if [[ $# -eq 1 ]]; then
        run_one "$1" "$update"
        exit
    fi

    if [[ $all -eq 0 ]]; then
        usage
        exit 2
    fi

    run_all "$update"
}

main "$@"
