package pkg

import (
	"strings"
)

func IsEmpty(str string) bool {
	if len(strings.TrimSpace(str)) == 0 || strings.TrimSpace(str) == "" {
		return true
	}
	return false
}
