package sexec

import (
	"bytes"
	"context"
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
	"path"

	//	"runtime"

	"time"
)

func ExecCommandShell(command string, timeout time.Duration, redirect2null ...bool) (stdOut, stdErr []byte, err error) {
	var stdout, stderr bytes.Buffer
	//	log.Printf("command:%v, timeout:%v", command, timeout)
	shellbin := ""
	shellrunoption := []string{}

	if runtime.GOOS == "windows" {
		shellrunoption = []string{"/c", command}

		if shellbin = os.Getenv("COMSPEC"); shellbin == "" {
			shellbin = "cmd"
		}
	} else { //linux
		shellrunoption = []string{"-c", "--", command}
		shellbin = os.Getenv("SHELL")
		for _, v := range []string{"bash", "sh"} {
			if _, err := exec.LookPath(v); err == nil {
				shellbin = v
				break
			}
		}
	}

	cmd := exec.Command(shellbin, shellrunoption...)

	if len(redirect2null) != 0 && redirect2null[0] {
		cmd.Stdout = nil
		cmd.Stderr = nil
		//			fmt.Println("Wating finish:\n", command)
	} else {
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
	}
	//	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true} //for linux only

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
				//				log.Printf("Timeout to kill process, %v", cmd.Process.Pid)
				cmd.Process.Kill()
				//				syscall.Kill(cmd.Process.Pid, syscall.SIGKILL)
			}
		}()
	}

	err = cmd.Wait()
	if len(redirect2null) != 0 && redirect2null[0] {
		return nil, nil, err
	}
	//    var result string
	return stdout.Bytes(), stderr.Bytes(), err
}

func ExecCommand(name string, arg ...string) (stdOut, stdErr []byte, err error) {
	var stdout, stderr bytes.Buffer

	cmd := exec.Command(name, arg...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Start()
	if err != nil {
		return stdOut, stdErr, err
	}
	err = cmd.Wait()
	//    var result string
	return stdout.Bytes(), stderr.Bytes(), err
}

func LookPath(efile string) string {
	if ret, err := exec.LookPath(efile); err == nil {
		return ret
	}
	return ""
}

func ExecCommandEnv(name string, moreenvs map[string]string, arg ...string) (stdOut, stdErr []byte, err error) {
	var stdout, stderr bytes.Buffer

	cmd := exec.Command(name, arg...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	for k, v := range moreenvs {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}
	err = cmd.Start()
	if err != nil {
		return stdOut, stdErr, err
	}
	err = cmd.Wait()
	//    var result string
	return stdout.Bytes(), stderr.Bytes(), err
}

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
	tmpslide := []string{path.Base(binary)}
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

func RunProgramBytes(byteprog []byte, progname, rootdir string, timeout time.Duration, args ...string) (retstdout, retstderr []byte, err error) {
	//	var err error
	var filePath string
	//	var isTemdir = false
	if len(rootdir) == 0 {
		//		isTemdir = true
		rootdir, err = ioutil.TempDir("", "system")
		if err != nil {
			return retstdout, retstderr, err
		} else {
			defer os.RemoveAll(rootdir)
		}
	} else {
		if err = os.MkdirAll(rootdir, 0700); err != nil {
			return retstdout, retstderr, err
		} else {
			defer os.Remove(filePath)
		}
	}

	filePath = path.Join(rootdir, progname)
	//	_, err = os.Create(filePath)
	//	if err != nil {
	//		return retstdout, retstderr, err
	//	}
	//	os.Chmod(filePath, 0744)
	//	f.Close()
	//	programBytes := byteprog // read bytes from somewhere
	err = ioutil.WriteFile(filePath, byteprog, 0755)
	//	be, err := byteexec.New(byteprog, filePath)

	//	defefunc := func() {
	//		if isTemdir {
	//			os.RemoveAll(rootdir)
	//		} else {
	//			os.Remove(filePath)
	//		}
	//	}
	//	defer defefunc()

	if err != nil {
		log.Errorf("Can not create new file to run: %v", err)
		return retstdout, retstderr, err
	}

	var stdout, stderr bytes.Buffer
	//sutils.PathHasFile(filepath, PATH)
	os.Setenv(sutils.PathGetEnvPathKey(), sutils.PathJointList(sutils.PathGetEnvPathValue(), path.Dir(filePath)))
	defer os.Setenv(sutils.PathGetEnvPathKey(), sutils.PathRemove(sutils.PathGetEnvPathValue(), path.Dir(filePath)))

	cmd := exec.Command(progname, args...)

	//	cmd := be.Command(args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	// cmd is an os/exec.Cmd
	if timeout != 0 {
		ctx, cancelFn := context.WithTimeout(context.Background(), timeout)
		defer cancelFn()

		go func() {
			<-ctx.Done()
			if ctx.Err() == context.DeadlineExceeded {
				log.Printf("timeout to kill process, %v", cmd.Process.Pid)
				cmd.Process.Kill()
			}
		}()
	}

	//	err = cmd.Run() //block at here
	err = cmd.Start()
	//	os.Remove(filePath)
	//	defefunc()
	if err != nil {
		log.Errorf("Can not start cmd: %v", err)
		//		sutils.FileCopy(filePath, "/tmp/run")
		return retstdout, retstderr, err
	}
	err = cmd.Wait()
	return stdout.Bytes(), stderr.Bytes(), err
}
