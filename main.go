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
	mountPath = flag.String("mount", "", "mountpoint of filesystem")
	logPath   = flag.String("log", "", "logfile to use, if any")
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

	v := &ValueFS{db.New()}

	if err = fs.Serve(c, v); err != nil {
		log.Fatal(err)
	}

	// check if the mount process has an error to report
	<-c.Ready
	if err := c.MountError; err != nil {
		log.Fatal(err)
	}
}
