package httpx

import (
	"net/http"
	"strconv"
)

const (
	defaultPerPage = 20
	maxPerPage     = 100
)

type Page struct {
	Page    int   `json:"page"`
	PerPage int   `json:"per_page"`
	Total   int64 `json:"total"`
}

func (p Page) Offset() int {
	return (p.Page - 1) * p.PerPage
}

func PageFromRequest(r *http.Request) Page {
	page, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil || page < 1 {
		page = 1
	}

	perPage, err := strconv.Atoi(r.URL.Query().Get("per_page"))
	if err != nil || perPage < 1 {
		perPage = defaultPerPage
	}
	if perPage > maxPerPage {
		perPage = maxPerPage
	}

	return Page{Page: page, PerPage: perPage}
}

type pageEnvelope struct {
	Data any  `json:"data"`
	Meta Page `json:"meta"`
}

func JSONPage(w http.ResponseWriter, r *http.Request, data any, page Page) {
	write(w, r, http.StatusOK, pageEnvelope{Data: data, Meta: page})
}
