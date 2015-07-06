package db

import (
	"log"
	"time"
)

// New returns a new instance implementing the API for this package. It should
// be used by a filesystem or other user-facing interface.
func New(c *Config, storage Storage) API {
	if c == nil {
		c = &Config{
			MemoryValues: 100,
		}
	}

	s := &store{
		values:  make(map[string]*storeValue),
		control: make(chan request),
		config:  *c,
		storage: storage,
	}
	go s.runner()

	return s
}

// store is the top-level database store for ValueFS. It implements API.
type store struct {
	values    map[string]*storeValue
	rows      []*row // sorted forwards
	watermark int // not including this row
	control   chan request
	sequence  TimeSequence
	config    Config
	storage   Storage
}

// write sends unwritten values to storage.
func (s *store) write() {
	if s.storage == nil {
		s.watermark = len(s.rows)
		return
	}

	for i := s.watermark; i < len(s.rows); i++ {
		row := s.rows[i]
		err := s.storage.Store(&row.sv.Record, &row.s)
		if err != nil {
			log.Printf("can't write to storage: err=%v", err)
			return
		}
		s.watermark = i+1
	}
}

// prune removes values.
func (s *store) prune() {
	clear := make(map[*storeValue]struct {
		rows []*row
	})

	var retain []*row // last-most values

	var i int
	for i = 0; len(s.rows)-i > s.config.MemoryValues; i++ {
		cand := s.rows[i]

		values := len(cand.sv.History)
		if values <= 1 {
			retain = append(retain, cand)
			// TODO: panic if == 0?
			continue
		}

		c := clear[cand.sv]
		c.rows = append(c.rows, cand)
		clear[cand.sv] = c
		s.watermark--
	}

	if s.watermark < 0 {
		log.Printf("warning: dropped %d unwritten values", -s.watermark)
		s.watermark = 0
	}

	remain := len(s.rows)-i+len(retain)
	if len(s.rows) != remain {
		log.Printf("pruned rows from %d => %d (%d last)", len(s.rows), remain, len(retain))
	}
	s.rows = append(retain, s.rows[i:]...)

	for sv, c := range clear {
		count := len(c.rows)
		log.Printf("pruning '%s': removing %d prefix", sv.Record.Name, count)
		sv.History = sv.History[count:]
	}
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
			sv.History = append(sv.History, &r.s)
		case reqGet:
			if sv == nil {
				break
			}
			r.Sample = sv.get(s.sequence.Next(), x.View)
		case reqClear:
			// unconditionally delete
			// TODO: maybe log this for later log updates
			// Note that this will keep the actual rows around, detached, until
			// pruned. If a new value by this name is added, it'll be unrelated.
			delete(s.values, x.name)
		case reqPrune:
			s.write()
			s.prune()
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
	reqPrune
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

func (s *store) Prune() bool {
	req := request{requestID: reqPrune}
	s.run(req)
	return true
}
