//go:build !windows
// +build !windows

package endec

import (
	"fmt"
	"io/fs"
	"syscall"
)

func getFileOwnership(info fs.FileInfo) (uint32, uint32, error) {
	if s, ok := info.Sys().(syscall.Stat_t); ok { // && s != nil && s.(*syscall.Stat_t) != nil { // && 		info.Sys().(*syscall.Stat_t)
		return s.Uid, s.Gid, nil
	} else {
		return 0, 0, fmt.Errorf("Unsupported system type")
	}
}
