package notify

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/kemalthoriq/healthcare_be/internal/booking"
	"github.com/kemalthoriq/healthcare_be/internal/platform/email"
	"github.com/kemalthoriq/healthcare_be/internal/setting"
)

var wib = time.FixedZone("WIB", 7*60*60)

type fakeQueue struct {
	sent []email.Message
}

func (f *fakeQueue) Enqueue(m email.Message) {
	f.sent = append(f.sent, m)
}

type fakeClinic struct {
	clinic setting.Clinic
	err    error
}

func (f fakeClinic) GetClinic(ctx context.Context) (setting.Clinic, error) {
	return f.clinic, f.err
}

func newTestNotifier(c fakeClinic) (*Notifier, *fakeQueue) {
	q := &fakeQueue{}
	return New(q, c, wib, "https://lumiere.test", slog.New(slog.NewTextHandler(io.Discard, nil))), q
}

func testEvent(kind string) booking.Event {
	return booking.Event{
		Kind:          kind,
		Code:          "LMR-AB12-CD34",
		PatientName:   "Dewi",
		Email:         "dewi@example.com",
		TreatmentName: "Facial Glow",
		TherapistName: "dr. Sari",
		StartAt:       time.Date(2026, 10, 26, 3, 0, 0, 0, time.UTC),
		EndAt:         time.Date(2026, 10, 26, 4, 0, 0, 0, time.UTC),
	}
}

func TestNotify_Created(t *testing.T) {
	n, q := newTestNotifier(fakeClinic{clinic: setting.Clinic{Name: "Lumière Aesthetic", Address: "Jl. Melati 1", WhatsApp: "6281234567890"}})

	n.Notify(context.Background(), testEvent(booking.EventCreated))

	if len(q.sent) != 1 {
		t.Fatalf("sent = %d, want 1", len(q.sent))
	}
	m := q.sent[0]
	if m.To != "dewi@example.com" || m.Ref != "LMR-AB12-CD34" || m.Subject != "Booking LMR-AB12-CD34 diterima — Lumière Aesthetic" {
		t.Errorf("message = %+v", m)
	}
	for _, want := range []string{
		"Halo Dewi,",
		"Jadwal       : Senin, 26 Oktober 2026, 10:00–11:00 WIB",
		"https://lumiere.test/booking/LMR-AB12-CD34",
		"Jl. Melati 1",
		"WhatsApp: +6281234567890",
	} {
		if !strings.Contains(m.Body, want) {
			t.Errorf("body missing %q:\n%s", want, m.Body)
		}
	}
}

func TestNotify_Rescheduled(t *testing.T) {
	n, q := newTestNotifier(fakeClinic{clinic: setting.Clinic{Name: "Lumière"}})
	e := testEvent(booking.EventRescheduled)
	e.PreviousStartAt = time.Date(2026, 10, 25, 7, 30, 0, 0, time.UTC)

	n.Notify(context.Background(), e)

	body := q.sent[0].Body
	if !strings.Contains(body, "Jadwal lama  : Minggu, 25 Oktober 2026, 14:30 WIB") || !strings.Contains(body, "Jadwal baru  : Senin, 26 Oktober 2026, 10:00–11:00 WIB") {
		t.Errorf("body:\n%s", body)
	}
}

func TestNotify_Cancelled(t *testing.T) {
	n, q := newTestNotifier(fakeClinic{clinic: setting.Clinic{Name: "Lumière"}})

	byClinic := testEvent(booking.EventCancelled)
	byClinic.Reason = "Terapis sakit"
	n.Notify(context.Background(), byClinic)

	byPatient := testEvent(booking.EventCancelled)
	byPatient.Reason = "Ada urusan"
	byPatient.CancelledByPatient = true
	n.Notify(context.Background(), byPatient)

	if !strings.Contains(q.sent[0].Body, "dibatalkan oleh klinik") || !strings.Contains(q.sent[0].Body, "Alasan       : Terapis sakit") {
		t.Errorf("clinic cancellation body:\n%s", q.sent[0].Body)
	}
	if strings.Contains(q.sent[1].Body, "Alasan") || strings.Contains(q.sent[1].Body, "/booking/") {
		t.Errorf("patient cancellation body:\n%s", q.sent[1].Body)
	}
}

func TestNotify_ClinicLookupFailsStillSends(t *testing.T) {
	n, q := newTestNotifier(fakeClinic{err: errors.New("db down")})

	n.Notify(context.Background(), testEvent(booking.EventConfirmed))

	if len(q.sent) != 1 || !strings.HasSuffix(q.sent[0].Subject, "— Lumière Clinic") {
		t.Errorf("sent = %+v", q.sent)
	}
}
