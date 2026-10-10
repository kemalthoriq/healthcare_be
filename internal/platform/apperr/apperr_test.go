package apperr

import (
	"errors"
	"fmt"
	"testing"
)

var errSlotTaken = Conflict("SLOT_TAKEN", "Jam ini baru saja dipesan.")

func TestError_IsMatchesWrappedSentinel(t *testing.T) {
	err := fmt.Errorf("create booking: %w", errSlotTaken)

	if !errors.Is(err, errSlotTaken) {
		t.Fatal("errors.Is should find the sentinel through wrapping")
	}
}

func TestError_AsExtractsWrappedError(t *testing.T) {
	err := fmt.Errorf("create booking: %w", errSlotTaken)

	var appErr *Error
	if !errors.As(err, &appErr) {
		t.Fatal("errors.As should extract *Error through wrapping")
	}
	if appErr.Kind != KindConflict || appErr.Code != "SLOT_TAKEN" {
		t.Errorf("got kind %v code %q, want KindConflict SLOT_TAKEN", appErr.Kind, appErr.Code)
	}
}

func TestError_SeparatelyCreatedErrorsAreNotEqual(t *testing.T) {
	other := Conflict("SLOT_TAKEN", "Jam ini baru saja dipesan.")

	if errors.Is(other, errSlotTaken) {
		t.Fatal("errors.Is compares identity, two separately created errors must not match")
	}
}

func TestError_Message(t *testing.T) {
	err := NotFound("Treatment tidak ditemukan.")

	if got, want := err.Error(), "NOT_FOUND: Treatment tidak ditemukan."; got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestValidation(t *testing.T) {
	err := Validation(map[string]string{"phone": "Nomor WhatsApp wajib diisi."})

	if err.Kind != KindUnprocessable || err.Code != "VALIDATION_ERROR" {
		t.Errorf("got kind %v code %q, want KindUnprocessable VALIDATION_ERROR", err.Kind, err.Code)
	}
	if got := err.Fields["phone"]; got != "Nomor WhatsApp wajib diisi." {
		t.Errorf("Fields[phone] = %q", got)
	}
}
