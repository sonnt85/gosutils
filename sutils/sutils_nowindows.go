//go:build !windows
// +build !windows

package sutils

import (
	"os"
	"syscall"
)

func DirIsWritable(path string) (isWritable bool) {
	isWritable = false
	info, err := os.Stat(path)
	if err != nil {
		return
	}
	if !info.IsDir() {
		return
	}

	// Check if the user bit is enabled in file permission
	if info.Mode().Perm()&(1<<(uint(7))) == 0 {
		//			fmt.Println("Write permission bit is not set on this file for user")
		return
	}
	var stat syscall.Stat_t
	if err = syscall.Stat(path, &stat); err != nil {
		//			fmt.Println("Unable to get stat")
		return
	}

	if uint32(os.Geteuid()) != stat.Uid {
		isWritable = false
		//fmt.Println("User doesn't have permission to write to this directory")
		return
	}
	isWritable = true
	return
}
