package db

import (
	"time"
	"log"
)

// New returns a new instance implementing the API for this package. It should
// be used by a filesystem or other user-facing interface.
func New(c *Config) API {
	s := &store{
		values:  make(map[string]*storeValue),
		control: make(chan request),
		config:  *c,
	}
	go s.runner()

	return s
}

// store is the top-level database store for ValueFS. It implements API.
type store struct {
	values   map[string]*storeValue
	rows     []*row
	control  chan request
	sequence TimeSequence
	config   Config
}

func (s *store) runner() {
	for x := range s.control {
		r := response{Time: s.sequence.Next()}
		var sv *storeValue

		if x.name != "" {
			sv = s.values[x.name]
		}

		switch x.requestID {
		case reqNone:
			// do nothing
		case reqList:
			r.RecordList = make(RecordList, 0) // always non-nil
			for _, v := range s.values {
				// TODO: copy v.Record?
				r.RecordList = append(r.RecordList, &v.Record)
			}
		case reqLoad:
			if sv == nil && x.b {
				// create if it doesn't exist
				sv = &storeValue{
					Record: Record{
						Name: x.name,
						When: s.sequence.Next(),
					},
				}
				s.values[x.name] = sv
			}
			if sv != nil {
				// TODO: copy?
				r.Record = &sv.Record
			}
		case reqWrite:
			if sv == nil {
				break
			}

			r := &row{
				s: Sample{
					Value: x.v,
					When:  s.sequence.Next(),
				},
				sv: sv,
			}
			s.rows = append(s.rows, r)

			// If we have too many values, prune the last other one.
			if len(s.rows) > s.config.MemoryValues {
				// TODO: use circ buffer?
				clear := s.rows[0]
				s.rows = s.rows[1:]
				log.Printf("got to prune: %v", clear)

				check := clear.sv.History[0]
				if check != &clear.s {
					log.Fatal("rec didn't match: %+v", clear.sv)
				}
				clear.sv.History = clear.sv.History[1:]
			}

			sv.History = append(sv.History, &r.s)
		case reqGet:
			if sv == nil {
				break
			}
			r.Sample = sv.get(s.sequence.Next(), x.View)
		case reqClear:
			// unconditionally delete
			// TODO: maybe log this for later log updates
			delete(s.values, x.name)
		default:
			panic("unhandled request")
		}

		x.ret <- r
	}
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
	v float64

	*View
}

// response is returned from the Store runner, an aggregate of all possible
// return values.
type response struct {
	time.Time
	RecordList
	*Record
	*Sample
}

func (s *store) run(r request) response {
	r.ret = make(chan response)
	s.control <- r
	return <-r.ret
}

// List returns the RecordList for all real data.
func (s *store) List() RecordList {
	req := request{requestID: reqList}
	resp := s.run(req)
	return resp.RecordList
}

// Load loads or creates a Record for the specified name.
func (s *store) Load(name string, create bool) *Record {
	if len(name) == 0 {
		return nil
	}

	req := request{requestID: reqLoad, name: name, b: create}
	resp := s.run(req)

	// FIXME: clearer clone
	if resp.Record == nil {
		return nil
	}
	var rec Record = *resp.Record
	return &rec
}

// Get retrieves a Sample for this Record of the given type.
func (s *store) Get(rec *Record, view *View) *Sample {
	if !rec.Valid() || !view.Valid() {
		return nil
	}

	req := request{requestID: reqGet, name: rec.Name, View: view}
	resp := s.run(req)

	// FIXME: clearer clone
	if resp.Sample == nil {
		return nil
	}
	var sample Sample = *resp.Sample
	return &sample
}

// Write unconditionally sets a new value for the specified Record.
func (s *store) Write(rec *Record, value float64) bool {
	if !rec.Valid() {
		return false
	}
	req := request{requestID: reqWrite, name: rec.Name, v: value}
	s.run(req)
	return true
}

// Clear unconditionally removes this Record. It may not delete historic
// values (or at least they may remain in logs).
func (s *store) Clear(rec *Record) bool {
	if !rec.Valid() {
		return false
	}
	req := request{requestID: reqClear, name: rec.Name}
	s.run(req)
	return true
}

