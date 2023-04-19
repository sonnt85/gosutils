package sexec

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/sonnt85/gosutils/cmdshellwords"
	"golang.org/x/sys/windows"
)

func makeCmdLine(args []string) string {
	var s string
	for i := 0; i < len(args); i++ {
		if s != "" {
			s += " "
		}
		s += syscall.EscapeArg(args[i])
	}
	return s
}

func execCommandShellElevatedEnvTimeout_(ctxc context.Context, exe string, showCmd int32, moreenvs map[string]string, timeout time.Duration, args ...string) (stdOut, stdErr []byte, err error) {
	var stdout, stderr bytes.Buffer
	if timeout == 0 || timeout == -1 {
		timeout = 1<<63 - 1
	}
	ctx, cancelFn := context.WithTimeout(context.Background(), timeout)
	defer cancelFn()
	cmd := exec.CommandContext(ctx, "cmd", append([]string{"/C", exe}, args...)...)

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	// script := makeCmdLine(append([]string{exe}, args...))
	// cmd.Stdin = bytes.NewBuffer([]byte(script))
	hidewindow := true
	if showCmd != 0 {
		hidewindow = false
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CmdLine: "runas /user:Administrator",
		// Token:      getAdminToken(),
		HideWindow: hidewindow,
	}
	if len(moreenvs) != 0 {
		// cmd.Env = os.Environ()
		for k, v := range moreenvs {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
		}
	}
	err = cmd.Start()
	if err != nil {
		return stdOut, stdErr, err
	}
	needKill := false
	if ctxc == nil {
		err = cmd.Wait()
	} else {
		c := make(chan error, 1)

		// Thực hiện cmd.Wait() trong một goroutine riêng
		go func() {
			c <- cmd.Wait()
		}()

		select {
		case err = <-c: // cmd.Wait()
		case <-ctxc.Done():
			err = errors.New("cancelled context")
			needKill = true
		}
	}

	if needKill {
		killChilds(cmd.Process.Pid)
		cmd.Process.Kill()
	}

	if ctx.Err() == context.DeadlineExceeded {
		err = fmt.Errorf("124:Timeout")
	}

	if err != nil {
		errstr := fmt.Sprintf("error code: [%s]", err)
		if stdout.Len() != 0 {
			errstr = fmt.Sprintf("%s,stdout: [%s]", errstr, stdout.String())
		}
		if stderr.Len() != 0 {
			errstr = fmt.Sprintf("%s,stderr: [%s]", errstr, stderr.String())
		}
		err = fmt.Errorf(errstr)
	}

	return stdout.Bytes(), stderr.Bytes(), err
}

func execCommandShellElevatedEnvTimeout(ctxc context.Context, exe string, showCmd int32, moreenvs map[string]string, timeout time.Duration, args ...string) (stdOut, stdErr []byte, err error) {

	// err = elevate.ShellExecute(exe, makeCmdLine(args), "", showCmd)
	// return
	verb := "runas"
	if timeout == 0 || timeout == -1 {
		timeout = 1<<63 - 1
	}
	ctx, cancelFn := context.WithTimeout(context.Background(), timeout)
	defer cancelFn()
	if len(exe) == 0 {
		exe, err = os.Executable()
		if err != nil {
			return
		}
	}
	cwd, _ := os.Getwd()
	argstr := cmdshellwords.Join(args...)
	// argstr := strings.Join(args, " ") //notwork
	// elevate.RunMeElevated()
	// argstr := makeCmdLine(args)
	// showCmd = 1
	// fmt.Println("Window elevate cmd: ", exe, showCmd, moreenvs, args, argstr)

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
	// err = windows.ShellExecute(0, verbPtr, exePtr, argPtr, cwdPtr, showCmd)
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	errc := make(chan error, 1)

	// needKill := false
	if ctxc == nil {
		ctxc = context.Background()
		// windows.ShellExecuteEx()
		// err = windows.ShellExecute(0, verbPtr, exePtr, argPtr, cwdPtr, showCmd)
	}
	//  else 	{
	go func() {
		errc <- windows.ShellExecute(0, verbPtr, exePtr, argPtr, cwdPtr, showCmd)
	}()
	select {
	case <-ctx.Done():
		err = errors.New("124:Timeout")
	//case err = <-errc: //
	case <-ctxc.Done():
		err = errors.New("cancelled context")
		// needKill = true
	}
	// }
	// time.Sleep(time.Second * 60)
	return nil, nil, err
}

func toEnvBlock(env []string) *uint16 {
	block := make([]uint16, 0)
	for _, e := range env {
		block = append(block, syscall.StringToUTF16(e)...)
		block = append(block, 0)
	}
	block = append(block, 0)
	return &block[0]
}

// func toCmdLine(executable string, args []string) *uint16 {
// 	args = append([]string{executable}, args...)
// 	cmdLine := syscall.StringToUTF16Ptr("")
// 	for _, arg := range args {
// 		cmdLine = append(cmdLine, syscall.StringToUTF16Ptr(arg)...)
// 		cmdLine = append(cmdLine, 0)
// 	}
// 	return &cmdLine[0]
// }

func syscallExec(binary string, argv []string, envv []string) (err error) {
	cmdLine := syscall.StringToUTF16Ptr(makeCmdLine(append([]string{binary}, argv...)))
	// cmdLine := toCmdLine(binary, argv)

	envBlock := toEnvBlock(envv)

	startupInfo := &syscall.StartupInfo{}
	processInfo := &syscall.ProcessInformation{}

	err = syscall.CreateProcess(nil, cmdLine, nil, nil, false, 0, envBlock, nil, startupInfo, processInfo)
	// if syscall.WaitForSingleObject(processInfo.Process, syscall.INFINITE) == syscall.WAIT_OBJECT_0 {
	// 	os.Exit()
	// }
	if err == nil {
		os.Exit(0)
	}
	return err
}

func cmdHiddenConsole(cmd *exec.Cmd) {
	if cmd.SysProcAttr != nil {
		cmd.SysProcAttr.HideWindow = true
	} else {
		cmd.SysProcAttr = &syscall.SysProcAttr{
			HideWindow: true,
		}
	}
}
