package sexec

import (
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/sonnt85/gosutils/cmdshellwords"
	"github.com/sonnt85/gosystem/elevate"
	"golang.org/x/sys/windows"
)

func makeCmdLine(args []string) string {
	var s string
	for _, v := range args {
		if s != "" {
			s += " "
		}
		s += syscall.EscapeArg(v)
	}
	return s
}

func execCommandShellElevatedEnvTimeout(exe string, showCmd int32, moreenvs map[string]string, timeout time.Duration, args ...string) (stdOut, stdErr []byte, err error) {
	verb := "runas"

	if len(exe) == 0 {
		exe, _ = os.Executable()
	}
	cwd, _ := os.Getwd()
	argstr := cmdshellwords.Join(args...)
	elevate.RunMeElevated()
	// strings.Join(args, " ")
	// argstr := makeCmdLine(args)
	// fmt.Println("Window elevate cmd: ", exe, showCmd, moreenvs, args)

	verbPtr, _ := syscall.UTF16PtrFromString(verb)
	exePtr, _ := syscall.UTF16PtrFromString(exe)
	cwdPtr, _ := syscall.UTF16PtrFromString(cwd)
	argPtr, _ := syscall.UTF16PtrFromString(argstr)

	// var showCmd int32 = 1 //SW_NORMAL
	if len(moreenvs) != 0 {
		storeenvs := os.Environ()
		for k, v := range moreenvs {
			os.Setenv(k, v)
		}
		defer func() {
			os.Clearenv()
			for _, e := range storeenvs {
				k, v, ok := strings.Cut(e, "=")
				if !ok {
					continue
				}
				os.Setenv(k, v)
			}
		}()
	}
	err = windows.ShellExecute(0, verbPtr, exePtr, argPtr, cwdPtr, showCmd)
	return nil, nil, err
}
