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

// get retrieves the value according to the passed Type. If it is Latest, this
// is simple; otherwise, generate typically based on historic data.
func (sv *storeValue) get(when time.Time, t Type, d time.Duration) *Sample {
	if len(sv.History) == 0 {
		return nil // nothing to do here!
	}
	if t == Latest {
		return sv.History.Last()
	}

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

	if t == Average || t == Total {
		if len(view) == 0 {
			return nil
		}
		s.Value = view.Total()
		if t == Average {
			s.Value /= int64(len(view))
		}
		return s
	}

	log.Printf("interal get got unknown type: %v", t)
	return nil
}

type requestID int

const (
	reqNone requestID = iota
	reqList
	reqLoad
	reqWrite
	reqGet
	reqClear
)

type request struct {
	requestID
	name string
	ret  chan response

	b bool
	v int64

	Type
	time.Duration
}

// response is returned from the Store runner, an aggregate of all possible
// return values.
type response struct {
	time.Time
	RecordList
	*Record
	*Sample
}
