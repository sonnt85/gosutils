package sexec

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/shirou/gopsutil/process"
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

//lint:ignore U1000 ignore this!
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
	cmd.Env = EnrovimentMergeWithCurrentEnv(moreenvs)
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
		errstr := err.Error()
		if strings.HasPrefix(errstr, "wait") && strings.HasSuffix(errstr, ": no child processes") {
			err = nil
		}
	}
	if err != nil {
		errstr := fmt.Sprintf("error code: [%s]", err)
		if stdout.Len() != 0 {
			errstr = fmt.Sprintf("%s, stdout: [%s]", errstr, stdout.String())
		}
		if stderr.Len() != 0 {
			errstr = fmt.Sprintf("%s, stderr: [%s]", errstr, stderr.String())
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
		// if timeout == -1 {
		timeout = 1<<63 - 1
	}
	ctx, cancelFn := context.WithTimeout(context.Background(), timeout)
	defer cancelFn()
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
	if moreenvs == nil {
		moreenvs = make(map[string]string)
	}
	checkprog, _ := moreenvs["LOOKUPEXE"]
	delete(moreenvs, "LOOKUPEXE")
	idprogKey := "__IDPROG__" + endec.GenerateRandomAssci(5) //"__IDPROG__"
	randomString := endec.GenerateRandomAssci(16)            //fmt.Sprintf("__CPID%d", os.Getpid())
	if len(checkprog) == 0 {
		moreenvs[idprogKey] = randomString
	}
	if len(moreenvs) != 0 {
		isClear := false
		script := createEnvSetxBatchFileContent(moreenvs, true)
		ExecCommandShell(script)
		clear := func() {
			if !isClear {
				script := createEnvSetxBatchFileContent(moreenvs, false)
				ExecCommandShell(script)
				isClear = true
			}
		}
		defer func() {
			clear()
		}()
		go func() {
			time.Sleep(time.Second * 5)
			clear()
		}()
	}

	if ctxc == nil {
		ctxc = context.Background()
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
	case <-ctx.Done():
		err = errors.New("124:Timeout")

	case <-ctxc.Done():
		err = errors.New("cancelled context")
	case <-errDone:
	}

	return nil, nil, err
}

func execCommandShellElevatedEnvTimeoutOld(ctxc context.Context, exe string, showCmd int32, moreenvs map[string]string, timeout time.Duration, args ...string) (stdout, stderr []byte, err error) {
	verb := "runas"
	if timeout == -1 {
		timeout = 1<<63 - 1
	}

	ctx, cancelFn := context.WithTimeout(context.Background(), timeout)
	defer cancelFn()

	if len(exe) == 0 {
		exe, err = os.Executable()
		if err != nil {
			return nil, nil, err
		}
	}

	cwd, _ := os.Getwd()
	if moreenvs == nil {
		moreenvs = make(map[string]string)
	}
	idprogKey := "__IDPROG__" + endec.GenerateRandomAssci(5)
	randomString := endec.GenerateRandomAssci(16)
	moreenvs[idprogKey] = randomString

	var processEnvMap = make(map[string]string, 0)
	for _, rawEnvLine := range os.Environ() {
		k, v, ok := strings.Cut(rawEnvLine, "=")
		if !ok {
			continue
		}
		processEnvMap[k] = v
	}

	for key, value := range moreenvs {
		processEnvMap[key] = value
	}

	batchContent := createBatchFileContent(exe, processEnvMap, args...)
	batchFile, err := os.CreateTemp("", "*e.bat")
	if err != nil {
		return nil, nil, err
	}
	_, err = batchFile.WriteString(batchContent)
	if err != nil {
		return nil, nil, err
	}
	exePath := batchFile.Name()
	batchFile.Close()
	verbPtr, _ := syscall.UTF16PtrFromString(verb)
	exePtr, _ := syscall.UTF16PtrFromString(exePath)
	cwdPtr, _ := syscall.UTF16PtrFromString(cwd)
	// argPtr, _ := syscall.UTF16PtrFromString("")
	argPtr, _ := syscall.UTF16PtrFromString("/savecred")
	err = windows.ShellExecute(0, verbPtr, exePtr, argPtr, cwdPtr, showCmd)
	defer func() {
		cmd := exec.Command("setx", idprogKey, "")
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		cmd.CombinedOutput()
	}()

	if err != nil {
		return nil, nil, err
	}
	if ctxc == nil {
		ctxc = context.Background()
	}
	ticker := time.NewTicker(time.Second) // Định kỳ mỗi giây

loop:
	for {
		select {
		case <-ctx.Done():
			err = errors.New("124:Timeout")
			break loop
		case <-ctxc.Done():
			err = errors.New("cancelled context")
			break loop
			//case err = <-errc: //
		case <-ticker.C:
			if len(PgrepWithEnv(filepath.Base(exe), idprogKey, randomString)) == 0 {
				break loop
			}
		}
		// needKill = true
	}
	return stdout, stderr, err
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
		} else {
			val = ""
		}
		builder.WriteString(fmt.Sprintf(`setx %s %s`, k, val))
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
