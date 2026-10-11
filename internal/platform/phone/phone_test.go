package phone

import "testing"

func TestNormalize(t *testing.T) {
	valid := map[string]string{
		"081234567890":      "6281234567890",
		"+62 812-3456-7890": "6281234567890",
		"6281234567890":     "6281234567890",
		"81234567890":       "6281234567890",
		"(0812) 3456.7890":  "6281234567890",
	}
	for in, want := range valid {
		if got, ok := Normalize(in); !ok || got != want {
			t.Errorf("Normalize(%q) = %q, %v; want %q", in, got, ok, want)
		}
	}

	for _, in := range []string{"", "12345", "0212345678", "+1 555 123 4567", "0812abc4567", "08123", "0812345678901234"} {
		if got, ok := Normalize(in); ok {
			t.Errorf("Normalize(%q) = %q, want rejected", in, got)
		}
	}
}
