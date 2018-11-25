package mirrorfs

import (
	"os"
	"log"
	"sync"
	"io"
	"path/filepath"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"golang.org/x/net/context"

	
)


var _ fs.Node = (*File)(nil)
var _ fs.NodeOpener = (*File)(nil)
var _ fs.HandleReader = (*File)(nil)
var _ fs.HandleWriter = (*File)(nil)
var _ fs.HandleReleaser = (*File)(nil)


type File struct {
	sync.RWMutex
	attr 	fuse.Attr
	path 	string
	fs   	*MirrorFS
	handler *os.File
} 

func (f *File) Attr(ctx context.Context, o *fuse.Attr) error {
	f.RLock()
	defer f.RUnlock()
	if err := f.readAttr();err != nil {
		return err
	}
	*o = f.attr
	return nil
}

func (f *File) readAttr() error {
	stats, err := os.Stat(f.path)
	if err != nil {
		//The real file does not exists. 
		log.Println("Read attr ERR: ", err, f.path)
		return err
	}
	f.attr.Size = uint64(stats.Size())
	f.attr.Mtime = stats.ModTime()
	f.attr.Mode = stats.Mode()
	// log.Println("Read attr: ", f.attr, f.path)
	return nil
}


func (f *File) Open(ctx context.Context, req *fuse.OpenRequest, resp *fuse.OpenResponse) (fs.Handle, error) {
	log.Println(req, filepath.Base(f.path))
	
	fsHandler, err := os.OpenFile(f.path, int(req.Flags), f.attr.Mode)
	if err != nil {
		log.Print("Open ERR: ", err)
		return nil, err
	}
	f.handler = fsHandler

	// resp.Flags |= fuse.OpenDirectIO

	return f, nil
}

func (f *File) Release(ctx context.Context, req *fuse.ReleaseRequest) error {
	log.Println(req)
	return f.handler.Close()
}


func (f *File) Read(ctx context.Context, req *fuse.ReadRequest, resp *fuse.ReadResponse) error {
	f.RLock()
	defer f.RUnlock()
	log.Println(req, filepath.Base(f.path))

	if f.handler == nil {
		log.Println("Read: File should be open, aborting request")
		return fuse.ENOTSUP
	}

	resp.Data = resp.Data[:req.Size]
	n, err := f.handler.ReadAt(resp.Data, req.Offset)
	if err != nil && err != io.EOF {
		log.Println("Read ERR: ", err) 
  		return err
  	}
	resp.Data = resp.Data[:n]

	return nil
}

func (f *File) Write(ctx context.Context, req *fuse.WriteRequest, resp *fuse.WriteResponse) error {
	f.Lock()
	defer f.Unlock()

	log.Println(req, filepath.Base(f.path))

	if f.handler == nil {
		log.Println("Write: File should be open, aborting request")
		return fuse.ENOTSUP
	}

	n, err := f.handler.WriteAt(req.Data, req.Offset)
	if err != nil {
		log.Println("Write ERR: ", err)
		return err
	}
	resp.Size = n

	return nil
}
