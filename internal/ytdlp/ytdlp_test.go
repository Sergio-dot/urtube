package ytdlp

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Sergio-dot/urtube/internal/media"
	"github.com/lrstanley/go-ytdlp"
	"github.com/stretchr/testify/assert"
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

func TestFetcher_Command(t *testing.T) {
	f := &Fetcher{}
	args := func(opts media.Options) []string {
		return f.command(opts, "/downloads", "/tmp/urtube/download-1").BuildCommand(context.Background(), "https://example.com/v").Args
	}

	t.Run("partial files go to the temp dir, finished files to the download dir", func(t *testing.T) {
		a := args(audioMP3)
		assert.Subset(t, a, []string{"--paths", "home:/downloads", "--paths", "temp:/tmp/urtube/download-1"})
		assert.Subset(t, a, []string{"--output", "%(title)s.%(ext)s"})
		assert.Equal(t, "https://example.com/v", a[len(a)-1])
	})

	t.Run("prints machine-readable progress", func(t *testing.T) {
		a := args(audioMP3)
		assert.Contains(t, a, "--newline")
		assert.Subset(t, a, []string{"--progress-template", "download:urtube-progress:%(progress)j"})
	})

	t.Run("audio extracts audio in the requested format", func(t *testing.T) {
		a := args(audioMP3)
		assert.Contains(t, a, "--extract-audio")
		assert.Subset(t, a, []string{"--audio-format", "mp3"})
		assert.NotContains(t, a, "--recode-video")
	})

	t.Run("video merges best streams and recodes", func(t *testing.T) {
		a := args(media.Options{Type: media.MediaVideo, Format: "mkv"})
		assert.Subset(t, a, []string{"--format", "bestvideo+bestaudio/best", "--recode-video", "mkv"})
		assert.NotContains(t, a, "--extract-audio")
	})

	t.Run("only allowlisted environment variables reach yt-dlp", func(t *testing.T) {
		t.Setenv("URTUBE_TEST_SECRET", "s3cret")
		t.Setenv("TZ", "Europe/Rome")

		env := f.command(audioMP3, "/downloads", "/tmp").BuildCommand(context.Background(), "https://example.com/v").Env

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
	req := media.Request{URL: "https://example.com/v", Options: audioMP3}

	t.Run("reports progress and cleans up the temp dir", func(t *testing.T) {
		f := newFetcher(t, `
echo '[youtube] example: Downloading webpage'
echo 'urtube-progress:{"downloaded_bytes":50,"total_bytes":100,"speed":10,"eta":5}'
echo 'urtube-progress:{"downloaded_bytes":100,"total_bytes":100,"speed":10,"eta":0}'`)

		var got []media.Progress
		err := f.Fetch(context.Background(), req, t.TempDir(), func(p media.Progress) { got = append(got, p) })

		assert.NoError(t, err)
		assert.Equal(t, []media.Progress{
			{DownloadedBytes: 50, TotalBytes: 100, BytesPerSecond: 10, ETA: 5 * time.Second},
			{DownloadedBytes: 100, TotalBytes: 100, BytesPerSecond: 10},
		}, got)
		entries, _ := os.ReadDir(f.TempDir)
		assert.Empty(t, entries, "per-download temp dir must be removed")
	})

	t.Run("returns yt-dlp's error message", func(t *testing.T) {
		f := newFetcher(t, `
echo 'WARNING: something minor' >&2
echo 'ERROR: [youtube] xyz: Video unavailable' >&2
exit 1`)

		err := f.Fetch(context.Background(), req, t.TempDir(), nil)
		assert.EqualError(t, err, "[youtube] xyz: Video unavailable")
	})

	t.Run("missing executable", func(t *testing.T) {
		f := &Fetcher{Executable: filepath.Join(t.TempDir(), "nope")}
		assert.Error(t, f.Fetch(context.Background(), req, t.TempDir(), nil))
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
		f := &Fetcher{TempDir: t.TempDir()}
		dir := t.TempDir()
		var progressed bool
		err := f.Fetch(context.Background(), media.Request{
			URL:     "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
			Options: audioMP3,
		}, dir, func(p media.Progress) { progressed = progressed || p.DownloadedBytes > 0 })

		assert.NoError(t, err)
		assert.True(t, progressed, "expected at least one progress update")
		files, _ := filepath.Glob(filepath.Join(dir, "*.mp3"))
		assert.Len(t, files, 1)
	})

	t.Run("download failure", func(t *testing.T) {
		f := &Fetcher{TempDir: t.TempDir()}
		err := f.Fetch(context.Background(), media.Request{URL: "https://example.com/invalid_video", Options: audioMP3}, t.TempDir(), nil)
		assert.Error(t, err)
	})
}
