package logger

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
)

func newTestLogger(buf *bytes.Buffer) *slog.Logger {
	return slog.New(ContextHandler(slog.NewJSONHandler(buf, nil)))
}

func TestContextHandler_AddsRequestID(t *testing.T) {
	var buf bytes.Buffer
	log := newTestLogger(&buf)

	log.InfoContext(WithRequestID(context.Background(), "req-123"), "hello")

	if !strings.Contains(buf.String(), `"request_id":"req-123"`) {
		t.Errorf("log missing request_id: %s", buf.String())
	}
}

func TestContextHandler_KeepsRequestIDAfterWith(t *testing.T) {
	var buf bytes.Buffer
	log := newTestLogger(&buf).With("service", "booking")

	log.InfoContext(WithRequestID(context.Background(), "req-123"), "hello")

	if !strings.Contains(buf.String(), `"request_id":"req-123"`) {
		t.Errorf("log missing request_id after With: %s", buf.String())
	}
}

func TestContextHandler_OmitsRequestIDWhenAbsent(t *testing.T) {
	var buf bytes.Buffer
	log := newTestLogger(&buf)

	log.InfoContext(context.Background(), "hello")

	if strings.Contains(buf.String(), "request_id") {
		t.Errorf("log should not contain request_id: %s", buf.String())
	}
}
