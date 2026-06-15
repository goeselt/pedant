# check=skip=FromPlatformFlagConstDisallowed

# pedant -- unified linting and formatting in a single Docker image.
# Usage (local): docker build -t pedant . && docker run --rm -v "$(pwd):/work" pedant [options]

# ---- Stage 1: build the Go orchestrator binary ----
# hadolint ignore=DL3029
FROM --platform=linux/amd64 golang:1.24-alpine AS builder

WORKDIR /build
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o pedant ./cmd/pedant

# ---- Stage 2: final image ----
# hadolint ignore=DL3029
FROM --platform=linux/amd64 alpine:3.21

SHELL ["/bin/ash", "-o", "pipefail", "-c"]

# ca-certificates needed for HTTPS downloads; curl for binary downloads; git for file discovery.
# safe.directory '*' lets git operate on bind-mounted workspaces owned by the host UID.
# Acceptable for an ephemeral lint container; do NOT reuse this image as a base for
# long-running services where untrusted users could plant a malicious .git config.
# hadolint ignore=DL3018
RUN apk add --no-cache ca-certificates curl git \
    && git config --system safe.directory '*'

# Copy Go 1.24 from the builder so golangci-lint sees the same toolchain version
# that projects targeting go 1.24 require. Alpine's packaged Go is 1.23.x.
COPY --from=builder /usr/local/go /usr/local/go
ENV GOROOT=/usr/local/go
ENV GOPATH=/go
ENV PATH=/usr/local/go/bin:${PATH}

# -- Node.js tools: prettier, eslint, markdownlint-cli2, textlint --

# renovate: datasource=npm depName=prettier
ARG PRETTIER_VERSION=3.8.3
# renovate: datasource=npm depName=eslint
ARG ESLINT_VERSION=10.4.1
# renovate: datasource=npm depName=eslint-plugin-unicorn
ARG UNICORN_VERSION=64.0.0
# renovate: datasource=npm depName=typescript-eslint
ARG TYPESCRIPT_ESLINT_VERSION=8.60.1
# renovate: datasource=npm depName=typescript
ARG TYPESCRIPT_VERSION=6.0.3
# renovate: datasource=npm depName=markdownlint-cli2
ARG MARKDOWNLINT_VERSION=0.22.1
# renovate: datasource=npm depName=textlint
ARG TEXTLINT_VERSION=15.7.1
# renovate: datasource=npm depName=stylelint
ARG STYLELINT_VERSION=16.14.0
# renovate: datasource=npm depName=stylelint-config-standard
ARG STYLELINT_CONFIG_STANDARD_VERSION=36.0.1

# hadolint ignore=DL3018
RUN apk add --no-cache nodejs npm \
    && npm install -g \
        "prettier@${PRETTIER_VERSION}" \
        "eslint@${ESLINT_VERSION}" \
        "jiti@2" \
        "eslint-plugin-unicorn@${UNICORN_VERSION}" \
        "@eslint/js@10" \
        "typescript-eslint@${TYPESCRIPT_ESLINT_VERSION}" \
        "typescript@${TYPESCRIPT_VERSION}" \
        "globals@17.6.0" \
        "markdownlint-cli2@${MARKDOWNLINT_VERSION}" \
        "textlint@${TEXTLINT_VERSION}" \
        "textlint-filter-rule-comments@1.3.0" \
        "textlint-rule-terminology@5.2.16" \
        "stylelint@${STYLELINT_VERSION}" \
        "stylelint-config-standard@${STYLELINT_CONFIG_STANDARD_VERSION}" \
    && npm cache clean --force \
    && apk del npm \
    && rm -rf /tmp/* /root/.npm \
    && prettier --version \
    && eslint --version \
    && markdownlint-cli2 --version \
    && textlint --version \
    && stylelint --version

# -- Python linter/formatter --

# renovate: datasource=github-releases depName=astral-sh/ruff
ARG RUFF_VERSION=0.15.16
ARG RUFF_SHA256=f52d90f8a6b1b3ad7d74301c3c796652e851d8f05b6ba26d139f05f4838cf412

RUN curl -fsSL -o /tmp/ruff.tar.gz \
        "https://github.com/astral-sh/ruff/releases/download/${RUFF_VERSION}/ruff-x86_64-unknown-linux-musl.tar.gz" \
    && printf '%s  /tmp/ruff.tar.gz\n' "${RUFF_SHA256}" | sha256sum -c - \
    && tar -xzf /tmp/ruff.tar.gz --strip-components=1 -C /usr/local/bin \
    && chmod 755 /usr/local/bin/ruff \
    && rm /tmp/ruff.tar.gz \
    && ruff version

# -- Shell tools --

# hadolint ignore=DL3018
RUN apk add --no-cache shfmt \
    && shfmt --version

# shellcheck (no Alpine package with current version; install from upstream release)
# renovate: datasource=github-releases depName=koalaman/shellcheck
ARG SHELLCHECK_VERSION=0.11.0
ARG SHELLCHECK_SHA256=8c3be12b05d5c177a04c29e3c78ce89ac86f1595681cab149b65b97c4e227198

RUN curl -fsSL -o /tmp/shellcheck.tar.xz \
        "https://github.com/koalaman/shellcheck/releases/download/v${SHELLCHECK_VERSION}/shellcheck-v${SHELLCHECK_VERSION}.linux.x86_64.tar.xz" \
    && printf '%s  /tmp/shellcheck.tar.xz\n' "${SHELLCHECK_SHA256}" | sha256sum -c - \
    && tar -xJ --strip-components=1 -C /usr/local/bin -f /tmp/shellcheck.tar.xz "shellcheck-v${SHELLCHECK_VERSION}/shellcheck" \
    && chmod 755 /usr/local/bin/shellcheck \
    && rm /tmp/shellcheck.tar.xz \
    && shellcheck --version

# -- Dockerfile linter --

# renovate: datasource=github-releases depName=hadolint/hadolint
ARG HADOLINT_VERSION=2.14.0
ARG HADOLINT_SHA256=6bf226944684f56c84dd014e8b979d27425c0148f61b3bd99bcc6f39e9dc5a47

RUN curl -fsSL \
        "https://github.com/hadolint/hadolint/releases/download/v${HADOLINT_VERSION}/hadolint-linux-x86_64" \
        -o /usr/local/bin/hadolint \
    && printf '%s  /usr/local/bin/hadolint\n' "${HADOLINT_SHA256}" | sha256sum -c - \
    && chmod 755 /usr/local/bin/hadolint \
    && hadolint --version

# -- GitHub Actions linter --

# renovate: datasource=github-releases depName=rhysd/actionlint
ARG ACTIONLINT_VERSION=1.7.12
ARG ACTIONLINT_SHA256=8aca8db96f1b94770f1b0d72b6dddcb1ebb8123cb3712530b08cc387b349a3d8

RUN curl -fsSL -o /tmp/actionlint.tar.gz \
        "https://github.com/rhysd/actionlint/releases/download/v${ACTIONLINT_VERSION}/actionlint_${ACTIONLINT_VERSION}_linux_amd64.tar.gz" \
    && printf '%s  /tmp/actionlint.tar.gz\n' "${ACTIONLINT_SHA256}" | sha256sum -c - \
    && tar -xzf /tmp/actionlint.tar.gz -C /usr/local/bin actionlint \
    && chmod 755 /usr/local/bin/actionlint \
    && rm /tmp/actionlint.tar.gz \
    && actionlint --version

# -- YAML linter --

# hadolint ignore=DL3018,DL3013
RUN apk add --no-cache python3 py3-pip \
    && pip install --no-cache-dir --break-system-packages yamllint==1.38.0 \
    && apk del py3-pip \
    && rm -rf /var/cache/apk/* \
    && yamllint --version

# -- EditorConfig checker --

# renovate: datasource=github-releases depName=editorconfig-checker/editorconfig-checker
ARG EC_VERSION=3.7.0
ARG EC_SHA256=9a0c3a5170bffa24f9e5f0def53d285777b6c5284a95367f40d399d0b76af552

RUN curl -fsSL -o /tmp/ec.tar.gz \
        "https://github.com/editorconfig-checker/editorconfig-checker/releases/download/v${EC_VERSION}/ec-linux-amd64.tar.gz" \
    && printf '%s  /tmp/ec.tar.gz\n' "${EC_SHA256}" | sha256sum -c - \
    && tar -xz -C /tmp -f /tmp/ec.tar.gz \
    && mv /tmp/bin/ec-linux-amd64 /usr/local/bin/ec \
    && chmod 755 /usr/local/bin/ec \
    && rm -rf /tmp/* \
    && ec --version

# -- Go linter --

# renovate: datasource=github-releases depName=golangci/golangci-lint
ARG GOLANGCI_VERSION=2.12.2
ARG GOLANGCI_SHA256=8df580d2670fed8fa984aac0507099af8df275e665215f5c7a2ae3943893a553

RUN curl -fsSL -o /tmp/golangci.tar.gz \
        "https://github.com/golangci/golangci-lint/releases/download/v${GOLANGCI_VERSION}/golangci-lint-${GOLANGCI_VERSION}-linux-amd64.tar.gz" \
    && printf '%s  /tmp/golangci.tar.gz\n' "${GOLANGCI_SHA256}" | sha256sum -c - \
    && tar -xzf /tmp/golangci.tar.gz \
        -C /usr/local/bin \
        --strip-components=1 \
        "golangci-lint-${GOLANGCI_VERSION}-linux-amd64/golangci-lint" \
    && chmod 755 /usr/local/bin/golangci-lint \
    && rm /tmp/golangci.tar.gz \
    && golangci-lint --version

# -- Text normalizer --

# renovate: datasource=github-releases depName=goeselt/plainify
ARG PLAINIFY_VERSION=1.0.1
ARG PLAINIFY_SHA256=cd14fd5796784e44e6d08c667f68f6538501d7c748d4cbce18c36134c340f212

RUN curl -fsSL -o /tmp/plainify.tar.gz \
        "https://github.com/goeselt/plainify/releases/download/v${PLAINIFY_VERSION}/plainify_linux_amd64.tar.gz" \
    && printf '%s  /tmp/plainify.tar.gz\n' "${PLAINIFY_SHA256}" | sha256sum -c - \
    && tar -xzf /tmp/plainify.tar.gz -C /usr/local/bin plainify \
    && chmod 755 /usr/local/bin/plainify \
    && rm /tmp/plainify.tar.gz \
    && plainify --version

# -- Remove curl now that all downloads are done --
RUN apk del curl && rm -rf /var/cache/apk/*

# -- Copy orchestrator binary, default configs, and entrypoint --
COPY --from=builder /build/pedant /usr/local/bin/pedant

COPY configs/ /etc/pedant/
COPY entrypoint.sh /usr/local/bin/entrypoint.sh
# hadolint ignore=DL3018
RUN apk add --no-cache bash && chmod 755 /usr/local/bin/entrypoint.sh

# Place bundled configs where tools can find them:
# - .editorconfig at / so ec finds it via upward traversal for repos without their own
# - node_modules symlink so the bundled eslint config can resolve its npm imports
RUN ln -s /etc/pedant/editorconfig/.editorconfig /.editorconfig \
    && ln -s /usr/local/lib/node_modules /etc/pedant/eslint/node_modules \
    && ln -s /usr/local/lib/node_modules /etc/pedant/stylelint/node_modules

# Intentionally running as root: pedant operates on a bind-mounted workspace
# whose files are owned by the host UID. A non-root user would need UID
# remapping or open permissions. Since this is an ephemeral lint container
# (no persistent state, no network services), the root default is acceptable.

WORKDIR /work

LABEL org.opencontainers.image.title="pedant" \
    org.opencontainers.image.description="Pedant: unified linting and formatting orchestrator" \
    org.opencontainers.image.source="https://github.com/goeselt/pedant"

ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
