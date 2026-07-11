# check=skip=FromPlatformFlagConstDisallowed

# pedant -- unified linting and formatting in a single Docker image.
# Usage (local): docker build -t pedant . && docker run --rm -v "$(pwd):/work" pedant [options]

# ---- Stage 1: build the Go orchestrator binary ----
# hadolint ignore=DL3029
FROM --platform=linux/amd64 golang:1.24-alpine@sha256:8bee1901f1e530bfb4a7850aa7a479d17ae3a18beb6e09064ed54cfd245b7191 AS builder

WORKDIR /build
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o pedant ./cmd/pedant

# ---- Stage 2: final image ----
# hadolint ignore=DL3029
FROM --platform=linux/amd64 alpine:3.21@sha256:48b0309ca019d89d40f670aa1bc06e426dc0931948452e8491e3d65087abc07d

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

COPY tools/node/package.json tools/node/package-lock.json /opt/pedant-node-tools/

# hadolint ignore=DL3018
RUN apk add --no-cache nodejs npm \
    && npm ci --prefix /opt/pedant-node-tools --omit=dev --ignore-scripts \
    && npm cache clean --force \
    && apk del npm \
    && rm -rf /tmp/* /root/.npm \
    && /opt/pedant-node-tools/node_modules/.bin/prettier --version \
    && /opt/pedant-node-tools/node_modules/.bin/eslint --version \
    && /opt/pedant-node-tools/node_modules/.bin/markdownlint-cli2 --version \
    && /opt/pedant-node-tools/node_modules/.bin/textlint --version \
    && /opt/pedant-node-tools/node_modules/.bin/stylelint --version

# -- Python linter/formatter --

# renovate: datasource=github-releases depName=astral-sh/ruff
ARG RUFF_VERSION=0.15.21

RUN curl -fsSL -o /tmp/ruff.tar.gz \
        "https://github.com/astral-sh/ruff/releases/download/${RUFF_VERSION}/ruff-x86_64-unknown-linux-musl.tar.gz" \
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

RUN curl -fsSL -o /tmp/shellcheck.tar.xz \
        "https://github.com/koalaman/shellcheck/releases/download/v${SHELLCHECK_VERSION}/shellcheck-v${SHELLCHECK_VERSION}.linux.x86_64.tar.xz" \
    && tar -xJ --strip-components=1 -C /usr/local/bin -f /tmp/shellcheck.tar.xz "shellcheck-v${SHELLCHECK_VERSION}/shellcheck" \
    && chmod 755 /usr/local/bin/shellcheck \
    && rm /tmp/shellcheck.tar.xz \
    && shellcheck --version

# -- Dockerfile linter --

# renovate: datasource=github-releases depName=hadolint/hadolint
ARG HADOLINT_VERSION=2.14.0

RUN curl -fsSL \
        "https://github.com/hadolint/hadolint/releases/download/v${HADOLINT_VERSION}/hadolint-linux-x86_64" \
        -o /usr/local/bin/hadolint \
    && chmod 755 /usr/local/bin/hadolint \
    && hadolint --version

# -- GitHub Actions linter --

# renovate: datasource=github-releases depName=rhysd/actionlint
ARG ACTIONLINT_VERSION=1.7.12

RUN curl -fsSL -o /tmp/actionlint.tar.gz \
        "https://github.com/rhysd/actionlint/releases/download/v${ACTIONLINT_VERSION}/actionlint_${ACTIONLINT_VERSION}_linux_amd64.tar.gz" \
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
ARG EC_VERSION=3.8.0

RUN curl -fsSL -o /tmp/ec.tar.gz \
        "https://github.com/editorconfig-checker/editorconfig-checker/releases/download/v${EC_VERSION}/ec-linux-amd64.tar.gz" \
    && tar -xz -C /tmp -f /tmp/ec.tar.gz \
    && mv /tmp/bin/ec-linux-amd64 /usr/local/bin/ec \
    && chmod 755 /usr/local/bin/ec \
    && rm -rf /tmp/* \
    && ec --version

# -- Go linter --

# renovate: datasource=github-releases depName=golangci/golangci-lint
ARG GOLANGCI_VERSION=2.12.2

RUN curl -fsSL -o /tmp/golangci.tar.gz \
        "https://github.com/golangci/golangci-lint/releases/download/v${GOLANGCI_VERSION}/golangci-lint-${GOLANGCI_VERSION}-linux-amd64.tar.gz" \
    && tar -xzf /tmp/golangci.tar.gz \
        -C /usr/local/bin \
        --strip-components=1 \
        "golangci-lint-${GOLANGCI_VERSION}-linux-amd64/golangci-lint" \
    && chmod 755 /usr/local/bin/golangci-lint \
    && rm /tmp/golangci.tar.gz \
    && golangci-lint --version

# -- Text normalizer --

# renovate: datasource=github-releases depName=goeselt/plainify
ARG PLAINIFY_VERSION=1.1.0

RUN curl -fsSL -o /tmp/plainify.tar.gz \
        "https://github.com/goeselt/plainify/releases/download/v${PLAINIFY_VERSION}/plainify_linux_amd64.tar.gz" \
    && tar -xzf /tmp/plainify.tar.gz -C /tmp \
    && mv /tmp/plainify /usr/local/bin/plainify \
    && chmod 755 /usr/local/bin/plainify \
    && rm -rf /tmp/* \
    && plainify --version

# -- TOML formatter --

# renovate: datasource=github-releases depName=tamasfe/taplo
ARG TAPLO_VERSION=0.10.0

RUN curl -fsSL -o /tmp/taplo.gz \
        "https://github.com/tamasfe/taplo/releases/download/${TAPLO_VERSION}/taplo-linux-x86_64.gz" \
    && gunzip -c /tmp/taplo.gz > /usr/local/bin/taplo \
    && chmod 755 /usr/local/bin/taplo \
    && rm /tmp/taplo.gz \
    && taplo --version

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
    && ln -s /opt/pedant-node-tools/node_modules /etc/pedant/eslint/node_modules \
    && ln -s /opt/pedant-node-tools/node_modules /etc/pedant/stylelint/node_modules

# Intentionally running as root: pedant operates on a bind-mounted workspace
# whose files are owned by the host UID. A non-root user would need UID
# remapping or open permissions. Since this is an ephemeral lint container
# (no persistent state, no network services), the root default is acceptable.

# Opt out of telemetry and automatic update checks made by Node.js tools
# at startup. These would result in unexpected outbound network calls from
# an ephemeral lint container.
ENV DO_NOT_TRACK=1 \
    DISABLE_OPENCOLLECTIVE=1 \
    NEXT_TELEMETRY_DISABLED=1 \
    GATSBY_TELEMETRY_DISABLED=1

ENV NODE_PATH=/opt/pedant-node-tools/node_modules
ENV PATH=/opt/pedant-node-tools/node_modules/.bin:${PATH}

WORKDIR /work

LABEL org.opencontainers.image.title="pedant" \
    org.opencontainers.image.description="Pedant: unified linting and formatting orchestrator" \
    org.opencontainers.image.source="https://github.com/goeselt/pedant"

ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
