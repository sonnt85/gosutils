//go:build !windows
// +build !windows

package sexec

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"syscall"

	"github.com/sonnt85/gosutils/cmdshellwords"
)

func makeCmdLine(args []string) string {
	return cmdshellwords.Join(args...)
}

func syscallExec(binary string, argv []string, envv []string) (err error) {
	return syscall.Exec(binary, argv, envv)
}

func cmdHiddenConsole(cmd *exec.Cmd) {
	return
	if !IsConsoleExecutable(cmd.Path) || runtime.GOOS == "darwin" {
		return
	}

	xterm := os.Getenv("TERM")
	if len(xterm) == 0 {
		xterm = "xterm-256color" //xterm-256color xterm
	}
	cmd.Env = append(cmd.Env, fmt.Sprintf("TERM=%s", xterm))
	if cmd.SysProcAttr != nil {
		cmd.SysProcAttr.Setctty = true
		cmd.SysProcAttr.Setsid = true
	} else {
		cmd.SysProcAttr = &syscall.SysProcAttr{
			Setctty: true,
			Setsid:  true,
		}
	}
}
