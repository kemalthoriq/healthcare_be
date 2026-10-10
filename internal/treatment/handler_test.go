package treatment

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newTestMux() *http.ServeMux {
	h := NewHandler(NewService(newFakeStore()))
	mux := http.NewServeMux()
	mux.HandleFunc("GET /treatments/{slug}", h.GetPublic)
	mux.HandleFunc("POST /admin/treatments", h.Create)
	mux.HandleFunc("PUT /admin/treatments/{id}", h.Update)
	mux.HandleFunc("DELETE /admin/treatments/{id}", h.Delete)
	return mux
}

func do(mux *http.ServeMux, method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func TestHandler_CreateThenGetBySlug(t *testing.T) {
	mux := newTestMux()

	rec := do(mux, http.MethodPost, "/admin/treatments", `{"name":"Facial Glow","price":350000,"duration_minutes":60}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"slug":"facial-glow"`) || !strings.Contains(rec.Body.String(), `"is_active":true`) {
		t.Errorf("unexpected create body: %s", rec.Body.String())
	}

	rec = do(mux, http.MethodGet, "/treatments/facial-glow", "")
	if rec.Code != http.StatusOK {
		t.Errorf("get status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestHandler_UpdateRequiresIsActive(t *testing.T) {
	mux := newTestMux()
	do(mux, http.MethodPost, "/admin/treatments", `{"name":"Facial Glow","price":350000,"duration_minutes":60}`)

	rec := do(mux, http.MethodPut, "/admin/treatments/1", `{"name":"Facial Glow","price":350000,"duration_minutes":60}`)

	if rec.Code != http.StatusUnprocessableEntity || !strings.Contains(rec.Body.String(), "is_active") {
		t.Errorf("got %d %s, want 422 with is_active field", rec.Code, rec.Body.String())
	}
}

func TestHandler_InvalidIDReturnsNotFound(t *testing.T) {
	mux := newTestMux()

	for _, path := range []string{"/admin/treatments/abc", "/admin/treatments/0", "/admin/treatments/-1"} {
		if rec := do(mux, http.MethodDelete, path, ""); rec.Code != http.StatusNotFound {
			t.Errorf("DELETE %s status = %d, want 404", path, rec.Code)
		}
	}
}
