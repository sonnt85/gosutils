package smerge_test

import (
	"testing"

	"github.com/sonnt85/gosutils/smerge"
)

type foz struct {
	A *bool
	B string
}

func TestIssue131MergeWithOverwriteWithEmptyValue(t *testing.T) {
	src := foz{
		A: func(v bool) *bool { return &v }(false),
		B: "src",
	}
	dest := foz{
		A: func(v bool) *bool { return &v }(true),
		B: "dest",
	}
	if err := smerge.Merge(&dest, src, smerge.WithOverwriteWithEmptyValue); err != nil {
		t.Error(err)
	}
	if *src.A != *dest.A {
		t.Errorf("dest.A not merged in properly: %v != %v", *src.A, *dest.A)
	}
	if src.B != dest.B {
		t.Errorf("dest.B not merged in properly: %v != %v", src.B, dest.B)
	}
}
