//go:build darwin || !linux
// +build darwin !linux

package sexec

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// func open(b []byte, progname string) (f *os.File, err error) {
// []byte, io.Reader
func (f *MemFile) close() error {
	if f.File == nil {
		return nil
	}
	f.File.Close()
	if f.tmpdir != "" {
		return os.RemoveAll(f.tmpdir)
	} else {
		return os.Remove(f.Name())
	}
	// return err
	// fullpathdir := filepath.Dir(f.Name())
	// dirname := filepath.Base(fullpathdir)
	// if strings.HasPrefix(dirname, prefixDirName) {
	// 	return os.RemoveAll(fullpathdir)
	// } else {
	// 	return os.Remove(f.Name())
	// }
	// if err := f.File.Close(); err != nil {
	// 	return err
	// }
	// if f.tmpdir != "" {
	// 	return os.RemoveAll(f.tmpdir)
	// }
	// return nil
	// fullpathdir := filepath.Dir(f.Name())
	// dirname := filepath.Base(fullpathdir)
	// if strings.HasPrefix(dirname, prefixDirName) {
	// 	return os.RemoveAll(fullpathdir)
	// } else {
	// 	return os.Remove(f.Name())
	// }
}
func open(bi interface{}, progname string, workdirs ...string) (mf *MemFile, err error) {
	var r io.Reader
	switch v := bi.(type) {
	case []byte:
		r = bytes.NewBuffer(v)
	case io.Reader:
		r = v
	}
	mf = new(MemFile)
	var filePath, workdir string
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
	if runtime.GOOS == "windows" && len(filepath.Ext(progname)) == 0 {
		progname = progname + ".exe"
	}
	filePath = filepath.Join(workdir, progname)
	var f *os.File
	f, err = os.Create(filePath)
	if err != nil {
		return nil, err
	}
	mf.File = f
	defer func() {
		if err != nil {
			_ = mf.close()
		}
	}()
	if err = os.Chmod(f.Name(), 0o500); err != nil {
		return nil, err
	}

	if _, err = io.Copy(f, r); err != nil {
		// if _, err := f.Write(b); err != nil {
		return nil, err
	} else {
		if _, err = f.Seek(0, 0); err != nil {
			return nil, err
		}
	}
	// if err = f.Close(); err != nil {
	// 	return nil, err
	// }
	return mf, nil

	// return f, nil
}

func clean(f *os.File) error {
	// fmt.Println(f.Name())
	f.Close()
	fullpathdir := filepath.Dir(f.Name())
	dirname := filepath.Base(fullpathdir)
	if strings.HasPrefix(dirname, prefixDirName) {
		return os.RemoveAll(fullpathdir)
	} else {
		return os.Remove(f.Name())
	}
}

func openMemFd(b interface{}, name string) (*MemFile, error) {
	return nil, fmt.Errorf("not support")
}
