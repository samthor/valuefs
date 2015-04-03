package db

import (
	"testing"
)

func TestTimeSequence(t *testing.T) {
	seq := &TimeSequence{}
	first := seq.Next()
	second := seq.Next()

	if !second.After(first) {
		t.Errorf("expected second after first; was first=%v second=%v", first, second)
	}
}
