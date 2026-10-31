package ytdlp

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Sergio-dot/urtube/internal/media"
	"github.com/lrstanley/go-ytdlp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeExecutable writes a shell script running body and returns its path.
func fakeExecutable(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "yt-dlp")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body+"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

var audioMP3 = media.Options{Type: media.MediaAudio, Format: "mp3"}

func testJob(opts media.Options) media.FetchJob {
	return media.FetchJob{
		Info:    media.VideoInfo{Raw: []byte(`{"id":"v1","title":"Song"}`)},
		Options: opts,
		Tags:    media.Tags{Artist: "Queen", Title: "Bohemian Rhapsody"},
		Dir:     "/downloads",
		Path:    []string{"Queen", "A Night At The Opera", "Bohemian Rhapsody"},
	}
}

func TestFetcher_Command(t *testing.T) {
	f := &Fetcher{}
	args := func(opts media.Options, trim bool) []string {
		return f.command(testJob(opts), "/tmp/urtube/d/info.json", "/tmp/urtube/d", trim).BuildCommand(context.Background()).Args
	}

	t.Run("downloads from the inspected info, into the planned path", func(t *testing.T) {
		a := args(audioMP3, false)
		assert.Subset(t, a, []string{"--load-info-json", "/tmp/urtube/d/info.json"})
		assert.Subset(t, a, []string{"--paths", "home:/downloads", "--paths", "temp:/tmp/urtube/d"})
		assert.Subset(t, a, []string{"--output", "Queen/A Night At The Opera/Bohemian Rhapsody.%(ext)s"})
		assert.Contains(t, a, "--windows-filenames")
		assert.Contains(t, a, "--embed-metadata")
	})

	t.Run("prints machine-readable progress", func(t *testing.T) {
		a := args(audioMP3, false)
		assert.Contains(t, a, "--newline")
		assert.Subset(t, a, []string{"--progress-template", "download:urtube-progress:%(progress)j"})
	})

	t.Run("audio: extracts, embeds square cover art", func(t *testing.T) {
		a := args(audioMP3, false)
		assert.Contains(t, a, "--extract-audio")
		assert.Subset(t, a, []string{"--audio-format", "mp3"})
		assert.Contains(t, a, "--embed-thumbnail")
		assert.Subset(t, a, []string{"--convert-thumbnails", "jpg", "--postprocessor-args", squareCover})
		assert.NotContains(t, a, "--recode-video")
	})

	t.Run("wav gets no cover art", func(t *testing.T) {
		a := args(media.Options{Type: media.MediaAudio, Format: "wav"}, false)
		assert.NotContains(t, a, "--embed-thumbnail")
	})

	t.Run("trimming removes non-music segments", func(t *testing.T) {
		assert.Subset(t, args(audioMP3, true), []string{"--sponsorblock-remove", "music_offtopic"})
		assert.NotContains(t, args(audioMP3, false), "--sponsorblock-remove")
	})

	t.Run("video merges best streams, recodes and embeds the thumbnail", func(t *testing.T) {
		a := args(media.Options{Type: media.MediaVideo, Format: "mkv"}, false)
		assert.Subset(t, a, []string{"--format", "bestvideo+bestaudio/best", "--recode-video", "mkv"})
		assert.Contains(t, a, "--embed-thumbnail")
		assert.NotContains(t, a, "--extract-audio")
		assert.NotContains(t, a, "--convert-thumbnails", "video keeps its 16:9 thumbnail")
	})

	t.Run("webm gets no thumbnail", func(t *testing.T) {
		assert.NotContains(t, args(media.Options{Type: media.MediaVideo, Format: "webm"}, false), "--embed-thumbnail")
	})

	t.Run("only allowlisted environment variables reach yt-dlp", func(t *testing.T) {
		t.Setenv("URTUBE_TEST_SECRET", "s3cret")
		t.Setenv("TZ", "Europe/Rome")

		env := f.command(testJob(audioMP3), "/tmp/i.json", "/tmp", false).BuildCommand(context.Background()).Env

		assert.Contains(t, env, "TZ=Europe/Rome")
		// PATH must survive, or yt-dlp cannot find ffmpeg.
		assert.True(t, slices.ContainsFunc(env, func(kv string) bool {
			return strings.HasPrefix(kv, "PATH=") && strings.HasSuffix(kv, os.Getenv("PATH"))
		}), "parent PATH not forwarded: %v", env)
		for _, kv := range env {
			assert.NotContains(t, kv, "URTUBE_TEST_SECRET")
		}
	})
}

func TestOutputTemplate(t *testing.T) {
	assert.Equal(t, "Artist/100%% Pure.%(ext)s", outputTemplate([]string{"Artist", "100% Pure"}))
}

func TestParseInfo(t *testing.T) {
	t.Run("YouTube Music song", func(t *testing.T) {
		raw := []byte(`{"id":"BSTsnWoslP4","title":"Bohemian Rhapsody","channel":"Queen Official","uploader":"Queen Official",
			"track":"Bohemian Rhapsody","artists":["Queen"],"album":"A Night At The Opera",
			"release_year":2011,"release_date":"19751121"}` + "\n")
		info, err := parseInfo(raw)
		require.NoError(t, err)
		assert.Equal(t, "Bohemian Rhapsody", info.Track)
		assert.Equal(t, []string{"Queen"}, info.Artists)
		assert.Equal(t, "A Night At The Opera", info.Album)
		assert.Equal(t, 1975, info.ReleaseYear, "the original release date wins over a remaster's year")
		assert.JSONEq(t, string(raw), string(info.Raw))
	})

	t.Run("legacy single artist field", func(t *testing.T) {
		info, err := parseInfo([]byte(`{"id":"x","artist":"Queen","release_year":1977}`))
		require.NoError(t, err)
		assert.Equal(t, []string{"Queen"}, info.Artists)
		assert.Equal(t, 1977, info.ReleaseYear)
	})

	t.Run("invalid JSON", func(t *testing.T) {
		_, err := parseInfo([]byte("not json"))
		assert.Error(t, err)
	})
}

func TestPatchInfo(t *testing.T) {
	raw := []byte(`{"id":"x","title":"Queen - Song (Official Video)","upload_date":"20190101","categories":["Music"],"filesize":12345678901234}`)
	out, err := patchInfo(raw, media.Tags{Artist: "Queen", Title: "Song", Album: "Album", Year: 1975})
	require.NoError(t, err)

	var got map[string]any
	dec := json.NewDecoder(strings.NewReader(string(out)))
	dec.UseNumber()
	require.NoError(t, dec.Decode(&got))
	assert.Equal(t, "Queen", got["meta_artist"])
	assert.Equal(t, "Queen", got["meta_album_artist"])
	assert.Equal(t, "Song", got["meta_title"])
	assert.Equal(t, "Album", got["meta_album"])
	assert.Equal(t, "1975", got["meta_date"])
	assert.Equal(t, "", got["meta_genre"], "the YouTube category isn't a genre")
	assert.Equal(t, "", got["meta_description"])
	assert.Equal(t, json.Number("12345678901234"), got["filesize"], "large numbers survive")
	assert.Equal(t, "Queen - Song (Official Video)", got["title"], "the original info is kept")

	out, err = patchInfo(raw, media.Tags{Artist: "Queen", Title: "Song"})
	require.NoError(t, err)
	assert.Contains(t, string(out), `"meta_date":""`, "an unknown year clears the upload date")
}

func TestParseProgress(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		expected media.Progress
		ok       bool
	}{
		{
			name:     "full progress",
			line:     `urtube-progress:{"status":"downloading","downloaded_bytes":1048576,"total_bytes":4194304,"speed":524288.5,"eta":6}`,
			expected: media.Progress{DownloadedBytes: 1048576, TotalBytes: 4194304, BytesPerSecond: 524288.5, ETA: 6 * time.Second},
			ok:       true,
		},
		{
			name:     "estimated total and unknown speed",
			line:     `urtube-progress:{"downloaded_bytes":10,"total_bytes":null,"total_bytes_estimate":100,"speed":null,"eta":null}`,
			expected: media.Progress{DownloadedBytes: 10, TotalBytes: 100},
			ok:       true,
		},
		{name: "other output", line: `[youtube] dQw4w9WgXcQ: Downloading webpage`},
		{name: "malformed JSON", line: `urtube-progress:{"downloaded_bytes":`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, ok := parseProgress([]byte(tt.line))
			assert.Equal(t, tt.ok, ok)
			assert.Equal(t, tt.expected, p)
		})
	}
}

func TestLineWriter(t *testing.T) {
	var lines []string
	w := &lineWriter{onLine: func(b []byte) { lines = append(lines, string(b)) }}

	_, _ = w.Write([]byte("first li"))
	_, _ = w.Write([]byte("ne\nsecond\r\n\nthird\rpartial"))

	assert.Equal(t, []string{"first line", "second", "third"}, lines)
}

func TestFetcher_Fetch(t *testing.T) {
	newFetcher := func(t *testing.T, script string) *Fetcher {
		return &Fetcher{
			Executable: fakeExecutable(t, script),
			TempDir:    t.TempDir(),
		}
	}
	job := testJob(audioMP3)
	job.Dir = t.TempDir()

	t.Run("reports progress and cleans up the temp dir", func(t *testing.T) {
		f := newFetcher(t, `
echo '[youtube] example: Downloading webpage'
echo 'urtube-progress:{"downloaded_bytes":50,"total_bytes":100,"speed":10,"eta":5}'
echo 'urtube-progress:{"downloaded_bytes":100,"total_bytes":100,"speed":10,"eta":0}'`)

		var got []media.Progress
		err := f.Fetch(context.Background(), job, func(p media.Progress) { got = append(got, p) })

		assert.NoError(t, err)
		assert.Equal(t, []media.Progress{
			{DownloadedBytes: 50, TotalBytes: 100, BytesPerSecond: 10, ETA: 5 * time.Second},
			{DownloadedBytes: 100, TotalBytes: 100, BytesPerSecond: 10},
		}, got)
		entries, _ := os.ReadDir(f.TempDir)
		assert.Empty(t, entries, "per-download temp dir must be removed")
	})

	t.Run("passes the patched info to yt-dlp", func(t *testing.T) {
		seen := filepath.Join(t.TempDir(), "info.json")
		f := newFetcher(t, `
while [ $# -gt 0 ]; do [ "$1" = --load-info-json ] && cp "$2" `+seen+`; shift; done`)

		require.NoError(t, f.Fetch(context.Background(), job, nil))
		info, err := os.ReadFile(seen)
		require.NoError(t, err)
		assert.Contains(t, string(info), `"meta_artist":"Queen"`)
	})

	t.Run("returns yt-dlp's error message", func(t *testing.T) {
		f := newFetcher(t, `
echo 'WARNING: something minor' >&2
echo 'ERROR: [youtube] xyz: Video unavailable' >&2
exit 1`)

		err := f.Fetch(context.Background(), job, nil)
		assert.EqualError(t, err, "[youtube] xyz: Video unavailable")
	})

	t.Run("retries without trimming when SponsorBlock is unreachable", func(t *testing.T) {
		calls := filepath.Join(t.TempDir(), "calls")
		f := newFetcher(t, `
echo "$*" >> `+calls+`
case "$*" in *--sponsorblock-remove*)
  echo 'ERROR: Postprocessing: Unable to communicate with SponsorBlock API: timed out' >&2; exit 1;;
esac`)
		trimmed := job
		trimmed.TrimNonMusic = true

		require.NoError(t, f.Fetch(context.Background(), trimmed, nil))
		lines, err := os.ReadFile(calls)
		require.NoError(t, err)
		runs := strings.Split(strings.TrimSpace(string(lines)), "\n")
		require.Len(t, runs, 2)
		assert.Contains(t, runs[0], "--sponsorblock-remove")
		assert.NotContains(t, runs[1], "--sponsorblock-remove")
	})

	t.Run("other errors are not retried", func(t *testing.T) {
		calls := filepath.Join(t.TempDir(), "calls")
		f := newFetcher(t, `echo x >> `+calls+`; echo 'ERROR: Video unavailable' >&2; exit 1`)
		trimmed := job
		trimmed.TrimNonMusic = true

		assert.Error(t, f.Fetch(context.Background(), trimmed, nil))
		lines, _ := os.ReadFile(calls)
		assert.Equal(t, "x\n", string(lines))
	})

	t.Run("invalid info is an error", func(t *testing.T) {
		bad := job
		bad.Info.Raw = []byte("nope")
		assert.Error(t, newFetcher(t, "exit 0").Fetch(context.Background(), bad, nil))
	})

	t.Run("missing executable", func(t *testing.T) {
		f := &Fetcher{Executable: filepath.Join(t.TempDir(), "nope")}
		assert.Error(t, f.Fetch(context.Background(), job, nil))
	})
}

func TestFetcher_Inspect(t *testing.T) {
	t.Run("parses yt-dlp's JSON", func(t *testing.T) {
		f := &Fetcher{Executable: fakeExecutable(t, `
for want in --dump-json --skip-download --no-playlist https://example.com/v; do
  case " $* " in *" $want "*) ;; *) echo "ERROR: missing $want in: $*" >&2; exit 1;; esac
done
echo '{"id":"v1","title":"Queen - Song","channel":"Queen"}'`)}

		info, err := f.Inspect(context.Background(), "https://example.com/v")
		require.NoError(t, err)
		assert.Equal(t, "v1", info.ID)
		assert.Equal(t, "Queen - Song", info.Title)
	})

	t.Run("returns yt-dlp's error", func(t *testing.T) {
		f := &Fetcher{Executable: fakeExecutable(t, `echo 'ERROR: [youtube] v1: Private video' >&2; exit 1`)}
		_, err := f.Inspect(context.Background(), "https://example.com/v")
		assert.EqualError(t, err, "[youtube] v1: Private video")
	})
}

func ptr[T any](v T) *T { return &v }

func TestFetcher_RemoveStaleTempDirs(t *testing.T) {
	t.Run("removes only per-download dirs", func(t *testing.T) {
		tempDir := t.TempDir()
		stale := filepath.Join(tempDir, "urtube-download-123")
		assert.NoError(t, os.MkdirAll(filepath.Join(stale, "nested"), 0o755))
		assert.NoError(t, os.WriteFile(filepath.Join(stale, "nested", "video.part"), []byte("x"), 0o644))
		other := filepath.Join(tempDir, "keep-me")
		assert.NoError(t, os.WriteFile(other, []byte("x"), 0o644))

		f := &Fetcher{TempDir: tempDir}
		assert.NoError(t, f.RemoveStaleTempDirs())

		assert.NoDirExists(t, stale)
		assert.FileExists(t, other)
	})

	t.Run("missing temp dir", func(t *testing.T) {
		f := &Fetcher{TempDir: filepath.Join(t.TempDir(), "missing")}
		assert.NoError(t, f.RemoveStaleTempDirs())
	})
}

func TestToVideo(t *testing.T) {
	t.Run("maps a regular video", func(t *testing.T) {
		v, ok := toVideo(&ytdlp.ExtractedInfo{
			ID:       "abc",
			Title:    ptr("Song"),
			URL:      ptr("https://youtube.com/watch?v=abc"),
			Channel:  ptr("Artist"),
			Duration: ptr(212.0),
			Thumbnails: []*ytdlp.ExtractedThumbnail{
				nil,
				{URL: "https://i.ytimg.com/a.jpg", Width: ptr(320), Height: ptr(180)},
			},
		})
		assert.True(t, ok)
		assert.Equal(t, media.Video{
			ID:         "abc",
			Title:      "Song",
			URL:        "https://youtube.com/watch?v=abc",
			Uploader:   "Artist",
			Duration:   212 * time.Second,
			Thumbnails: []media.Thumbnail{{URL: "https://i.ytimg.com/a.jpg", Width: 320, Height: 180}},
		}, v)
	})

	t.Run("detects live streams", func(t *testing.T) {
		v, ok := toVideo(&ytdlp.ExtractedInfo{
			Title:      ptr("Live"),
			URL:        ptr("https://youtube.com/watch?v=live"),
			LiveStatus: ptr(ytdlp.ExtractedLiveStatusIsLive),
		})
		assert.True(t, ok)
		assert.True(t, v.IsLive)
		assert.Zero(t, v.Duration)
	})

	t.Run("skips results without title or URL", func(t *testing.T) {
		_, ok := toVideo(&ytdlp.ExtractedInfo{Title: ptr("no url")})
		assert.False(t, ok)
		_, ok = toVideo(&ytdlp.ExtractedInfo{URL: ptr("https://youtube.com")})
		assert.False(t, ok)
		_, ok = toVideo(nil)
		assert.False(t, ok)
	})
}

func TestSearcher_Search(t *testing.T) {
	t.Run("returns mapped results up to the limit", func(t *testing.T) {
		s := &Searcher{Executable: fakeExecutable(t, `
echo '{"_type": "url", "id": "1", "title": "Video 1", "url": "https://youtube.com/watch?v=1"}'
echo '{"_type": "url", "id": "2", "title": "Video 2"}'
echo '{"_type": "url", "id": "3", "title": "Video 3", "url": "https://youtube.com/watch?v=3"}'
echo '{"_type": "url", "id": "4", "title": "Video 4", "url": "https://youtube.com/watch?v=4"}'`)}

		videos, err := s.Search(context.Background(), "test", 2, true)
		assert.NoError(t, err)
		assert.Len(t, videos, 2)
		assert.Equal(t, "Video 1", videos[0].Title)
		assert.Equal(t, "Video 3", videos[1].Title, "results without a URL are skipped")
	})

	t.Run("no output is an empty result", func(t *testing.T) {
		s := &Searcher{Executable: fakeExecutable(t, `true`)}
		videos, err := s.Search(context.Background(), "test", 5, true)
		assert.NoError(t, err)
		assert.Empty(t, videos)
	})

	t.Run("failed execution is an error", func(t *testing.T) {
		s := &Searcher{Executable: fakeExecutable(t, `exit 1`)}
		_, err := s.Search(context.Background(), "test", 5, true)
		assert.Error(t, err)
	})
}

func TestIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping yt-dlp integration test in short mode")
	}

	t.Run("search", func(t *testing.T) {
		videos, err := (&Searcher{}).Search(context.Background(), "Rick Roll", 5, true)
		assert.NoError(t, err)
		assert.Len(t, videos, 5)
	})

	t.Run("download audio", func(t *testing.T) {
		ctx := context.Background()
		f := &Fetcher{TempDir: t.TempDir()}
		dir := t.TempDir()

		info, err := f.Inspect(ctx, "https://www.youtube.com/watch?v=dQw4w9WgXcQ")
		require.NoError(t, err)
		tags := media.GuessTags(info)
		assert.Equal(t, "Rick Astley", tags.Artist)
		assert.Equal(t, "Never Gonna Give You Up", tags.Title)

		job := media.FetchJob{
			Info: info, Options: audioMP3, Tags: tags, Dir: dir,
			Path:         media.PlanPath(os.DirFS(dir), tags, media.MediaAudio),
			TrimNonMusic: true,
		}
		var progressed bool
		err = f.Fetch(ctx, job, func(p media.Progress) { progressed = progressed || p.DownloadedBytes > 0 })
		require.NoError(t, err)
		assert.True(t, progressed, "expected at least one progress update")

		file := filepath.Join(dir, "Rick Astley", "Never Gonna Give You Up.mp3")
		require.FileExists(t, file)

		// The tags and cover art must be inside the file, as Jellyfin reads them.
		out, err := exec.Command("ffprobe", "-v", "error", "-print_format", "json",
			"-show_entries", "format_tags=artist,album_artist,title:stream=codec_type,codec_name:stream_disposition=attached_pic",
			file).Output()
		require.NoError(t, err)
		var probe struct {
			Format struct {
				Tags map[string]string `json:"tags"`
			} `json:"format"`
			Streams []struct {
				CodecType   string         `json:"codec_type"`
				Disposition map[string]int `json:"disposition"`
			} `json:"streams"`
		}
		require.NoError(t, json.Unmarshal(out, &probe))
		assert.Equal(t, "Rick Astley", probe.Format.Tags["artist"])
		assert.Equal(t, "Rick Astley", probe.Format.Tags["album_artist"])
		assert.Equal(t, "Never Gonna Give You Up", probe.Format.Tags["title"])
		assert.True(t, slices.ContainsFunc(probe.Streams, func(s struct {
			CodecType   string         `json:"codec_type"`
			Disposition map[string]int `json:"disposition"`
		}) bool {
			return s.CodecType == "video" && s.Disposition["attached_pic"] == 1
		}), "expected embedded cover art: %s", out)
	})

	t.Run("lookup failure", func(t *testing.T) {
		f := &Fetcher{TempDir: t.TempDir()}
		_, err := f.Inspect(context.Background(), "https://example.com/invalid_video")
		assert.Error(t, err)
	})
}
