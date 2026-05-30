#!/bin/bash
# Intentionally malformed shell script for pedant e2e testing.
#
# Expected findings (rule prefixes "SC" / "shfmt" deliberately not preceded by
# the literal word that shellcheck treats as a directive):
#   SC2006   -- backtick command substitution
#   SC2086   -- unquoted variable in echo
#   SC2292   -- POSIX [ ] instead of [[ ]] (require-double-brackets)
#   shfmt    -- bad indentation (echo not indented inside if)

RESULT=`date +%s`

if [ "$RESULT" = "" ]; then
echo "empty result"
fi

echo $RESULT
