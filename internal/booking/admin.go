package booking

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/kemalthoriq/healthcare_be/internal/platform/apperr"
)

const maxNoteContentLength = 2000

var transitions = map[string][]string{
	StatusPending:   {StatusConfirmed, StatusCancelled},
	StatusConfirmed: {StatusCompleted, StatusNoShow, StatusCancelled},
}

var statusLabels = map[string]string{
	StatusPending:   "Menunggu Konfirmasi",
	StatusConfirmed: "Terkonfirmasi",
	StatusCompleted: "Selesai",
	StatusNoShow:    "Tidak Datang",
	StatusCancelled: "Dibatalkan",
}

var (
	ErrBookingNotStarted = apperr.Unprocessable("BOOKING_NOT_STARTED", "Booking belum dimulai. Tandai selesai atau tidak datang setelah jam mulai lewat.")
	ErrStatusChanged     = apperr.Conflict("STATUS_CHANGED", "Status booking baru saja berubah. Muat ulang halaman.")
	ErrNotRescheduleable = apperr.Conflict("INVALID_TRANSITION", "Hanya booking yang menunggu konfirmasi atau terkonfirmasi yang bisa dijadwalkan ulang.")
)

type Actor struct {
	UserID      int64
	TherapistID int64
}

func (a Actor) isTherapist() bool {
	return a.TherapistID != 0
}

func (a Actor) canAccess(b AdminBooking) bool {
	return !a.isTherapist() || b.TherapistID == a.TherapistID
}

type AdminFilter struct {
	Status      *string
	TherapistID *int64
	DateFrom    *time.Time
	DateTo      *time.Time
}

type AdminBooking struct {
	Booking
	PatientPhone  string
	PatientEmail  string
	TreatmentNote string
	TreatmentName string
	TherapistName string
	NeedsAction   bool
}

func IsValidStatus(status string) bool {
	_, ok := statusLabels[status]
	return ok
}

func canTransition(from, to string) bool {
	return slices.Contains(transitions[from], to)
}

func (s *Service) TherapistActor(ctx context.Context, userID int64) (Actor, error) {
	th, err := s.availability.therapists.GetByUserID(ctx, userID)
	if err != nil {
		return Actor{}, apperr.Forbidden()
	}
	return Actor{UserID: userID, TherapistID: th.ID}, nil
}

func (s *Service) ListAdmin(ctx context.Context, actor Actor, f AdminFilter, limit, offset int) ([]AdminBooking, int64, error) {
	if actor.isTherapist() {
		f.TherapistID = &actor.TherapistID
	}

	items, total, err := s.store.ListAdmin(ctx, f, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	names := newNameCache(s)
	for i := range items {
		if err := names.fill(ctx, &items[i]); err != nil {
			return nil, 0, err
		}
	}
	return items, total, nil
}

func (s *Service) GetAdmin(ctx context.Context, actor Actor, id int64) (AdminBooking, error) {
	b, err := s.store.GetAdmin(ctx, id)
	if errors.Is(err, errNotFound) {
		return AdminBooking{}, ErrBookingNotFound
	}
	if err != nil {
		return AdminBooking{}, err
	}
	if !actor.canAccess(b) {
		return AdminBooking{}, apperr.Forbidden()
	}

	if err := newNameCache(s).fill(ctx, &b); err != nil {
		return AdminBooking{}, err
	}
	return b, nil
}

func (s *Service) Transition(ctx context.Context, actor Actor, id int64, to, reason string) (AdminBooking, error) {
	reason = strings.TrimSpace(reason)
	if to == StatusCancelled {
		switch {
		case reason == "":
			return AdminBooking{}, apperr.Validation(map[string]string{"reason": "Alasan pembatalan wajib diisi."})
		case utf8.RuneCountInString(reason) > maxCancelReasonLength:
			return AdminBooking{}, apperr.Validation(map[string]string{"reason": fmt.Sprintf("Alasan maksimal %d karakter.", maxCancelReasonLength)})
		}
	}

	current, err := s.GetAdmin(ctx, actor, id)
	if err != nil {
		return AdminBooking{}, err
	}

	if !canTransition(current.Status, to) {
		return AdminBooking{}, apperr.Conflict("INVALID_TRANSITION",
			fmt.Sprintf("Booking berstatus %s tidak bisa diubah menjadi %s.", statusLabels[current.Status], statusLabels[to]))
	}

	if (to == StatusCompleted || to == StatusNoShow) && s.availability.now().Before(current.StartAt) {
		return AdminBooking{}, ErrBookingNotStarted
	}

	err = s.store.Transition(ctx, id, current.Status, to, reason)
	if errors.Is(err, errStatusChanged) {
		return AdminBooking{}, ErrStatusChanged
	}
	if err != nil {
		return AdminBooking{}, err
	}

	updated, err := s.GetAdmin(ctx, actor, id)
	if err != nil {
		return AdminBooking{}, err
	}
	switch to {
	case StatusConfirmed:
		s.notify(ctx, eventFor(EventConfirmed, updated))
	case StatusCancelled:
		s.notify(ctx, eventFor(EventCancelled, updated))
	}
	return updated, nil
}

func (s *Service) Reschedule(ctx context.Context, actor Actor, id int64, startAt time.Time) (AdminBooking, error) {
	if startAt.IsZero() {
		return AdminBooking{}, apperr.Validation(map[string]string{"start_at": "Jam baru wajib dipilih."})
	}

	current, err := s.GetAdmin(ctx, actor, id)
	if err != nil {
		return AdminBooking{}, err
	}
	if !isActive(current.Status) {
		return AdminBooking{}, ErrNotRescheduleable
	}

	tr, err := s.availability.treatments.Get(ctx, current.TreatmentID)
	if err != nil {
		return AdminBooking{}, fmt.Errorf("get treatment: %w", err)
	}
	th, err := s.availability.therapists.Get(ctx, current.TherapistID)
	if err != nil {
		return AdminBooking{}, fmt.Errorf("get therapist: %w", err)
	}

	p, err := s.availability.check(ctx, tr.Slug, th.Slug, startAt, id)
	if err != nil {
		return AdminBooking{}, err
	}

	endAt := startAt.Add(p.duration)
	err = s.store.Reschedule(ctx, id, startAt, endAt, endAt.Add(p.buffer))
	switch {
	case errors.Is(err, errSlotTaken):
		return AdminBooking{}, ErrSlotTaken
	case errors.Is(err, errStatusChanged):
		return AdminBooking{}, ErrStatusChanged
	case err != nil:
		return AdminBooking{}, err
	}

	updated, err := s.GetAdmin(ctx, actor, id)
	if err != nil {
		return AdminBooking{}, err
	}
	e := eventFor(EventRescheduled, updated)
	e.PreviousStartAt = current.StartAt
	s.notify(ctx, e)
	return updated, nil
}

func (s *Service) SaveNote(ctx context.Context, actor Actor, id int64, content string) (AdminBooking, error) {
	content = strings.TrimSpace(content)
	switch {
	case content == "":
		return AdminBooking{}, apperr.Validation(map[string]string{"content": "Catatan wajib diisi."})
	case utf8.RuneCountInString(content) > maxNoteContentLength:
		return AdminBooking{}, apperr.Validation(map[string]string{"content": fmt.Sprintf("Catatan maksimal %d karakter.", maxNoteContentLength)})
	}

	if _, err := s.GetAdmin(ctx, actor, id); err != nil {
		return AdminBooking{}, err
	}

	if err := s.store.SaveNote(ctx, id, actor.UserID, content); err != nil {
		return AdminBooking{}, err
	}

	return s.GetAdmin(ctx, actor, id)
}

type nameCache struct {
	s          *Service
	treatments map[int64]string
	therapists map[int64]string
}

func newNameCache(s *Service) *nameCache {
	return &nameCache{s: s, treatments: map[int64]string{}, therapists: map[int64]string{}}
}

func (c *nameCache) fill(ctx context.Context, b *AdminBooking) error {
	if _, ok := c.treatments[b.TreatmentID]; !ok {
		tr, err := c.s.availability.treatments.Get(ctx, b.TreatmentID)
		if err != nil {
			return fmt.Errorf("get treatment: %w", err)
		}
		c.treatments[b.TreatmentID] = tr.Name
	}
	if _, ok := c.therapists[b.TherapistID]; !ok {
		th, err := c.s.availability.therapists.Get(ctx, b.TherapistID)
		if err != nil {
			return fmt.Errorf("get therapist: %w", err)
		}
		c.therapists[b.TherapistID] = th.Name
	}

	b.TreatmentName = c.treatments[b.TreatmentID]
	b.TherapistName = c.therapists[b.TherapistID]
	b.NeedsAction = b.Status == StatusPending && b.StartAt.Before(c.s.availability.now())
	return nil
}
