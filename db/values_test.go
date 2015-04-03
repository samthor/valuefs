package db

import (
	"testing"
)

func TestAPI(t *testing.T) {
	// TODO: replace with real test
	s := &Store{}
	var sa StoreAPI
	sa = s
	t.Logf("Store is a StoreAPI", sa)
}
