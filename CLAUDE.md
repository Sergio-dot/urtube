# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

urtube is a self-hosted YouTube search/download web app: a Go HTTP server (chi) wrapping the `yt-dlp` CLI via `github.com/lrstanley/go-ytdlp`, plus a React 19 + Vite + Tailwind v4 SPA in `web/` that is embedded into the Go binary.

Runtime requirements: `yt-dlp` on `PATH` (and `ffmpeg` for merging/conversion).

## Commands

Backend (repo root):

```sh
make test        # go test -short ./...  (skips yt-dlp integration tests)
make vet         # go vet ./...
make build       # go build ./...
make coverage    # coverage report
go test -race -v ./...                             # what CI runs (needs yt-dlp + network)
go test -short -run TestDownloadMedia ./internal/handlers   # single test
go run .         # start server (default localhost:8080)
```

Frontend (`web/`):

```sh
npm ci
npm run dev      # Vite dev server; proxies /api -> http://localhost:8080
npm run build    # tsc -b && vite build -> web/dist
npm run lint     # eslint
```

**`web/dist` must exist before any Go build/vet/test of the root package**: `main.go` has `//go:embed all:web/dist`, and `web/dist` is gitignored. Run `npm run build` in `web/` first (CI does this).

## Configuration

`internal/config` uses viper: reads `.env` from the CWD (optional, gitignored), overridden by environment variables. Keys: `SERVER_HOST`, `SERVER_PORT`, `DOWNLOAD_DIR` (default `./downloads`), `LOG_LEVEL`, `JSON`, `CONCISE`, `REQUEST_HEADERS` (the last three configure httplog).

## Architecture

Request flow: `main.go` → `config.Load` → builds `download.DownloadManager` (wrapping `YtdlpDownloader`) and `search.YtdlpSearcher` → `router.NewRouter(Dependencies{...})` → `server.NewServer` (binds the listener eagerly so `Addr()` works in tests).

- **Router** (`internal/router`): `/api/v1/*` for the API, `/healthz` heartbeat, and a catch-all that serves the embedded SPA with fallback to `index.html` for client-side routes. The search/download routes sit under a 30s `middleware.Timeout`; `/events` (SSE) is deliberately outside it.
- **Handler convention** (`pkg/httputils`): handlers have signature `func(w, r) error` and are wrapped with `httputils.MakeHandler`. Return `httputils.APIError{StatusCode, Message}` for client-facing errors; any other error becomes a logged 500. Write responses with `httputils.WriteJSON` / `httputils.Error` (JSON body `{"error": "..."}`).
- **Interfaces at the consumer**: each handler declares the narrow interface it needs (`handlers.DownloadManager`, `handlers.EventsDownloadManager`, `router.DownloadManager`, `search.Searcher`, `download.Downloader`). Tests use hand-written mocks of these plus `testify/assert`; there is no mocking library.
- **Async downloads** (`internal/download/manager.go`): `POST /api/v1/download` returns `202` with a UUID immediately; the download runs in a goroutine on a `context.Background()`-derived context (so it outlives the request) whose cancel func is stored by UUID. `DELETE /api/v1/download/{uuid}` calls it. Terminal status (`cancelled` / `error` / `finished`) is decided by checking the background context's `Err()` first, not by inspecting the returned error.
- **Progress via SSE**: the manager fans out `ProgressUpdate`s to subscriber channels (buffer 10); `GET /api/v1/events` subscribes one channel per client. `broadcast` intentionally drops updates for full channels rather than blocking the download goroutine.
- **Download request shape**: `download.DownloadRequest` passes `ytdlp.FlagConfig` straight through from the client JSON, so the frontend controls yt-dlp flags; output path is always joined under `DOWNLOAD_DIR`.
- **Search**: `GET /api/v1/search/{searchParam}?limit=&wantLiveStreams=` uses `ytsearchN:`; when excluding live streams it over-fetches 4× and filters.

Frontend: state lives mostly in `web/src/App.tsx`; `hooks/useDownloadEvents.ts` consumes the SSE stream; `types.ts` mirrors the Go JSON (`ProgressUpdate` must stay in sync with `download.ProgressUpdate`). All API calls use relative `/api/v1/...` paths.

## Testing notes

Tests that actually invoke `yt-dlp` (in `internal/search` and `internal/download`) guard with `if testing.Short() { t.Skip(...) }` — follow this for any new test that shells out or hits the network.

## Repository agent rules

`.agents/rules/no-modifications.md` (always on): do not modify, create, or delete files unless the user explicitly asks; review tasks produce a written report only. `.agents/workflows/be-review.md` and `fe-review.md` define read-only Go and React review checklists that output `review_report.md` at the repo root.
