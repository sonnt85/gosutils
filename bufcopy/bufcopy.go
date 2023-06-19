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

func (b *BufCopy) Copy(dst io.Writer, src io.Reader) (written int64, err error) {
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
	defer b.Put(buf)
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
