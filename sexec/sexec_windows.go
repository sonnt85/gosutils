package sexec

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/shirou/gopsutil/v4/process"
	"github.com/sonnt85/gosutils/cmdshellwords"
	"github.com/sonnt85/gosutils/endec"

	// "github.com/sonnt85/gosutils/slogrus"
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

func execCommandShellElevatedEnvTimeout(ctxc context.Context, exe string, showCmd int32, moreenvs map[string]string, timeout time.Duration, args ...interface{}) (stdOut, stdErr []byte, err error) {

	// err = elevate.ShellExecute(exe, makeCmdLine(args), "", showCmd)
	// return
	verb := "runas"
	if ctxc == nil {
		ctxc = context.Background()
	}
	if timeout > 0 {
		var cancelFn context.CancelFunc
		ctxc, cancelFn = context.WithTimeout(ctxc, timeout)
		defer cancelFn()
	}

	// ctx, cancelFn := context.WithTimeout(context.Background(), timeout)
	// defer cancelFn()
	var arg []string
	for _, a := range args {
		switch v := a.(type) {
		case string:
			arg = append(arg, v)
		case io.Writer:
		default:
			err = fmt.Errorf("unsupported type: %#v", v)
			return
		}
	}
	if len(exe) == 0 {
		exe, err = os.Executable()
		if err != nil {
			return
		}
	} else {
		exe, err = exec.LookPath(exe)
		if err != nil {
			return
		}
	}
	cwd, _ := os.Getwd()
	argstr := cmdshellwords.Join(arg...)
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
	if moreenvs == nil {
		moreenvs = make(map[string]string)
	}
	checkprog := moreenvs["LOOKUPEXE"]
	delete(moreenvs, "LOOKUPEXE")
	idprogKey := "__IDPROG__" + endec.GenerateRandomAssci(5) //"__IDPROG__"
	randomString := endec.GenerateRandomAssci(16)            //fmt.Sprintf("__CPID%d", os.Getpid())
	if len(checkprog) == 0 {
		moreenvs[idprogKey] = randomString
	}
	if len(moreenvs) != 0 {
		cmdenv := exec.Command("cmd.exe")
		CmdHiddenConsole(cmdenv)
		cmdenv.Stdin = bytes.NewBuffer([]byte(createEnvSetxBatchFileContent(moreenvs, true)))
		cmdenv.Run()
		onceClear := sync.Once{}
		clear := func() {
			onceClear.Do(func() {
				cmdenv := exec.Command("cmd.exe")
				cmdenv.Stdin = bytes.NewBuffer([]byte(createEnvSetxBatchFileContent(moreenvs, false)))
				cmdenv.Run()
			})
		}
		defer func() {
			clear()
		}()
		go func() {
			time.Sleep(time.Second * 5)
			clear()
		}()
	}

	exeBaseName := filepath.Base(exe)
	err = windows.ShellExecute(0, verbPtr, exePtr, argPtr, cwdPtr, showCmd)
	if err != nil {
		return nil, nil, err
	}
	errDone := make(chan struct{}, 1)

	if len(checkprog) != 0 {
		go func() {
			for {
				if _, e := exec.LookPath(checkprog); e == nil {
					// time.Sleep(time.Second)
					errDone <- struct{}{}
					break
				}
				time.Sleep(time.Second)
			}
		}()
	} else {
		var ps []*process.Process

		cntCheck := 0
		for {
			pst := PgrepWithEnv(exeBaseName, idprogKey, randomString)
			if len(pst) != 0 || cntCheck > 50 {
				ps = PgrepWithEnv("*", idprogKey, randomString)
				break
			}
			cntCheck++
			time.Sleep(time.Millisecond * 10)
		}

		if len(ps) == 0 {
			return
		}
		var process []*os.Process
		for _, p := range ps {
			var pt *os.Process
			if pt, err = os.FindProcess(int(p.Pid)); err == nil {
				if n, _ := p.Name(); n != "explorer.exe" {
					process = append(process, pt)
				}
			}
		}

		go func() {
			for _, p := range process {
				p.Wait()
			}
			errDone <- struct{}{}
		}()

		defer func() {
			if err != nil {
				for _, p := range process {
					p.Kill()
				}
			}
		}()
	}
	select {
	case <-ctxc.Done():
		err = errors.New("124:Timeout")
	case <-ctxc.Done():
		err = errors.New("cancelled context")
	case <-errDone:
	}

	return nil, nil, err
}

func createEnvSetxBatchFileContent(env map[string]string, isSet bool) string {
	var builder strings.Builder
	builder.WriteString("@echo off\n")
	var val string
	for k, v := range env {
		if v == "PATH" {
			continue
		}
		if isSet {
			val = syscall.EscapeArg(v)
			builder.WriteString(fmt.Sprintf(`setx %s %s`, k, val))
		} else {
			// val = ""
			// builder.WriteString(fmt.Sprintf(`REG DELETE "HKLM\SYSTEM\CurrentControlSet\Control\Session Manager\Environment" /v %s /f`, k))
			//HKEY_CURRENT_USER\Environment HKCU\Environment
			// builder.WriteString(fmt.Sprintf(`REG DELETE "HKEY_CURRENT_USER\Environment" /v %s /f\n`, k))
			builder.WriteString(fmt.Sprintf(`REG DELETE "HKCU\Environment" /v %s /f`, k))
		}

		builder.WriteString("\n")
	}
	builder.WriteString("exit\n")
	return builder.String()
}

func createBatchFileContent(exe string, env map[string]string, argstr ...string) string {
	var builder strings.Builder
	builder.WriteString("@echo off\n")
	for k, v := range env {
		if v == "PATH" {
			continue
		}
		builder.WriteString(fmt.Sprintf(`setx %s %s`, k, syscall.EscapeArg(v)))
		builder.WriteString("\n")
	}

	builder.WriteString(makeCmdLine(append([]string{exe}, argstr...)))

	for k, v := range env {
		if v == "PATH" {
			continue
		}
		builder.WriteString(fmt.Sprintf(`setx %s %s`, k, syscall.EscapeArg(v)))
		builder.WriteString("\n")
	}
	builder.WriteString("exit\n")

	return builder.String()
}

func toEnvBlock(env []string) (s16s *uint16, err error) {
	block := make([]uint16, 0)
	var u16tmp []uint16
	for _, e := range env {
		u16tmp, err = syscall.UTF16FromString(e)
		if err != nil {
			return
		}
		block = append(block, u16tmp...)
		block = append(block, 0)
	}
	block = append(block, 0)
	return &block[0], nil
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
	// cmdLine := syscall.StringToUTF16Ptr(makeCmdLine(append([]string{binary}, argv...)))
	// var cmdline *uint16
	cmdLine, err := syscall.UTF16PtrFromString(makeCmdLine(append([]string{binary}, argv...)))
	if err != nil {
		return err
	}
	// cmdLine := toCmdLine(binary, argv)
	var envBlock *uint16
	if envBlock, err = toEnvBlock(envv); err != nil {
		return
	}

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
	if !IsConsoleExecutable(cmd.Path) {
		return
	}
	if cmd.SysProcAttr != nil {
		cmd.SysProcAttr.HideWindow = true
	} else {
		cmd.SysProcAttr = &syscall.SysProcAttr{
			HideWindow: true,
		}
	}
}
