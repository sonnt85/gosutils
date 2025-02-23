//go:build (!windows && !linux) || openbsd || netbsd
// +build !windows,!linux openbsd netbsd

package sexec

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

// darwin no check
func execCommandShellElevatedEnvTimeout(ctxc context.Context, name string, showCmd int32, moreenvs map[string]string, timeout time.Duration, args ...interface{}) (stdOut, stdErr []byte, err error) {
	var stdout, stderr io.Writer
	var cmd *exec.Cmd
	var cmdPointer **exec.Cmd
	if ctxc == nil {
		ctxc = context.Background()
	}
	if timeout > 0 {
		var cancelFn context.CancelFunc
		ctxc, cancelFn = context.WithTimeout(ctxc, timeout)
		defer cancelFn()
	}
	useWriterInArgs := false
	// internal/fuzz.workerTimeoutDuration (1000000000)
	var arg []string
	var arg0 string
	for _, a := range args {
		switch v := a.(type) {
		case string:
			if strings.HasPrefix(v, prefixarg0) {
				arg0 = strings.TrimPrefix(v, prefixarg0)
			} else if !strings.HasPrefix(v, prefixScriptName) {
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
	}
	cmd = exec.CommandContext(ctxc, name, arg...)
	if len(arg0) != 0 {
		cmd.Args[0] = arg0
	}
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{}
	cmd.SysProcAttr.Credential = &syscall.Credential{Uid: 0, Gid: 0}
	if len(moreenvs) != 0 {
		for k, v := range moreenvs {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
		}
	}
	err = cmd.Start()
	if cmdPointer != nil {
		*cmdPointer = cmd
	}
	if err != nil {
		return
	}
	err = cmd.Wait()
	// err = cmd.Run()
	return
	// return ExecCommand(exe, args...)
}
