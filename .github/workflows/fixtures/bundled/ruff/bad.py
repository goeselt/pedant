# Intentionally flawed Python file for pedant e2e testing.
#
# Expected findings with bundled (default) ruff config:
#   F401  -- unused import (os)
#   F841  -- unused variable (x)
#   ruff-format -- bad formatting (missing blank lines, inconsistent spacing)

import os
import sys

def main():
    x = 42
    print( sys.argv )

if __name__ == "__main__":
    main()
