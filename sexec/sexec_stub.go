//go:build (!windows && !darwin && !linux) || openbsd || netbsd
// +build !windows,!darwin,!linux openbsd netbsd

package sexec

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"
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
	return nil, errors.New("not suport")
}

func clean(f *os.File) error {
	return f.Close()
}
