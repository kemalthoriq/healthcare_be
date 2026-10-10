package booking

import (
	"slices"
	"testing"
	"time"
)

var wib = time.FixedZone("WIB", 7*60*60)

func at(hhmm string) time.Time {
	t, err := time.ParseInLocation("2006-01-02 15:04", "2026-10-20 "+hhmm, wib)
	if err != nil {
		panic(err)
	}
	return t
}

func span(from, to string) Interval {
	return Interval{Start: at(from), End: at(to)}
}

func clocks(ts []time.Time) []string {
	out := make([]string, 0, len(ts))
	for _, t := range ts {
		out = append(out, t.Format("15:04"))
	}
	return out
}

func TestAvailableSlots(t *testing.T) {
	base := SlotInput{
		Shifts:        []Interval{span("10:00", "13:00")},
		Duration:      60 * time.Minute,
		Buffer:        15 * time.Minute,
		Step:          30 * time.Minute,
		EarliestStart: at("00:00"),
	}

	tests := []struct {
		name   string
		modify func(*SlotInput)
		want   []string
	}{
		{
			name: "buffer may run past shift end",
			want: []string{"10:00", "10:30", "11:00", "11:30", "12:00"},
		},
		{
			name:   "treatment must fit inside shift",
			modify: func(in *SlotInput) { in.Duration = 90 * time.Minute },
			want:   []string{"10:00", "10:30", "11:00", "11:30"},
		},
		{
			name:   "treatment longer than shift",
			modify: func(in *SlotInput) { in.Duration = 4 * time.Hour },
			want:   nil,
		},
		{
			name:   "split shifts",
			modify: func(in *SlotInput) { in.Shifts = []Interval{span("10:00", "11:30"), span("14:00", "15:00")} },
			want:   []string{"10:00", "10:30", "14:00"},
		},
		{
			name:   "existing booking blocks overlapping starts including our buffer",
			modify: func(in *SlotInput) { in.Busy = []Interval{span("11:30", "12:15")} },
			want:   []string{"10:00"},
		},
		{
			name:   "booking right after our buffer is fine",
			modify: func(in *SlotInput) { in.Busy = []Interval{span("12:15", "13:30")} },
			want:   []string{"10:00", "10:30", "11:00"},
		},
		{
			name:   "time off blocks slots",
			modify: func(in *SlotInput) { in.Busy = []Interval{span("10:00", "11:00")} },
			want:   []string{"11:00", "11:30", "12:00"},
		},
		{
			name:   "slots before earliest start are hidden",
			modify: func(in *SlotInput) { in.EarliestStart = at("10:45") },
			want:   []string{"11:00", "11:30", "12:00"},
		},
		{
			name:   "no shifts",
			modify: func(in *SlotInput) { in.Shifts = nil },
			want:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := base
			if tt.modify != nil {
				tt.modify(&in)
			}

			got := clocks(AvailableSlots(in))
			if !slices.Equal(got, tt.want) && !(len(got) == 0 && len(tt.want) == 0) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}
