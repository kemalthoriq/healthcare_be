package httpx

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPageFromRequest(t *testing.T) {
	tests := []struct {
		query string
		want  Page
	}{
		{"", Page{Page: 1, PerPage: 20}},
		{"?page=3&per_page=10", Page{Page: 3, PerPage: 10}},
		{"?page=0&per_page=-5", Page{Page: 1, PerPage: 20}},
		{"?page=abc&per_page=500", Page{Page: 1, PerPage: 100}},
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			got := PageFromRequest(httptest.NewRequest(http.MethodGet, "/"+tt.query, nil))
			if got != tt.want {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}

	if offset := (Page{Page: 3, PerPage: 20}).Offset(); offset != 40 {
		t.Errorf("Offset() = %d, want 40", offset)
	}
}

func TestJSONPage(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	JSONPage(rec, req, []int{1, 2}, Page{Page: 2, PerPage: 2, Total: 5})

	assertJSON(t, rec, http.StatusOK, `{"data":[1,2],"meta":{"page":2,"per_page":2,"total":5}}`)
}
