package apperr

type Kind int

const (
	KindInternal Kind = iota
	KindBadRequest
	KindUnauthenticated
	KindForbidden
	KindNotFound
	KindConflict
	KindTooLarge
	KindUnsupportedMediaType
	KindUnprocessable
	KindRateLimited
)

type Error struct {
	Kind    Kind
	Code    string
	Message string
	Fields  map[string]string
}

func (e *Error) Error() string {
	return e.Code + ": " + e.Message
}

func New(kind Kind, code, message string) *Error {
	return &Error{Kind: kind, Code: code, Message: message}
}

func BadRequest(code, message string) *Error {
	return New(KindBadRequest, code, message)
}

func Unauthenticated() *Error {
	return New(KindUnauthenticated, "UNAUTHENTICATED", "Silakan login terlebih dahulu.")
}

func Forbidden() *Error {
	return New(KindForbidden, "FORBIDDEN", "Anda tidak memiliki akses.")
}

func NotFound(message string) *Error {
	return New(KindNotFound, "NOT_FOUND", message)
}

func Conflict(code, message string) *Error {
	return New(KindConflict, code, message)
}

func Unprocessable(code, message string) *Error {
	return New(KindUnprocessable, code, message)
}

func Validation(fields map[string]string) *Error {
	return &Error{
		Kind:    KindUnprocessable,
		Code:    "VALIDATION_ERROR",
		Message: "Data belum sesuai.",
		Fields:  fields,
	}
}

func RateLimited() *Error {
	return New(KindRateLimited, "RATE_LIMITED", "Terlalu banyak permintaan. Silakan coba lagi nanti.")
}
