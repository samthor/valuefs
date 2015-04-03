package db

import (
	"time"
)

// TimeSequence generates ever-increasing time.Time values.
type TimeSequence struct {
	last time.Time
}

// Next returns the next time.Time, which will be either now or in the future.
func (ts *TimeSequence) Next() time.Time {
	out := time.Now()

	if out.After(ts.last) {
		// all good
	} else {
		out = ts.last.Add(time.Duration(1))
	}

	ts.last = out
	return out
}
