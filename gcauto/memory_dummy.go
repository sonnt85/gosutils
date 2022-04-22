// +build !linux,!darwin,!windows,!freebsd,!dragonfly,!netbsd,!openbsd

package gcauto

func TotalMemory() uint64 {
	return 0
}
func FreeMemory() uint64 {
	return 0
}
