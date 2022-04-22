package malloc

import (
	"sync"
)

const maxSize = 64

// index contains []byte which cap is 1<<index
var caches [maxSize]sync.Pool

func init() {
	for i := 0; i < maxSize; i++ {
		size := 1 << i
		caches[i].New = func() interface{} {
			buf := make([]byte, 0, size)
			return buf
		}
	}
}

// calculates which pool to get from
func calcIndex(size int) int {
	if size == 0 {
		return 0
	}
	if isPowerOfTwo(size) {
		return bsr(size)
	}
	return bsr(size) + 1
}

// Malloc supports one or two integer argument.
// The size specifies the length of the returned slice, which means len(ret) == size.
// A second integer argument may be provided to specify the minimum capacity, which means cap(ret) >= cap.
func Malloc(size int, capacity ...int) []byte {
	if len(capacity) > 1 {
		panic("too many arguments to Malloc")
	}
	var c = size
	if len(capacity) > 0 && capacity[0] > size {
		c = capacity[0]
	}
	var ret = caches[calcIndex(c)].Get().([]byte)
	ret = ret[:size]
	return ret
}

// Free should be called when the buf is no longer used.
func Free(buf []byte) {
	size := cap(buf)
	if !isPowerOfTwo(size) {
		return
	}
	buf = buf[:0]
	caches[bsr(size)].Put(buf)
}

func isPowerOfTwo(x int) bool {
	return (x & (-x)) == x
}
