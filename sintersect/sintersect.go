package sintersect

import (
	"reflect"
	"sort"

	"github.com/google/go-cmp/cmp"
)

// SlideSimple has complexity: O(n^2)
func SlideSimple(a interface{}, b interface{}) []interface{} {
	set := make([]interface{}, 0)
	av := reflect.ValueOf(a)

	for i := 0; i < av.Len(); i++ {
		el := av.Index(i).Interface()
		if Contains(b, el) {
			set = append(set, el)
		}
	}

	return set
}

// SlideSorted has complexity: O(n * log(n)), a needs to be sorted
func SlideSorted(a interface{}, b interface{}) []interface{} {
	set := make([]interface{}, 0)
	av := reflect.ValueOf(a)
	bv := reflect.ValueOf(b)

	for i := 0; i < av.Len(); i++ {
		el := av.Index(i).Interface()
		idx := sort.Search(bv.Len(), func(i int) bool {
			return bv.Index(i).Interface() == el
		})
		if idx < bv.Len() && bv.Index(idx).Interface() == el {
			set = append(set, el)
		}
	}

	return set
}

// SlideHash has complexity: O(n * x) where x is a factor of hash function efficiency (between 1 and 2)
func SlideHash(a interface{}, b interface{}) []interface{} {
	set := make([]interface{}, 0)
	hash := make(map[interface{}]bool)
	av := reflect.ValueOf(a)
	bv := reflect.ValueOf(b)

	for i := 0; i < av.Len(); i++ {
		el := av.Index(i).Interface()
		hash[el] = true
	}

	for i := 0; i < bv.Len(); i++ {
		el := bv.Index(i).Interface()
		if _, found := hash[el]; found {
			set = append(set, el)
		}
	}

	return set
}

func Contains(a interface{}, e interface{}) bool {
	v := reflect.ValueOf(a)

	for i := 0; i < v.Len(); i++ {
		if cmp.Equal(v.Index(i).Interface(), e) {
			// if v.Index(i).Interface() == e {
			return true
		}
	}
	return false
}
