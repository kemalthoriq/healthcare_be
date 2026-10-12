package slug

import "testing"

func TestMake(t *testing.T) {
	tests := []struct {
		text string
		want string
	}{
		{"Facial Glow", "facial-glow"},
		{"  Laser  CO2  Fractional ", "laser-co2-fractional"},
		{"Peeling & Brightening", "peeling-brightening"},
		{"Lumière Signature", "lumiere-signature"},
		{"dr. Sari, Sp.KK", "dr-sari-sp-kk"},
		{"!!!", "fallback"},
	}

	for _, tt := range tests {
		t.Run(tt.text, func(t *testing.T) {
			if got := Make(tt.text, "fallback"); got != tt.want {
				t.Errorf("Make(%q) = %q, want %q", tt.text, got, tt.want)
			}
		})
	}
}
