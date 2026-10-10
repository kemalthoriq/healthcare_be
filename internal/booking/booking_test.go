package booking

import (
	"regexp"
	"testing"
)

func TestNormalizePhone(t *testing.T) {
	valid := map[string]string{
		"081234567890":      "6281234567890",
		"+62 812-3456-7890": "6281234567890",
		"6281234567890":     "6281234567890",
		"81234567890":       "6281234567890",
		"(0812) 3456.7890":  "6281234567890",
	}
	for in, want := range valid {
		if got, ok := NormalizePhone(in); !ok || got != want {
			t.Errorf("NormalizePhone(%q) = %q, %v; want %q", in, got, ok, want)
		}
	}

	for _, in := range []string{"", "12345", "0212345678", "+1 555 123 4567", "0812abc4567", "08123", "0812345678901234"} {
		if got, ok := NormalizePhone(in); ok {
			t.Errorf("NormalizePhone(%q) = %q, want rejected", in, got)
		}
	}
}

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
