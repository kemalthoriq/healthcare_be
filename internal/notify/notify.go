package notify

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/kemalthoriq/healthcare_be/internal/booking"
	"github.com/kemalthoriq/healthcare_be/internal/platform/email"
	"github.com/kemalthoriq/healthcare_be/internal/setting"
)

const fallbackClinicName = "Lumière Clinic"

var (
	days   = []string{"Minggu", "Senin", "Selasa", "Rabu", "Kamis", "Jumat", "Sabtu"}
	months = []string{"Januari", "Februari", "Maret", "April", "Mei", "Juni", "Juli", "Agustus", "September", "Oktober", "November", "Desember"}
)

type queue interface {
	Enqueue(m email.Message)
}

type clinicGetter interface {
	GetClinic(ctx context.Context) (setting.Clinic, error)
}

type Notifier struct {
	queue          queue
	clinic         clinicGetter
	loc            *time.Location
	frontendOrigin string
	log            *slog.Logger
}

func New(queue queue, clinic clinicGetter, loc *time.Location, frontendOrigin string, log *slog.Logger) *Notifier {
	return &Notifier{queue: queue, clinic: clinic, loc: loc, frontendOrigin: frontendOrigin, log: log}
}

func (n *Notifier) Notify(ctx context.Context, e booking.Event) {
	c, err := n.clinic.GetClinic(ctx)
	if err != nil {
		n.log.Warn("load clinic profile for email", "ref", e.Code, "err", err)
	}
	if c.Name == "" {
		c.Name = fallbackClinicName
	}

	subject, intro := n.headline(e)
	if subject == "" {
		return
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Halo %s,\n\n%s\n\n", e.PatientName, intro)
	fmt.Fprintf(&b, "Kode booking : %s\n", e.Code)
	fmt.Fprintf(&b, "Treatment    : %s\n", e.TreatmentName)
	fmt.Fprintf(&b, "Terapis      : %s\n", e.TherapistName)
	if e.Kind == booking.EventRescheduled {
		fmt.Fprintf(&b, "Jadwal lama  : %s\n", n.formatTime(e.PreviousStartAt, time.Time{}))
		fmt.Fprintf(&b, "Jadwal baru  : %s\n", n.formatTime(e.StartAt, e.EndAt))
	} else {
		fmt.Fprintf(&b, "Jadwal       : %s\n", n.formatTime(e.StartAt, e.EndAt))
	}
	if e.Kind == booking.EventCancelled && !e.CancelledByPatient && e.Reason != "" {
		fmt.Fprintf(&b, "Alasan       : %s\n", e.Reason)
	}
	if e.Kind != booking.EventCancelled {
		fmt.Fprintf(&b, "\nSimpan kode booking ini. Cek status atau batalkan booking di:\n%s/booking/%s\n", n.frontendOrigin, e.Code)
	}

	fmt.Fprintf(&b, "\nSalam,\n%s\n", c.Name)
	if c.Address != "" {
		fmt.Fprintf(&b, "%s\n", c.Address)
	}
	if c.WhatsApp != "" {
		fmt.Fprintf(&b, "WhatsApp: +%s\n", c.WhatsApp)
	}
	if c.Phone != "" {
		fmt.Fprintf(&b, "Telepon: %s\n", c.Phone)
	}

	n.queue.Enqueue(email.Message{
		To:      e.Email,
		Subject: fmt.Sprintf("%s — %s", subject, c.Name),
		Body:    b.String(),
		Ref:     e.Code,
	})
}

func (n *Notifier) headline(e booking.Event) (string, string) {
	switch e.Kind {
	case booking.EventCreated:
		return "Booking " + e.Code + " diterima",
			"Terima kasih, booking Anda sudah kami terima dan sedang menunggu konfirmasi dari klinik."
	case booking.EventConfirmed:
		return "Booking " + e.Code + " dikonfirmasi",
			"Booking Anda sudah dikonfirmasi. Sampai jumpa di klinik."
	case booking.EventRescheduled:
		return "Jadwal booking " + e.Code + " diubah",
			"Jadwal booking Anda telah diubah oleh klinik."
	case booking.EventCancelled:
		if e.CancelledByPatient {
			return "Booking " + e.Code + " dibatalkan",
				"Booking Anda sudah berhasil dibatalkan sesuai permintaan Anda."
		}
		return "Booking " + e.Code + " dibatalkan",
			"Mohon maaf, booking Anda dibatalkan oleh klinik. Silakan hubungi kami untuk membuat jadwal baru."
	}
	return "", ""
}

func (n *Notifier) formatTime(start, end time.Time) string {
	s := start.In(n.loc)
	out := fmt.Sprintf("%s, %d %s %d, %s", days[s.Weekday()], s.Day(), months[s.Month()-1], s.Year(), s.Format("15:04"))
	if !end.IsZero() {
		out += "–" + end.In(n.loc).Format("15:04")
	}
	return out + " WIB"
}
