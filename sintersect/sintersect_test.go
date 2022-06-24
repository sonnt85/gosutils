package sintersect

import (
	"testing"

	"github.com/go-playground/assert"
)

func TestSimple(t *testing.T) {
	s := SlideSimple([]int{1}, []int{2})
	assert.Equal(t, len(s), 0)
	assert.Equal(t, s, []interface{}{})

	s = SlideSimple([]int{1, 2}, []int{2})
	assert.Equal(t, s, []interface{}{2})
}

func TestSorted(t *testing.T) {
	s := SlideSorted([]int{1}, []int{2})
	assert.Equal(t, len(s), 0)
	assert.Equal(t, s, []interface{}{})

	s = SlideSorted([]int{1, 2}, []int{2})
	assert.Equal(t, s, []interface{}{2})
}

func TestHash(t *testing.T) {
	s := SlideHash([]int{1}, []int{2})
	assert.Equal(t, len(s), 0)
	assert.Equal(t, s, []interface{}{})

	s = SlideHash([]int{1, 2}, []int{2})
	assert.Equal(t, s, []interface{}{2})
}
