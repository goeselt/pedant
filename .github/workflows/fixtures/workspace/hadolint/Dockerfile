# Custom hadolint scenario: missing required 'maintainer' LABEL.
# FROM is properly tagged and RUN is clean, so the bundled config passes.
# The custom config requires a maintainer label via strict-labels, so it fails.
#
# Expected finding with custom config:
#   DL3049  -- Label `maintainer` is missing.
# No findings with bundled config.

FROM alpine:3.21
RUN echo "hello"
