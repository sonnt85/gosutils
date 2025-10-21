package smerge_test

import (
	"testing"
	"time"

	"github.com/sonnt85/gosutils/smerge"
)

type testStruct struct {
	time.Duration
}

func TestIssue50Merge(t *testing.T) {
	to := testStruct{}
	from := testStruct{}

	if err := smerge.Merge(&to, from); err != nil {
		t.Fail()
	}
}
