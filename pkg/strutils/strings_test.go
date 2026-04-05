package strutils_test

import (
	"testing"

	"github.com/Sergio-dot/urtube/pkg/strutils"
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
			assert.Equal(t, tt.expected, strutils.IsEmpty(tt.input))
		})
	}
}

func TestStringPtr(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *string
	}{
		{name: "empty string", input: "", expected: nil},
		{name: "non-empty string", input: "hello", expected: strutils.StringPtr("hello")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, strutils.StringPtr(tt.input))
		})
	}
}
