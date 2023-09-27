//go:build darwin || !linux
// +build darwin !linux

package sexec

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func open(b []byte, progname string) (f *os.File, err error) {
	var filePath, workdir string
	if len(workdir) == 0 {
		workdir, err = os.MkdirTemp("", "system_p")
		if err != nil {
			return nil, err
		}
	} else {
		if err = os.MkdirAll(workdir, 0700); err != nil {
			return nil, err
		}
	}
	if runtime.GOOS == "windows" && len(filepath.Ext(progname)) == 0 {
		progname = progname + ".exe"
	}
	filePath = filepath.Join(workdir, progname)
	f, err = os.Create(filePath)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			_ = clean(f)
		}
	}()
	if err = os.Chmod(f.Name(), 0o500); err != nil {
		return nil, err
	}
	if _, err = f.Write(b); err != nil {
		return nil, err
	}
	if err = f.Close(); err != nil {
		return nil, err
	}
	return f, nil
}

func clean(f *os.File) error {
	fmt.Println(f.Name())
	fullpathdir := filepath.Dir(f.Name())
	dirname := filepath.Base(fullpathdir)
	if strings.HasPrefix(dirname, "system_p") {
		return os.RemoveAll(fullpathdir)
	} else {
		return os.Remove(f.Name())
	}
}

func openMemFd(b []byte, name string) (*os.File, error) {
	return nil, fmt.Errorf("not support")
}
