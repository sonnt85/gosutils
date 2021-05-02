// +build !windows

package sutils

import (
	"os"
	"syscall"

	dbus "github.com/godbus/dbus"
	//	dbus "github.com/guelfey/go.dbus"
)

// dbusCall calls a D-Bus method that has no return value.
func DbusCall(bus *dbus.Object, path string) error {
	//	log.WithFields(log.Fields{"path": path}).Debug("omxplayer: dbus call")
	return bus.Call(path, 0).Err
}

// dbusGetBool calls a D-Bus method that will return a boolean value.
func DbusGetBool(bus *dbus.Object, path string) (bool, error) {
	//	log.WithFields(log.Fields{"path": path}).Debug("omxplayer: dbus call")
	call := bus.Call(path, 0)
	if call.Err != nil {
		return false, call.Err
	}
	return call.Body[0].(bool), nil
}

// dbusGetFloat64 calls a D-Bus method that will return an int64 value.
func DbusGetFloat64(bus *dbus.Object, path string) (float64, error) {
	//	log.WithFields(log.Fields{"path": path}).Debug("omxplayer: dbus call")
	call := bus.Call(path, 0)
	if call.Err != nil {
		return 0, call.Err
	}
	return call.Body[0].(float64), nil
}

// dbusGetInt64 calls a D-Bus method that will return an int64 value.
func DbusGetInt64(bus *dbus.Object, path string) (int64, error) {
	//	log.WithFields(log.Fields{"path": path}).Debug("omxplayer: dbus call")
	call := bus.Call(path, 0)
	if call.Err != nil {
		return 0, call.Err
	}
	return call.Body[0].(int64), nil
}

// dbusGetString calls a D-Bus method that will return a string value.
func DbusGetString(bus *dbus.Object, path string) (string, error) {
	//	log.WithFields(log.Fields{"path": path}).Debug("omxplayer: dbus call")
	call := bus.Call(path, 0)
	if call.Err != nil {
		return "", call.Err
	}
	return call.Body[0].(string), nil
}

// dbusGetStringArray calls a D-Bus method that will return a string array.
func DbusGetStringArray(bus *dbus.Object, path string) ([]string, error) {
	//	log.WithFields(log.Fields{"path": path}).Debug("omxplayer: dbus call")
	call := bus.Call(path, 0)
	if call.Err != nil {
		return nil, call.Err
	}
	return call.Body[0].([]string), nil
}

func DirIsWritable(path string) (isWritable bool) {
	isWritable = false
	info, err := os.Stat(path)
	if err != nil {
		//		fmt.Println("Path doesn't exist")
		return
	}
	if !info.IsDir() {
		//		fmt.Println("Path isn't a directory")
		return
	}

	// Check if the user bit is enabled in file permission
	if info.Mode().Perm()&(1<<(uint(7))) == 0 {
		//			fmt.Println("Write permission bit is not set on this file for user")
		return
	}
	var stat syscall.Stat_t
	if err = syscall.Stat(path, &stat); err != nil {
		//			fmt.Println("Unable to get stat")
		return
	}

	if uint32(os.Geteuid()) != stat.Uid {
		isWritable = false
		//			fmt.Println("User doesn't have permission to write to this directory")
		return
	}
	isWritable = true
	return
}

func FileIWriteable(path string) (isWritable bool) {
	isWritable = false
	err := syscall.Access(path, syscall.O_RDWR)
	if err != nil {
		// fmt.Println(err.Error())
		return
	}
	//	fmt.Println("access ok")
	return true
}

func IsDoubleClickRun() bool {
	return true
}
