/*
  Examples:

  $ go run golo.go -timeout 10  -port 4040 --no-bind -- /usr/bin/ssh MyServer -o "LocalForward localhost:4040 localhost:8888" -fN
  :: Port is available. App is not running
  :: Now staring application '/usr/bin/ssh' from .

  $ go run golo.go -timeout 10  -port 4040 --no-bind -- /usr/bin/ssh MyServer -o "LocalForward localhost:4040 localhost:8888" -fN
  :: Port is not available. App is running?
*/

package runonce

import (
	"encoding/json"
	//	"flag"
	"fmt"
	"github.com/sonnt85/gosutils/sutils"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type RunOnceConf struct {
	IpAddress string
	WorkDir   string
	port      int
	timeout   int
	noBind    bool
	noLog     bool
	loopCall  bool
	cmd       *exec.Cmd
	args      []string
	doneExec  chan struct{}
	finished  chan struct{}
	isFunc    bool
}

func warnf(format string, a ...interface{}) {
	//if nolog != true {
	fmt.Fprintln(os.Stderr, "[", time.Now().Format(time.StampMilli), "]", fmt.Sprintf(format, a...))
	//}
}

/*
  Set close-on-exec state for all fds >= 3
  The idea comes from
    https://github.com/golang/gofrontend/commit/651e71a729e5dcbd9dc14c1b59b6eff05bfe3d26
*/
func closeOnExec(state bool) {

	out, err := exec.Command("ls", fmt.Sprintf("/proc/%d/fd/", syscall.Getpid())).Output()
	if err != nil {
		log.Fatal(err)
	}
	pids := regexp.MustCompile("[ \t\n]").Split(fmt.Sprintf("%s", out), -1)
	i := 0
	for i < len(pids) {
		if len(pids[i]) < 1 {
			i++
			continue
		}
		pid, err := strconv.Atoi(pids[i])
		if err != nil {
			log.Fatal(err)
		}
		if pid > 2 {
			// FIXME: Check if fd is close
			if state {
				syscall.Syscall(syscall.SYS_FCNTL, uintptr(pid), syscall.FD_CLOEXEC, 0)
			} else {
				syscall.Syscall(syscall.SYS_FCNTL, uintptr(pid), 0, 0)
			}
		}
		i++
	}
}

func cmdtable(buf []byte, gVar *RunOnceConf) {
	var dat map[string]interface{}
	//	fmt.Println(len(buf))
	if err := json.Unmarshal(buf[:len(buf)], &dat); err != nil {
		warnf("Not is json: ", err.Error())
		return
	}

	var cmd = dat["cmd"]
	var token = dat["token"]
	if token != strconv.Itoa(syscall.Getpid()) {
		warnf("Missing token!")
		return
	}
	switch cmd {
	case "quit":
		if gVar.isFunc == false {
			syscall.Kill(syscall.Getpid(), syscall.SIGKILL)
		}
		gVar.finished <- struct{}{}

	case "restart":
		//	warnf("%d/%d", gVar.cmd.Process.Pid, syscall.Getpid())
		//syscall.Kill(gVar.cmd.Process.Pid, syscall.SIGKILL)
		if gVar.isFunc == false {
			gVar.cmd.Process.Kill()
			gVar.cmd.Process.Release()
		}
	case "getpid":

	case "eval":

	case "debug":

	case "delayget":

	case "echo":
		fmt.Print(dat["data"])
	default:
	}
}

func handleRequest(conn net.Conn, gVar *RunOnceConf) {
	// Make a buffer to hold incoming data.
	defer conn.Close()
	tmp := make([]byte, 1024)
	data := make([]byte, 0)
	// Read the incoming connection into the buffer.
	length := 0

	// loop through the connection stream, appending tmp to data
	for {
		// read to the tmp var
		n, err := conn.Read(tmp)
		if err != nil {
			// log if not normal error
			if err != io.EOF {
				warnf("Read error - %s\n", err)
			}
			break
		} else {
			// append read data to full data
			data = append(data, tmp[:n]...)
			length += n
			break
		}

	}

	//	fmt.Println(length)
	//	cmdtable(data[:length])
	cmdtable(data, gVar)
	// Send a response back to person contacting us.
	conn.Write([]byte(data))
	// Close the connection when you're done with it.
}

func NewRunOnce(IpAddress, WorkDir string, port, timeout int, noBind, noLog, isFunc, loopCall bool, args []string) *RunOnceConf {
	return &RunOnceConf{
		cmd:       nil,
		IpAddress: IpAddress,
		WorkDir:   WorkDir,
		port:      port,
		timeout:   timeout,
		noBind:    noBind,
		noLog:     noLog,
		isFunc:    isFunc,
		loopCall:  loopCall,
		doneExec:  make(chan struct{}, 1),
		finished:  make(chan struct{}, 1),
		args:      args,
	}
}

func NewRunOnceFuncPort(port int) *RunOnceConf {
	return &RunOnceConf{
		cmd:       nil,
		IpAddress: "localhost",
		WorkDir:   "",
		port:      port,
		timeout:   1,
		noBind:    false,
		noLog:     true,
		isFunc:    true,
		loopCall:  false,
		doneExec:  make(chan struct{}, 1),
		finished:  make(chan struct{}, 1),
		args:      []string{},
	}
}

func NewRunOnceExecPort(port int, loopCall bool, args []string) *RunOnceConf {
	return &RunOnceConf{
		cmd:       nil,
		IpAddress: "localhost",
		WorkDir:   "",
		port:      port,
		timeout:   1,
		noBind:    false,
		noLog:     true,
		isFunc:    false,
		loopCall:  loopCall,
		doneExec:  make(chan struct{}, 1),
		finished:  make(chan struct{}, 1),
		args:      args,
	}
}

func Run(gVar *RunOnceConf) {
	//
	//	flag.StringVar(&gVar.IpAddress, "address", "127.0.0.1", "Address to listen on or to check")
	//	flag.StringVar(&gVar.WorkDir, "dir", "", "Working diretory")
	//	flag.IntVar(&gVar.port, "port", 0, "Port to listen on or to check")
	//	flag.IntVar(&gVar.timeout, "timeout", 1, "Timeout when checking. Default: 1 second.")
	//	flag.BoolVar(&gVar.noBind, "no-bind", false, "Do not bind on address:port specified")
	//	flag.BoolVar(&gVar.noLog, "no-log", false, "Do not print logs")
	//	flag.BoolVar(&gVar.loopCall, "loop", false, "Loop call command flag")
	//
	//	gVar.doneExec = make(chan struct{}, 1)
	//	flag.Parse()
	//	envtmp := os.Getenv("MP_PORT")
	//	os.Unsetenv("MP_PORT")
	//	if envtmp != "" && gVar.port == 0 {
	//		gVar.port, _ = strconv.Atoi(envtmp)
	//	}
	//
	//	envtmp = os.Getenv("MP_DIR")
	//	os.Unsetenv("MP_DIR")
	//	if envtmp != "" && gVar.WorkDir == "" {
	//		gVar.WorkDir = envtmp
	//	}
	//
	//	envtmp = os.Getenv("MP_ENLOG")
	//	os.Unsetenv("MP_ENLOG")
	//	if envtmp != "" && gVar.noLog == true {
	//		gVar.noLog = false
	//	}
	//

	if gVar.noBind {
		if sutils.IsPortAvailable(gVar.IpAddress, gVar.port, gVar.timeout) {
			warnf("Port is available. App is not running")
		} else {
			warnf("Port is not available. App is running?")
			os.Exit(1)
		}
	} else {
		l, err := net.Listen("tcp", fmt.Sprintf("%s:%d", gVar.IpAddress, gVar.port))

		if err != nil {
			warnf("Unable to bind on %s:%d. App is running?", gVar.IpAddress, gVar.port)
			os.Exit(1)
		}
		defer l.Close() //Close the connection if it's Open, otherwise doesn't do anything
		go func() {
			for {
				conn, err := l.Accept()

				if err != nil {
					// handle error
					warnf("Error tcp connections!")
					time.Sleep(500 * time.Millisecond)
				}
				// Make a buffer to hold incoming data.
				warnf("New tcp connections!")
				go handleRequest(conn, gVar)
			}
		}()

		warnf("Bind successfully on %s", l.Addr().String())
		gVar.port, _ = strconv.Atoi(strings.Split(l.Addr().String(), ":")[1])
	}

	if _, err := os.Stat(gVar.WorkDir); os.IsNotExist(err) {
		// path/to/whatever does not exist
		gVar.WorkDir = os.TempDir()
		warnf("Auto change workdir to '%s", gVar.WorkDir)
	}

	err := syscall.Chdir(gVar.WorkDir)
	if err != nil {
		warnf("Switching to '%s' got error '%s'", gVar.WorkDir, err)
		return
	}

	if len(gVar.args) < 1 {
		warnf("You must specify a command\n")
		return

	}

	execPath := gVar.args[0]
	cmdArgs := gVar.args[1:]

	execPath, err = exec.LookPath(execPath)

	if err != nil {
		warnf("Program is not exits: %s", execPath)
		return
	}
	if !gVar.noBind && !gVar.isFunc {
		warnf("Making sure all fd >= 3 is not close-on-exec")
		closeOnExec(false)
	}
	warnf("Now staring application '%s' %v from %s\n", execPath, cmdArgs, gVar.WorkDir)
	fmt.Println(gVar.port)

	if gVar.isFunc {
		return
	}

	go func() {
		for {
			//			sttyArgs := syscall.ProcAttr{
			//				"",
			//				syscall.Environ(),
			//				[]uintptr{os.Stdin.Fd(), os.Stdout.Fd(), os.Stderr.Fd()},
			//				nil,
			//			}
			//			pid, _ := syscall.ForkExec(execPath, cmdArgs, &sttyArgs)
			//			process, err := os.FindProcess(int(pid))
			//   			process.Wait()

			//err = syscall.Exec(execPath, cmdArgs, syscall.Environ())

			gVar.cmd = exec.Command(execPath, cmdArgs...)
			gVar.cmd.Env = os.Environ()
			gVar.cmd.SysProcAttr = &syscall.SysProcAttr{
				Pdeathsig:  syscall.SIGTERM,
				Foreground: false,
				Setsid:     true,
			}
			//			warnf("Staring application '%s'", execPath)
			err = gVar.cmd.Start()
			warnf("Wait finished pid: %d", gVar.cmd.Process.Pid)
			err = gVar.cmd.Wait()
			//			warnf("%t:%s", gVar.loopCall, err.Error())
			// && gVar.cmd.ProcessState.ExitCode() == -1

			if gVar.loopCall && ((err != nil && "signal: killed" == err.Error()) || (err == nil)) {
				warnf("Reload program: %s[%d]", execPath, gVar.cmd.ProcessState.Pid())
			} else if err != nil {
				warnf("Executing got error '%s'", err.Error())
				gVar.finished <- struct{}{}
				return
			}

			//			gVar.doneExec <- struct{}{}
			warnf("Done staring application '%s'", execPath)
			time.Sleep(1 * time.Second)
		}

	}()
	_ = <-gVar.finished

}
