package bufcopy

import (
	"io"
	"time"

	"github.com/sonnt85/goramcache"
)

type BufCopy struct {
	*goramcache.Pool[*[]byte]
}

func New(default_size ...int) *BufCopy {
	size := 32 * 1024 // large objects(> 32 kB) are allocated straight from the heap
	if len(default_size) != 0 {
		size = default_size[0]
	}
	p := goramcache.NewPool(1024*10, time.Minute, func() *[]byte {
		nb := make([]byte, size)
		return &nb
	})
	return &BufCopy{p}
}

func (b *BufCopy) Copy(dst io.Writer, src io.Reader, checkCloser ...bool) (written int64, err error) {
	// If the reader has a WriteTo method, use it to do the copy.
	// Avoids an allocation and a copy.
	if wt, ok := src.(io.WriterTo); ok {
		return wt.WriteTo(dst)
	}
	// Similarly, if the writer has a ReadFrom method, use it to do the copy.
	if rt, ok := dst.(io.ReaderFrom); ok {
		return rt.ReadFrom(src)
	}

	buf := b.Get()
	defer func() {
		b.Put(buf)
		if len(checkCloser) != 0 && checkCloser[0] {
			if c, ok := src.(io.Closer); ok {
				c.Close()
			}
			if c, ok := dst.(io.Closer); ok {
				c.Close()
			}
		}
	}()
	var nr, nw int
	var er, ew error
	for {
		nr, er = src.Read(*buf)
		if nr > 0 {
			nw, ew = dst.Write((*buf)[0:nr])
			if nw > 0 {
				written += int64(nw)
			}

			if ew != nil {
				err = ew
				break
			}

			if nr != nw {
				err = io.ErrShortWrite
				break
			}
		}

		if er == io.EOF {
			break
		}

		if er != nil {
			err = er
			break
		}
	}
	return
}

var _bufcopy *BufCopy

func Copy(dst io.Writer, src io.Reader, checkCloser ...bool) (written int64, err error) {
	if _bufcopy == nil {
		_bufcopy = New()
	}
	return _bufcopy.Copy(dst, src, checkCloser...)
}

func Copy2Way(rw1 io.ReadWriter, rw2 io.ReadWriter, checkCloser ...bool) (written int64, err error) {
	if _bufcopy == nil {
		_bufcopy = New()
	}
	return _bufcopy.Copy2Way(rw1, rw2, checkCloser...)
}

func (b *BufCopy) Copy2Way(rw1 io.ReadWriter, rw2 io.ReadWriter, checkCloser ...bool) (written int64, err error) {
	var n1, n2 int64
	var err1, err2 error
	errorChannel := make(chan error, 1)
	defer func() {
		if len(checkCloser) != 0 && checkCloser[0] {
			if c, ok := rw1.(io.Closer); ok {
				c.Close()
			}
			if c, ok := rw2.(io.Closer); ok {
				c.Close()
			}
		}
	}()
	go func() {
		n1, err1 = b.Copy(rw1, rw2)
		errorChannel <- err1
	}()
	go func() {
		n2, err2 = b.Copy(rw2, rw1)
		errorChannel <- err2
	}()
	err = <-errorChannel
	return n1 + n2, err
}
