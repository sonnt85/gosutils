package smerge_test

import (
	"encoding/json"
	"testing"

	"github.com/sonnt85/gosutils/smerge"
)

func TestIssue17MergeWithOverwrite(t *testing.T) {
	var (
		request    = `{"timestamp":null, "name": "foo"}`
		maprequest = map[string]interface{}{
			"timestamp": nil,
			"name":      "foo",
			"newStuff":  "foo",
		}
	)

	var something map[string]interface{}
	if err := json.Unmarshal([]byte(request), &something); err != nil {
		t.Errorf("Error while Unmarshalling maprequest: %s", err)
	}

	if err := smerge.MergeWithOverwrite(&something, maprequest); err != nil {
		t.Errorf("Error while merging: %s", err)
	}
}
