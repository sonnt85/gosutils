//go:build !amd64
// +build !amd64

package malloc

func bsr(x int) int {
	r := 0
	for x != 0 {
		x = x >> 1
		r += 1
	}
	return r - 1
}
