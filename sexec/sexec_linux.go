//go:build !darwin && !openbsd && !netbsd && !solaris
// +build !darwin,!openbsd,!netbsd,!solaris

package sexec

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/sonnt85/gosutils/pty"
	"github.com/sonnt85/gosutils/sutils"
)

func execCommandShellElevatedEnvTimeout1(name string, showCmd int32, moreenvs map[string]string, timeout time.Duration, args ...string) (stdOut, stdErr []byte, err error) {
	var stdout, stderr bytes.Buffer
	if len(name) == 0 {
		name, err = os.Executable()
		if err != nil {
			return
		}
	}
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

func execCommandShellElevatedEnvTimeout(ctx context.Context, name string, showCmd int32, moreenvs map[string]string, timeout time.Duration, args ...string) (stdOut, stdErr []byte, err error) {
	var stdout, stderr bytes.Buffer
	if len(name) == 0 {
		name, err = os.Executable()
		if err != nil {
			return
		}
	}
	args = append([]string{"-S", name}, args...)
	cmd := exec.CommandContext(ctx, "sudo", args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{}
	// cmd.SysProcAttr.Credential = &syscall.Credential{Uid: 0, Gid: 0} //error
	if len(moreenvs) != 0 {
		for k, v := range moreenvs {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
		}
	}

	var tty *os.File
	tty, err = pty.Start(cmd)
	if err != nil {
		return
	}
	defer tty.Close()
	go sutils.TeeReadWriterOsFile(tty, os.Stdin, &stderr, &stdout, nil)
	// go sutils.CopyReadWriters(tty, os.Stdin, nil)

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
	if err != nil {
		errstr := fmt.Sprintf("error code: [%s]", err)
		if stdout.Len() != 0 {
			errstr = fmt.Sprintf("%s,stdout: [%s]", errstr, stdout.String())
		}
		if stderr.Len() != 0 {
			errstr = fmt.Sprintf("%s,stderr: [%s]", errstr, stderr.String())
		}
		err = fmt.Errorf(errstr)
	}
	return stdout.Bytes(), stderr.Bytes(), err
}
