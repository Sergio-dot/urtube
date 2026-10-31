# urtube

A small self-hosted web app to search YouTube and download videos or audio
with [yt-dlp](https://github.com/yt-dlp/yt-dlp), straight into a folder such
as a Jellyfin library. It's a single Go binary with the React UI embedded,
meant for one household: one instance, no accounts.

## Running with Docker Compose

1. Pick the folder downloads should go to. It must already exist (Docker
   would otherwise create it owned by root, where urtube can't write), and
   note its owner: `stat -c '%u:%g' /path/to/library`.
2. Next to `compose.yaml`, create a `.env` file:

   ```sh
   LIBRARY_DIR=/srv/media/jellyfin/data/music
   PUID=1000
   PGID=1000
   # MEDIA_ROOT=/srv/media
   # HOST_PORT=8080
   ```

3. Start it and open `http://<host>:8080`:

   ```sh
   docker compose up -d --build
   ```

The container runs as `PUID:PGID` with a read-only root filesystem and no
capabilities. Settings changed in the app are kept in the `data` volume.
Finished files appear in the library only when complete:
in-progress downloads live in the `work` volume and leftovers from a crash are
removed on the next start.

**Download folder.** `LIBRARY_DIR` is the default download folder, a path on
the host. compose mounts host folders at the same path inside the container,
so the app's Settings (gear icon) shows and accepts the host's own paths. A
container can only write to folders mounted into it: Settings can pick any
folder inside `MEDIA_ROOT` (the folder mounted; defaults to `LIBRARY_DIR`)
and refuses others. With `MEDIA_ROOT=/srv/media`, for example, you can switch
between `/srv/media/jellyfin/data/music` and `/srv/media/podcasts`. Keep
`MEDIA_ROOT` as narrow as practical: anyone who can open urtube can save into
it. The folder chosen in Settings is saved and takes precedence over
`LIBRARY_DIR`; **Use default** goes back to it. If the folder isn't writable,
urtube still starts: the app shows the problem and refuses downloads until it
is fixed.

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
| `URTUBE_DOWNLOAD_DIR` | `./downloads` (`/downloads` in the image) | Default folder for finished files; a folder chosen in Settings takes precedence |
| `URTUBE_DOWNLOAD_ROOT` | *(none)* | If set, download folders (including one chosen in Settings) must be inside it |
| `URTUBE_TEMP_DIR` | `$TMPDIR/urtube` (`/var/tmp/urtube` in the image) | Where in-progress downloads are written |
| `URTUBE_DATA_DIR` | `./data` (`/data` in the image) | Where settings changed in the app are saved |
| `URTUBE_MAX_CONCURRENT_DOWNLOADS` | `2` | Downloads running at once; the rest wait in a queue |
| `URTUBE_SHUTDOWN_TIMEOUT` | `5s` | How long running downloads may continue after a stop signal |
| `URTUBE_LOG_LEVEL` | `info` | `debug`, `info`, `warn` or `error` |
| `URTUBE_LOG_JSON` | `false` (`true` in the image) | Log as JSON instead of text (application and request logs) |
| `URTUBE_LOG_CONCISE` | `true` | Shorter request logs |
| `URTUBE_LOG_REQUEST_HEADERS` | `true` | Include request headers in request logs |

Health endpoints: `/livez` and `/readyz` (503 while shutting down). The image's
`HEALTHCHECK` runs `urtube healthcheck`, which requests `/readyz`.

## Development

Requires Go 1.27, Node.js 22, and `yt-dlp` plus `ffmpeg` on `PATH`.

```sh
(cd web && npm ci && npm run build)  # writes internal/ui/dist, which the Go build embeds
go run ./cmd/urtube                  # http://localhost:8080
make test                            # unit tests (skips network tests)
```

For frontend work, run `npm run dev` in `web/`; Vite proxies `/api` to
`localhost:8080`.

## License

[MIT](LICENSE)
