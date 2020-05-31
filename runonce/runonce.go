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
	"errors"
	"fmt"
	"github.com/getlantern/byteexec"
	//	"github.com/getlantern/daemon"

	"github.com/sonnt85/gosutils/daemon"
	"github.com/sonnt85/gosutils/sutils"

	"context"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type Runtype int

const (
	RUNFUNC Runtype = iota
	RUNBYTES
	RUNFILE
)

func (rt Runtype) String() string {
	return [...]string{"RUNFUNC", "RUNBYTES", "RUNFILE"}[rt]
}

type RunOnceConf struct {
	IpAddress                                       string
	WorkDir                                         string
	runtype                                         Runtype
	Port                                            int
	PortRuntime                                     int
	timeout                                         int
	noBind                                          bool
	NoLog                                           bool
	LoopCall                                        bool
	cmd                                             *exec.Cmd
	Args                                            []string
	exebytes                                        []byte
	Exename, ExeFullPathRuntime, ExeRealNameRuntime string

	doneExec chan struct{}
	ctx      context.Context
	PATH     string
	cancel   context.CancelFunc
}

func (gVar *RunOnceConf) Reset() {
	gVar.IpAddress = ""
	gVar.WorkDir = ""
	gVar.runtype = RUNFUNC
	gVar.Port = 0
	gVar.PortRuntime = 0
	gVar.timeout = 0
	gVar.noBind = false
	gVar.NoLog = false
	gVar.LoopCall = false
	gVar.cmd = nil
	gVar.Args = []string{}
	gVar.exebytes = []byte{}
	gVar.Exename = ""
	gVar.ExeRealNameRuntime = ""

	//	gVar.doneExec chan struct{}
	//	gVar.ctx      context.Context
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

func (gVar *RunOnceConf) cmdtable(buf []byte, conn net.Conn) {
	//	var dat map[string]interface{}
	var dat map[string]string

	//	fmt.Println(len(buf))
	if err := json.Unmarshal(buf[:len(buf)], &dat); err != nil {
		cmd := strings.TrimRight(string(buf), "\r\n")
		gVar.Log("cmd: '%s'", cmd)

		switch cmd {
		case "getpid":
			conn.Write([]byte(strconv.Itoa(syscall.Getpid())))
		case "ping":
			conn.Write([]byte("pong"))

		default:
		}
		return
	}
	//json parser pass with private command
	var cmd = dat["cmd"]
	var token = dat["token"]
	if token != strconv.Itoa(syscall.Getpid()) {
		gVar.Log("Missing token!")
		return
	}

	switch cmd {
	case "quit":
		if gVar.runtype != RUNFUNC {
			gVar.cancel()
			//syscall.Kill(syscall.Getpid(), syscall.SIGKILL)
		}
		gVar.cancel()

	case "restart":
		//gVar.Log("%d/%d", gVar.cmd.Process.Pid, syscall.Getpid())
		//syscall.Kill(gVar.cmd.Process.Pid, syscall.SIGKILL)
		if gVar.runtype != RUNFUNC {
			gVar.cmd.Process.Signal(syscall.SIGTERM)
			time.Sleep(time.Microsecond * 100)
			gVar.cmd.Process.Kill()
			gVar.cmd.Process.Release()
		}
	case "eval":

	case "debug":

	case "delayget":

	case "echo":
		data := dat["data"]
		conn.Write([]byte(data))
		return
	default:
	}
	//	conn.Write([]byte(data))
}

func (gVar *RunOnceConf) handleRequest(conn net.Conn) {
	// Make a buffer to hold incoming data.
	defer conn.Close()
	var isClosed = false
	data := make([]byte, 1024)
	// Read the incoming connection into the buffer.
	// loop through the connection stream, appending tmp to data
	for {

		n, err := conn.Read(data)
		//		gVar.Log("New data from client")
		if err == io.EOF {
			isClosed = true
		}

		gVar.cmdtable(data[:n], conn)

		if isClosed {
			return
		}
	}
}

func NewRunOnce(IpAddress, WorkDir string, port, timeout int, noBind, NoLog, LoopCall bool, runtype Runtype, args []string, exebytes []byte) *RunOnceConf {
	return &RunOnceConf{
		cmd:       nil,
		IpAddress: IpAddress,
		WorkDir:   WorkDir,
		Port:      port,
		timeout:   timeout,
		noBind:    noBind,
		NoLog:     NoLog,
		LoopCall:  LoopCall,
		doneExec:  make(chan struct{}, 1),
		Args:      args,
		runtype:   runtype,
		exebytes:  exebytes,
		//		ctx:       context.WithCancel(context.Background()),
	}
}

func NewRunOnceFuncPort(port int) *RunOnceConf {
	return &RunOnceConf{
		cmd:       nil,
		IpAddress: "localhost",
		WorkDir:   "",
		Port:      port,
		timeout:   1,
		noBind:    false,
		NoLog:     true,
		LoopCall:  false,
		runtype:   RUNFUNC,
		doneExec:  make(chan struct{}, 1),
		Args:      []string{},
		//		ctx:       context.WithCancel(context.Background()),
	}
}

func NewRunOnceExecPort(port int, LoopCall bool, args []string) *RunOnceConf {
	return &RunOnceConf{
		cmd:       nil,
		IpAddress: "localhost",
		WorkDir:   "",
		Port:      port,
		timeout:   1,
		noBind:    false,
		NoLog:     true,
		runtype:   RUNFILE,
		LoopCall:  LoopCall,
		doneExec:  make(chan struct{}, 1),
		Args:      args,
		//		ctx:       context.WithCancel(context.Background()),
	}
}

func NewRunOnceBytesPort(port int, LoopCall bool, exebytes []byte) *RunOnceConf {
	return &RunOnceConf{
		cmd:       nil,
		IpAddress: "localhost",
		WorkDir:   "",
		Port:      port,
		timeout:   1,
		noBind:    false,
		NoLog:     true,
		runtype:   RUNBYTES,
		LoopCall:  LoopCall,
		doneExec:  make(chan struct{}, 1),
		exebytes:  exebytes,
		//		ctx:       context.WithCancel(context.Background()),
	}
}

func (gVar *RunOnceConf) Log(format string, a ...interface{}) {
	if !gVar.NoLog {
		fmt.Fprintln(os.Stderr, "[", time.Now().Format(time.StampMilli), "]", fmt.Sprintf(format, a...))
	}
}

func (gVar *RunOnceConf) GenerateCmd() (err error) {
	if gVar.runtype == RUNBYTES {
		be, err := byteexec.New(gVar.exebytes, gVar.Exename)
		if err != nil {
			return err
		}
		gVar.ExeRealNameRuntime = be.Filename
		gVar.cmd = be.Command(gVar.Args...)

	} else {
		if len(gVar.Args) < 1 {
			gVar.Log("You must specify a command\n")
			return errors.New("Need has command arguments")
		}

		execPathOrg := gVar.Args[0]
		cmdArgs := gVar.Args[1:]

		gVar.ExeFullPathRuntime, err = exec.LookPath(execPathOrg)

		if err != nil {
			gVar.Log("Program is not exits: %s", execPathOrg)
			return err
		}
		gVar.ExeRealNameRuntime = execPathOrg
		os.Setenv(sutils.PathGetEnvPathKey(), gVar.PATH)
		if len(gVar.Exename) > 0 {
			tmppath := sutils.TempFileCreateInNewTemDir(gVar.Exename)
			if len(tmppath) != 0 {
				if os.Symlink(gVar.ExeFullPathRuntime, tmppath) == nil {
					//					gVar.Log("Use fake name")
					os.Setenv(sutils.PathGetEnvPathKey(), sutils.PathJointList(gVar.PATH, path.Dir(tmppath)))
					//					os.Setenv("PATH", gVar.PATH+":"+path.Dir(tmppath))
					gVar.ExeFullPathRuntime = tmppath
					gVar.ExeRealNameRuntime = gVar.Exename
					//					execPathOrg = gVar.Exename
				}
			}
		}

		gVar.cmd = exec.Command(gVar.ExeRealNameRuntime, cmdArgs...)
	}
	return nil
}

func (gVar *RunOnceConf) Poll() bool {

	if retbytes, err := sutils.NetTCPClientSend(fmt.Sprintf("localhost:%d", gVar.Port), []byte("ping")); err == nil {
		//		gVar.Log("retbytes: %s", retbytes)
		if string(retbytes) == "pong" {
			gVar.Log("App is running....")
			return true
		} else {
			gVar.Log("Port is using but not for app")
		}
	}

	return false
}

func (gVar *RunOnceConf) Run() (err error) {
	//
	//	flag.StringVar(&gVar.IpAddress, "address", "127.0.0.1", "Address to listen on or to check")
	//	flag.StringVar(&gVar.WorkDir, "dir", "", "Working diretory")
	//	flag.IntVar(&gVar.Port, "port", 0, "Port to listen on or to check")
	//	flag.IntVar(&gVar.timeout, "timeout", 1, "Timeout when checking. Default: 1 second.")
	//	flag.BoolVar(&gVar.noBind, "no-bind", false, "Do not bind on address:port specified")
	//	flag.BoolVar(&gVar.NoLog, "no-log", false, "Do not print logs")
	//	flag.BoolVar(&gVar.LoopCall, "loop", false, "Loop call command flag")
	//
	//	gVar.doneExec = make(chan struct{}, 1)
	//	flag.Parse()
	//	envtmp := os.Getenv("MP_PORT")
	//	os.Unsetenv("MP_PORT")
	//	if envtmp != "" && gVar.Port == 0 {
	//		gVar.Port, _ = strconv.Atoi(envtmp)
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
	//	if envtmp != "" && gVar.NoLog == true {
	//		gVar.NoLog = false
	//	}
	//
	gVar.PATH = sutils.PathGetEnvPathValue()
	gVar.ctx, gVar.cancel = context.WithCancel(context.Background())
	defer func() {
		if gVar.ctx.Err() == nil {
			gVar.cancel()
		}
	}()

	if gVar.noBind {
		if sutils.IsPortAvailable(gVar.IpAddress, gVar.Port, gVar.timeout) {
			gVar.Log("Port is available. App is not running")
		} else {
			if gVar.Poll() {
				os.Exit(1)
				return errors.New("App is running....")
			} else {
				return errors.New("Port is used but not for app!")
			}
		}
	} else {
		l, err := net.Listen("tcp", fmt.Sprintf("%s:%d", gVar.IpAddress, gVar.Port))

		if err != nil {
			if gVar.Poll() {
				os.Exit(1)
				return errors.New("App is running....")
			} else {
				return errors.New("Port is used but not for app!")
			}

			//			gVar.Log("Unable to bind on %s:%d. App is running?", gVar.IpAddress, gVar.Port)
			//			return errors.New("Can not bind to listens")
		}
		if gVar.runtype != RUNFUNC {
			defer l.Close() //Close the connection if it's Open, otherwise doesn't do anything
		}
		go func() {
			for {
				//				select {
				//				case <-gVar.ctx.Done():
				//					return gVar.ctx.Err()
				//				case
				conn, err := l.Accept()
				if err != nil {
					return
					//					continue
				}
				// Make a buffer to hold incoming data.
				//				gVar.Log("New tcp connections!")
				go gVar.handleRequest(conn)
				//				}
			}
		}()

		gVar.Log("Bind successfully on %s", l.Addr().String())
		gVar.PortRuntime, _ = strconv.Atoi(strings.Split(l.Addr().String(), ":")[1])
	}

	if _, err := os.Stat(gVar.WorkDir); os.IsNotExist(err) {
		// path/to/whatever does not exist
		gVar.WorkDir = os.TempDir()
		gVar.Log("Auto change workdir to '%s", gVar.WorkDir)
	}

	err = syscall.Chdir(gVar.WorkDir)
	if err != nil {
		gVar.Log("Switching to '%s' got error '%s'", gVar.WorkDir, err)
		return errors.New("Can not change workdir")
	}

	if gVar.runtype == RUNFUNC {
		return nil
	} else if gVar.runtype == RUNFILE || gVar.runtype == RUNBYTES {
		if gVar.runtype == RUNBYTES {
			defer os.Remove(gVar.ExeRealNameRuntime)
		}

		if !gVar.noBind {
			gVar.Log("Making sure all fd >= 3 is not close-on-exec")

			closeOnExec(false)
		}

		gVar.Log("Now staring application monitor exec on port %d", gVar.PortRuntime)

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
				if err = gVar.GenerateCmd(); err != nil {
					return
				}

				gVar.cmd.Env = os.Environ()
				gVar.cmd.SysProcAttr = &syscall.SysProcAttr{
					Pdeathsig:  syscall.SIGTERM,
					Foreground: false,
					Setsid:     true,
					//					Setpgid:    true, // create group pid
				}
				//			gVar.Log("Staring application '%s'", execPath)
				err = gVar.cmd.Start()                                                                       //start no block
				if len(gVar.Exename) != 0 && path.Base(gVar.ExeRealNameRuntime) == path.Base(gVar.Exename) { //fake name
					os.RemoveAll(path.Dir(gVar.ExeFullPathRuntime))
				}

				gVar.Log("Reload program: %s %+q [%d]", gVar.ExeFullPathRuntime, gVar.Args[:], gVar.cmd.Process.Pid)

				//				gVar.Log("Wait finished pid: %d", gVar.cmd.Process.Pid)
				//				err = gVar.cmd.Wait() //block waiting finish or error
				go func() {
					err = gVar.cmd.Wait()
					gVar.doneExec <- struct{}{}
				}()

				select {
				case <-gVar.ctx.Done():
					if !sutils.IsProcessAlive(gVar.cmd.Process.Pid) {
						//					if gVar.cmd.ProcessState != nil {
						gVar.cmd.Process.Kill()
					} else {
						if !gVar.cmd.ProcessState.Success() {
							gVar.Log("Program error exit")
						}
					}
					//					gVar.Log("Done excute application '%s'", gVar.ExeRealNameRuntime)
					return
				case <-gVar.doneExec:
					if gVar.LoopCall {
						gVar.Log("Done excute application '%s'", gVar.ExeFullPathRuntime)
						gVar.cmd.Process.Release()
						time.Sleep(1 * time.Second)
						continue
					} else {
						gVar.cancel()
						return
					}
				}
			}
		}()
		_ = <-gVar.ctx.Done()
		time.Sleep(time.Millisecond * 100)
	}
	return nil
}

func NewAndRunOnce(port int) (*RunOnceConf, error) {
	econf := NewRunOnceFuncPort(port)
	//	econf.NoLog = false
	return econf, econf.Run()
}

func RebornNewProgram(port int, newname string) bool {
	var cntxt = &daemon.Context{
		//	PidFileName: "/tmp/sample.pid",
		PidFilePerm: 0644,
		LogFileName: "/tmp/sample.log",
		LogFilePerm: 0640,
		WorkDir:     "./",
		Umask:       027,
		NameProg:    newname,
		//	Args:        []string{"[go-daemon sample]"},
	}

	child, err := cntxt.Reborn()

	//	if err != nil && child == nil {
	//		if port != 0 {
	//			NewAndRunOnce(port)
	//		}
	//	}
	//	fmt.Println(child, err)
	if err != nil { //error
		//		fmt.Println("Unable to run: ", err)
		if port != -1 {
			NewAndRunOnce(port)
		}
		return false
	}

	if child != nil { //current is parrent [firt call]
		//			updateDeamon()
		return true //exit parrent
	} else {
		//current is child run [secon call]
		//		defer cntxt.Release()
		if port != -1 {
			NewAndRunOnce(port)
		}
		return false
	}
}
