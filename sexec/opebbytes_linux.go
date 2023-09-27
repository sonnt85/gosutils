//go:build !darwin
// +build !darwin

package sexec

import (
	"fmt"
	"golang.org/x/sys/unix"
	"os"
	"syscall"
)

func open(b []byte, name string) (*os.File, error) {
	fd, err := unix.MemfdCreate("", unix.MFD_CLOEXEC)
	if err != nil {
		return nil, err
	}
	// if len(name) == 0 {
	filePath := fmt.Sprintf("/proc/%d/fd/%d", os.Getpid(), fd)
	// }
	f := os.NewFile(uintptr(fd), filePath)
	if _, err := f.Write(b); err != nil {
		_ = f.Close()
		return nil, err
	}
	return f, nil
}

func clean(f *os.File) error {
	return f.Close()
}

func openMemFd(b []byte, name string) (*os.File, error) {
	return open(b, name)
}

func readMemfdFile(fd int) ([]byte, error) {
	const bufferSize = 4096

	// file := os.NewFile(uintptr(fd), "memfd") // Tạo đối tượng *os.File từ file descriptor

	buffer := make([]byte, bufferSize)
	var data []byte

	for {
		n, err := syscall.Read(fd, buffer)
		if err != nil {
			return nil, err
		}
		if n == 0 {
			break
		}

		data = append(data, buffer[:n]...)
	}

	return data, nil
}
