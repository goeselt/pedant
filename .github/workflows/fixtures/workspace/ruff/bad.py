# Custom ruff config scenario: enables additional rule sets (UP, SIM)
# that are not enabled in the default configuration.
#
# This file passes the default ruff config but fails with the custom
# ruff.toml that enables UP (pyupgrade) and SIM (flake8-simplify).
#
# Expected findings with custom config:
#   UP031  -- use format specifiers instead of percent format
#   SIM108 -- use ternary operator instead of if-else block

import sys


def greet(name):
    message = "Hello, %s" % name
    return message


def classify(value):
    if value > 0:
        result = "positive"
    else:
        result = "negative"
    return result


if __name__ == "__main__":
    print(greet(sys.argv[1] if len(sys.argv) > 1 else "World"))
