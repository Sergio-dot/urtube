# syntax=docker/dockerfile:1

# Base images are pinned by digest for reproducible builds; Dependabot
# (.github/dependabot.yml) proposes updates to both tags and digests.

# The frontend and Go stages run on the build machine's platform and
# cross-compile; only the runtime stage is per target platform.
FROM --platform=$BUILDPLATFORM node:22-alpine@sha256:fe4f6d74e20370acb04ad889077d14d0f64a9c8ea6a42bc60cddf1e844699914 AS web
WORKDIR /src/web
COPY web/package.json web/package-lock.json ./
RUN --mount=type=cache,target=/root/.npm npm ci
COPY web/ ./
RUN npm run build

FROM --platform=$BUILDPLATFORM golang:1.27-alpine@sha256:d200d2dd76b03a99b52651fd1325aff7dcd35422c35c01bac56571b0507f3deb AS build
ARG TARGETOS TARGETARCH
WORKDIR /src
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY . .
COPY --from=web /src/internal/ui/dist ./internal/ui/dist
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath -ldflags="-s -w" -o /out/urtube ./cmd/urtube

# YouTube needs a JavaScript runtime for yt-dlp to get all formats; Deno is
# the one yt-dlp enables by default.
FROM denoland/deno:bin-2.9.8@sha256:b507583d3c68ba69f3ed2ce5969ae5b8b70f43421c781279c7aea06435263543 AS deno

FROM debian:trixie-slim@sha256:f6b08b09a05725f9431502d3b5714f5c4576e82c1a84fa1debe0c24a7e0c3ec4

# yt-dlp breaks whenever YouTube changes, so bump these together often: the
# checksum is the "yt-dlp" line of the release's SHA2-256SUMS file.
# .github/workflows/update-ytdlp.yml checks for new releases daily.
ARG YTDLP_VERSION=2026.10.26
ARG YTDLP_SHA256=e51e69e89489f199530f13d32a4342da3ddb2cf34842f364603a066db05afbf3

# ffmpeg merges and converts; python3 runs the yt-dlp zipapp; tini reaps the
# ffmpeg processes orphaned when a cancelled download's process group is killed.
# Versions come from the Debian release: pinning exact ones breaks the build
# as soon as a security update replaces them.
# hadolint ignore=DL3008
RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates ffmpeg python3 tini \
    && rm -rf /var/lib/apt/lists/*

COPY --from=deno /deno /usr/local/bin/deno
ADD --checksum=sha256:${YTDLP_SHA256} --chmod=755 \
    https://github.com/yt-dlp/yt-dlp/releases/download/${YTDLP_VERSION}/yt-dlp /usr/local/bin/yt-dlp
COPY --from=build /out/urtube /usr/local/bin/urtube

# /data holds settings saved from the app; world-writable (like /var/tmp) so
# a volume mounted there works whatever user the container runs as.
RUN install -d -o 1000 -g 1000 /downloads && install -d -m 1777 /data

# The root filesystem can be read-only: caches go under /tmp (a tmpfs in
# compose.yaml), in-progress downloads under /var/tmp and settings under /data
# (volumes).
ENV URTUBE_DOWNLOAD_DIR=/downloads \
    URTUBE_TEMP_DIR=/var/tmp/urtube \
    URTUBE_DATA_DIR=/data \
    URTUBE_LOG_JSON=true \
    HOME=/tmp \
    XDG_CACHE_HOME=/tmp/.cache

LABEL org.opencontainers.image.title="urtube" \
      org.opencontainers.image.description="Self-hosted YouTube search and download web app" \
      org.opencontainers.image.source="https://github.com/Sergio-dot/urtube" \
      org.opencontainers.image.licenses="MIT"

USER 1000:1000
WORKDIR /tmp
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=5s --start-period=20s --retries=3 \
    CMD ["/usr/local/bin/urtube", "healthcheck"]
ENTRYPOINT ["/usr/bin/tini", "--", "/usr/local/bin/urtube"]
