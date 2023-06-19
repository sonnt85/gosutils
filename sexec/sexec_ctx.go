package sexec

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/shirou/gopsutil/process"

	log "github.com/sirupsen/logrus"
)

func ExecCommandCtxShellEnvTimeout(ctx context.Context, script string, moreenvs map[string]string, timeout time.Duration, scriptrunoption ...string) (stdOut, stdErr []byte, err error) {
	shellbin := ""
	if runtime.GOOS == "windows" {
		if shellbin = os.Getenv("COMSPEC"); shellbin == "" {
			shellbin = "cmd"
		}
	} else { //linux
		// shellrunoption = []string{"-c", "--", script}
		shellbin = os.Getenv("SHELL")
		for _, v := range []string{"bash", "sh"} {
			if _, err := exec.LookPath(v); err == nil {
				shellbin = v
				break
			}
		}
	}
	if len(shellbin) == 0 {
		return nil, nil, errors.New("missing binary shell")
	}
	return ExecCommandCtxScriptEnvTimeout(ctx, shellbin, script, moreenvs, timeout, scriptrunoption...)
}

func killChilds(pid int) {
	if p, err := process.NewProcess(int32(pid)); err == nil {
		if ps, err := p.Children(); err == nil && len(ps) != 0 {
			for _, p := range ps {
				killChilds(int(p.Pid))
			}
		} else {
			// p.Terminate()
			p.Kill()
		}
	}
}

func ExecCommandCtxScriptEnvTimeout(ctxc context.Context, scriptbin, script string, moreenvs map[string]string, timeout time.Duration, arg ...string) (stdOut, stdErr []byte, err error) {
	var stdout, stderr bytes.Buffer
	if timeout == 0 || timeout == -1 {
		timeout = 1<<63 - 1
	}
	ctx, cancelFn := context.WithTimeout(context.Background(), timeout)
	defer cancelFn()
	cmd := exec.CommandContext(ctx, scriptbin, arg...)
	CmdHiddenConsole(cmd)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if !strings.HasSuffix(script, "\n") {
		script += "\n"
	}
	cmd.Stdin = bytes.NewBuffer([]byte(script))
	cmd.Env = EnrovimentMergeWithCurrentEnv(moreenvs)
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
			err = errors.New("cancelled context")
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

	return stdout.Bytes(), stderr.Bytes(), err
}

// func ExecCommandCtxShellEnvTimeoutAs(script string, moreenvs map[string]string, timeout time.Duration) (stdOut, stdErr []byte, err error) {
// }

func ExecCommandCtxShell(ctx context.Context, script string, timeout time.Duration) (stdOut, stdErr []byte, err error) {
	return ExecCommandCtxShellEnvTimeout(ctx, script, map[string]string{}, timeout)
}

func ExecCommandCtxEnvTimeout(ctxc context.Context, name string, moreenvs map[string]string, timeout time.Duration, arg ...string) (stdOut, stdErr []byte, err error) {
	var stdout, stderr bytes.Buffer
	// var cmd *exec.Cmd
	// var ctx context.Context
	// var cancelFn context.CancelFunc
	if timeout == 0 || timeout == -1 {
		timeout = 1<<63 - 1
	}
	ctx, cancelFn := context.WithTimeout(context.Background(), timeout)
	defer cancelFn()
	cmd := exec.CommandContext(ctx, name, arg...)
	CmdHiddenConsole(cmd)

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Env = EnrovimentMergeWithCurrentEnv(moreenvs)
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
			err = errors.New("cancelled context")
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
	return stdout.Bytes(), stderr.Bytes(), err
}

// run command without timeout
func ExecCommandCtx(ctx context.Context, name string, arg ...string) (stdOut, stdErr []byte, err error) {
	return ExecCommandCtxEnvTimeout(ctx, name, map[string]string{}, -1, arg...)
}

// run command with timeout
func ExecCommandCtxTimeout(ctx context.Context, name string, timeout time.Duration, arg ...string) (stdOut, stdErr []byte, err error) {
	return ExecCommandCtxEnvTimeout(ctx, name, map[string]string{}, timeout, arg...)
}

func ExecCommandCtxEnv(ctx context.Context, name string, moreenvs map[string]string, arg ...string) (stdOut, stdErr []byte, err error) {
	return ExecCommandCtxEnvTimeout(ctx, name, moreenvs, -1, arg...)
}

// spaw father to  child via syscall, merge executablePath to executableArgs if first executableArgs[0] is diffirence executablePath
func ExecCommandCtxSyscall(ctx context.Context, executablePath string, executableArgs []string, moreEnvs map[string]string) error {
	//  var result string
	if len(executableArgs) == 0 {
		executableArgs = make([]string, 0)
		if len(executablePath) == 0 {
			executableArgs = os.Args[1:]
		}
	}

	if len(executablePath) == 0 {
		executablePath, _ = GetExecPath()
	} else {
		executablePath, _ = filepath.Abs(executablePath)
	}
	//need config executableEnv if not its empty
	// executableEnv := []string{}
	executableEnv := EnrovimentMergeWithCurrentEnv(moreEnvs)
	var binary string
	var err error
	binary, err = exec.LookPath(executablePath)

	// if _, err = os.Stat(executablePath); err == nil {
	// 	binary, err = filepath.Abs(executablePath)
	// } else {
	// 	binary, err = exec.LookPath(executablePath)
	// 	log.Errorf("Error LookPath: %s", err)
	// }

	if err != nil {
		log.Errorf("Error: %s", err)
		return err
	}
	//	time.Sleep(1 * time.Second)
	tmpslide := []string{filepath.Base(binary)}
	if tmpslide[0] == executableArgs[0] {
	} else {
		executableArgs = append(tmpslide, executableArgs...)
	}
	err = syscallExec(binary, executableArgs, executableEnv)
	if err != nil {
		log.Errorf("error: %s %v", binary, err)
	}
	return err
}

func ExecBytesCtxEnvTimeout(ctx context.Context, byteprog []byte, name string, moreenvs map[string]string, timeout time.Duration, args ...string) (retstdout, retstderr []byte, err error) {
	var f *os.File
	f, err = open(byteprog, name)
	if err != nil {
		return nil, nil, err
	}
	defer clean(f)
	return ExecCommandCtxEnvTimeout(ctx, f.Name(), moreenvs, timeout, args...)
}

func ExecBytesCtxEnv(ctx context.Context, byteprog []byte, name string, moreenvs map[string]string, args ...string) (retstdout, retstderr []byte, err error) {
	return ExecBytesCtxEnvTimeout(ctx, byteprog, name, moreenvs, -1, args...)
}

func ExecBytesCtxTimeout(ctx context.Context, byteprog []byte, name string, timeout time.Duration, args ...string) (retstdout, retstderr []byte, err error) {
	return ExecBytesCtxEnvTimeout(ctx, byteprog, name, nil, timeout, args...)
}

func ExecBytesCtx(ctx context.Context, byteprog []byte, name string, args ...string) (retstdout, retstderr []byte, err error) {
	return ExecBytesCtxEnvTimeout(ctx, byteprog, name, nil, -1, args...)
}

// exe is empty will run current program
func ExecCommandCtxShellElevated(ctx context.Context, exe string, showCmd int32, args ...string) (stdOut, stdErr []byte, err error) {
	return ExecCommandCtxShellElevatedEnvTimeout(ctx, exe, showCmd, map[string]string{}, -1, args...)
}

// exe is empty will run current program
func ExecCommandCtxShellElevatedEnvTimeout(ctx context.Context, exe string, showCmd int32, moreenvs map[string]string, timeout time.Duration, args ...string) (stdOut, stdErr []byte, err error) {
	return execCommandShellElevatedEnvTimeout(ctx, exe, showCmd, moreenvs, timeout, args...)
}
