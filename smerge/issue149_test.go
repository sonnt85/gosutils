package smerge_test

import (
	"testing"

	"github.com/sonnt85/gosutils/smerge"
)

type user struct {
	Name string
}

type token struct {
	User  *user
	Token *string
}

func TestIssue149(t *testing.T) {
	dest := &token{
		User: &user{
			Name: "destination",
		},
		Token: nil,
	}
	tokenValue := "Issue149"
	src := &token{
		User:  nil,
		Token: &tokenValue,
	}
	if err := smerge.Merge(dest, src, smerge.WithOverwriteWithEmptyValue); err != nil {
		t.Error(err)
	}
	if dest.User != nil {
		t.Errorf("expected nil User, got %q", dest.User)
	}
	if dest.Token == nil {
		t.Errorf("expected not nil Token, got %q", *dest.Token)
	}
}
