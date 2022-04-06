//go:build !linux
// +build !linux

package slogrus

import (
	"os"
)

func chown(_ string, _ os.FileInfo) error {
	return nil
}
