package invoice

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/kemalthoriq/healthcare_be/internal/booking"
	"github.com/kemalthoriq/healthcare_be/internal/platform/apperr"
)

var wib = time.FixedZone("WIB", 7*60*60)

type fakeStore struct {
	items   map[int64]Invoice
	periods []string
	payErr  error
}

func (f *fakeStore) Create(ctx context.Context, period string, in NewInvoice) (int64, error) {
	for _, inv := range f.items {
		if inv.BookingID == in.BookingID {
			return 0, errInvoiceExists
		}
	}
	f.periods = append(f.periods, period)
	id := int64(len(f.items) + 1)
	f.items[id] = Invoice{ID: id, Number: FormatNumber(period, int32(id)), BookingID: in.BookingID, Subtotal: in.Subtotal,
		Discount: in.Discount, Total: in.Subtotal - in.Discount, Status: StatusUnpaid}
	return id, nil
}

func (f *fakeStore) Get(ctx context.Context, id int64) (Invoice, error) {
	inv, ok := f.items[id]
	if !ok {
		return Invoice{}, errNotFound
	}
	return inv, nil
}

func (f *fakeStore) List(ctx context.Context, filter Filter, pattern string, limit, offset int) ([]Invoice, int64, error) {
	return nil, 0, nil
}

func (f *fakeStore) UpdateDiscount(ctx context.Context, id, discount int64) error {
	inv := f.items[id]
	inv.Discount, inv.Total = discount, inv.Subtotal-discount
	f.items[id] = inv
	return nil
}

func (f *fakeStore) Pay(ctx context.Context, id int64, method string) error {
	if f.payErr != nil {
		return f.payErr
	}
	inv := f.items[id]
	inv.Status, inv.PaymentMethod = StatusPaid, method
	f.items[id] = inv
	return nil
}

type fakeBookings map[int64]booking.AdminBooking

func (f fakeBookings) GetAdmin(ctx context.Context, actor booking.Actor, id int64) (booking.AdminBooking, error) {
	b, ok := f[id]
	if !ok {
		return booking.AdminBooking{}, booking.ErrBookingNotFound
	}
	return b, nil
}

func newTestService() (*Service, *fakeStore) {
	store := &fakeStore{items: map[int64]Invoice{}}
	bookings := fakeBookings{
		1: {Booking: booking.Booking{ID: 1, Status: booking.StatusCompleted, Price: 350000}},
		2: {Booking: booking.Booking{ID: 2, Status: booking.StatusConfirmed, Price: 350000}},
	}
	svc := NewService(store, bookings, wib)
	svc.now = func() time.Time { return time.Date(2026, 10, 31, 18, 0, 0, 0, time.UTC) }
	return svc, store
}

func wantCode(t *testing.T, err error, code string) {
	t.Helper()
	var appErr *apperr.Error
	if !errors.As(err, &appErr) || appErr.Code != code {
		t.Fatalf("err = %v, want %s", err, code)
	}
}

func TestCreate(t *testing.T) {
	svc, store := newTestService()

	inv, err := svc.Create(context.Background(), 9, 1, 50000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if inv.Subtotal != 350000 || inv.Discount != 50000 || inv.Total != 300000 {
		t.Errorf("amounts = %d - %d = %d", inv.Subtotal, inv.Discount, inv.Total)
	}
	if store.periods[0] != "202611" {
		t.Errorf("period = %q, want 202611 (31 Oct 18:00 UTC is already November in WIB)", store.periods[0])
	}

	_, err = svc.Create(context.Background(), 9, 1, 0)
	wantCode(t, err, "INVOICE_EXISTS")
}

func TestCreate_Rejects(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	_, err := svc.Create(ctx, 9, 2, 0)
	wantCode(t, err, "BOOKING_NOT_COMPLETED")

	_, err = svc.Create(ctx, 9, 1, 350001)
	wantCode(t, err, "VALIDATION_ERROR")

	_, err = svc.Create(ctx, 9, 1, -1)
	wantCode(t, err, "VALIDATION_ERROR")

	if _, err := svc.Create(ctx, 9, 99, 0); !errors.Is(err, booking.ErrBookingNotFound) {
		t.Errorf("err = %v, want booking.ErrBookingNotFound", err)
	}
}

func TestUpdateDiscountAndPay(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()
	inv, _ := svc.Create(ctx, 9, 1, 0)

	updated, err := svc.UpdateDiscount(ctx, inv.ID, 100000)
	if err != nil || updated.Total != 250000 {
		t.Fatalf("update discount: total = %d, err = %v", updated.Total, err)
	}

	_, err = svc.Pay(ctx, inv.ID, "qris")
	wantCode(t, err, "VALIDATION_ERROR")

	paid, err := svc.Pay(ctx, inv.ID, MethodTransfer)
	if err != nil || paid.Status != StatusPaid {
		t.Fatalf("pay: status = %q, err = %v", paid.Status, err)
	}

	_, err = svc.UpdateDiscount(ctx, inv.ID, 0)
	wantCode(t, err, "INVOICE_PAID")
	_, err = svc.Pay(ctx, inv.ID, MethodCash)
	wantCode(t, err, "INVOICE_PAID")

	if _, err := svc.Get(ctx, 99); !errors.Is(err, ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

func TestPay_ConcurrentPayment(t *testing.T) {
	svc, store := newTestService()
	inv, _ := svc.Create(context.Background(), 9, 1, 0)
	store.payErr = errAlreadyPaid

	_, err := svc.Pay(context.Background(), inv.ID, MethodCash)
	wantCode(t, err, "INVOICE_PAID")
}
