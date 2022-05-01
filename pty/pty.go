// +build !windows

package pty

import (
	"os"
	"os/exec"
	"syscall"
	"unsafe"

	opty "github.com/creack/pty"
)

func SetWinsizeTerminal(f *os.File, w, h int) {
	syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), uintptr(syscall.TIOCSWINSZ),
		uintptr(unsafe.Pointer(&struct{ h, w, x, y uint16 }{uint16(h), uint16(w), 0, 0})))
	return
}
func Start(c *exec.Cmd) (pty *os.File, err error) {
	return opty.Start(c)
}
