# urtube

A small self-hosted web app to search YouTube and download videos or audio
with [yt-dlp](https://github.com/yt-dlp/yt-dlp), straight into a folder such
as a Jellyfin library. It's a single Go binary with the React UI embedded,
meant for one household: one instance, no accounts.

## Running with Docker Compose

1. Find the owner of the folder downloads should go to: `stat -c '%u:%g' /path/to/library`.
2. Next to `compose.yaml`, create a `.env` file:

   ```sh
   LIBRARY_DIR=/srv/media/jellyfin/data/music
   PUID=1000
   PGID=1000
   # HOST_PORT=8080
   ```

3. Start it and open `http://<host>:8080`:

   ```sh
   docker compose up -d --build
   ```

The container runs as `PUID:PGID` with a read-only root filesystem and no
capabilities. Finished files appear in the library only when complete:
in-progress downloads live in the `work` volume and leftovers from a crash are
removed on the next start.

**Updating yt-dlp.** YouTube changes often break older yt-dlp releases. The
image pins yt-dlp with `YTDLP_VERSION` and `YTDLP_SHA256` in the `Dockerfile`;
take the new checksum from the `yt-dlp` line of the release's `SHA2-256SUMS`
and rebuild. To try a release without editing the file:

```sh
docker compose build --build-arg YTDLP_VERSION=<version> --build-arg YTDLP_SHA256=<sha256>
```

**Stopping.** On `docker compose stop`/`down`, running downloads get up to
`URTUBE_SHUTDOWN_TIMEOUT` (60s in `compose.yaml`) to finish before they are
cancelled; queued ones are cancelled immediately.

## Configuration

Environment variables (or a `.env` file in the working directory when running
the binary directly):

| Variable | Default | Description |
| --- | --- | --- |
| `URTUBE_HTTP_HOST` | *(all interfaces)* | Address to listen on |
| `URTUBE_HTTP_PORT` | `8080` | Port to listen on |
| `URTUBE_DOWNLOAD_DIR` | `./downloads` (`/downloads` in the image) | Where finished files are saved |
| `URTUBE_TEMP_DIR` | `$TMPDIR/urtube` (`/var/tmp/urtube` in the image) | Where in-progress downloads are written |
| `URTUBE_MAX_CONCURRENT_DOWNLOADS` | `2` | Downloads running at once; the rest wait in a queue |
| `URTUBE_SHUTDOWN_TIMEOUT` | `5s` | How long running downloads may continue after a stop signal |
| `URTUBE_LOG_LEVEL` | `info` | `debug`, `info`, `warn` or `error` |
| `URTUBE_LOG_JSON` | `false` | Log requests as JSON |
| `URTUBE_LOG_CONCISE` | `true` | Shorter request logs |
| `URTUBE_LOG_REQUEST_HEADERS` | `true` | Include request headers in request logs |

Health endpoints: `/livez` and `/readyz` (503 while shutting down). The image's
`HEALTHCHECK` runs `urtube healthcheck`, which requests `/readyz`.

## Development

Requires Go 1.25, Node.js 22, and `yt-dlp` plus `ffmpeg` on `PATH`.

```sh
(cd web && npm ci && npm run build)  # the Go build embeds web/dist
go run ./cmd/urtube                  # http://localhost:8080
make test                            # unit tests (skips network tests)
```

For frontend work, run `npm run dev` in `web/`; Vite proxies `/api` to
`localhost:8080`.

## License

[MIT](LICENSE)
