package captcha

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newTestVerifier(t *testing.T, handler http.HandlerFunc) *Verifier {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	v := New("rahasia", slog.New(slog.NewTextHandler(io.Discard, nil)))
	v.endpoint = srv.URL
	return v
}

func TestVerify(t *testing.T) {
	var form map[string]string
	v := newTestVerifier(t, func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		form = map[string]string{"secret": r.PostForm.Get("secret"), "response": r.PostForm.Get("response"), "remoteip": r.PostForm.Get("remoteip")}
		if r.PostForm.Get("response") == "token-ok" {
			io.WriteString(w, `{"success":true}`)
			return
		}
		io.WriteString(w, `{"success":false,"error-codes":["invalid-input-response"]}`)
	})

	if err := v.Verify(context.Background(), "token-ok", "198.51.100.7"); err != nil {
		t.Fatalf("valid token: %v", err)
	}
	if form["secret"] != "rahasia" || form["remoteip"] != "198.51.100.7" {
		t.Errorf("form sent = %v", form)
	}

	if err := v.Verify(context.Background(), "token-palsu", ""); !errors.Is(err, ErrFailed) {
		t.Errorf("rejected token: err = %v, want ErrFailed", err)
	}
	if err := v.Verify(context.Background(), "", ""); !errors.Is(err, ErrFailed) {
		t.Errorf("empty token: err = %v, want ErrFailed", err)
	}
	if err := v.Verify(context.Background(), strings.Repeat("a", 2049), ""); !errors.Is(err, ErrFailed) {
		t.Errorf("oversized token: err = %v, want ErrFailed", err)
	}
}

func TestVerify_FailsClosed(t *testing.T) {
	v := newTestVerifier(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	if err := v.Verify(context.Background(), "token", ""); !errors.Is(err, ErrFailed) {
		t.Errorf("server error: err = %v, want ErrFailed", err)
	}

	v.endpoint = "http://127.0.0.1:1"
	if err := v.Verify(context.Background(), "token", ""); !errors.Is(err, ErrFailed) {
		t.Errorf("unreachable: err = %v, want ErrFailed", err)
	}
}

func TestVerify_Disabled(t *testing.T) {
	v := New("", slog.New(slog.NewTextHandler(io.Discard, nil)))
	if v.Enabled() {
		t.Fatal("empty secret must disable verification")
	}
	if err := v.Verify(context.Background(), "", ""); err != nil {
		t.Errorf("disabled: err = %v", err)
	}
}
