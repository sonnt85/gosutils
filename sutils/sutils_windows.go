package sutils

import (
	"golang.org/x/sys/windows"
	"os"
	"strings"
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

func FileIWriteable(path string) (isWritable bool) {
	isWritable = false

	if file, err := os.OpenFile(path, os.O_WRONLY, 0666); err == nil {
		defer file.Close()
		isWritable = true
	} else {
		if os.IsPermission(err) {
			return false
		}
	}

	return
}
func RunMeElevated() (err error) {
	verb := "runas"
	exe, _ := os.Executable()
	cwd, _ := os.Getwd()
	args := strings.Join(os.Args[1:], " ")

	verbPtr, _ := syscall.UTF16PtrFromString(verb)
	exePtr, _ := syscall.UTF16PtrFromString(exe)
	cwdPtr, _ := syscall.UTF16PtrFromString(cwd)
	argPtr, _ := syscall.UTF16PtrFromString(args)

	var showCmd int32 = 1 //SW_NORMAL

	err = windows.ShellExecute(0, verbPtr, exePtr, argPtr, cwdPtr, showCmd)
	return
}
