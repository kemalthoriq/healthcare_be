package captcha

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/kemalthoriq/healthcare_be/internal/platform/apperr"
)

const (
	siteverifyURL  = "https://challenges.cloudflare.com/turnstile/v0/siteverify"
	verifyTimeout  = 5 * time.Second
	maxTokenLength = 2048
)

var ErrFailed = apperr.Unprocessable("CAPTCHA_FAILED", "Verifikasi keamanan gagal. Muat ulang halaman lalu coba lagi.")

type Verifier struct {
	secret   string
	endpoint string
	client   *http.Client
	log      *slog.Logger
}

func New(secret string, log *slog.Logger) *Verifier {
	return &Verifier{secret: secret, endpoint: siteverifyURL, client: &http.Client{Timeout: verifyTimeout}, log: log}
}

func (v *Verifier) Enabled() bool {
	return v.secret != ""
}

func (v *Verifier) Verify(ctx context.Context, token, remoteIP string) error {
	if !v.Enabled() {
		return nil
	}
	if token == "" || len(token) > maxTokenLength {
		return ErrFailed
	}

	form := url.Values{"secret": {v.secret}, "response": {token}}
	if remoteIP != "" {
		form.Set("remoteip", remoteIP)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, v.endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("build siteverify request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := v.client.Do(req)
	if err != nil {
		v.log.ErrorContext(ctx, "turnstile siteverify unreachable", "err", err)
		return ErrFailed
	}
	defer resp.Body.Close()

	var out struct {
		Success    bool     `json:"success"`
		ErrorCodes []string `json:"error-codes"`
	}
	if resp.StatusCode != http.StatusOK || json.NewDecoder(resp.Body).Decode(&out) != nil {
		v.log.ErrorContext(ctx, "turnstile siteverify bad response", "status", resp.StatusCode)
		return ErrFailed
	}
	if !out.Success {
		v.log.InfoContext(ctx, "turnstile rejected token", "codes", out.ErrorCodes)
		return ErrFailed
	}
	return nil
}
