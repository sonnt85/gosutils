package cmdshellwords

import (
	"errors"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"golang.org/x/sys/windows"
)

func parser(cmd string, p *Parser) ([]string, error) {
	return windows.DecomposeCommandLine(cmd)
}

func join(args ...string) string {
	var s string
	for _, v := range args {
		if s != "" {
			s += " "
		}
		s += syscall.EscapeArg(v)
	}
	return s
}

func shellRun(line, dir string) (string, error) {
	var shell string
	if shell = os.Getenv("COMSPEC"); shell == "" {
		shell = "cmd"
	}
	cmd := exec.Command(shell, "/c", line)
	if dir != "" {
		cmd.Dir = dir
	}
	b, err := cmd.Output()
	if err != nil {
		if eerr, ok := err.(*exec.ExitError); ok {
			b = eerr.Stderr
		}
		return "", errors.New(err.Error() + ":" + string(b))
	}
	return strings.TrimSpace(string(b)), nil
}
