package storage

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestSign_MatchesCloudinaryDocsExample(t *testing.T) {
	params := map[string]string{
		"timestamp": "1315060510",
		"public_id": "sample_image",
		"eager":     "w_400,h_300,c_pad|w_260,h_200,c_crop",
	}
	if got := sign(params, "abcd"); got != "bfd09f95f331f558cbd1320e67aa8d488770583e" {
		t.Errorf("signature = %s", got)
	}
}

func TestCloudinarySign(t *testing.T) {
	c := NewCloudinary("lumiere-demo", "123456", "rahasia")
	c.now = func() time.Time { return time.Unix(1760155200, 0) }

	s, err := c.Sign("treatments")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := sign(map[string]string{"allowed_formats": "jpg,png,webp", "folder": "lumiere/treatments", "timestamp": "1760155200"}, "rahasia")
	if s.Signature != want || s.Folder != "lumiere/treatments" || s.Timestamp != 1760155200 || s.APIKey != "123456" ||
		s.UploadURL != "https://api.cloudinary.com/v1_1/lumiere-demo/image/upload" {
		t.Errorf("signature = %+v", s)
	}

	if _, err := c.Sign("../rahasia"); !errors.Is(err, ErrInvalidFolder) {
		t.Errorf("bad folder: err = %v", err)
	}
	if _, err := NewCloudinary("", "", "").Sign("promos"); !errors.Is(err, ErrNotConfigured) {
		t.Errorf("not configured: err = %v", err)
	}
}

func TestCheckImageURL(t *testing.T) {
	configured := NewCloudinary("lumiere-demo", "123456", "rahasia")
	unconfigured := NewCloudinary("", "", "")
	good := "https://res.cloudinary.com/lumiere-demo/image/upload/v1760155200/lumiere/treatments/abc123.jpg"

	tests := []struct {
		name         string
		url          string
		configured   bool
		unconfigured bool
	}{
		{"empty", "", true, true},
		{"own cloudinary upload", good, true, true},
		{"with transformation", "https://res.cloudinary.com/lumiere-demo/image/upload/f_auto,q_auto/v1/lumiere/promos/x.webp", true, true},
		{"other cloud", "https://res.cloudinary.com/orang-lain/image/upload/v1/lumiere/treatments/x.jpg", false, true},
		{"outside lumiere folder", "https://res.cloudinary.com/lumiere-demo/image/upload/v1/sample.jpg", false, true},
		{"other host", "https://images.example.com/lumiere-demo/image/upload/lumiere/x.jpg", false, true},
		{"query string", good + "?x=1", false, true},
		{"http", "http://res.cloudinary.com/lumiere-demo/image/upload/v1/lumiere/x.jpg", false, false},
		{"userinfo", "https://a@res.cloudinary.com/lumiere-demo/image/upload/v1/lumiere/x.jpg", false, false},
		{"too long", "https://res.cloudinary.com/lumiere-demo/image/upload/lumiere/" + strings.Repeat("a", 500), false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if ok := configured.CheckImageURL(tt.url) == ""; ok != tt.configured {
				t.Errorf("configured: ok = %v, want %v", ok, tt.configured)
			}
			if ok := unconfigured.CheckImageURL(tt.url) == ""; ok != tt.unconfigured {
				t.Errorf("unconfigured: ok = %v, want %v", ok, tt.unconfigured)
			}
		})
	}
}

func TestSignHandler(t *testing.T) {
	do := func(c *Cloudinary, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/uploads/sign", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		NewHandler(c).Sign(rec, req)
		return rec
	}

	rec := do(NewCloudinary("lumiere-demo", "123456", "rahasia"), `{"folder":"promos"}`)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"folder":"lumiere/promos"`) || strings.Contains(rec.Body.String(), "rahasia") {
		t.Errorf("got %d %s", rec.Code, rec.Body.String())
	}

	if rec := do(NewCloudinary("", "", ""), `{"folder":"promos"}`); rec.Code != http.StatusServiceUnavailable {
		t.Errorf("not configured: status = %d, want 503", rec.Code)
	}
	if rec := do(NewCloudinary("lumiere-demo", "123456", "rahasia"), `{"folder":"lain"}`); rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("bad folder: status = %d, want 422", rec.Code)
	}
}
