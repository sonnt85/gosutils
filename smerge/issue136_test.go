package smerge_test

import (
	"testing"

	"github.com/sonnt85/gosutils/smerge"
)

type embeddedTestA struct {
	Name string
	Age  uint8
}

type embeddedTestB struct {
	embeddedTestA
	Address string
}

func TestMergeEmbedded(t *testing.T) {
	var (
		err error
		a   = &embeddedTestA{
			"Suwon", 16,
		}
		b = &embeddedTestB{}
	)

	if err := smerge.Merge(&b.embeddedTestA, *a); err != nil {
		t.Error(err)
	}

	if b.Name != "Suwon" {
		t.Errorf("%v %v", b.Name, err)
	}
}
