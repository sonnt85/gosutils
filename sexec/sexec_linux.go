//go:build !windows && !darwin && !openbsd && !netbsd && !solaris
// +build !windows,!darwin,!openbsd,!netbsd,!solaris

package sexec

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

// func ExecCommandEnvTimeout(name string, moreenvs map[string]string, timeout time.Duration, arg ...string) (stdOut, stdErr []byte, err error) {
func ExecCommandShellElevated(exe string, showCmd int32, args ...string) (stdOut, stdErr []byte, err error) {
	return ExecCommandShellElevatedEnvTimeout(exe, showCmd, map[string]string{}, 0, args...)
}

func ExecCommandShellElevatedEnvTimeout(name string, showCmd int32, moreenvs map[string]string, timeout time.Duration, args ...string) (stdOut, stdErr []byte, err error) {
	var stdout, stderr bytes.Buffer
	cmd := exec.Command(name, args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{}
	cmd.SysProcAttr.Credential = &syscall.Credential{Uid: 0, Gid: 0}
	if len(moreenvs) != 0 {
		for k, v := range moreenvs {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
		}
	}
	err = cmd.Start()
	if err != nil {
		return stdOut, stdErr, err
	}
	if timeout != 0 {
		ctx, cancelFn := context.WithTimeout(context.Background(), timeout)
		defer cancelFn()
		go func() {
			<-ctx.Done()
			if ctx.Err() == context.DeadlineExceeded {
				cmd.Process.Kill()
			}
		}()
		err = cmd.Wait()
		if ctx.Err() == nil {
			cancelFn()
		}
	} else {
		err = cmd.Wait()
	}
	return stdout.Bytes(), stderr.Bytes(), err
	// return ExecCommand(exe, args...)
}

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
