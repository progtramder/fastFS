package fastFS

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"sync"
)

const (
	KB = 1024
	MB = 1024*KB
	GB = 1024*MB
)

const __defaultPoolSize = 128

var (
	errNotFound = errors.New("not found")
	errNotDir = errors.New("not directory")
	errInvalidOffset = errors.New("invalid seek offset")
	errInvalidWhence = errors.New("invalid seek whence")
)

type FileSystem struct {
	//
	// Root directory of file server
	//
	Root       string

	//
	// Memory quota for cache
	//
	MemQuota   int64

	//
	// Files can be cached if size is below CacheLimit and memory
	// quota is not used up
	//
	CacheLimit int64

	muFP sync.Mutex
	filePool map[string]*fileStack

	muMFP sync.RWMutex
	memFilePool map[string]*memFile

	//
	// Memory used for cache
	//
	memUsed   int64

	//
	// anonymous mutex for accessing memUsed
	//
	sync.Mutex
}

type fileStack struct {
	files []*file
}

func newFileStack() *fileStack {
	return &fileStack{
		make([]*file, 0, __defaultPoolSize),
	}
}

func (self *fileStack) push(f *file) {
	self.files = append(self.files, f)
}

func (self *fileStack) pop() *file {
	size := len(self.files)
	if size == 0 {
		return nil
	}

	f := self.files[size-1]
	self.files = self.files[0:size-1]
	return f
}

func (fs *FileSystem) getFile(name string) (http.File, error) {
	fs.muFP.Lock()
	defer fs.muFP.Unlock()

	files, ok := fs.filePool[name]
	if ok {
		f := files.pop()
		if f != nil {
			f.Seek(0, io.SeekStart)
			return f, nil
		}
	}

	return nil, errNotFound
}

func (fs *FileSystem) putFile(name string, f *file) error {
	fs.muFP.Lock()
	defer fs.muFP.Unlock()

	if fs.filePool == nil {
		fs.filePool = make(map[string]*fileStack, __defaultPoolSize)
	}

	fStack, ok := fs.filePool[name]
	if !ok {
		fStack = newFileStack()
		fs.filePool[name] = fStack
	}

	fStack.push(f)
	return nil
}

func (fs *FileSystem) getMemFile(name string) (http.File, error) {
	fs.muMFP.RLock()
	defer fs.muMFP.RUnlock()

	f, ok := fs.memFilePool[name]
	if ok {
		memF := _memPool.Get()
		*memF = *f
		return memF, nil
	}

	return nil, errNotFound
}

func (fs *FileSystem) putMemFile(name string, f *memFile) error {

	fs.muMFP.Lock()
	defer fs.muMFP.Unlock()
	if fs.memFilePool == nil {
		fs.memFilePool = make(map[string]*memFile, __defaultPoolSize)
	}

	fs.memFilePool[name] = f

	return nil
}

func (fs *FileSystem) Open(name string) (http.File, error) {

	f, err := fs.getMemFile(name)
	if err == nil {
		return f, nil
	}

	f, err = fs.getFile(name)
	if err == nil {
		return f, nil
	}

	//
	// Cant find in pool, open the file
	//
	f, err = http.Dir(fs.Root).Open(name)
	if err != nil {
		return nil, err
	}

	fInfo, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, err
	}

	//
	// if it is a directory or memory quota used up or file size is greater than CacheLimit,
	// then wrapper into file{}
	//
	fs.Lock()
	defer fs.Unlock()
	if fInfo.IsDir() || fInfo.Size() > fs.CacheLimit || fs.memUsed > fs.MemQuota {
		return &file{f, name, fs}, nil
	}

	//
	// Otherwise wrapper the file into teeFile
	//

	fs.memUsed += fInfo.Size()
	return &teeFile {
		f,
		name,
		fs,
		bytes.NewBuffer(make([]byte, 0, fInfo.Size())),
	}, nil
}
