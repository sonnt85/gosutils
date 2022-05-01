//go:build windows || aix || solaris
// +build windows aix solaris

package pty

import (
	opty "github.com/creack/pty"
	"os"
)

func setWinsizeTerminal(f *os.File, w, h int) error {
	return opty.ErrUnsupported
}
