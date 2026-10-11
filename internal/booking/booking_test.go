package booking

import (
	"regexp"
	"testing"
)

func TestNewCode(t *testing.T) {
	pattern := regexp.MustCompile(`^LMR-[0-9A-HJKMNP-TV-Z]{4}-[0-9A-HJKMNP-TV-Z]{4}$`)
	seen := map[string]bool{}

	for range 1000 {
		code := NewCode()
		if !pattern.MatchString(code) {
			t.Fatalf("code %q does not match the expected format", code)
		}
		if seen[code] {
			t.Fatalf("duplicate code %q in 1000 draws", code)
		}
		seen[code] = true
	}
}
