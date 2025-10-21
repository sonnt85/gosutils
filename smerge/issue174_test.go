package smerge_test

import (
	"testing"

	"github.com/sonnt85/gosutils/smerge"
)

type structWithBlankField struct {
	_ struct{}
	A struct{}
}

func TestIssue174(t *testing.T) {
	dst := structWithBlankField{}
	src := structWithBlankField{}

	if err := smerge.Merge(&dst, src, smerge.WithOverride); err != nil {
		t.Error(err)
	}
}
