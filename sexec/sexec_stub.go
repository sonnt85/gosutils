//go:build (!windows && !linux) || openbsd || netbsd
// +build !windows,!linux openbsd netbsd

package sexec

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"syscall"
	"time"
)

//darwin no check
func execCommandShellElevatedEnvTimeout(ctxc context.Context, name string, showCmd int32, moreenvs map[string]string, timeout time.Duration, args ...string) (stdOut, stdErr []byte, err error) {
	var stdout, stderr bytes.Buffer
	ctx, cancelFn := context.WithTimeout(context.Background(), timeout)
	defer cancelFn()
	cmd := exec.CommandContext(ctx, name, args...)

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
	needKill := false

	if ctxc == nil {
		err = cmd.Wait()
	} else {
		c := make(chan error, 1)

		// Thực hiện cmd.Wait() trong một goroutine riêng
		go func() {
			c <- cmd.Wait()
		}()

		select {
		case err = <-c: // cmd.Wait()
		case <-ctxc.Done():
			needKill = true
		}
	}

	if needKill {
		killChilds(cmd.Process.Pid)
		cmd.Process.Kill()
	}

	if ctx.Err() == context.DeadlineExceeded {
		err = fmt.Errorf("124:Timeout")
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
	return
	// return ExecCommand(exe, args...)
}
