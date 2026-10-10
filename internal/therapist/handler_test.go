package therapist

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newTestMux() *http.ServeMux {
	svc, _ := newTestService()
	h := NewHandler(svc)
	mux := http.NewServeMux()
	mux.HandleFunc("POST /admin/therapists", h.Create)
	mux.HandleFunc("PUT /admin/treatments/{id}/therapists", h.SetForTreatment)
	mux.HandleFunc("GET /treatments/{slug}/therapists", h.ListForTreatmentPublic)
	return mux
}

func do(mux *http.ServeMux, method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func TestHandler_SetAndListTherapistsForTreatment(t *testing.T) {
	mux := newTestMux()
	do(mux, http.MethodPost, "/admin/therapists", `{"name":"dr. Sari","title":"Dokter Estetika"}`)

	rec := do(mux, http.MethodPut, "/admin/treatments/1/therapists", `{"therapist_ids":[1]}`)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"slug":"dr-sari"`) {
		t.Fatalf("set: got %d %s", rec.Code, rec.Body.String())
	}

	rec = do(mux, http.MethodGet, "/treatments/facial-glow/therapists", "")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"slug":"dr-sari"`) {
		t.Errorf("public list: got %d %s", rec.Code, rec.Body.String())
	}
}

func TestHandler_SetForTreatmentRequiresList(t *testing.T) {
	rec := do(newTestMux(), http.MethodPut, "/admin/treatments/1/therapists", `{}`)

	if rec.Code != http.StatusUnprocessableEntity || !strings.Contains(rec.Body.String(), "therapist_ids") {
		t.Errorf("got %d %s, want 422 on therapist_ids", rec.Code, rec.Body.String())
	}
}

func TestHandler_UnknownTreatmentReturnsNotFound(t *testing.T) {
	mux := newTestMux()

	if rec := do(mux, http.MethodGet, "/treatments/tidak-ada/therapists", ""); rec.Code != http.StatusNotFound {
		t.Errorf("public: status = %d, want 404", rec.Code)
	}
	if rec := do(mux, http.MethodPut, "/admin/treatments/abc/therapists", `{"therapist_ids":[]}`); rec.Code != http.StatusNotFound {
		t.Errorf("admin: status = %d, want 404", rec.Code)
	}
}
