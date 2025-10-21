package smerge_test

import (
	"testing"

	"github.com/sonnt85/gosutils/smerge"
)

type issue100s struct {
	Member interface{}
}

func TestIssue100(t *testing.T) {
	m := make(map[string]interface{})
	m["Member"] = "anything"

	st := &issue100s{}
	if err := smerge.Map(st, m); err != nil {
		t.Error(err)
	}
}
