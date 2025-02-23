package sexec

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v4/process"

	log "github.com/sirupsen/logrus"
)

func ExecCommandCtxShellEnvTimeout(ctx context.Context, script string, moreenvs map[string]string, timeout time.Duration, scriptrunoption ...interface{}) (stdOut, stdErr []byte, err error) {
	shellbin := ""
	if runtime.GOOS == "windows" {
		if shellbin = os.Getenv("COMSPEC"); shellbin == "" {
			shellbin = "cmd"
		}
	} else { //linux
		// shellrunoption = []string{"-c", "--", script}
		shellbin = os.Getenv("SHELL")
		for _, v := range []string{"bash", "sh"} {
			if _, err := exec.LookPath(v); err == nil {
				shellbin = v
				break
			}
		}
	}
	if len(shellbin) == 0 {
		return nil, nil, errors.New("missing binary shell")
	}
	return ExecCommandCtxScriptEnvTimeout(ctx, shellbin, script, moreenvs, timeout, scriptrunoption...)
}

func killChilds(pid int) {
	if p, err := process.NewProcess(int32(pid)); err == nil {
		if ps, err := p.Children(); err == nil && len(ps) != 0 {
			for _, p := range ps {
				killChilds(int(p.Pid))
			}
		} else {
			// p.Terminate()
			p.Kill()
		}
	}
}
func HasArg(args ...interface{}) bool {
	for _, a := range args {
		switch v := a.(type) {
		case string:
			if !strings.HasPrefix(v, prefixarg0) && !strings.HasPrefix(v, prefixScriptName) {
				return true
			}
		}
	}
	return false
}

func GetScriptPath(filePath interface{}) (scriptbin string) {
	var r io.Reader
	var isReader bool
	switch v := filePath.(type) {
	case string:
		file, err := os.Open(v)
		if err != nil {
			return
		}
		r = file
		defer file.Close()
	case io.Reader:
		r = v
		isReader = true
	}

	buffer := make([]byte, 2)
	_, err := r.Read(buffer)
	if err != nil {
		return
	}
	if f, ok := r.(io.Seeker); ok && isReader {
		defer f.Seek(0, 0)
	}
	if string(buffer) == "#!" {
		scanner := bufio.NewScanner(r)
		isFirst := true
		ret := ""
		for {
			if scanner.Scan() {
				t := scanner.Text()
				if isFirst {
					isFirst = false
					if spaceIndex := strings.Index(t, " "); spaceIndex != -1 {
						t = t[spaceIndex+1:]
					}
					ret = strings.TrimSpace(t)
				}
				if strings.Contains(t, "BASH_SOURCE") {
					if !strings.HasPrefix(strings.TrimSpace(t), "#") {
						return "BASH_SOURCE"
					}
				}
				// t = strings.TrimPrefix(t, "/usr/bin/env")
				// t = strings.TrimSpace(t)
				// return t
			} else {
				return ret
			}
		}
	}
	return ""
}

// script is strings or io.Reader
func ExecCommandCtxScriptEnvTimeout(ctxc context.Context, scriptbin string, script interface{}, moreenvs map[string]string, timeout time.Duration, args ...interface{}) (stdOut, stdErr []byte, err error) {
	var stdout, stderr io.Writer //
	var cmdPointer **exec.Cmd
	var hookEndFunc func(*exec.Cmd)

	var cmd *exec.Cmd
	if ctxc == nil {
		ctxc = context.Background()
	}
	if timeout > 0 {
		var cancelFn context.CancelFunc
		ctxc, cancelFn = context.WithTimeout(ctxc, timeout)
		defer cancelFn()
	}
	var arg []string
	var arg0 string
	var scriptname string
	var cleanEnv bool
	useWriterInArgs := false
	for _, a := range args {
		switch v := a.(type) {
		case string:
			if strings.HasPrefix(v, prefixarg0) {
				arg0 = strings.TrimPrefix(v, prefixarg0)
			} else if strings.HasPrefix(v, prefixScriptName) {
				scriptname = strings.TrimPrefix(v, prefixScriptName)
			} else {
				// if len(arg) == 0 {
				// 	arg = []string{"-s"}
				// }
				arg = append(arg, v)
			}
		case io.Writer:
			useWriterInArgs = true
			if stdout == nil {
				stdout = v
			} else if stderr == nil {
				stderr = v
			}
		case **exec.Cmd:
			cmdPointer = v
		case func(*exec.Cmd):
			hookEndFunc = v
		case bool:
			cleanEnv = v
		default:
			err = fmt.Errorf("unsupported type: %v", v)
			return
		}
	}
	cmd = exec.CommandContext(ctxc, scriptbin, arg...)
	if len(scriptname) != 0 {
		cmd.Args[0] = scriptname
	} else if len(arg0) != 0 {
		cmd.Args[0] = arg0
	}
	CmdHiddenConsole(cmd)
	if !useWriterInArgs {
		stdout = &bytes.Buffer{}
		stderr = &bytes.Buffer{}
		defer func() {
			stdOut = stdout.(*bytes.Buffer).Bytes()
			stdErr = stderr.(*bytes.Buffer).Bytes()
		}()
	}
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	GOOS := runtime.GOOS

	var NewLine = "\n"
	if GOOS == "windows" {
		NewLine = "\r\n"
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
	} else if GOOS == "darwin" {
		NewLine = "\r"
	}
	var stdin io.Reader
	switch v := script.(type) {
	case string:
		if !strings.HasSuffix(v, "\n") || !strings.HasSuffix(v, "\r") {
			v += NewLine
		}
		if GOOS == "windows" && filepath.Base(scriptbin) == "cmd.exe" {
			v += "exit" + NewLine
		}
		stdin = bytes.NewBuffer([]byte(v))
	case io.Reader:
		stdin = v
	}

	cmd.Stdin = stdin
	if cleanEnv {
		cmd.Env = EnrovimentMapToStrings(moreenvs)
	} else {
		cmd.Env = EnrovimentMergeWithCurrentEnv(moreenvs)
	}
	err = cmd.Start()
	if cmdPointer != nil {
		*cmdPointer = cmd
	}
	if err != nil {
		return
	}
	err = cmd.Wait()
	// err = cmd.Run()
	// if err != nil {
	// 	return stdOut, stdErr, err
	// }
	// err = cmd.Wait()
	if hookEndFunc != nil {
		hookEndFunc(cmd)
	}
	return
}

// func ExecCommandCtxShellEnvTimeoutAs(script string, moreenvs map[string]string, timeout time.Duration) (stdOut, stdErr []byte, err error) {
// }

func ExecCommandCtxShell(ctx context.Context, script string, timeout time.Duration, args ...interface{}) (stdOut, stdErr []byte, err error) {
	return ExecCommandCtxShellEnvTimeout(ctx, script, map[string]string{}, timeout, args...)
}

func ExecCommandCtxEnvTimeout(ctxc context.Context, name string, moreenvs map[string]string, timeout time.Duration, args ...interface{}) (stdOut, stdErr []byte, err error) {
	var stdout, stderr io.Writer //
	var cmdPointer **exec.Cmd
	var cmd *exec.Cmd
	var hookEndFunc func(*exec.Cmd)
	if ctxc == nil {
		ctxc = context.Background()
	}
	if timeout > 0 {
		var cancelFn context.CancelFunc
		ctxc, cancelFn = context.WithTimeout(ctxc, timeout)
		defer cancelFn()
	}
	useWriterInArgs := false
	var arg []string
	var arg0 string
	var cleanEnv bool
	for _, a := range args {
		switch v := a.(type) {
		case string:
			if strings.HasPrefix(v, prefixarg0) {
				arg0 = strings.TrimPrefix(v, prefixarg0)
			} else if !strings.HasPrefix(v, prefixScriptName) {
				arg = append(arg, v)
			}
		case *os.File:
			useWriterInArgs = true
			if stdout == nil {
				stdout = v
			} else if stderr == nil {
				stderr = v
			}
			defer func() {
				v.Close()
			}()
			continue
		case io.Writer:
			useWriterInArgs = true
			if stdout == nil {
				stdout = v
			} else if stderr == nil {
				stderr = v
			}
		case **exec.Cmd:
			cmdPointer = v
		case func(*exec.Cmd):
			hookEndFunc = v
		case bool:
			cleanEnv = v
		default:
			err = fmt.Errorf("unsupported type: %v", v)
			return
		}
	}
	if !useWriterInArgs {
		stdout = &bytes.Buffer{}
		stderr = &bytes.Buffer{}
		defer func() {
			stdOut = stdout.(*bytes.Buffer).Bytes()
			stdErr = stderr.(*bytes.Buffer).Bytes()
		}()
	}
	cmd = exec.CommandContext(ctxc, name, arg...)

	CmdHiddenConsole(cmd)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if len(arg0) != 0 {
		cmd.Args[0] = arg0
	}
	if cleanEnv {
		cmd.Env = EnrovimentMapToStrings(moreenvs)
	} else {
		cmd.Env = EnrovimentMergeWithCurrentEnv(moreenvs)
	}
	err = cmd.Start()
	if cmdPointer != nil {
		*cmdPointer = cmd
	}
	if err != nil {
		return
	}
	err = cmd.Wait()
	// err = cmd.Run()

	// if err != nil {
	// 	return
	// }
	// err = cmd.Wait()
	if hookEndFunc != nil {
		hookEndFunc(cmd)
	}
	return
}

// run command without timeout
func ExecCommandCtx(ctx context.Context, name string, args ...interface{}) (stdOut, stdErr []byte, err error) {
	return ExecCommandCtxEnvTimeout(ctx, name, map[string]string{}, -1, args...)
}

// run command with timeout
func ExecCommandCtxTimeout(ctx context.Context, name string, timeout time.Duration, args ...interface{}) (stdOut, stdErr []byte, err error) {
	return ExecCommandCtxEnvTimeout(ctx, name, map[string]string{}, timeout, args...)
}

func ExecCommandCtxEnv(ctx context.Context, name string, moreenvs map[string]string, args ...interface{}) (stdOut, stdErr []byte, err error) {
	return ExecCommandCtxEnvTimeout(ctx, name, moreenvs, -1, args...)
}

// spaw father to  child via syscall, merge executablePath to executableArgs if first executableArgs[0] is diffirence executablePath
func ExecCommandCtxSyscall(ctx context.Context, executablePath string, executableArgs []string, moreEnvs map[string]string) error {
	//  var result string
	if len(executableArgs) == 0 {
		executableArgs = make([]string, 0)
		if len(executablePath) == 0 {
			executableArgs = os.Args[1:]
		}
	}

	if len(executablePath) == 0 {
		executablePath, _ = GetExecPath()
	} else {
		executablePath, _ = filepath.Abs(executablePath)
	}
	//need config executableEnv if not its empty
	// executableEnv := []string{}
	executableEnv := EnrovimentMergeWithCurrentEnv(moreEnvs)
	var binary string
	var err error
	binary, err = exec.LookPath(executablePath)

	// if _, err = os.Stat(executablePath); err == nil {
	// 	binary, err = filepath.Abs(executablePath)
	// } else {
	// 	binary, err = exec.LookPath(executablePath)
	// 	log.Errorf("Error LookPath: %s", err)
	// }

	if err != nil {
		log.Errorf("Error: %s", err)
		return err
	}
	//	time.Sleep(1 * time.Second)
	tmpslide := []string{filepath.Base(binary)}
	if tmpslide[0] == executableArgs[0] {
	} else {
		executableArgs = append(tmpslide, executableArgs...)
	}
	err = syscallExec(binary, executableArgs, executableEnv)
	if err != nil {
		log.Errorf("error: %s %v", binary, err)
	}
	return err
}

// byteprog interface{} is io.Reader or []byte
func ExecBytesCtxEnvTimeout(ctx context.Context, byteprog interface{}, name string, moreenvs map[string]string, timeout time.Duration, args ...interface{}) (retstdout, retstderr []byte, err error) {
	var f *MemFile
	f, err = open(byteprog, name)
	if err != nil {
		return nil, nil, err
	}
	nodel := false
	if !HasArg(args...) {
		if scriptBin := GetScriptPath(f.Name()); len(scriptBin) != 0 {
			if scriptBin == "BASH_SOURCE" {
				nodel = true
			} else {
				var b []byte
				b, err = os.ReadFile(f.Name())
				if err != nil {
					return
				}
				f.Close()
				return ExecCommandCtxScriptEnvTimeout(ctx, scriptBin, string(b), moreenvs, timeout, args...)
			}
		}
	}
	if !nodel {
		defer f.Close()
	}
	return ExecCommandCtxEnvTimeout(ctx, f.Name(), moreenvs, timeout, args...)
}

func ExecBytesCtxEnv(ctx context.Context, byteprog interface{}, name string, moreenvs map[string]string, args ...interface{}) (retstdout, retstderr []byte, err error) {
	return ExecBytesCtxEnvTimeout(ctx, byteprog, name, moreenvs, -1, args...)
}

func ExecBytesCtxTimeout(ctx context.Context, byteprog interface{}, name string, timeout time.Duration, args ...interface{}) (retstdout, retstderr []byte, err error) {
	return ExecBytesCtxEnvTimeout(ctx, byteprog, name, nil, timeout, args...)
}

func ExecBytesCtx(ctx context.Context, byteprog interface{}, name string, args ...interface{}) (retstdout, retstderr []byte, err error) {
	return ExecBytesCtxEnvTimeout(ctx, byteprog, name, nil, -1, args...)
}

// exe is empty will run current program
func ExecCommandCtxShellElevated(ctx context.Context, exe string, showCmd int32, args ...interface{}) (stdOut, stdErr []byte, err error) {
	return ExecCommandCtxShellElevatedEnvTimeout(ctx, exe, showCmd, map[string]string{}, -1, args...)
}

// exe is empty will run current program
func ExecCommandCtxShellElevatedEnvTimeout(ctx context.Context, exe string, showCmd int32, moreenvs map[string]string, timeout time.Duration, args ...interface{}) (stdOut, stdErr []byte, err error) {
	return execCommandShellElevatedEnvTimeout(ctx, exe, showCmd, moreenvs, timeout, args...)
}
