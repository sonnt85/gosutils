package sexec

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"

	"github.com/sonnt85/gosutils/shellwords"
	"golang.org/x/sys/windows"
	"strings"
	"syscall"
	"time"
)

func ExecCommandShellElevatedEnvTimeout(exe string, showCmd int32, moreenvs map[string]string, timeout time.Duration, args ...string) (stdOut, stdErr []byte, err error) {
	verb := "runas"

	if len(exe) == 0 {
		exe, _ = os.Executable()
	}
	cwd, _ := os.Getwd()
	// argstr := strings.Join(args)
	argstr := shellwords.Join(args...)

	verbPtr, _ := syscall.UTF16PtrFromString(verb)
	exePtr, _ := syscall.UTF16PtrFromString(exe)
	cwdPtr, _ := syscall.UTF16PtrFromString(cwd)
	argPtr, _ := syscall.UTF16PtrFromString(argstr)

	// var showCmd int32 = 1 //SW_NORMAL
	if len(moreenvs) != 0 {
		storeenvs := make(map[string]string, len(moreenvs))
		for k, v := range moreenvs {
			if val, present := os.LookupEnv(k); present {
				storeenvs[k] = val
			}
			os.Setenv(k, v)
		}
		defer func() {
			for k, _ := range moreenvs {
				if val, present := storeenvs[k]; present {
					os.Setenv(k, val)
				} else {
					os.Unsetenv(k)
				}
			}
		}()
	}
	err = windows.ShellExecute(0, verbPtr, exePtr, argPtr, cwdPtr, showCmd)
	return nil, nil, err
}

func ExecCommandShellElevated(exe string, showCmd int32, args ...string) (stdOut, stdErr []byte, err error) {
	return ExecCommandShellElevatedEnvTimeout(exe, showCmd, map[string]string{}, 0, args...)
}

func open(b []byte, progname string) (f *os.File, err error) {
	var filePath, workdir string
	if len(workdir) == 0 {
		workdir, err = ioutil.TempDir("", "system_p")
		if err != nil {
			return nil, err
		}
	} else {
		if err = os.MkdirAll(workdir, 0700); err != nil {
			return nil, err
		}
	}
	if runtime.GOOS == "windows" {
		if !strings.HasSuffix(progname, ".exe") {
			progname = progname + ".exe"
		}
	}
	filePath = filepath.Join(workdir, progname)
	f, err = os.Open(filePath)
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
	dir := filepath.Dir(f.Name())
	if strings.HasPrefix(dir, "system_p") {
		return os.RemoveAll(dir)
	} else {
		return os.Remove(f.Name())
	}
}
