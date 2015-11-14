package mirrorfs

import (
	"os"
	"log"
	"sync"
	"io/ioutil"
	"path/filepath"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"golang.org/x/net/context"

	
)

var _ fs.Node = (*Dir)(nil)
var _ fs.NodeCreater = (*Dir)(nil)
var _ fs.NodeMkdirer = (*Dir)(nil)
var _ fs.NodeRemover = (*Dir)(nil)
var _ fs.NodeRenamer = (*Dir)(nil)
var _ fs.NodeStringLookuper = (*Dir)(nil)


type Dir struct {
	sync.RWMutex
	attr fuse.Attr

	path   string
	fs     *MirrorFS
	parent *Dir
}

func (d *Dir) Attr(ctx context.Context, o *fuse.Attr) error {
	d.RLock()
	*o = d.attr
	d.RUnlock()
	return nil
}

func (d *Dir) Lookup(ctx context.Context, name string) (fs.Node, error) {
	d.RLock()
	defer d.RUnlock()
	// n, exist := d.nodes[name]
	path := filepath.Join(d.path, name)
	stats, err := os.Stat(path)
	if err != nil {
		//The real file does not exists. 
		// log.Println("Lookup ERR: ", err)
		return nil, fuse.ENOENT
	}

	// log.Println("Lookup", path)
	

	switch {
		case stats.IsDir():
			// log.Printf("DRIVE | DIR LOOKUP: %s\n", virtualPath)
			return d.fs.newDir(path, stats.Mode()), nil
		case stats.Mode().IsRegular():
			return d.fs.newFile(path, stats.Mode()), nil
		default:
			panic("Unknown type in filesystem")
	}
}

func (d *Dir) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	d.RLock()
	log.Println("ReadDirAll ", d.path)
	var out []fuse.Dirent
	files, err := ioutil.ReadDir(d.path)
	if err != nil {
		log.Println("ReadDirAll ERR: ", err)
		return nil, err
	}
	
	for _, node := range files {
		de := fuse.Dirent{Name: node.Name()}
		if node.IsDir() {
			de.Type = fuse.DT_Dir
		}
		if node.Mode().IsRegular() {
			de.Type = fuse.DT_File
		}
		log.Println("ReadDirAll file", de)
		out = append(out, de)
	}

	d.RUnlock()
	return out, nil
}

func (d *Dir) Mkdir(ctx context.Context, req *fuse.MkdirRequest) (fs.Node, error) {
	d.Lock()
	defer d.Unlock()
	log.Println(req)
	if exists := d.exists(req.Name); exists {
		log.Println("Mkdir ERR: EEXIST")
		return nil, fuse.EEXIST
	}

	path := filepath.Join(d.path, req.Name)
	n := d.fs.newDir(path, req.Mode)

	if err := os.Mkdir(path,  req.Mode); err != nil{
		log.Println("Mkdir ERR:  ", err)
		return nil, err
	}
	return n, nil
}


func (d *Dir) Create(ctx context.Context, req *fuse.CreateRequest,resp *fuse.CreateResponse) (fs.Node, fs.Handle, error) {
	d.Lock()
	defer d.Unlock()
	log.Println(req)
	if exists := d.exists(req.Name) ; exists {
		log.Println("Create open ERR: EEXIST")
		return nil, nil, fuse.EEXIST
	}
	// log.Println(req.Mode, DEF_MODE, req.Flags)
	path := filepath.Join(d.path, req.Name)
    fHandler, err := os.OpenFile(path, int(req.Flags), req.Mode)
	if err != nil {
		log.Println("Create open ERR: ", err)
		return nil, nil, err
	}

	n := d.fs.newFile(path, req.Mode)
	n.fs = d.fs
	n.handler = fHandler

	resp.Attr = n.attr

	return n, n, nil
}

func (d *Dir) Rename(ctx context.Context, req *fuse.RenameRequest, newDir fs.Node) error {
	nd := newDir.(*Dir)
	log.Println(req)
	if d.attr.Inode == nd.attr.Inode {
		d.Lock()
		defer d.Unlock()
	} else if d.attr.Inode < nd.attr.Inode {
		d.Lock()
		defer d.Unlock()
		nd.Lock()
		defer nd.Unlock()
	} else {
		nd.Lock()
		defer nd.Unlock()
		d.Lock()
		defer d.Unlock()
	}

	if exists := d.exists(req.OldName); !exists {
		log.Println("Rename ERR: ENOENT")
		return fuse.ENOENT
	}

	oldPath := filepath.Join(d.path, req.OldName)
	newPath := filepath.Join(nd.path, req.NewName)

	if err := os.Rename(oldPath, newPath); err != nil{
		log.Println("Rename ERR: ", err)
		return err
	}
	return nil
}

func (d *Dir) Remove(ctx context.Context, req *fuse.RemoveRequest) error {
	d.Lock()
	defer d.Unlock()
	log.Println(req, filepath.Base(d.path), req.Name)

	if exists := d.exists(req.Name); !exists {
		log.Println("Remove ERR: ENOENT")
		return fuse.ENOENT
	} 
	// else if hasChildren() {
	// 	log.Println("Remove ERR: ENOENT")
	// 	return fuse.ENOENT
	// }

	path := filepath.Join(d.path, req.Name)
	if err := os.Remove(path); err != nil{
		log.Println("Remove ERR: ", err)
		return err
	}
	return nil
}

func (d *Dir) exists(name string) (bool) {
	path := filepath.Join(d.path, name)
	_, err := os.Stat(path)
	if err != nil {
		// log.Println("Children " + name + " does not exists")
		return false
	}
	
	return true
}
