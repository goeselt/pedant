#!/bin/bash
# Custom config enables require-variable-braces (SC2250).
# The bundled default does NOT enable this check, so this file
# would pass with the bundled config but fail with the custom one.
#
# Expected finding with custom config (prefix "SC" deliberately not preceded by
# the literal word that shellcheck treats as a directive):
#   SC2250  -- $RESULT and $COUNT should use ${RESULT} and ${COUNT}

RESULT=$(date +%s)
COUNT=42

echo "$RESULT"
echo "$COUNT"
