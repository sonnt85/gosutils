//go:build !windows && !solaris && !aix
// +build !windows,!solaris,!aix

package pty

import (
	"os"
	"syscall"
	"unsafe"
)

func setWinsizeTerminal(f *os.File, w, h int) (err error) {
	_, _, err = syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), uintptr(syscall.TIOCSWINSZ),
		uintptr(unsafe.Pointer(&struct{ h, w, x, y uint16 }{uint16(h), uint16(w), 0, 0})))
	return
}
