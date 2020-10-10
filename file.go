package fastFS

import (
	"bytes"
	"io"
	"net/http"
	"os"
)

type file struct {
	http.File
	name string
	fs *FileSystem
}

func (f *file) Close() error {
	return f.fs.putFile(f.name, f)
}

type memFile struct {
	name string
	fs *FileSystem
	fileInfo os.FileInfo
	content []byte
	rIndex  int
}

func (f *memFile) Close() error {
	_memPool.put(f)
	return nil
}

func (f *memFile) Read(p []byte) (n int, err error) {
	if f.rIndex >= len(f.content) {
		return 0, io.EOF
	}

	if p == nil {
		return 0, nil
	}

	n = len(p)
	if n == 0 {
		return 0, nil
	}

	remain := len(f.content) - f.rIndex
	if n > remain {
		n = remain
	}

	copySlice := f.content[f.rIndex:f.rIndex + n]
	copy(p, copySlice)
	f.rIndex += n
	return n, nil
}

func (f *memFile) Seek(offset int64, whence int) (int64, error) {
	if offset < 0 {
		return 0, errInvalidOffset
	}

	size := int64(len(f.content))

	if whence == io.SeekStart {
		if offset < 0 {
			return 0, errInvalidOffset
		}
		if offset > size {
			offset = size
		}
		f.rIndex = int(offset)
		return offset, nil
	} else if whence == io.SeekCurrent {
		index := offset + int64(f.rIndex)
		if index < 0 {
			return 0, errInvalidWhence
		}
		if index > size {
			index = size
		}
		f.rIndex = int(index)
		return index, nil
	} else if whence == io.SeekEnd {
		if offset >= 0 {
			return size, nil
		}

		index := size + offset
		if index < 0 {
			return 0, errInvalidWhence
		} else {
			f.rIndex = int(index)
			return index, nil
		}
	} else {
		return 0, errInvalidWhence
	}
}

func (f *memFile) Readdir(count int) ([]os.FileInfo, error) {
	return nil, errNotDir
}

func (f memFile) Stat() (os.FileInfo, error) {
	return f.fileInfo, nil
}

type teeFile struct {
	http.File
	name string
	fs   *FileSystem
	buf  *bytes.Buffer
}

func (f *teeFile) Close() error {
	defer f.File.Close()
	_, err := f.File.Seek(0, io.SeekStart)
	if err != nil {
		return err
	}
	n, err := io.Copy(f.buf, f.File)
	if err != nil {
		return err
	}

	info, _ := f.File.Stat()
	if n != info.Size() {
		return nil
	}

	return f.fs.putMemFile(f.name, &memFile{
		f.name,
		f.fs,
		info,
		f.buf.Bytes(),
		0,
	})
}
