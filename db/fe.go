package db

import (
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
	control  chan request
	sequence TimeSequence
	count    int
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
			sample := &Sample{
				Value: x.v,
				When:  s.sequence.Next(),
			}
			s.count++
			sv.History = append(sv.History, sample)
		case reqGet:
			if sv == nil {
				break
			}
			r.Sample = sv.get(s.sequence.Next(), x.View)
		case reqClear:
			// unconditionally delete
			// TODO: maybe log this for later log updates
			s.count -= len(sv.History)
			delete(s.values, x.name)
		case reqPrune:
			log.Printf("prune; got %v values (of %v)", s.count, s.config.MemoryValues)
		default:
			panic("unhandled request")
		}

		x.ret <- r
	}
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

// Prune prunes values from this Store.
func (s *store) Prune() bool {
	req := request{requestID: reqPrune}
	s.run(req)
	return true
}
