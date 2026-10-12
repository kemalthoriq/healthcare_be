package like

import "testing"

func TestContains(t *testing.T) {
	tests := map[string]string{
		"":         "",
		"Sari":     "%Sari%",
		`50%_off\`: `%50\%\_off\\%`,
	}
	for in, want := range tests {
		if got := Contains(in); got != want {
			t.Errorf("Contains(%q) = %q, want %q", in, got, want)
		}
	}
}
