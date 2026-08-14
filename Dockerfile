# Remotune Docker build environment
# All project toolchains live here; the Windows host is editor + orchestrator only.
#
# Base: Go 1.26.1 on Debian Bookworm (stable, good cross-compile support)
# Toolchain: Go 1.26.1, Bun (frontend, Phase 5), Windows cross-compile via CGO_ENABLED=0
# Wails CLI: deferred to Phase 5 (requires GTK3 dev libs for Linux compilation;
#            binding generation is only needed once a Vue frontend exists)
#
# Build:  docker compose build
# Use:    docker compose run --rm verify
#         docker compose run --rm build
#         docker compose run --rm shell

FROM golang:1.26.1-bookworm

# Pinned tool versions — bump deliberately, not automatically.
ARG BUN_VERSION=1.2.17

# System packages for general tooling.
# No mingw-w64: Wails v3 Windows builds are CGO-free (go-winloader).
# GTK/WebKit libs will be added in Phase 5 when wails3 CLI is needed for bindings.
RUN apt-get update && apt-get install -y --no-install-recommends \
        ca-certificates \
        curl \
        unzip \
        git \
    && rm -rf /var/lib/apt/lists/*

# Install Bun (frontend package manager and bundler for Vue/Vite, Phase 5).
RUN curl -fsSL https://bun.sh/install | bash -s "bun-v${BUN_VERSION}"
ENV BUN_INSTALL=/root/.bun
ENV PATH="${BUN_INSTALL}/bin:${PATH}"

# Pre-warm Go module cache with project dependencies.
# Copy only go.mod/go.sum first for layer caching.
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

# Default environment for Windows cross-compilation.
# CGO_ENABLED=0 because Wails v3 uses go-winloader (pure Go WebView2 bindings).
ENV GOOS=windows
ENV GOARCH=amd64
ENV CGO_ENABLED=0

# Copy full source. In compose, we bind-mount instead for live iteration.
COPY . .

# Default: build the Windows executable.
CMD ["go", "build", "-trimpath", "-ldflags=-s -w", "-o", "out/Remotune.exe", "."]
