package sexec

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
	"syscall"

	//	"fmt"
	//"https://github.com/jpillora/overseer
	//	"github.com/getlantern/byteexec"
	log "github.com/sirupsen/logrus"
	"github.com/sonnt85/gosutils/sutils"

	"io/ioutil"
	"os"
	"os/exec"

	//	"runtime"

	"time"
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
			batfile := sutils.TempFileCreateInNewTemDir("scriptbytes.bat")
			if len(batfile) != 0 {
				defer os.RemoveAll(filepath.Dir(batfile))
			}
			err = os.WriteFile(batfile, []byte(script), os.FileMode(755))
			if err != nil {
				return nil, nil, err
			}
			// fmt.Println("================> Run bat file", batfile)
			// return ExecCommandTimeout(shellbin, timeout, "/c", batfile)
			return ExecCommandTimeout(batfile, timeout)
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
		return nil, nil, errors.New("Missing binary shell")
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

//spaw father to  child via syscall, merge executablePath to executableArgs if first executableArgs[0] is diffirence executablePath
func ExecCommandSyscall(executablePath string, executableArgs []string, executableEnvs []string) error {
	//  var result string
	//	executableArgs = os.Args
	//	executableEnvs = os.Environ()
	executablePath, _ = filepath.Abs(executablePath)

	binary, err := exec.LookPath(executablePath)
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
