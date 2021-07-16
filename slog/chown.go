// +build !linux

package slog

import (
	"os"
)

func chown(_ string, _ os.FileInfo) error {
	return nil
}
