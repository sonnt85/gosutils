//go:build windows
// +build windows

package endec

import (
	"fmt"
	"io/fs"
)

func getFileOwnership(info fs.FileInfo) (uint32, uint32, error) {
	return 0, 0, fmt.Errorf("not implemented")
}
