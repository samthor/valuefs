// valuefs implements a value-based filesystem (?)
package main

import (
	"flag"
	"log"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"

	"github.com/samthor/valuefs/db"
)

var (
	mountPath    = flag.String("mount", "", "mountpoint of filesystem")
	logPath      = flag.String("log", "", "logfile to use, if any")
	memoryValues = flag.Int("values", 100, "number of values to hold in memory")
)

func main() {
	flag.Parse()

	c, err := fuse.Mount(
		*mountPath,
		fuse.FSName("valuefs"),
		fuse.LocalVolume(),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	config := &db.Config{
		MemoryValues: *memoryValues,
	}
	v := &ValueFS{db.New(config)}

	if err = fs.Serve(c, v); err != nil {
		log.Fatal(err)
	}

	// check if the mount process has an error to report
	<-c.Ready
	if err := c.MountError; err != nil {
		log.Fatal(err)
	}
}
