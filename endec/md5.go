// Copyright 2020 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package endec

import (
	"crypto/md5"
	"encoding/hex"
)

// MD5 encodes string to hexadecimal of MD5 checksum.
func MD5(data []byte) string {
	return hex.EncodeToString(MD5Bytes(data))
}

// MD5Bytes encodes string to MD5 checksum.
func MD5Bytes(data []byte) []byte {
	m := md5.New()
	_, _ = m.Write([]byte(data))
	return m.Sum(nil)
}
