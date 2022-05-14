//go:build !windows
// +build !windows

package sexec

import "github.com/sonnt85/gosutils/cmdshellwords"

func makeCmdLine(args []string) string {
	return cmdshellwords.Join(args...)
}
