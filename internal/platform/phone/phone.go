package phone

import "strings"

func Normalize(raw string) (string, bool) {
	s := strings.Map(func(r rune) rune {
		switch r {
		case ' ', '-', '.', '(', ')':
			return -1
		}
		return r
	}, strings.TrimSpace(raw))
	s = strings.TrimPrefix(s, "+")

	switch {
	case strings.HasPrefix(s, "62"):
	case strings.HasPrefix(s, "0"):
		s = "62" + s[1:]
	case strings.HasPrefix(s, "8"):
		s = "62" + s
	default:
		return "", false
	}

	if !strings.HasPrefix(s, "628") || len(s) < 10 || len(s) > 15 {
		return "", false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return "", false
		}
	}
	return s, true
}
