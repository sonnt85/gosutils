package smerge_test

import (
	"fmt"
	"testing"

	"github.com/sonnt85/gosutils/smerge"
)

func TestIssue143(t *testing.T) {
	testCases := []struct {
		options  []func(*smerge.Config)
		expected func(map[string]interface{}) error
	}{
		{
			options: []func(*smerge.Config){smerge.WithOverride},
			expected: func(m map[string]interface{}) error {
				properties := m["properties"].(map[string]interface{})
				if properties["field1"] != "wrong" {
					return fmt.Errorf("expected %q, got %v", "wrong", properties["field1"])
				}
				return nil
			},
		},
		{
			options: []func(*smerge.Config){},
			expected: func(m map[string]interface{}) error {
				properties := m["properties"].(map[string]interface{})
				if properties["field1"] == "wrong" {
					return fmt.Errorf("expected a map, got %v", "wrong")
				}
				return nil
			},
		},
	}
	for _, tC := range testCases {
		base := map[string]interface{}{
			"properties": map[string]interface{}{
				"field1": map[string]interface{}{
					"type": "text",
				},
			},
		}

		err := smerge.Map(
			&base,
			map[string]interface{}{
				"properties": map[string]interface{}{
					"field1": "wrong",
				},
			},
			tC.options...,
		)
		if err != nil {
			t.Error(err)
		}
		if err := tC.expected(base); err != nil {
			t.Error(err)
		}
	}
}
