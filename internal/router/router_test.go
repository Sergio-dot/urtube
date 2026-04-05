package router

import (
	"testing"

	"github.com/Sergio-dot/urtube/internal/search"
	"github.com/stretchr/testify/assert"
)

func TestNewRouter(t *testing.T) {
	mockSearcher := &search.YtdlpSearcher{}
	r := NewRouter(Dependencies{
		Searcher: mockSearcher,
	})
	assert.NotNil(t, r)
}
