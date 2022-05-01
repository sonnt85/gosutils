// Copyright 2020 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package endec

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMD5(t *testing.T) {
	tests := []struct {
		input  []byte
		output string
	}{
		{input: []byte(""), output: "d41d8cd98f00b204e9800998ecf8427e"},
		{input: []byte("The quick brown fox jumps over the lazy dog"), output: "9e107d9d372bb6826bd81d3542a419d6"},
		{input: []byte("The quick brown fox jumps over the lazy dog."), output: "e4d909c290d0fb1ca068ffaddf22cbd0"},
	}
	for _, test := range tests {
		t.Run(string(test.input), func(t *testing.T) {
			assert.Equal(t, test.output, MD5(test.input))
		})
	}
}
