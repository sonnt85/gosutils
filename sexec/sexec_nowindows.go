//go:build !windows
// +build !windows

package sexec

import (
	"syscall"

	"github.com/sonnt85/gosutils/cmdshellwords"
)

func makeCmdLine(args []string) string {
	return cmdshellwords.Join(args...)
}

func syscallExec(binary string, argv []string, envv []string) (err error) {
	return syscall.Exec(binary, argv, envv)
}
