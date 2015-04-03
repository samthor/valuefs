package db

// Store is the top-level database store for ValueFS. It implements StoreAPI.
type Store struct {
	values   map[string]*storeValue
	control  chan request
	sequence TimeSequence
}

func (s *Store) Run() {
	if s.control != nil {
		panic("should only be Run once")
	}
	s.values = make(map[string]*storeValue)
	s.control = make(chan request)
	go func() {
		for x := range s.control {
			r := response{Time: s.sequence.Next()}

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
				sv := s.values[x.name]
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
				sv := s.values[x.name]
				if sv == nil {
					break
				}
				s := &Sample{
					Value: x.v,
					When:  s.sequence.Next(),
				}
				sv.History = append(sv.History, s)
			case reqGet:
				sv := s.values[x.name]
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
	}()
}

func (s *Store) run(r request) response {
	r.ret = make(chan response)
	s.control <- r
	return <-r.ret
}

// List returns the RecordList for all real data.
func (s *Store) List() RecordList {
	req := request{requestID: reqList}
	resp := s.run(req)
	return resp.RecordList
}

// Load loads or creates a Record for the specified name.
func (s *Store) Load(name string, create bool) *Record {
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
func (s *Store) Get(rec *Record, view *View) *Sample {
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
func (s *Store) Write(rec *Record, value int64) bool {
	if !rec.Valid() {
		return false
	}
	req := request{requestID: reqWrite, name: rec.Name, v: value}
	s.run(req)
	return true
}

// Clear unconditionally removes this Record. It may not delete historic
// values (or at least they may remain in logs).
func (s *Store) Clear(rec *Record) bool {
	if !rec.Valid() {
		return false
	}
	req := request{requestID: reqClear, name: rec.Name}
	s.run(req)
	return true
}
