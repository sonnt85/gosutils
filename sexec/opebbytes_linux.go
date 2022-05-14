//go:build !darwin
// +build !darwin

package sexec

import (
	"fmt"
	"os"

	"golang.org/x/sys/unix"
)

func open(b []byte, name string) (*os.File, error) {
	fd, err := unix.MemfdCreate("", unix.MFD_CLOEXEC)
	if err != nil {
		return nil, err
	}
	// fmt.Println("fd: ", fd)
	// if len(name) == 0 {
	name = fmt.Sprintf("/proc/self/fd/%d", fd)
	// }
	f := os.NewFile(uintptr(fd), name)
	if _, err := f.Write(b); err != nil {
		_ = f.Close()
		return nil, err
	}
	return f, nil
}

func clean(f *os.File) error {
	return f.Close()
}
