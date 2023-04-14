// Copyright 2020 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package endec

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"os"
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

func MD5File(filename string, chunkSizes ...int64) (string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := md5.New()
	var chunkSize int64 = 8192
	if len(chunkSizes) != 0 {
		chunkSize = chunkSizes[0]
	}
	buffer := make([]byte, chunkSize)
	for {
		readBytes, err := file.Read(buffer)
		if readBytes == 0 && err == io.EOF {
			break
		} else if err != nil {
			return "", err
		}

		hash.Write(buffer[:readBytes])
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}
