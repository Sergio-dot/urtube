package download

import (
	"context"
	"errors"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestValidate(t *testing.T) {
	validURL := "https://www.youtube.com/watch?v=dQw4w9WgXcQ"
	videoMP4 := DownloadOptions{Type: MediaVideo, Format: "mp4"}

	tests := []struct {
		name          string
		url           string
		options       DownloadOptions
		expectedError error
	}{
		{name: "URL is required", url: "", options: videoMP4, expectedError: ErrURLRequired},
		{name: "URL must not be a flag", url: "--batch-file=/etc/passwd", options: videoMP4, expectedError: ErrInvalidURL},
		{name: "URL must be http or https", url: "file:///etc/passwd", options: videoMP4, expectedError: ErrInvalidURL},
		{name: "URL must have a host", url: "https://", options: videoMP4, expectedError: ErrInvalidURL},
		{name: "options are required", url: validURL, expectedError: ErrInvalidMediaType},
		{name: "unknown media type", url: validURL, options: DownloadOptions{Type: "exec", Format: "mp4"}, expectedError: ErrInvalidMediaType},
		{name: "format not allowed for type", url: validURL, options: DownloadOptions{Type: MediaAudio, Format: "mp4"}, expectedError: ErrInvalidFormat},
		{name: "valid video request", url: validURL, options: videoMP4},
		{name: "valid audio request", url: "http://example.com/v", options: DownloadOptions{Type: MediaAudio, Format: "mp3"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &DownloadRequest{
				URL:     tt.url,
				Options: tt.options,
			}
			err := req.Validate()
			if tt.expectedError != nil {
				assert.ErrorIs(t, err, tt.expectedError)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestYtdlpDownloader_NewCommand(t *testing.T) {
	d := &YtdlpDownloader{DownloadDir: "/downloads"}

	t.Run("audio extracts audio in the requested format", func(t *testing.T) {
		args := d.newCommand(DownloadOptions{Type: MediaAudio, Format: "mp3"}).
			BuildCommand(context.Background(), "https://example.com/v").Args

		assert.Contains(t, args, "--extract-audio")
		assert.Subset(t, args, []string{"--audio-format", "mp3"})
		assert.NotContains(t, args, "--recode-video")
		assert.Subset(t, args, []string{"--output", "/downloads/%(title)s.%(ext)s"})
		assert.Equal(t, "https://example.com/v", args[len(args)-1])
	})

	t.Run("video merges best streams and recodes", func(t *testing.T) {
		args := d.newCommand(DownloadOptions{Type: MediaVideo, Format: "mkv"}).
			BuildCommand(context.Background(), "https://example.com/v").Args

		assert.Subset(t, args, []string{"--format", "bestvideo+bestaudio/best", "--recode-video", "mkv"})
		assert.NotContains(t, args, "--extract-audio")
	})

	t.Run("only allowlisted environment variables reach yt-dlp", func(t *testing.T) {
		t.Setenv("URTUBE_TEST_SECRET", "s3cret")
		t.Setenv("TZ", "Europe/Rome")

		env := d.newCommand(DownloadOptions{Type: MediaVideo, Format: "mp4"}).
			BuildCommand(context.Background(), "https://example.com/v").Env

		assert.Contains(t, env, "TZ=Europe/Rome")
		// PATH must survive, or yt-dlp cannot find ffmpeg for audio extraction/recoding.
		assert.True(t, slices.ContainsFunc(env, func(kv string) bool {
			return strings.HasPrefix(kv, "PATH=") && strings.HasSuffix(kv, os.Getenv("PATH"))
		}), "parent PATH not forwarded: %v", env)
		for _, kv := range env {
			assert.NotContains(t, kv, "URTUBE_TEST_SECRET")
		}
	})
}

func TestYtdlpDownloader_Integration(t *testing.T) {
	// Skip if developer is just running fast unit tests
	if testing.Short() {
		t.Skip("skipping yt-dlp integration test in short mode")
	}

	tests := []struct {
		name          string
		url           string
		expectedError error
	}{
		{
			name:          "Download with success",
			url:           "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
			expectedError: nil,
		},
		{
			name:          "Download failure - invalid URL",
			url:           "https://example.com/invalid_video",
			expectedError: errors.New("download failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			downloader := &YtdlpDownloader{
				DownloadDir: t.TempDir(),
			}

			req := &DownloadRequest{
				URL:     tt.url,
				Options: DownloadOptions{Type: MediaAudio, Format: "mp3"},
			}

			// Passing a nil callback for integration test
			err := downloader.Download(context.Background(), req, nil)
			if tt.expectedError != nil {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

type mockDownloader struct {
	DownloadFunc func(ctx context.Context, body *DownloadRequest, onProgress func(ProgressUpdate)) error
}

func (m *mockDownloader) Download(ctx context.Context, body *DownloadRequest, onProgress func(ProgressUpdate)) error {
	return m.DownloadFunc(ctx, body, onProgress)
}

func TestDownloadManager(t *testing.T) {
	t.Run("StartDownload triggers download and broadcasts", func(t *testing.T) {
		mock := &mockDownloader{
			DownloadFunc: func(ctx context.Context, body *DownloadRequest, onProgress func(ProgressUpdate)) error {
				onProgress(ProgressUpdate{Percent: "50%"})
				return nil
			},
		}

		manager := NewDownloadManager(mock)
		ch := manager.Subscribe()
		defer manager.Unsubscribe(ch)

		uuid, err := manager.StartDownload(context.Background(), &DownloadRequest{URL: "test"})
		assert.NoError(t, err)
		assert.NotEmpty(t, uuid)

		// We expect at least: downloading (start), 50% (from mock), and finished (end)
		updates := []string{}
		timeout := time.After(2 * time.Second)

	loop:
		for {
			select {
			case p := <-ch:
				updates = append(updates, p.Status)
				if p.Status == "finished" {
					break loop
				}
			case <-timeout:
				t.Fatal("timed out waiting for updates")
			}
		}

		assert.Contains(t, updates, "downloading")
		assert.Contains(t, updates, "finished")
	})

	t.Run("Manager handles download error", func(t *testing.T) {
		mock := &mockDownloader{
			DownloadFunc: func(ctx context.Context, body *DownloadRequest, onProgress func(ProgressUpdate)) error {
				return errors.New("failed")
			},
		}

		manager := NewDownloadManager(mock)
		ch := manager.Subscribe()
		defer manager.Unsubscribe(ch)

		uuid, err := manager.StartDownload(context.Background(), &DownloadRequest{URL: "test"})
		assert.NoError(t, err)
		assert.NotEmpty(t, uuid)

		timeout := time.After(1 * time.Second)
		var lastStatus string
		var errMsg string

	loop:
		for {
			select {
			case p := <-ch:
				lastStatus = p.Status
				errMsg = p.ErrorMessage
				if p.Status == "error" {
					break loop
				}
			case <-timeout:
				t.Fatal("timed out waiting for error")
			}
		}

		assert.Equal(t, "error", lastStatus)
		assert.Equal(t, "failed", errMsg)
	})
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		name     string
		input    int
		expected string
	}{
		{name: "negative value", input: -10, expected: "0 B"},
		{name: "zero bytes", input: 0, expected: "0 B"},
		{name: "bytes", input: 500, expected: "500 B"},
		{name: "KB", input: 1024, expected: "1.0 KiB"},
		{name: "MB", input: 1024 * 1024 * 5, expected: "5.0 MiB"},
		{name: "GB", input: 1024 * 1024 * 1024 * 2, expected: "2.0 GiB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, formatBytes(tt.input))
		})
	}
}

func TestDownloadManager_CancelDownload(t *testing.T) {
	t.Run("cancels active download and broadcasts cancelled status", func(t *testing.T) {
		mock := &mockDownloader{
			DownloadFunc: func(ctx context.Context, body *DownloadRequest, onProgress func(ProgressUpdate)) error {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(2 * time.Second):
					return nil
				}
			},
		}

		manager := NewDownloadManager(mock)
		ch := manager.Subscribe()
		defer manager.Unsubscribe(ch)

		uuid, err := manager.StartDownload(context.Background(), &DownloadRequest{URL: "test"})
		assert.NoError(t, err)
		assert.NotEmpty(t, uuid)

		// Wait a moment for goroutine to start
		time.Sleep(10 * time.Millisecond)

		// Cancel it
		ok := manager.CancelDownload(uuid)
		assert.True(t, ok)

		// We expect the status to be "cancelled" with no error message
		timeout := time.After(1 * time.Second)
		var lastStatus string

	loop:
		for {
			select {
			case p := <-ch:
				lastStatus = p.Status
				if p.Status == statusCancelled {
					break loop
				}
			case <-timeout:
				t.Fatal("timed out waiting for cancelled status")
			}
		}

		assert.Equal(t, statusCancelled, lastStatus)
	})

	t.Run("returns false for unknown uuid", func(t *testing.T) {
		manager := NewDownloadManager(&mockDownloader{})
		ok := manager.CancelDownload("non-existent-uuid")
		assert.False(t, ok)
	})

	t.Run("returns false after download already finished", func(t *testing.T) {
		doneCh := make(chan struct{})
		mock := &mockDownloader{
			DownloadFunc: func(ctx context.Context, body *DownloadRequest, onProgress func(ProgressUpdate)) error {
				close(doneCh)
				return nil
			},
		}

		manager := NewDownloadManager(mock)
		ch := manager.Subscribe()
		defer manager.Unsubscribe(ch)

		uuid, err := manager.StartDownload(context.Background(), &DownloadRequest{URL: "test"})
		assert.NoError(t, err)

		// Wait for the download goroutine to finish and clean up the cancellation entry
		<-doneCh
		// Drain channel to ensure the goroutine's deferred cleanup has run
		var finished bool
		timeout := time.After(1 * time.Second)
		for !finished {
			select {
			case p := <-ch:
				if p.Status == statusFinished {
					finished = true
				}
			case <-timeout:
				t.Fatal("timed out waiting for finished status")
			}
		}

		ok := manager.CancelDownload(uuid)
		assert.False(t, ok)
	})
}

func TestDownloadManager_Broadcast(t *testing.T) {
	t.Run("drops progress ticks for a full subscriber", func(t *testing.T) {
		manager := NewDownloadManager(&mockDownloader{})
		ch := manager.Subscribe()
		defer manager.Unsubscribe(ch)

		for range cap(ch) + 5 {
			manager.broadcast(ProgressUpdate{UUID: "a", Status: statusDownloading})
		}

		assert.Len(t, ch, cap(ch))
	})

	t.Run("terminal update is delivered even when the subscriber is full", func(t *testing.T) {
		manager := NewDownloadManager(&mockDownloader{})
		ch := manager.Subscribe()
		defer manager.Unsubscribe(ch)

		for range cap(ch) {
			manager.broadcast(ProgressUpdate{UUID: "a", Status: statusDownloading})
		}
		manager.broadcast(ProgressUpdate{UUID: "a", Status: statusFinished})

		var last ProgressUpdate
		for len(ch) > 0 {
			last = <-ch
		}
		assert.Equal(t, statusFinished, last.Status)
	})
}
