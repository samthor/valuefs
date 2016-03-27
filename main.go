// valuefs implements a value-based filesystem (?)
package main

import (
	"flag"
	"fmt"
	"log"
	"time"
	"os"
	"os/signal"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"

	"github.com/samthor/valuefs/db"
)

var (
	mountPath    = flag.String("mount", "", "mountpoint of filesystem")
	logPath      = flag.String("log", "", "logfile to use, if any")
	memoryValues = flag.Int("values", 100, "number of values to hold in memory")
	writeDelay   = flag.Duration("writeDelay", time.Minute, "delay to write values")
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
	writer := &LogWriter{}
	v := &ValueFS{db.New(config, writer)}
	go signalWait(v.Store)
	go func() {
		for range time.Tick(*writeDelay) {
			v.Store.Prune()
		}
	}()

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

type LogWriter struct{}

func (lw *LogWriter) Load() map[string]*db.Sample {
	return make(map[string]*db.Sample)
}

func (lw *LogWriter) Store(rec *db.Record, sample *db.Sample) error {
	// TODO: Just writes to stdout for now.
	fmt.Printf("%v\t%v\t%v\n", rec.Name, sample.When.Format(time.RFC3339), sample.Value)
	return nil
}
