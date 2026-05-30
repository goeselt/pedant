# Intentionally malformed Dockerfile for pedant e2e testing.
#
# Expected findings:
#   hadolint DL3006  -- missing version tag on FROM
#   hadolint DL3018  -- apk packages without version pins
#   hadolint DL3019  -- apk add without --no-cache

FROM alpine
RUN apk add curl git
