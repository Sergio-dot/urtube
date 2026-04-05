package search

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

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
	}{
		{
			name:            "Search with success",
			query:           "Rick Roll",
			expectedError:   nil,
			expectedResults: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			searcher := &YtdlpSearcher{}
			results, err := searcher.Search(context.Background(), tt.query)
			if err != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, results, tt.expectedResults)
			}
		})
	}
}
