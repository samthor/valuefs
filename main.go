// valuefs implements a value-based filesystem (?)
package main

import (
	"flag"
	"log"
	"os"
	"os/signal"

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

	config := &db.Config{
		MemoryValues: 100,
	}
	v := &ValueFS{db.New(config)}
	go signalWait(v.Store)

	if err = fs.Serve(c, v); err != nil {
		log.Fatal(err)
	}

	// check if the mount process has an error to report
	<-c.Ready
	if err := c.MountError; err != nil {
		log.Fatal(err)
	}
}

// signalWait translates all Interrupt signals to a call to the Prune function
// on Store.
func signalWait(store db.API) {
	ch := make(chan os.Signal)
	signal.Notify(ch, os.Interrupt)
	for s := range ch {
		log.Printf("signal: %v", s)
		store.Prune()
	}
}
