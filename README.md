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
   docker compose up -d
   ```

This runs the published image, `ghcr.io/sergio-dot/urtube`, built for amd64
and arm64 by CI from `main`. Tags:
- `latest`: the newest build of main.
- `sha-<commit>`: one specific build, to pin or roll back with
  `IMAGE_TAG=sha-1a2b3c4` in `.env`.
- `1.2.3` / `1.2`: releases, once `v*` tags are pushed.

Update with `docker compose pull && docker compose up -d`. To run your own
changes instead, build locally with `docker compose up -d --build`.

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

**Music search.** Next to the search box, *Videos* searches regular YouTube
and *Music* searches YouTube Music songs. Songs come with artist, album and
year, so their tags are right without editing; results show artist and
album. Music searches look each song up and take a few seconds (at most 20
results).

**How files are organized.** Audio goes to `Artist/Album/Title.ext`, or
`Artist/Title.ext` when the album is unknown; video is saved as `Title.ext`.
The download dialog prefills Artist, Title and Album and lets you correct
them (Download all uses the guesses):

- YouTube Music songs come with artist, album and year.
- Otherwise "Artist - Title" is read from the video title, dropping noise
  such as "(Official Video)" or "[Lyrics]", or the channel name is used
  without " - Topic", "VEVO" or "Official".
- Featured artists ("feat.", "ft.") move into the title, so they share the
  main artist's folder; "&" and "," are left alone (Simon & Garfunkel).
- An existing artist or album folder that differs only by case or spacing is
  reused, and the tags take its spelling.

The tags (title, artist, album artist, album, year) and the thumbnail,
cropped square as cover art, are embedded in the file, which is what
Jellyfin reads; WAV can't hold cover art. Audio downloads skip segments
SponsorBlock users marked as non-music (intros, skits), when the
SponsorBlock API is reachable. File names avoid characters Windows and SMB
shares reject (`AC/DC` is saved as `AC⧸DC`).

**Updating yt-dlp.** YouTube changes often break older yt-dlp releases. The
image pins yt-dlp with `YTDLP_VERSION` and `YTDLP_SHA256` in the `Dockerfile`.
A daily workflow opens a pull request when a new release is out, and the
image is republished once it is merged; then `docker compose pull`. To try a
release locally without editing the file:

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
