//go:build !darwin
// +build !darwin

package sexec

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"golang.org/x/sys/unix"
)

// []byte, io.Reader
func open(bi interface{}, name string, workdirs ...string) (mf *MemFile, err error) {
	// memfs, err := syscall.Mount("memfs", "none", "memfs", 0)
	// var b []bytes
	var r io.Reader
	switch v := bi.(type) {
	case []byte:
		r = bytes.NewBuffer(v)
	case string:
		r = bytes.NewBuffer([]byte(v))
	case io.Reader:
		r = v
	}
	var fd int
	mf = new(MemFile)
	fd, err = unix.MemfdCreate("", unix.MFD_CLOEXEC) // unix.MFD_CLOEXEC unix.MFD_EXEC
	if err != nil {
		return nil, err
	}

	filePath := fmt.Sprintf("/proc/%d/fd/%d", os.Getpid(), fd)
	defer func() {
		if err != nil {
			_ = mf.close()
		}
	}()
	if len(name) != 0 {
		var workdir string
		if len(workdirs) == 0 {
			workdir, err = os.MkdirTemp("", prefixDirName)
			if err != nil {
				return nil, err
			}
			mf.tmpdir = workdir
		} else {
			workdir = workdirs[0]
			if err = os.MkdirAll(workdir, 0700); err != nil {
				return nil, err
			}
		}

		if err = os.Symlink(filePath, filepath.Join(workdir, name)); err == nil {
			// if err = os.Rename(filePath, fmt.Sprintf("/proc/%d/fd/%s", os.Getpid(), name)); err == nil {
			filePath = filepath.Join(workdir, name)
		}
	}
	// err =
	unix.Fchmod(fd, 0755)
	// if err != nil {
	// 	return nil, err
	// }
	f := os.NewFile(uintptr(fd), filePath)
	mf.File = f
	if _, err = io.Copy(f, r); err != nil {
		return nil, err
	} else {

		if _, err = f.Seek(0, 0); err != nil {
			return nil, err
		}
	}
	return mf, nil
}

func (f *MemFile) close() error {
	// unix.Close(12)
	if f.File == nil {
		return nil
	}
	err := f.File.Close()

	if f.tmpdir != "" {
		return os.RemoveAll(f.tmpdir)
	}
	return err
	// fullpathdir := filepath.Dir(f.Name())
	// dirname := filepath.Base(fullpathdir)
	// if strings.HasPrefix(dirname, prefixDirName) {
	// 	return os.RemoveAll(fullpathdir)
	// } else {
	// 	return os.Remove(f.Name())
	// }
}

func clean(f *os.File) error {
	if err := f.Close(); err != nil {
		return err
	}
	fullpathdir := filepath.Dir(f.Name())
	dirname := filepath.Base(fullpathdir)
	if strings.HasPrefix(dirname, prefixDirName) {
		return os.RemoveAll(fullpathdir)
	} else {
		return os.Remove(f.Name())
	}
}

func openMemFd(b interface{}, name string) (*MemFile, error) {
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
