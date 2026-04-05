package strutils

import (
	"strings"
)

// IsEmpty checks if a string is empty or contains only whitespace.
func IsEmpty(str string) bool {
	return len(strings.TrimSpace(str)) == 0
}

// StringPtr returns a pointer to the passed string
func StringPtr(str string) *string {
	if str == "" {
		return nil
	}
	return &str
}
