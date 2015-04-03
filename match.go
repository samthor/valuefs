package main

import (
	"regexp"

	"github.com/samthor/valuefs/db"
)

var (
	modes  = `#%@^`
	pathRe = regexp.MustCompile(`^([\w][\w\-\.]*)(([` + regexp.QuoteMeta(modes) + `])(\w+))?$`)
)

// matchPath matches the given path against the ValueFS regex. Ensures that the
// path has the required length.
func matchPath(path string) (base, mode, ext string, ok bool) {
	m := pathRe.FindStringSubmatch(path)
	if m == nil {
		return // default values are fine
	}
	return m[1], m[3], m[4], true
}

// matchLatestPath matches only the latest-form of a path. Ensures that it has
// the required length.
func matchLatestPath(path string) (base string, ok bool) {
	var mode, ext string
	base, mode, ext, ok = matchPath(path)
	if mode != "" || ext != "" {
		return "", false
	}
	return
}

// matchMode matches the mode string to a db.Type.
func matchMode(mode string) (t db.Type, ok bool) {
	ok = true

	switch mode {
	case "":
		t = db.Latest
	case "#":
		t = db.Average
	case "%":
		t = db.Total
	case "@":
		t = db.ValueAt
	default:
		ok = false
	}

	return
}
