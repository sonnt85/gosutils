//go:build !windows
// +build !windows

package runonce

import (
	"fmt"
	"log"
	"os/exec"
	"regexp"
	"strconv"
	"syscall"

	"github.com/sonnt85/sdaemon"
	// "github.com/sonnt85/sdaemon"
)

func RebornNewProgram(port int, newname string, cloneflags ...bool) bool {
	cloneflag := true
	if len(cloneflags) != 0 {
		cloneflag = cloneflags[0]
	}
	var cntxt = &sdaemon.Context{
		//	PidFileName: "/tmp/sample.pid",
		PidFilePerm: 0644,
		LogFileName: "",
		LogFilePerm: 0640,
		WorkDir:     "",
		Umask:       027,
		NameProg:    newname,
		CloneFlag:   cloneflag,
		//	Args:        []string{"[go-daemon sample]"},
	}

	child, err := cntxt.Reborn()

	//	if err != nil && child == nil {
	//		if port != 0 {
	//			NewAndRunOnce(port)
	//		}
	//	}
	//	log.Println(child, err)
	if err != nil { //error
		//		log.Println("Unable to run: ", err)
		if port != -1 {
			NewAndRunOnce(port)
		}
		return false
	}

	if child != nil { //current is parrent [firt call]
		// updateDeamon()
		return true //exit parrent
	} else {
		//current is child run [secon call]
		//		defer cntxt.Release()
		if port != -1 {
			NewAndRunOnce(port)
		}
		return false
	}
}

/*
  Set close-on-exec state for all fds >= 3
  The idea comes from
    https://github.com/golang/gofrontend/commit/651e71a729e5dcbd9dc14c1b59b6eff05bfe3d26
*/
func closeOnExec(state bool) {

	out, err := exec.Command("ls", fmt.Sprintf("/proc/%d/fd/", syscall.Getpid())).Output()
	if err != nil {
		log.Fatal(err)
	}
	pids := regexp.MustCompile("[ \t\n]").Split(fmt.Sprintf("%s", out), -1)
	i := 0
	for i < len(pids) {
		if len(pids[i]) < 1 {
			i++
			continue
		}
		pid, err := strconv.Atoi(pids[i])
		if err != nil {
			log.Fatal(err)
		}
		if pid > 2 {
			// FIXME: Check if fd is close
			if state {
				syscall.Syscall(syscall.SYS_FCNTL, uintptr(pid), syscall.FD_CLOEXEC, 0)
			} else {
				syscall.Syscall(syscall.SYS_FCNTL, uintptr(pid), 0, 0)
			}
		}
		i++
	}
}
