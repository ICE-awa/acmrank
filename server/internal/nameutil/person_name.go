package nameutil

import (
	"strings"
	"unicode"
)

func NormalizePersonName(input string) string {
	normalized := strings.ToLower(strings.TrimSpace(input))
	if normalized == "" {
		return ""
	}

	var builder strings.Builder
	builder.Grow(len(normalized))
	for _, r := range normalized {
		switch {
		case unicode.IsSpace(r), unicode.IsPunct(r), unicode.IsSymbol(r):
			continue
		default:
			builder.WriteRune(r)
		}
	}

	return builder.String()
}
