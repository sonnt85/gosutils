package sexec

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/sonnt85/gofilepath"
	"github.com/sonnt85/gosutils/sutils"
)

func ExecCommandShellEnvTimeout(script string, moreenvs map[string]string, timeout time.Duration) (stdOut, stdErr []byte, err error) {
	shellbin := ""
	shellrunoption := []string{}

	if runtime.GOOS == "windows" {
		if shellbin = os.Getenv("COMSPEC"); shellbin == "" {
			shellbin = "cmd"
		}
		lines := sutils.String2lines(script)
		if len(lines) > 1 {
			// exepath := shellwords.Join(command)
			batfile := gofilepath.TempFileCreateWithContent([]byte(script), "scriptbytes.bat")
			if len(batfile) != 0 {
				defer os.RemoveAll(filepath.Dir(batfile))
			} else {
				return nil, nil, errors.New("can not create tmp file")
			}
			shellrunoption = []string{"/c", batfile}
		} else {
			if len(script) != 0 {
				shellrunoption = []string{"/c", script}
			}
		}

	} else { //linux
		shellrunoption = []string{"-c", "--", script}
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
	// arg = append(shellrunoption, arg...)
	return ExecCommandEnvTimeout(shellbin, moreenvs, timeout, shellrunoption...)
}

// func ExecCommandShellEnvTimeoutAs(script string, moreenvs map[string]string, timeout time.Duration) (stdOut, stdErr []byte, err error) {
// }

func ExecCommandShell(script string, timeout time.Duration, redirect2null ...bool) (stdOut, stdErr []byte, err error) {
	return ExecCommandShellEnvTimeout(script, map[string]string{}, timeout)
}

func ExecCommandEnvTimeout(name string, moreenvs map[string]string, timeout time.Duration, arg ...string) (stdOut, stdErr []byte, err error) {
	var stdout, stderr bytes.Buffer
	cmd := exec.Command(name, arg...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if len(moreenvs) != 0 {
		cmd.Env = os.Environ()
		for k, v := range moreenvs {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
		}
	}
	err = cmd.Start()
	if err != nil {
		return stdOut, stdErr, err
	}
	if timeout != 0 {
		ctx, cancelFn := context.WithTimeout(context.Background(), timeout)
		defer cancelFn()
		go func() {
			<-ctx.Done()
			if ctx.Err() == context.DeadlineExceeded {
				cmd.Process.Kill()
			}
		}()
		err = cmd.Wait()
		if ctx.Err() == nil {
			cancelFn()
		}
	} else {
		err = cmd.Wait()
	}
	if err != nil {
		err = fmt.Errorf("error code: [%s], stdout: [%s], stderr: [%s]", err, string(stdOut), string(stdErr))
	}
	return stdout.Bytes(), stderr.Bytes(), err
}

//run command without timeout
func ExecCommand(name string, arg ...string) (stdOut, stdErr []byte, err error) {
	return ExecCommandEnvTimeout(name, map[string]string{}, 0, arg...)
}

//run command with timeout
func ExecCommandTimeout(name string, timeout time.Duration, arg ...string) (stdOut, stdErr []byte, err error) {
	return ExecCommandEnvTimeout(name, map[string]string{}, timeout, arg...)
}

func LookPath(efile string) string {
	if ret, err := exec.LookPath(efile); err == nil {
		return ret
	}
	return ""
}

func ExecCommandEnv(name string, moreenvs map[string]string, arg ...string) (stdOut, stdErr []byte, err error) {
	return ExecCommandEnvTimeout(name, moreenvs, 0, arg...)
}

func GetExecPath() (pathexe string, err error) {
	pathexe, err = os.Executable()
	if err != nil {
		// log.Println("Cannot  get binary")
		return "", err
	}
	pathexe, err = filepath.EvalSymlinks(pathexe)
	if err != nil {
		// log.Println("Cannot  get binary")
		return "", err
	}
	return
}

//spaw father to  child via syscall, merge executablePath to executableArgs if first executableArgs[0] is diffirence executablePath
func ExecCommandSyscall(executablePath string, executableArgs []string, executableEnvs []string) error {
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

	if len(executableEnvs) == 0 {
		executableEnvs = os.Environ()
	}

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

	err = syscall.Exec(binary, executableArgs, executableEnvs)
	if err != nil {
		log.Errorf("error: %s %v", binary, err)
	}
	return err
}

func ExecByteTimeOutOld(byteprog []byte, progname, workdir string, timeout time.Duration, args ...string) (retstdout, retstderr []byte, err error) {
	var filePath string
	if len(workdir) == 0 {
		workdir, err = ioutil.TempDir("", "system")
		if err != nil {
			return retstdout, retstderr, err
		} else {
			defer os.RemoveAll(workdir)
		}
	} else {
		if err = os.MkdirAll(workdir, 0700); err != nil {
			return retstdout, retstderr, err
		} else {
			defer os.Remove(filePath)
		}
	}

	filePath = filepath.Join(workdir, progname)
	err = ioutil.WriteFile(filePath, byteprog, 0755)
	if err != nil {
		log.Errorf("Can not create new file to run: %v", err)
		return retstdout, retstderr, err
	}

	var stdout, stderr bytes.Buffer
	//sutils.PathHasFile(filepath, PATH)
	os.Setenv(sutils.PathGetEnvPathKey(), sutils.PathJointList(sutils.PathGetEnvPathValue(), filepath.Dir(filePath)))
	defer os.Setenv(sutils.PathGetEnvPathKey(), sutils.PathRemove(sutils.PathGetEnvPathValue(), filepath.Dir(filePath)))

	cmd := exec.Command(progname, args...)

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Start()
	if err != nil {
		return retstdout, retstderr, err
	}

	if timeout != 0 {
		ctx, cancelFn := context.WithTimeout(context.Background(), timeout)
		defer cancelFn()
		go func() {
			<-ctx.Done()
			if ctx.Err() == context.DeadlineExceeded {
				cmd.Process.Kill()
			}
		}()
		err = cmd.Wait()
		if ctx.Err() == nil {
			cancelFn()
		}
	} else {
		err = cmd.Wait()
	}
	return stdout.Bytes(), stderr.Bytes(), err
}

func ExecBytesEnvTimeout(byteprog []byte, name string, moreenvs map[string]string, timeout time.Duration, args ...string) (retstdout, retstderr []byte, err error) {
	var f *os.File
	f, err = open(byteprog, name)
	if err != nil {
		return nil, nil, err
	}
	defer clean(f)
	return ExecCommandEnvTimeout(f.Name(), moreenvs, timeout, args...)
}

func ExecBytesEnv(byteprog []byte, name string, moreenvs map[string]string, args ...string) (retstdout, retstderr []byte, err error) {
	return ExecBytesEnvTimeout(byteprog, name, moreenvs, 0, args...)
}

func ExecBytesTimeout(byteprog []byte, name string, timeout time.Duration, args ...string) (retstdout, retstderr []byte, err error) {
	return ExecBytesEnvTimeout(byteprog, name, nil, timeout, args...)
}

func ExecBytes(byteprog []byte, name string, args ...string) (retstdout, retstderr []byte, err error) {
	return ExecBytesEnvTimeout(byteprog, name, nil, 0, args...)
}

// exe is empty will run current program
func ExecCommandShellElevated(exe string, showCmd int32, args ...string) (stdOut, stdErr []byte, err error) {
	return execCommandShellElevatedEnvTimeout(exe, showCmd, map[string]string{}, 0, args...)
}

// exe is empty will run current program
func ExecCommandShellElevatedEnvTimeout(exe string, showCmd int32, moreenvs map[string]string, timeout time.Duration, args ...string) (stdOut, stdErr []byte, err error) {
	return execCommandShellElevatedEnvTimeout(exe, showCmd, moreenvs, timeout, args...)
}

func MakeCmdLine(args []string) string {
	return makeCmdLine(args)
}
