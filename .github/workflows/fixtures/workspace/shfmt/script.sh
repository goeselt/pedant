#!/bin/bash
# Custom shfmt scenario: file uses 4-space indent (bundled default),
# workspace .editorconfig overrides indent_size to 2 so shfmt re-formats.
#
# Expected finding with custom config:
#   shfmt  -- "needs formatting"
# No findings with bundled config.

if true; then
    echo "four spaces"
fi
