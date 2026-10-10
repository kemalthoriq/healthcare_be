package slug

import (
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

func Make(text, fallback string) string {
	var b strings.Builder
	dash := false

	for _, r := range norm.NFD.String(strings.ToLower(text)) {
		switch {
		case unicode.Is(unicode.Mn, r):
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
			dash = false
		case b.Len() > 0 && !dash:
			b.WriteByte('-')
			dash = true
		}
	}

	s := strings.TrimSuffix(b.String(), "-")
	if s == "" {
		return fallback
	}
	return s
}
