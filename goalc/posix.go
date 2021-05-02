//+build !windows

package goacl

import "os"

// Chmod is os.Chmod.
var Chmod = os.Chmod
