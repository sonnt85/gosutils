package sexec
import (
	"github.com/getlantern/byteexec"
	"time"
	"log"
	"context"
	"os/exec"
	"bytes"
	"syscall"
	"io/ioutil"
	"os"
)

func ExecCommand(command string, timeout time.Duration) (result string, err error) {
	log.Printf("command:%v, timeout:%v", command, timeout)
	ctx, cancelFn := context.WithTimeout(context.Background(), timeout)
	defer cancelFn()

	var stdout, stderr bytes.Buffer

	cmd := exec.Command("bash", "-c", "--", command)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	err = cmd.Start()
	if err != nil {
		return "", err
	}

	go func() {
		<-ctx.Done()
		if ctx.Err() == context.DeadlineExceeded {
			log.Printf("timeout to kill process, %v", cmd.Process.Pid)
			syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		}
	}()

	err = cmd.Wait()
	//    var result string
	if err != nil {
		result = stderr.String()
	} else {
		result = stdout.String()
	}
	return result, err
}

func RunProgramBytes(byteprog []byte, progname string, args ...string) {
	var rootdir string
	rootdir = "" //default os.TempDir
	f, err := ioutil.TempFile(rootdir, progname)
	programBytes := byteprog // read bytes from somewhere
	be, err := byteexec.New(programBytes, f.Name())
	if err != nil {
		log.Fatalf("Uh oh: %s", err)
	}
	cmd := be.Command(args...)
	// cmd is an os/exec.Cmd
	defer os.Remove(f.Name())
	err = cmd.Run()
}