package runonce

import (
	"encoding/json"
	//	"flag"
	"errors"
	"fmt"

	//	"github.com/getlantern/byteexec"
	//	"github.com/getlantern/daemon"

	"github.com/sonnt85/gosutils/sutils"

	"context"
	"io"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"path"

	log "github.com/sirupsen/logrus"

	//	"path/filepath"

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
	gVar.LoopCall = false
	gVar.cmd = nil
	gVar.Args = []string{}
	gVar.exebytes = []byte{}
	gVar.Exename = ""
	gVar.ExeRealNameRuntime = ""

	//	gVar.doneExec chan struct{}
	//	gVar.ctx      context.Context
}

func (gVar *RunOnceConf) cmdtable(buf []byte, conn net.Conn) {
	//	var dat map[string]interface{}
	var dat map[string]string

	//	log.Println(len(buf))
	if err := json.Unmarshal(buf[:len(buf)], &dat); err != nil {
		cmd := strings.TrimRight(string(buf), "\r\n")
		log.Printf("cmd: '%s'", cmd)

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
		log.Errorln("Missing token!")
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
		//log.Printf("%d/%d", gVar.cmd.Process.Pid, syscall.Getpid())
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
		//		log.Printf("New data from client")
		if err == io.EOF {
			isClosed = true
		}

		gVar.cmdtable(data[:n], conn)

		if isClosed {
			return
		}
	}
}

func NewRunOnce(IpAddress, WorkDir string, port, timeout int, noBind, LoopCall bool, runtype Runtype, args []string, exebytes []byte) *RunOnceConf {
	return &RunOnceConf{
		cmd:       nil,
		IpAddress: IpAddress,
		WorkDir:   WorkDir,
		Port:      port,
		timeout:   timeout,
		noBind:    noBind,
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
		runtype:   RUNBYTES,
		LoopCall:  LoopCall,
		doneExec:  make(chan struct{}, 1),
		exebytes:  exebytes,
		//		ctx:       context.WithCancel(context.Background()),
	}
}

func (gVar *RunOnceConf) GenerateCmd() (err error) {
	if gVar.runtype == RUNBYTES {

		rootdir, err := ioutil.TempDir("", "system")
		if err != nil {
			return err
		}

		filePath := path.Join(rootdir, gVar.Exename)
		err = ioutil.WriteFile(filePath, gVar.exebytes, 0755)

		if err != nil {
			log.Errorf("Can not create new file to run: %v", err)
			return err
		}

		os.Setenv(sutils.PathGetEnvPathKey(), sutils.PathJointList(sutils.PathGetEnvPathValue(), path.Dir(filePath)))

		if err != nil {
			return err
		}
		gVar.ExeRealNameRuntime = filePath
		gVar.cmd = exec.Command(gVar.Exename, gVar.Args...)

	} else {
		if len(gVar.Args) < 1 {
			log.Println("You must specify a command\n")
			return errors.New("Need has command arguments")
		}

		execPathOrg := gVar.Args[0]
		cmdArgs := gVar.Args[1:]

		gVar.ExeFullPathRuntime, err = exec.LookPath(execPathOrg)

		if err != nil {
			log.Errorf("Program is not exits: %s", execPathOrg)
			return err
		}
		gVar.ExeRealNameRuntime = execPathOrg
		os.Setenv(sutils.PathGetEnvPathKey(), gVar.PATH)
		if len(gVar.Exename) > 0 {
			tmppath := sutils.TempFileCreateInNewTemDir(gVar.Exename)
			if len(tmppath) != 0 {
				if os.Symlink(gVar.ExeFullPathRuntime, tmppath) == nil {
					//					log.Println("Use fake name")
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
		//		log.Printf("retbytes: %s", retbytes)
		if string(retbytes) == "pong" {
			log.Warningln("App is running....")
			return true
		} else {
			log.Warningln("Port is using but not for this app")
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
			log.Warningln("Port is available. App is not running")
		} else {
			if gVar.Poll() {
				log.Errorln("App is running....\nExit!")
				os.Exit(1)
				//				return errors.New("App is running....")
			} else {
				return errors.New("Port is used but not for this app!")
			}
		}
	} else {
		l, err := net.Listen("tcp", fmt.Sprintf("%s:%d", gVar.IpAddress, gVar.Port))

		if err != nil {
			if gVar.Poll() {
				log.Errorln("App is running....\nExit!")
				os.Exit(1)
			} else {
				return errors.New("Port is used but not for this app!")
			}

			//			log.Printf("Unable to bind on %s:%d. App is running?", gVar.IpAddress, gVar.Port)
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
				//				log.Printf("New tcp connections!")
				go gVar.handleRequest(conn)
				//				}
			}
		}()

		log.Infof("Bind successfully on %s\n", l.Addr().String())
		gVar.PortRuntime, _ = strconv.Atoi(strings.Split(l.Addr().String(), ":")[1])
	}

	if _, err := os.Stat(gVar.WorkDir); os.IsNotExist(err) {
		// path/to/whatever does not exist
		gVar.WorkDir = os.TempDir()
		log.Infof("Auto change workdir to '%s", gVar.WorkDir)
	}

	err = syscall.Chdir(gVar.WorkDir)
	if err != nil {
		log.Errorf("Switching to '%s' got error '%s'", gVar.WorkDir, err)
		return errors.New("Can not change workdir")
	}

	if gVar.runtype == RUNFUNC {
		return nil
	} else if gVar.runtype == RUNFILE || gVar.runtype == RUNBYTES {
		if gVar.runtype == RUNBYTES {
			defer os.Remove(gVar.ExeRealNameRuntime)
		}

		if !gVar.noBind {
			log.Infoln("Making sure all fd >= 3 is not close-on-exec")

			closeOnExec(false)
		}

		log.Infof("Now staring application monitor exec on port %d", gVar.PortRuntime)

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
				//					Pdeathsig:  syscall.SIGTERM,

				gVar.cmd.SysProcAttr = &syscall.SysProcAttr{
					//					Foreground: false,
					//					Setsid: true,
					//					Setpgid:    true, // create group pid
				}
				//			log.Printf("Staring application '%s'", execPath)
				err = gVar.cmd.Start()                                                                       //start no block
				if len(gVar.Exename) != 0 && path.Base(gVar.ExeRealNameRuntime) == path.Base(gVar.Exename) { //fake name
					os.Setenv(sutils.PathGetEnvPathKey(), sutils.PathRemove(sutils.PathGetEnvPathValue(), path.Dir(gVar.ExeFullPathRuntime)))
					os.RemoveAll(path.Dir(gVar.ExeFullPathRuntime))
				}

				log.Infof("Reload program: %s %+q [%d]\n", gVar.ExeFullPathRuntime, gVar.Args[:], gVar.cmd.Process.Pid)

				//				log.Printf("Wait finished pid: %d", gVar.cmd.Process.Pid)
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
							log.Errorln("Program error exit")
						}
					}
					//					log.Printf("Done excute application '%s'", gVar.ExeRealNameRuntime)
					return
				case <-gVar.doneExec:
					if gVar.LoopCall {
						log.Infof("Done excute application '%s'", gVar.ExeFullPathRuntime)
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
	return econf, econf.Run()
}
