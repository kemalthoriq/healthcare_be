package treatment

import (
	"strings"
	"time"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

type Treatment struct {
	ID              int64
	Name            string
	Slug            string
	Description     string
	ImageURL        string
	Price           int64
	DurationMinutes int32
	IsActive        bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type Input struct {
	Name            string
	Description     string
	ImageURL        string
	Price           int64
	DurationMinutes int32
	IsActive        bool
}

func Slugify(name string) string {
	var b strings.Builder
	dash := false

	for _, r := range norm.NFD.String(strings.ToLower(name)) {
		switch {
		case unicode.Is(unicode.Mn, r):
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
			dash = false
		case b.Len() > 0 && !dash:
			b.WriteByte('-')
			dash = true
		}
	}

	slug := strings.TrimSuffix(b.String(), "-")
	if slug == "" {
		return "treatment"
	}
	return slug
}
