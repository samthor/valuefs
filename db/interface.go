package db

import (
	"strconv"
	"time"
)

// API is a database interface.
type API interface {
	List() RecordList
	Load(string, bool) *Record
	Write(*Record, float64) bool
	Get(*Record, *View) *Sample
	Clear(*Record) bool
	Prune() bool
}

// Storage is provided as a storage mechanism. Passing a nil store will treat
// all writes as successful, aka transient.
type Storage interface {
	Store(*Record, *Sample) error
}

// Config is the configuration for API.
type Config struct {
	MemoryValues int
}

// Type describes an approach to a View.
type Type int

const (
	// None will be ignored.
	None Type = iota

	// Average finds the average over a duration.
	Average

	// Total adds the values in the specified duration.
	Total

	// ValueAt finds the value at given duration in the past, aka the value set
	// most recently before the resolved time.
	ValueAt

	// SafeLatest finds the last set value, assuming that it was set after the
	// specified duration in the past. Prevents dead values.
	SafeLatest
)

// View is a request for a view over a Record.
type View struct {
	Type     Type
	Duration time.Duration
}

// Valid determines whether this View is valid. The nil View is valid.
func (v *View) Valid() bool {
	return v == nil || (v.Type != None && v.Duration >= time.Duration(0))
}

// Record is a header/file combination.
type Record struct {
	Name string
	When time.Time // creation time
}

// Valid determines whether this Record is valid.
func (r *Record) Valid() bool {
	return r != nil && len(r.Name) != 0
}

// Node returns the creation time of this Record as a uint64.
func (r *Record) Node() uint64 {
	if r == nil {
		return 0
	}
	return uint64(r.When.UnixNano())
}

// RecordList is a list of Record objects.
type RecordList []*Record

// Sample is a vector: a value valid at a specific time. Each Sample is
// independent and immutable. In cases where Sample represents a historic
// value, When will typically represent the time it was generated.
type Sample struct {
	Value float64
	When  time.Time
}

// Bytes converts this Sample to a byte array.
func (s *Sample) Bytes() []byte {
	if s == nil {
		return nil
	}
	out := make([]byte, 0, 16)
	out = strconv.AppendFloat(out, s.Value, 'f', -1, 64)
	return append(out, '\n')
}
