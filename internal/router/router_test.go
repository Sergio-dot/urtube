package router

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewRouter(t *testing.T) {
	r := NewRouter()
	assert.NotNil(t, r)
}
