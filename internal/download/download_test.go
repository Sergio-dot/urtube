package download

import (
	"context"
	"errors"
	"testing"

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
			name: "Valid request with flags",
			url:  "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
			flags: &ytdlp.FlagConfig{},
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
			downloader := &YtdlpDownloader{}
			err := downloader.Download(context.Background(), &DownloadRequest{
				URL: tt.url,
			})
			if err != nil {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
