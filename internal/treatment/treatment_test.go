package treatment

import "testing"

func TestSlugify(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"Facial Glow", "facial-glow"},
		{"  Laser  CO2  Fractional ", "laser-co2-fractional"},
		{"Peeling & Brightening", "peeling-brightening"},
		{"Lumière Signature", "lumiere-signature"},
		{"Botox (Dahi)", "botox-dahi"},
		{"!!!", "treatment"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Slugify(tt.name); got != tt.want {
				t.Errorf("Slugify(%q) = %q, want %q", tt.name, got, tt.want)
			}
		})
	}
}
