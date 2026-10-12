package email

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

const (
	queueSize   = 100
	sendTimeout = 15 * time.Second
)

type Sender interface {
	Send(ctx context.Context, m Message) error
}

type Queue struct {
	sender Sender
	log    *slog.Logger
	jobs   chan Message
	done   chan struct{}
	mu     sync.Mutex
	closed bool
}

func NewQueue(sender Sender, log *slog.Logger) *Queue {
	q := &Queue{sender: sender, log: log, jobs: make(chan Message, queueSize), done: make(chan struct{})}
	if sender == nil {
		close(q.done)
		return q
	}
	go q.work()
	return q
}

func (q *Queue) Enqueue(m Message) {
	if q.sender == nil {
		q.log.Info("email disabled, not sent", "subject", m.Subject, "ref", m.Ref)
		return
	}

	q.mu.Lock()
	defer q.mu.Unlock()
	if q.closed {
		q.log.Warn("email queue closed, dropped", "ref", m.Ref)
		return
	}

	select {
	case q.jobs <- m:
	default:
		q.log.Warn("email queue full, dropped", "ref", m.Ref)
	}
}

func (q *Queue) Close(ctx context.Context) error {
	q.mu.Lock()
	if !q.closed && q.sender != nil {
		close(q.jobs)
	}
	q.closed = true
	q.mu.Unlock()

	select {
	case <-q.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (q *Queue) work() {
	defer close(q.done)
	for m := range q.jobs {
		ctx, cancel := context.WithTimeout(context.Background(), sendTimeout)
		err := q.sender.Send(ctx, m)
		cancel()
		if err != nil {
			q.log.Error("send email failed", "ref", m.Ref, "err", err)
			continue
		}
		q.log.Info("email sent", "subject", m.Subject, "ref", m.Ref)
	}
}
