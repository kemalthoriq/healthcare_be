package booking

import "time"

type Interval struct {
	Start time.Time
	End   time.Time
}

func (a Interval) overlaps(b Interval) bool {
	return a.Start.Before(b.End) && b.Start.Before(a.End)
}

type SlotInput struct {
	Shifts        []Interval
	Busy          []Interval
	Duration      time.Duration
	Buffer        time.Duration
	Step          time.Duration
	EarliestStart time.Time
}

func AvailableSlots(in SlotInput) []time.Time {
	var slots []time.Time

	for _, shift := range in.Shifts {
		for start := shift.Start; !start.Add(in.Duration).After(shift.End); start = start.Add(in.Step) {
			if start.Before(in.EarliestStart) {
				continue
			}

			candidate := Interval{Start: start, End: start.Add(in.Duration + in.Buffer)}
			if !overlapsAny(candidate, in.Busy) {
				slots = append(slots, start)
			}
		}
	}

	return slots
}

func overlapsAny(candidate Interval, busy []Interval) bool {
	for _, b := range busy {
		if candidate.overlaps(b) {
			return true
		}
	}
	return false
}
