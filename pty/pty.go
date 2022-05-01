package pty

import (
	"os"
	"os/exec"

	opty "github.com/creack/pty"
)

//set size terminal file
func SetWinsizeTerminal(f *os.File, w, h int) error {
	return setWinsizeTerminal(f, w, h)
}

// Start assigns a pseudo-terminal tty os.File to c.Stdin, c.Stdout,
// and c.Stderr, calls c.Start, and returns the File of the tty's
// corresponding pty.
//
// Starts the process in a new session and sets the controlling terminal.
func Start(c *exec.Cmd) (pty *os.File, err error) {
	return opty.Start(c)
}

// StartWithSize assigns a pseudo-terminal tty os.File to c.Stdin, c.Stdout,
// and c.Stderr, calls c.Start, and returns the File of the tty's
// corresponding pty.
//
// This will resize the pty to the specified size before starting the command.
// Starts the process in a new session and sets the controlling terminal.
func StartWithSize(c *exec.Cmd, ws *opty.Winsize) (pty *os.File, err error) {
	return opty.StartWithSize(c, ws)
}
