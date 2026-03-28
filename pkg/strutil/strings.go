package strutil

import (
	"strings"
)

// IsEmpty checks if a string is empty or contains only whitespace.
func IsEmpty(str string) bool {
	return len(strings.TrimSpace(str)) == 0
}
