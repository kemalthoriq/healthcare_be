package email

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"mime"
	"net/mail"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestBuild(t *testing.T) {
	from, _ := mail.ParseAddress("Lumière Clinic <noreply@lumiere.test>")
	raw, err := build(from, Message{To: "dewi@example.com", Subject: "Booking LMR-AB12-CD34 diterima", Body: "Halo Dewi,\nJadwal: Senin – 10:00"}, time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	msg, err := mail.ReadMessage(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("parse built message: %v", err)
	}
	var dec mime.WordDecoder
	subject, _ := dec.DecodeHeader(msg.Header.Get("Subject"))
	if subject != "Booking LMR-AB12-CD34 diterima" {
		t.Errorf("subject = %q", subject)
	}
	sender, err := msg.Header.AddressList("From")
	if err != nil || sender[0].Name != "Lumière Clinic" || sender[0].Address != "noreply@lumiere.test" {
		t.Errorf("from = %v, %v", sender, err)
	}
	if !strings.HasSuffix(msg.Header.Get("Message-ID"), "@lumiere.test>") {
		t.Errorf("message id = %q", msg.Header.Get("Message-ID"))
	}
	if !bytes.Contains(raw, []byte("\r\nJadwal: Senin =E2=80=93 10:00")) {
		t.Errorf("body not quoted-printable with CRLF:\n%s", raw)
	}
}

func TestBuild_RejectsHeaderInjection(t *testing.T) {
	from, _ := mail.ParseAddress("noreply@lumiere.test")
	if _, err := build(from, Message{To: "a@example.com\r\nBcc: b@example.com"}, time.Now()); err == nil {
		t.Error("expected error for recipient with newline")
	}
}

type fakeSender struct {
	mu      sync.Mutex
	sent    []string
	block   chan struct{}
	failRef string
}

func (f *fakeSender) Send(ctx context.Context, m Message) error {
	if f.block != nil {
		<-f.block
	}
	if m.Ref == f.failRef {
		return errors.New("smtp down")
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sent = append(f.sent, m.Ref)
	return nil
}

func quietLog() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestQueue_DrainsOnClose(t *testing.T) {
	sender := &fakeSender{failRef: "B"}
	q := NewQueue(sender, quietLog())
	for _, ref := range []string{"A", "B", "C"} {
		q.Enqueue(Message{To: "x@example.com", Ref: ref})
	}

	if err := q.Close(context.Background()); err != nil {
		t.Fatalf("close: %v", err)
	}
	if strings.Join(sender.sent, ",") != "A,C" {
		t.Errorf("sent = %v, want A,C (B failed without stopping the worker)", sender.sent)
	}

	q.Enqueue(Message{Ref: "late"})
	if len(sender.sent) != 2 {
		t.Error("message enqueued after close must be dropped")
	}
}

func TestQueue_FullQueueDropsInsteadOfBlocking(t *testing.T) {
	sender := &fakeSender{block: make(chan struct{})}
	q := NewQueue(sender, quietLog())

	done := make(chan struct{})
	go func() {
		for range queueSize + 10 {
			q.Enqueue(Message{Ref: "x"})
		}
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Enqueue blocked on a full queue")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	if err := q.Close(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("close with stuck sender: err = %v, want deadline exceeded", err)
	}
	close(sender.block)
}

func TestQueue_Disabled(t *testing.T) {
	q := NewQueue(nil, quietLog())
	q.Enqueue(Message{Ref: "A"})
	if err := q.Close(context.Background()); err != nil {
		t.Errorf("close: %v", err)
	}
}
