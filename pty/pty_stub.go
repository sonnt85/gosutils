package pty

import (
	"os"
	"os/exec"
)

// Start assigns a pseudo-terminal tty os.File to c.Stdin, c.Stdout,
// and c.Stderr, calls c.Start, and returns the File of the tty's
// corresponding pty.
func Start(c *exec.Cmd) (pty *os.File, err error) {
	return pty, err
}

// StartWithSize assigns a pseudo-terminal tty os.File to c.Stdin, c.Stdout,
// and c.Stderr, calls c.Start, and returns the File of the tty's
// corresponding pty.
//
// This will resize the pty to the specified size before starting the command
func StartWithSize(c *exec.Cmd, sz interface{}) (pty *os.File, err error) {
	return pty, err
}

func SetWinsizeTerminal(f *os.File, w, h int) {
	return
}
