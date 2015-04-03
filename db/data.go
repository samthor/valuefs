package db

import (
	"fmt"
	"strconv"
	"time"
)

// StoreAPI is a database interface.
type StoreAPI interface {
	List() RecordList
	Load(string, bool) *Record
	Write(*Record, int64) bool
	GetLatest(*Record) *Sample
	Get(*Record, Type, time.Duration) *Sample
	Clear(*Record)
}

// Type describes an approach to Get.
type Type int

const (
	// Latest is the default, fetching the most recent value.
	Latest Type = iota

	// Average finds the average over a duration.
	Average

	// Total adds the values in the specified duration.
	Total

	// ValueAt finds the value at given duration in the past, aka the value set
	// most recently before the resolved time.
	ValueAt
)

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
	Value int64
	When  time.Time
}

// String converts this Sample to its output.
func (s *Sample) String() string {
	if s == nil {
		return ""
	}
	return fmt.Sprintf("%d\n", s.Value)
}

// Bytes converts this Sample to a byte array.
func (s *Sample) Bytes() []byte {
	if s == nil {
		return nil
	}
	out := make([]byte, 0, 16)
	out = strconv.AppendInt(out, s.Value, 10)
	return append(out, '\n')
}
