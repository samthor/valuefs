package main

import (
	"regexp"
	"time"

	"github.com/samthor/valuefs/db"
)

var (
	modes  = `#%@^`
	pathRe = regexp.MustCompile(`^([\w][\w\-\.]*)(([` + regexp.QuoteMeta(modes) + `])(\w+))?$`)
)

// matchPath matches the given path against the ValueFS regex. Ensures that the
// path has the required length.
func matchPath(path string) (base string, view *db.View, ok bool) {
	m := pathRe.FindStringSubmatch(path)
	if m == nil {
		return // default values are fine
	}

	base = m[1]
	mode, ext := m[3], m[4]
	var t db.Type

	if mode == "" {
		return base, nil, true
	}

	switch mode {
	default:
		return
	case "#":
		t = db.Average
	case "%":
		t = db.Total
	case "@":
		t = db.ValueAt
	}

	d, err := time.ParseDuration(ext)
	if err != nil {
		return
	}

	view = &db.View{Type: t, Duration: d}
	return base, view, true
}

// matchLatestPath matches only the latest-form of a path. Ensures that it has
// the required length.
func matchLatestPath(path string) (base string, ok bool) {
	var view *db.View
	base, view, ok = matchPath(path)
	if view != nil {
		return base, false
	}
	return
}
