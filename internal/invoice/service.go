package invoice

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/kemalthoriq/healthcare_be/internal/booking"
	"github.com/kemalthoriq/healthcare_be/internal/platform/apperr"
	"github.com/kemalthoriq/healthcare_be/internal/platform/like"
)

const maxQueryLength = 100

var (
	ErrNotFound            = apperr.NotFound("Invoice tidak ditemukan.")
	ErrInvoiceExists       = apperr.Conflict("INVOICE_EXISTS", "Booking ini sudah memiliki invoice.")
	ErrInvoicePaid         = apperr.Conflict("INVOICE_PAID", "Invoice sudah lunas sehingga tidak bisa diubah.")
	ErrBookingNotCompleted = apperr.Unprocessable("BOOKING_NOT_COMPLETED", "Invoice hanya bisa dibuat untuk booking yang sudah selesai.")
)

type store interface {
	Create(ctx context.Context, period string, in NewInvoice) (int64, error)
	Get(ctx context.Context, id int64) (Invoice, error)
	List(ctx context.Context, f Filter, pattern string, limit, offset int) ([]Invoice, int64, error)
	UpdateDiscount(ctx context.Context, id, discount int64) error
	Pay(ctx context.Context, id int64, method string) error
}

type bookingFinder interface {
	GetAdmin(ctx context.Context, actor booking.Actor, id int64) (booking.AdminBooking, error)
}

type Service struct {
	store    store
	bookings bookingFinder
	loc      *time.Location
	now      func() time.Time
}

func NewService(store store, bookings bookingFinder, loc *time.Location) *Service {
	return &Service{store: store, bookings: bookings, loc: loc, now: time.Now}
}

func (s *Service) Create(ctx context.Context, adminID, bookingID, discount int64) (Invoice, error) {
	b, err := s.bookings.GetAdmin(ctx, booking.Actor{UserID: adminID}, bookingID)
	if err != nil {
		return Invoice{}, err
	}
	if b.Status != booking.StatusCompleted {
		return Invoice{}, ErrBookingNotCompleted
	}
	if err := validateDiscount(discount, b.Price); err != nil {
		return Invoice{}, err
	}

	period := s.now().In(s.loc).Format("200601")
	id, err := s.store.Create(ctx, period, NewInvoice{
		BookingID: bookingID,
		Subtotal:  b.Price,
		Discount:  discount,
		CreatedBy: adminID,
	})
	if errors.Is(err, errInvoiceExists) {
		return Invoice{}, ErrInvoiceExists
	}
	if err != nil {
		return Invoice{}, err
	}

	return s.Get(ctx, id)
}

func (s *Service) Get(ctx context.Context, id int64) (Invoice, error) {
	inv, err := s.store.Get(ctx, id)
	if errors.Is(err, errNotFound) {
		return Invoice{}, ErrNotFound
	}
	return inv, err
}

func (s *Service) List(ctx context.Context, f Filter, limit, offset int) ([]Invoice, int64, error) {
	f.Query = strings.TrimSpace(f.Query)
	if utf8.RuneCountInString(f.Query) > maxQueryLength {
		return nil, 0, apperr.Validation(map[string]string{"q": fmt.Sprintf("Kata kunci maksimal %d karakter.", maxQueryLength)})
	}
	return s.store.List(ctx, f, like.Contains(f.Query), limit, offset)
}

func (s *Service) UpdateDiscount(ctx context.Context, id, discount int64) (Invoice, error) {
	inv, err := s.unpaid(ctx, id)
	if err != nil {
		return Invoice{}, err
	}
	if err := validateDiscount(discount, inv.Subtotal); err != nil {
		return Invoice{}, err
	}

	if err := s.store.UpdateDiscount(ctx, id, discount); err != nil {
		return Invoice{}, mapPaid(err)
	}
	return s.Get(ctx, id)
}

func (s *Service) Pay(ctx context.Context, id int64, method string) (Invoice, error) {
	if !isValidMethod(method) {
		return Invoice{}, apperr.Validation(map[string]string{"payment_method": "Metode pembayaran harus cash atau transfer."})
	}

	if _, err := s.unpaid(ctx, id); err != nil {
		return Invoice{}, err
	}

	if err := s.store.Pay(ctx, id, method); err != nil {
		return Invoice{}, mapPaid(err)
	}
	return s.Get(ctx, id)
}

func (s *Service) unpaid(ctx context.Context, id int64) (Invoice, error) {
	inv, err := s.Get(ctx, id)
	if err != nil {
		return Invoice{}, err
	}
	if inv.Status == StatusPaid {
		return Invoice{}, ErrInvoicePaid
	}
	return inv, nil
}

func mapPaid(err error) error {
	if errors.Is(err, errAlreadyPaid) {
		return ErrInvoicePaid
	}
	return err
}

func validateDiscount(discount, subtotal int64) error {
	if discount < 0 || discount > subtotal {
		return apperr.Validation(map[string]string{"discount": "Diskon tidak boleh negatif atau melebihi subtotal."})
	}
	return nil
}
