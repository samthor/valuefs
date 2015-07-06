package db

import (
	"time"
)

// row contains a Sample (and its pointer to the owning storeValue), along
// with whether it has been written to storage.
type row struct {
	s  Sample
	sv *storeValue
}

// sampleList is a list of Samples used in internal storage. Assumed to be
// in ascending order (earliest times first).
type sampleList []*Sample

// Last returns the last Sample, if any.
func (sl sampleList) Last() *Sample {
	if len(sl) == 0 {
		return nil
	}
	return sl[len(sl)-1]
}

// Slice returns the samples from the specified time forward, plus the sample
// immediately prior.
func (sl sampleList) Slice(from time.Time) (out sampleList, prev *Sample) {
	var start int
	end := len(sl)
	for start = end - 1; start >= 0; start-- {
		if from.After(sl[start].When) {
			break // found enough
		}
	}
	if start >= 0 {
		prev = sl[start]
	}
	return sl[start+1 : end], prev
}

// Total sums the values in this sampleList.
func (sl sampleList) Total() (out float64) {
	for _, x := range sl {
		out += x.Value
	}
	return
}
