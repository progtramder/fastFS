package fastFS

import (
	"sync"
)

type Pool struct {
	p *sync.Pool
}

// NewPool constructs a new memory Pool for memFile type.
func NewPool() Pool {
	return Pool{
		p: &sync.Pool{
			New: func() interface{} {
				return &memFile{}
			},
		},
	}
}

func (p Pool) Get() *memFile {
	f := p.p.Get().(*memFile)
	return f
}

func (p Pool) put(f *memFile) {
	p.p.Put(f)
}

var (
	_memPool = NewPool()
)
