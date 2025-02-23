//go:build !darwin && !openbsd && !netbsd && !solaris
// +build !darwin,!openbsd,!netbsd,!solaris

package sexec

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/sonnt85/gosutils/pty"
	"github.com/sonnt85/gosutils/sutils"
)

func execCommandShellElevatedEnvTimeout(ctxc context.Context, name string, showCmd int32, moreenvs map[string]string, timeout time.Duration, args ...interface{}) (stdOut, stdErr []byte, err error) {
	var stdout, stderr io.Writer //
	var cmd *exec.Cmd
	var cmdPointer **exec.Cmd
	// var stdoutb, stderrb bytes.Buffer
	if len(name) == 0 {
		name, err = os.Executable()
		if err != nil {
			return
		}
	}
	if ctxc == nil {
		ctxc = context.Background()
	}
	if timeout > 0 {
		var cancelFn context.CancelFunc
		ctxc, cancelFn = context.WithTimeout(ctxc, timeout)
		defer cancelFn()
	}
	useWriterInArgs := false

	// ctx, cancelFn := context.WithTimeout(context.Background(), timeout)
	// defer cancelFn()
	var arg []string
	var arg0 string
	for _, a := range args {
		switch v := a.(type) {
		case string:
			if strings.HasPrefix(v, prefixarg0) {
				arg0 = strings.TrimPrefix(v, prefixarg0)
			} else {
				arg = append(arg, v)
			}
		case io.Writer:
			useWriterInArgs = true
			if stdout == nil {
				stdout = v
			} else if stderr == nil {
				stderr = v
			}
		case **exec.Cmd:
			cmdPointer = v
		default:
			err = fmt.Errorf("unsupported type: %v", v)
			return
		}
	}
	if !useWriterInArgs {
		stdout = &bytes.Buffer{}
		stderr = &bytes.Buffer{}
		defer func() {
			stdOut = stdout.(*bytes.Buffer).Bytes()
			stdErr = stderr.(*bytes.Buffer).Bytes()
		}()
	} else {

	}

	arg = append([]string{"-S", name}, arg...)
	cmd = exec.CommandContext(ctxc, "sudo", arg...)
	if len(arg0) != 0 {
		cmd.Args[0] = arg0
	}
	// cmd.Stdout = stdout
	// cmd.Stderr = stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{}
	// cmd.SysProcAttr.Credential = &syscall.Credential{Uid: 0, Gid: 0} //error
	if len(moreenvs) != 0 {
		for k, v := range moreenvs {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
		}
	}

	var tty *os.File
	if cmdPointer != nil {
		*cmdPointer = cmd
	}
	tty, err = pty.Start(cmd)
	if err != nil {
		return
	}
	defer tty.Close()
	go sutils.TeeReadWriterOsFile(tty, os.Stdin, stderr, stdout, nil)
	// go sutils.CopyReadWriters(tty, os.Stdin, nil)
	err = cmd.Wait()
	return
}
