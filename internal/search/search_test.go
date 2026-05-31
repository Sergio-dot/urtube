package search

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func createMockExecutable(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "mock-ytdlp")
	err := os.WriteFile(path, []byte(content), 0755)
	if err != nil {
		t.Fatalf("failed to write mock executable: %v", err)
	}
	return path
}

func TestYtdlpSearcher_Search_Unit(t *testing.T) {
	t.Run("successful search returns results", func(t *testing.T) {
		mockScript := `#!/bin/sh
echo '{"_type": "url", "title": "Video 1", "url": "https://youtube.com/watch?v=1"}'
echo '{"_type": "url", "title": "Video 2", "url": "https://youtube.com/watch?v=2"}'
`
		exe := createMockExecutable(t, mockScript)
		searcher := &YtdlpSearcher{Executable: exe}

		results, err := searcher.Search(context.Background(), "test", 2, true)
		assert.NoError(t, err)
		assert.Len(t, results, 2)
		assert.Equal(t, "Video 1", *results[0].Title)
		assert.Equal(t, "https://youtube.com/watch?v=1", *results[0].URL)
	})

	t.Run("limit <= 0 normalisation uses default of 5", func(t *testing.T) {
		mockScript := `#!/bin/sh
echo '{"_type": "url", "title": "Video 1", "url": "https://youtube.com/watch?v=1"}'
echo '{"_type": "url", "title": "Video 2", "url": "https://youtube.com/watch?v=2"}'
`
		exe := createMockExecutable(t, mockScript)
		searcher := &YtdlpSearcher{Executable: exe}

		// limit <= 0 should use default which is 5. Since we only output 2 results, it should succeed and return both.
		results, err := searcher.Search(context.Background(), "test", 0, true)
		assert.NoError(t, err)
		assert.Len(t, results, 2)
	})

	t.Run("nil or missing title/url is skipped", func(t *testing.T) {
		mockScript := `#!/bin/sh
echo '{"_type": "url", "title": "Video 1"}' # missing url
echo '{"_type": "url", "url": "https://youtube.com/watch?v=2"}' # missing title
echo '{"_type": "url", "title": "Video 3", "url": "https://youtube.com/watch?v=3"}'
`
		exe := createMockExecutable(t, mockScript)
		searcher := &YtdlpSearcher{Executable: exe}

		results, err := searcher.Search(context.Background(), "test", 5, true)
		assert.NoError(t, err)
		assert.Len(t, results, 1)
		assert.Equal(t, "Video 3", *results[0].Title)
	})

	t.Run("no results returns ErrNoResults", func(t *testing.T) {
		mockScript := `#!/bin/sh
# output nothing
`
		exe := createMockExecutable(t, mockScript)
		searcher := &YtdlpSearcher{Executable: exe}

		results, err := searcher.Search(context.Background(), "test", 5, true)
		assert.ErrorIs(t, err, ErrNoResults)
		assert.Nil(t, results)
	})

	t.Run("failed execution returns ErrSearchFailed", func(t *testing.T) {
		mockScript := `#!/bin/sh
exit 1
`
		exe := createMockExecutable(t, mockScript)
		searcher := &YtdlpSearcher{Executable: exe}

		results, err := searcher.Search(context.Background(), "test", 5, true)
		assert.ErrorIs(t, err, ErrSearchFailed)
		assert.Nil(t, results)
	})
}

func TestYtdlpSearcher_Integration(t *testing.T) {
	// Skip if developer is just running fast unit tests
	if testing.Short() {
		t.Skip("skipping yt-dlp integration test in short mode")
	}

	tests := []struct {
		name            string
		query           string
		expectedError   error
		expectedResults int
		wantLiveStreams bool
	}{
		{
			name:            "Search with success 5 results include live streams",
			query:           "Rick Roll",
			expectedError:   nil,
			expectedResults: 5,
			wantLiveStreams: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			searcher := &YtdlpSearcher{}
			results, err := searcher.Search(context.Background(), tt.query, tt.expectedResults, tt.wantLiveStreams)
			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, results, tt.expectedResults)
			}
		})
	}
}
