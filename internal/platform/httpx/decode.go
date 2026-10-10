package httpx

import (
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"strings"

	"github.com/kemalthoriq/healthcare_be/internal/platform/apperr"
)

const (
	maxBodyBytes       = 1 << 20
	unknownFieldPrefix = "json: unknown field "
)

var (
	errUnsupportedMediaType = apperr.New(apperr.KindUnsupportedMediaType, "UNSUPPORTED_MEDIA_TYPE", "Content-Type harus application/json.")
	errBodyTooLarge         = apperr.New(apperr.KindTooLarge, "PAYLOAD_TOO_LARGE", "Ukuran body request maksimal 1 MB.")
	errEmptyBody            = apperr.BadRequest("INVALID_JSON", "Body request tidak boleh kosong.")
	errMalformedJSON        = apperr.BadRequest("INVALID_JSON", "Format JSON tidak valid.")
	errMultipleJSONValues   = apperr.BadRequest("INVALID_JSON", "Body request hanya boleh berisi satu objek JSON.")
)

func DecodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		return errUnsupportedMediaType
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(dst); err != nil {
		return decodeError(err)
	}

	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			return errBodyTooLarge
		}
		return errMultipleJSONValues
	}

	return nil
}

func decodeError(err error) error {
	var syntaxErr *json.SyntaxError
	var typeErr *json.UnmarshalTypeError
	var maxBytesErr *http.MaxBytesError

	switch {
	case errors.Is(err, io.EOF):
		return errEmptyBody
	case errors.As(err, &maxBytesErr):
		return errBodyTooLarge
	case errors.As(err, &syntaxErr), errors.Is(err, io.ErrUnexpectedEOF):
		return errMalformedJSON
	case errors.As(err, &typeErr) && typeErr.Field != "":
		return apperr.Validation(map[string]string{typeErr.Field: "Tipe data tidak sesuai."})
	case strings.HasPrefix(err.Error(), unknownFieldPrefix):
		field := strings.Trim(strings.TrimPrefix(err.Error(), unknownFieldPrefix), `"`)
		return apperr.Validation(map[string]string{field: "Field tidak dikenal."})
	default:
		return errMalformedJSON
	}
}
