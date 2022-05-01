//go:build windows
// +build windows

package sutils

import (
	// "golang.org/x/sys/windows"
	"os"
	// "strings"
	"syscall"
	"unsafe"
)

func DirIsWritable(path string) (isWritable bool) {
	isWritable = false
	info, err := os.Stat(path)
	if err != nil {
		//		fmt.Println("Path doesn't exist")
		return
	}
	if !info.IsDir() {
		//		fmt.Println("Path isn't a directory")
		return
	}

	// Check if the user bit is enabled in file permission
	if info.Mode().Perm()&(1<<(uint(7))) == 0 {
		//			fmt.Println("Write permission bit is not set on this file for user")
		return
	}
	isWritable = true
	return
}

func IsDoubleClickRun() bool {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	lp := kernel32.NewProc("GetConsoleProcessList")
	if lp != nil {
		var pids [2]uint32
		var maxCount uint32 = 2
		ret, _, _ := lp.Call(uintptr(unsafe.Pointer(&pids)), uintptr(maxCount))
		if ret > 1 {
			return false
		}
	}
	return true
}
