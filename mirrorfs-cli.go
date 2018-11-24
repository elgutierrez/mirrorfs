package main

import (
	// "io"
	"os"
	"fmt"
	"log"
	"flag"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	// "golang.org/x/net/context"
	"github.com/elgutierrez/mirrorfs/fs"
	
)

// debug flag enables logging of debug messages to stderr.
var debug = flag.Bool("debug", false, "enable debug log messages to stderr")
var mirror = flag.String("mirror", "", "path to mirror contents (required)")
var mount = flag.String("mount", "", "path to mount volume (required)")

func usage() {
	fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
	flag.PrintDefaults()
}

func debugLog(msg interface{}) {
	fmt.Printf("%s", msg)
}

func main() {
	flag.Usage = usage
	flag.Parse()

	if *mount == "" ||  *mirror == "" {
		usage()
		os.Exit(2)
	}

	c, err := fuse.Mount(
		*mount,
		fuse.FSName("mirrorfs"),
		fuse.Subtype("mirrorfs"),
		fuse.VolumeName("Mirror FS"),
		// fuse.LocalVolume(),
		fuse.AllowOther(),
	)

	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	cfg := &fs.Config{}
	if *debug {
		cfg.Debug = debugLog
	}
	srv := fs.New(c, cfg)
	filesys := mirrorfs.NewMirrorFS(*mirror)

	if err := srv.Serve(filesys); err != nil {
		log.Fatal(err)
	}

	// Check if the mount process has an error to report.
	<-c.Ready
	if err := c.MountError; err != nil {
		log.Fatal(err)
	}
}
