package handlers

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func emptyHandler(w http.ResponseWriter, r *http.Request) error {
	return nil
}

func TestWrapper(t *testing.T) {
	t.Run("wrap handler", func(t *testing.T) {
		handlerFunc := MakeHandler(emptyHandler)
		assert.NotNil(t, handlerFunc)
	})
}
