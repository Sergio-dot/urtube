# syntax=docker/dockerfile:1

ARG GO_VERSION=1.25
ARG NODE_VERSION=22
ARG DEBIAN_RELEASE=trixie
ARG DENO_VERSION=2.9.8

# The frontend and Go stages run on the build machine's platform and
# cross-compile; only the runtime stage is per target platform.
FROM --platform=$BUILDPLATFORM node:${NODE_VERSION}-alpine AS web
WORKDIR /src/web
COPY web/package.json web/package-lock.json ./
RUN --mount=type=cache,target=/root/.npm npm ci
COPY web/ ./
RUN npm run build

FROM --platform=$BUILDPLATFORM golang:${GO_VERSION}-alpine AS build
ARG TARGETOS TARGETARCH
WORKDIR /src
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY . .
COPY --from=web /src/web/dist ./web/dist
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath -ldflags="-s -w" -o /out/urtube ./cmd/urtube

FROM denoland/deno:bin-${DENO_VERSION} AS deno

FROM debian:${DEBIAN_RELEASE}-slim

# yt-dlp breaks whenever YouTube changes, so bump these together often: the
# checksum is the "yt-dlp" line of the release's SHA2-256SUMS file.
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

# YouTube needs a JavaScript runtime for yt-dlp to get all formats; Deno is
# the one yt-dlp enables by default.
COPY --from=deno /deno /usr/local/bin/deno
ADD --checksum=sha256:${YTDLP_SHA256} --chmod=755 \
    https://github.com/yt-dlp/yt-dlp/releases/download/${YTDLP_VERSION}/yt-dlp /usr/local/bin/yt-dlp
COPY --from=build /out/urtube /usr/local/bin/urtube

RUN install -d -o 1000 -g 1000 /downloads

# The root filesystem can be read-only: caches go under /tmp (a tmpfs in
# compose.yaml) and in-progress downloads under /var/tmp (a volume).
ENV URTUBE_DOWNLOAD_DIR=/downloads \
    URTUBE_TEMP_DIR=/var/tmp/urtube \
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
