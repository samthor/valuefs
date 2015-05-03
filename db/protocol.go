package db

import (
	"log"
	"time"
)

// storeValue is a Record plus its historic sampleList.
type storeValue struct {
	Record
	History sampleList
}

// get retrieves the value for the given View. If the View is nil, then this is
// simple; otherwise generate based on historic data.
func (sv *storeValue) get(when time.Time, v *View) *Sample {
	if len(sv.History) == 0 {
		return nil // nothing to do here!
	}
	if v == nil {
		return sv.History.Last()
	}

	t, d := v.Type, v.Duration
	view, prev := sv.History.Slice(when.Add(-d))
	log.Printf("using values: %+v, prev: %v", view, prev)
	s := &Sample{When: when}

	if t == ValueAt {
		if prev == nil {
			return nil
		}
		s.Value = prev.Value
		return s
	}

	if t == SafeLatest {
		// TODO: could be faster, just look at last in sv.History
		if len(view) == 0 {
			return nil
		}
		s.Value = view[len(view)-1].Value
		return s
	}

	if t == Average || t == Total {
		if len(view) == 0 {
			return nil
		}
		s.Value = view.Total()
		if t == Average {
			s.Value /= float64(len(view))
		}
		return s
	}

	log.Printf("internal get got unknown type: %v", t)
	return nil
}

