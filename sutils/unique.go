// Package unique provides primitives for finding unique elements of types that
// implement sort.Interface.
package sutils

import (
	"sort"

	"golang.org/x/exp/constraints"
)

// Types that implement unique.Interface can have duplicate elements removed by
// the functionality in this package.
type Interface interface {
	sort.Interface

	// Truncate reduces the length to the first n elements.
	Truncate(n int)
}

// UniqueSorted removes duplicate elements from data. It assumes sort.IsSorted(data).
func UniqueSorted(data Interface) {
	data.Truncate(ToFront(data))
}

// ToFront reports the number of unique elements of data which it moves to the
// first n positions. It assumes sort.IsSorted(data).
func ToFront(data sort.Interface) (n int) {
	n = data.Len()
	if n == 0 {
		return
	}
	k := 0
	for i := 1; i < n; i++ {
		if data.Less(k, i) {
			k++
			data.Swap(k, i)
		}
	}
	return k + 1
}

// SortThenUnique sorts and removes duplicate entries from data.
func SortThenUnique(data Interface) {
	sort.Sort(data)
	UniqueSorted(data)
}

// SortThenUnique sorts and removes duplicate entries from data.
func Sort(data Interface) {
	sort.Sort(data)
}

// IsUniqued reports whether the elements in data are sorted and unique.
func IsUniqued(data sort.Interface) bool {
	n := data.Len()
	for i := n - 1; i > 0; i-- {
		if !data.Less(i-1, i) {
			return false
		}
	}
	return true
}

type SlideSortable[T constraints.Ordered] struct {
	P *[]T
}

func (p SlideSortable[T]) Len() int           { return len(*p.P) }
func (p SlideSortable[T]) Swap(i, j int)      { (*p.P)[i], (*p.P)[j] = (*p.P)[j], (*p.P)[i] }
func (p SlideSortable[T]) Less(i, j int) bool { return (*p.P)[i] < (*p.P)[j] }
func (p SlideSortable[T]) Truncate(n int)     { *p.P = (*p.P)[:n] }

func (p SlideSortable[T]) Sort() SlideSortable[T]   { Sort(p); return p }
func (p SlideSortable[T]) Unique() SlideSortable[T] { SortThenUnique(p); return p }
func (p SlideSortable[T]) AreUnique() bool          { return IsUniqued(p) }

func UniqueSlide[T constraints.Ordered](a *[]T) { SortThenUnique(SlideSortable[T]{a}) }

// SlideAreUnique tests whether a slice of strings is sorted and its elements
// are unique.
func SlideAreUnique[T constraints.Ordered](a *[]T) bool { return IsUniqued(SlideSortable[T]{a}) }
func SlideSort[T constraints.Ordered](a *[]T)           { Sort(SlideSortable[T]{a}) }

func UniqueFloat64s(a *[]float64)         { UniqueSlide[float64](a) }
func Float64sAreUnique(a *[]float64) bool { return SlideAreUnique[float64](a) }
func Float64sSort(a *[]float64)           { SlideSort[float64](a) }

func UniqueInts(a *[]int)         { UniqueSlide[int](a) }
func IntsAreUnique(a *[]int) bool { return SlideAreUnique[int](a) }
func IntsSort(a *[]int)           { SlideSort[int](a) }

func UniqueInt64s(a *[]int64)         { UniqueSlide[int64](a) }
func Int64sAreUnique(a *[]int64) bool { return SlideAreUnique[int64](a) }
func Int64sSort(a *[]int64)           { SlideSort[int64](a) }

func UniqueStrings(a *[]string)         { UniqueSlide[string](a) }
func StringsAreUnique(a *[]string) bool { return SlideAreUnique[string](a) }
func StringsSort(a *[]string)           { SlideSort[string](a) }
