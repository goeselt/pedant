#!/bin/bash
# Intentionally malformed shell script for pedant e2e testing.
#
# Expected finding:
#   shfmt  -- "needs formatting" (echo not indented inside if)

if true; then
echo "no indent"
fi
