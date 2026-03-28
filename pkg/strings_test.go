package pkg

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsEmpty(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{name: "empty string", input: "", expected: true},
		{name: "string with spaces", input: "   ", expected: true},
		{name: "non-empty string", input: "hello", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, IsEmpty(tt.input))
		})
	}
}
