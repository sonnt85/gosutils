package sexec

import (
	"bytes"
	"context"
	"github.com/getlantern/byteexec"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"syscall"
	"time"
)

func ExecCommandShell(command string, timeout time.Duration) (stdOut, stdErr []byte, err error) {
	var stdout, stderr bytes.Buffer
	log.Printf("command:%v, timeout:%v", command, timeout)

	cmd := exec.Command("bash", "-c", "--", command)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

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
				log.Printf("timeout to kill process, %v", cmd.Process.Pid)
				syscall.Kill(cmd.Process.Pid, syscall.SIGKILL)
			}
		}()
	}
	err = cmd.Wait()
	//    var result string
	return stdout.Bytes(), stderr.Bytes(), err
}

func ExecCommand(name string, arg ...string) (err error) {
	cmd := exec.Command(name, arg...)
	return cmd.Run()
}

func RunProgramBytes(byteprog []byte, progname, rootdir string, timeout time.Duration, args ...string) (retstdout, retstderr []byte, err error) {
	//	var err error
	var filePath string
	if len(rootdir) == 0 {
		rootdir, err = ioutil.TempDir("", "system")
		if err != nil {
			return retstdout, retstderr, err
		} else {
			//			defer os.RemoveAll(rootdir)
		}
	} else {
		if err = os.MkdirAll(rootdir, 0700); err != nil {
			return retstdout, retstderr, err
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
	be, err := byteexec.New(byteprog, filePath)
	if err != nil {
		return retstdout, retstderr, err
	}
	defer func() {
		os.Remove(filePath)
	}()

	var stdout, stderr bytes.Buffer

	cmd := be.Command(args...)
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
				syscall.Kill(cmd.Process.Pid, syscall.SIGKILL)
			}
		}()
	}

	//	err = cmd.Run() //block at here
	if err := cmd.Start(); err != nil {
		return retstdout, retstderr, err
	}
	os.Remove(filePath)
	err = cmd.Wait()

	return stdout.Bytes(), stderr.Bytes(), err
}
