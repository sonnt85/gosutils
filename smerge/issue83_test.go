package smerge_test

import (
	"testing"

	"github.com/sonnt85/gosutils/smerge"
)

type issue83My struct {
	Data []int
}

func TestIssue83(t *testing.T) {
	dst := issue83My{Data: []int{1, 2, 3}}
	new := issue83My{}
	if err := smerge.Merge(&dst, new, smerge.WithOverwriteWithEmptyValue); err != nil {
		t.Error(err)
	}
	if len(dst.Data) > 0 {
		t.Errorf("expected empty slice, got %v", dst.Data)
	}
}
