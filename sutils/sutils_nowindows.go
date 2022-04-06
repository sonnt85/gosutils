//go:build !windows
// +build !windows

package sutils

import (
	"fmt"
	"os"
	"syscall"

	dbus "github.com/godbus/dbus"
)

// const ErrDusObjectIsNil = "dbus opject is nil"
var ErrDusObjectIsNil = fmt.Errorf("dbus opject is nil")

// dbusCall calls a D-Bus method that has no return value.
func DbusCall(bus dbus.BusObject, path string) error {
	if bus == nil {
		return ErrDusObjectIsNil
	}
	return bus.Call(path, 0).Err
}

// dbusGetBool calls a D-Bus method that will return a boolean value.
func DbusGetBool(bus dbus.BusObject, path string) (bool, error) {
	if bus == nil {
		return false, ErrDusObjectIsNil
	}
	call := bus.Call(path, 0)
	if call.Err != nil {
		return false, call.Err
	}
	return call.Body[0].(bool), nil
}

// dbusGetFloat64 calls a D-Bus method that will return an int64 value.
func DbusGetFloat64(bus dbus.BusObject, path string) (float64, error) {
	if bus == nil {
		return 0, ErrDusObjectIsNil
	}
	call := bus.Call(path, 0)
	if call.Err != nil {
		return 0, call.Err
	}
	return call.Body[0].(float64), nil
}

// dbusGetInt64 calls a D-Bus method that will return an int64 value.
func DbusGetInt64(bus dbus.BusObject, path string) (int64, error) {
	if bus == nil {
		return 0, ErrDusObjectIsNil
	}
	call := bus.Call(path, 0)
	if call.Err != nil {
		return 0, call.Err
	}
	return call.Body[0].(int64), nil
}

// dbusGetString calls a D-Bus method that will return a string value.
func DbusGetString(bus dbus.BusObject, path string) (string, error) {
	if bus == nil {
		return "", ErrDusObjectIsNil
	}
	call := bus.Call(path, 0)
	if call.Err != nil {
		return "", call.Err
	}
	return call.Body[0].(string), nil
}

// dbusGetStringArray calls a D-Bus method that will return a string array.
func DbusGetStringArray(bus dbus.BusObject, path string) ([]string, error) {
	if bus == nil {
		return []string{}, ErrDusObjectIsNil
	}
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
		return
	}
	if !info.IsDir() {
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
		//fmt.Println("User doesn't have permission to write to this directory")
		return
	}
	isWritable = true
	return
}

func FileIWriteable(path string) (isWritable bool) {
	isWritable = false
	err := syscall.Access(path, syscall.O_RDWR)
	if err != nil {
		return
	}
	return true
}
