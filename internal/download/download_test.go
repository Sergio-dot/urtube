package download

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/lrstanley/go-ytdlp"
	"github.com/stretchr/testify/assert"
)

func TestValidate(t *testing.T) {
	tests := []struct {
		name          string
		url           string
		flags         *ytdlp.FlagConfig
		expectedError error
	}{
		{
			name:          "URL is required",
			url:           "",
			expectedError: errors.New("url is required"),
		},
		{
			name:          "Valid request with no flags",
			url:           "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
			expectedError: nil,
		},
		{
			name:          "Valid request with flags",
			url:           "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
			flags:         &ytdlp.FlagConfig{},
			expectedError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &DownloadRequest{
				URL:   tt.url,
				Flags: tt.flags,
			}
			err := req.Validate()
			if tt.expectedError != nil {
				assert.EqualError(t, err, tt.expectedError.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			downloader := &YtdlpDownloader{
				DownloadDir: t.TempDir(),
			}

			req := &DownloadRequest{
				URL: tt.url,
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

		manager.StartDownload(context.Background(), &DownloadRequest{URL: "test"})

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
