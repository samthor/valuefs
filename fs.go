package main

import (
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"golang.org/x/net/context"

	"github.com/samthor/valuefs/db"
)

// ValueFS implements the file system and its root directory.
type ValueFS struct {
	Store *db.Store
}

func (vfs *ValueFS) Root() (fs.Node, error) {
	return vfs, nil
}

func (vfs *ValueFS) Attr() fuse.Attr {
	return fuse.Attr{Inode: 1, Mode: os.ModeDir | 0555}
}

func (vfs *ValueFS) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	list := vfs.Store.List()
	out := make([]fuse.Dirent, len(list))
	for i, rec := range list {
		out[i] = fuse.Dirent{
			Name:  rec.Name,
			Inode: rec.Node(),
			Type:  fuse.DT_File,
		}
	}
	return out, nil
}

func (vfs *ValueFS) Create(ctx context.Context, req *fuse.CreateRequest, resp *fuse.CreateResponse) (fs.Node, fs.Handle, error) {
	name, ok := matchLatestPath(req.Name)
	if !ok {
		// Only allow creation of real files without mode (and that have a name).
		return nil, nil, fuse.ENOENT
	}

	rec := vfs.Store.Load(name, true)
	if rec == nil {
		return nil, nil, fuse.ENOENT
	}

	vf := &ValueFile{
		ValueFS: vfs,
		Record:  rec,
	}
	return vf, vf, nil
}

func (vfs *ValueFS) Remove(ctx context.Context, req *fuse.RemoveRequest) error {
	if req.Dir {
		return fuse.EIO // should never happen, do nothing
	}
	name, ok := matchLatestPath(req.Name)
	if !ok {
		return fuse.ENOENT
	}

	// TODO: remove round-trip of loading data
	rec := vfs.Store.Load(name, true)
	if rec != nil {
		vfs.Store.Clear(rec)
		req.Respond()
	}
	return nil
}

func (vfs *ValueFS) Lookup(ctx context.Context, name string) (fs.Node, error) {
	name, mode, ext, ok := matchPath(name)
	if !ok {
		return nil, fuse.ENOENT
	}

	var t db.Type
	var d time.Duration
	if mode != "" {
		var ok bool
		t, ok = matchMode(mode)
		if !ok {
			log.Printf("ignoring mode/ext: %v/%v", mode, ext)
			return nil, fuse.ENOENT
		}
		var err error
		d, err = time.ParseDuration(ext)
		if err != nil {
			log.Printf("invalid duration: %v", ext)
			return nil, fuse.ENOENT
		}
	}

	rec := vfs.Store.Load(name, false)
	if rec == nil {
		return nil, fuse.ENOENT
	}

	// TODO: If mode is not Latest, and the Sample is nil, then this could
	// return ENOENT although it would require another round-trip.

	vf := &ValueFile{
		ValueFS:  vfs,
		Record:   rec,
		Type:     t,
		Duration: d,
	}
	return vf, nil
}

// ValueFile wraps the visible db.Sample, and also holds a reference to the
// underlying ValueFS.
type ValueFile struct {
	*ValueFS

	*db.Record
	db.Type // Latest is default, only writable
	time.Duration

	*db.Sample // possibly nil

	Bytes []byte // rendered bytes
}

func (vf *ValueFile) Attr() fuse.Attr {
	// reload on every Attr call.
	// we MUST do this because if data is written, it becomes new...
	store := vf.ValueFS.Store
	sample := store.Get(vf.Record, vf.Type, vf.Duration)
	vf.Sample = sample
	vf.Bytes = sample.Bytes()

	when := vf.Record.When
	node := vf.Record.Node()
	if vf.Sample != nil {
		when = vf.Sample.When // using inode of sample
		node = uint64(vf.Sample.When.UnixNano())
	} else {
		// Use the inode of the record itself. This happens in two cases-
		//   1) no sample data
		//   2) no result from mode
		// Both are fine, although
	}

	return fuse.Attr{
		Inode: node,
		Mode:  0664,
		Size:  uint64(len(vf.Bytes)),
		Mtime: when,
		Ctime: when,
	}
}

func (vf *ValueFile) ReadAll(ctx context.Context) ([]byte, error) {
	return vf.Bytes, nil
}

func (vf *ValueFile) Write(ctx context.Context, req *fuse.WriteRequest, resp *fuse.WriteResponse) error {
	if req.Offset != 0 {
		return fuse.EIO
	}

	s := strings.TrimSpace(string(req.Data))
	parsed, err := strconv.ParseInt(s, 0, 64)
	if err != nil {
		return fuse.EIO
	}

	log.Printf("got write: %v => %v", vf.Name, parsed)

	store := vf.ValueFS.Store
	store.Write(vf.Record, parsed)
	resp.Size = len(req.Data)
	return nil
}
