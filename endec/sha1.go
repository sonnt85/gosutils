// Copyright 2020 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package endec

import (
	"crypto/sha1"
	"encoding/hex"
)

// SHA1 encodes string to hexadecimal of SHA1 checksum.
func SHA1(data []byte) string {
	h := sha1.New()
	_, _ = h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}
