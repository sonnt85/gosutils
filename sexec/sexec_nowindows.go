//go:build !windows
// +build !windows

package sexec

import (
	"github.com/sonnt85/gosutils/cmdshellwords"
	"os/exec"
	"syscall"
)

func makeCmdLine(args []string) string {
	return cmdshellwords.Join(args...)
}

func syscallExec(binary string, argv []string, envv []string) (err error) {
	return syscall.Exec(binary, argv, envv)
}

func cmdHiddenConsole(cmd *exec.Cmd) {
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
