package therapist

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/kemalthoriq/healthcare_be/internal/platform/apperr"
	"github.com/kemalthoriq/healthcare_be/internal/platform/httpx"
)

type shiftJSON struct {
	DayOfWeek int    `json:"day_of_week"`
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
}

type schedulesRequest struct {
	Shifts []shiftJSON `json:"shifts"`
}

type timeOffRequest struct {
	StartAt time.Time `json:"start_at"`
	EndAt   time.Time `json:"end_at"`
	Reason  string    `json:"reason"`
}

type timeOffResponse struct {
	ID        int64     `json:"id"`
	StartAt   time.Time `json:"start_at"`
	EndAt     time.Time `json:"end_at"`
	Reason    string    `json:"reason"`
	CreatedAt time.Time `json:"created_at"`
}

func toShiftJSON(shifts []Shift) []shiftJSON {
	out := make([]shiftJSON, 0, len(shifts))
	for _, s := range shifts {
		out = append(out, shiftJSON{DayOfWeek: s.DayOfWeek, StartTime: FormatClock(s.Start), EndTime: FormatClock(s.End)})
	}
	return out
}

func toTimeOffResponse(t TimeOff) timeOffResponse {
	return timeOffResponse{ID: t.ID, StartAt: t.StartAt, EndAt: t.EndAt, Reason: t.Reason, CreatedAt: t.CreatedAt}
}

func (h *Handler) ListSchedules(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r)
	if !ok {
		httpx.Error(w, r, ErrNotFound)
		return
	}

	shifts, err := h.service.ListSchedules(r.Context(), id)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, r, http.StatusOK, toShiftJSON(shifts))
}

func (h *Handler) SetSchedules(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r)
	if !ok {
		httpx.Error(w, r, ErrNotFound)
		return
	}

	var req schedulesRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	if req.Shifts == nil {
		httpx.Error(w, r, apperr.Validation(map[string]string{"shifts": "Daftar shift wajib diisi (boleh kosong: [])."}))
		return
	}

	shifts := make([]Shift, 0, len(req.Shifts))
	fields := map[string]string{}
	for i, s := range req.Shifts {
		start, okStart := ParseClock(s.StartTime)
		end, okEnd := ParseClock(s.EndTime)
		if !okStart || !okEnd {
			fields[fmt.Sprintf("shifts[%d]", i)] = "Format jam harus HH:MM, contoh 09:30."
			continue
		}
		shifts = append(shifts, Shift{DayOfWeek: s.DayOfWeek, Start: start, End: end})
	}
	if len(fields) > 0 {
		httpx.Error(w, r, apperr.Validation(fields))
		return
	}

	saved, err := h.service.SetSchedules(r.Context(), id, shifts)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, r, http.StatusOK, toShiftJSON(saved))
}

func (h *Handler) ListTimeOff(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r)
	if !ok {
		httpx.Error(w, r, ErrNotFound)
		return
	}

	items, err := h.service.ListUpcomingTimeOff(r.Context(), id)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}

	out := make([]timeOffResponse, 0, len(items))
	for _, t := range items {
		out = append(out, toTimeOffResponse(t))
	}
	httpx.JSON(w, r, http.StatusOK, out)
}

func (h *Handler) CreateTimeOff(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r)
	if !ok {
		httpx.Error(w, r, ErrNotFound)
		return
	}

	var req timeOffRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}

	t, err := h.service.CreateTimeOff(r.Context(), id, TimeOffInput{StartAt: req.StartAt, EndAt: req.EndAt, Reason: req.Reason})
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, r, http.StatusCreated, toTimeOffResponse(t))
}

func (h *Handler) DeleteTimeOff(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r)
	if !ok {
		httpx.Error(w, r, ErrNotFound)
		return
	}
	timeOffID, err := strconv.ParseInt(r.PathValue("timeOffId"), 10, 64)
	if err != nil || timeOffID <= 0 {
		httpx.Error(w, r, ErrTimeOffNotFound)
		return
	}

	if err := h.service.DeleteTimeOff(r.Context(), id, timeOffID); err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.NoContent(w)
}
