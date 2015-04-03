package db

import (
	"testing"
)

func TestView(t *testing.T) {
	var v *View
	if !v.Valid() {
		t.Errorf("expected nil view to be Valid, was invalid")
	}
}
