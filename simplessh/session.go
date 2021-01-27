package simplessh

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"unsafe"

	"github.com/kr/pty"
	"golang.org/x/crypto/ssh"
)

// Request types used in sessions - RFC 4254 6.X
const (
	SessionRequest               = "session"       // RFC 4254 6.1 login ssh
	PTYRequest                   = "pty-req"       // RFC 4254 6.2
	X11Request                   = "x11-req"       // RFC 4254 6.3.1
	X11ChannelRequest            = "x11"           // RFC 4254 6.3.2
	EnvironmentRequest           = "env"           // RFC 4254 6.4
	ShellRequest                 = "shell"         // RFC 4254 6.5
	ExecRequest                  = "exec"          // RFC 4254 6.5
	SubsystemRequest             = "subsystem"     // RFC 4254 6.5
	WindowDimensionChangeRequest = "window-change" // RFC 4254 6.7
	FlowControlRequest           = "xon-off"       // RFC 4254 6.8
	SignalRequest                = "signal"        // RFC 4254 6.9
	ExitStatusRequest            = "exit-status"   // RFC 4254 6.10
	ExitSignalRequest            = "exit-signal"   // RFC 4254 6.10
)

// SessionHandler returns a ChannelHandler that implements standard SSH Sessions for PTY, shell, and exec capabilities
func SessionHandler() ChannelHandler { return ChannelHandlerFunc(SessionChannel) }

// SessionChannel acts as an SSH Session ChannelHandler
func SessionChannel(newChannel ssh.NewChannel, channel ssh.Channel, reqs <-chan *ssh.Request, sshConn ssh.Conn) {
	// Get system shell
	shell := os.Getenv("SHELL")
	c := exec.Command(shell)
	f, err := pty.Start(c)
	if err != nil {
		logger.Printf("Unable to start shell: %s", shell)
		return
	}

	terminalModes := ssh.TerminalModes{}
	// TODO: Find out if I should do anything with the terminal modes. Do any of the ptys/ttys know what to do with them

	close := func() {
		channel.Close()
		err := c.Wait()
		if err != nil {
			logger.Printf("failed to exit bash (%s)", err)
		}
		logger.Printf("session closed")
	}

	go func(in <-chan *ssh.Request) {

		env := []string{}
		var command *exec.Cmd

		for req := range in {
			ok := false

			switch req.Type {
			case PTYRequest:

				ok = true
				pty := ptyReq{}
				err = ssh.Unmarshal(req.Payload, &pty)
				if err != nil {
					logger.Printf("Unable to decode pty request: %s", err.Error())
				}

				setWinsize(f.Fd(), pty.Width, pty.Height)
				logger.Printf("pty-req '%s'", pty.Term)

				type termModeStruct struct {
					//Key byte
					Key uint8
					Val uint32
				}
				termModes := []termModeStruct{}
				working := []byte(pty.TermModes)
				for {
					if len(working) < 5 {
						break
					}
					tm := termModeStruct{}

					tm.Key = working[0]
					tm.Val = binary.BigEndian.Uint32(working[1:5])

					/*

						err = ssh.Unmarshal(working, &tm)
						if err != nil {
							log.Println(err.Error())
							break
						}
					*/

					termModes = append(termModes, tm)
					terminalModes[tm.Key] = tm.Val
					working = working[5:]

				}
				go CopyReadWriters(channel, f, close)

			case WindowDimensionChangeRequest:
				win := windowDimensionReq{}
				err = ssh.Unmarshal(req.Payload, &win)
				if err != nil {
					logger.Printf("Error reading window dimension change request: %s", err.Error())
				}
				setWinsize(f.Fd(), win.Width, win.Height)
				continue //no response according to RFC 4254 6.7
			case ShellRequest: // Shell requests should not have a payload - RFC 4254 6.7
				ok = true
				go CopyReadWriters(channel, f, close)

			case ExecRequest:
				ok = true
				var cmd execRequest
				err = ssh.Unmarshal(req.Payload, &cmd)
				if err != nil {
					continue
				}

				command = exec.Command("sh", "-c", cmd.Command) // Let shell do the parsing
				logger.Printf("exec starting: %s", cmd.Command)
				//c.Env = append(c.Env, env...)

				exitStatus := exitStatusReq{}

				fd, err := pty.Start(command)
				if err != nil {
					logger.Printf("Unable to wrap exec command in pty\n")
					return
				}

				execClose := func() {
					channel.Close()
					logger.Printf("exec finished: %s", cmd.Command)
				}

				defer fd.Close()
				go CopyReadWriters(channel, fd, execClose)
				err = command.Wait()

				/*
					command.Stdout = channel
					command.Stderr = channel
				*/

				//command.Stdin = channel // TODO: test how stdin works on exec on openssh server
				//err = command.Run()
				if err != nil {
					logger.Printf("Error running exec : %s", err.Error())
					e, ok := err.(*exec.ExitError)
					errVal := 1
					if ok {
						status := e.Sys().(syscall.WaitStatus)
						if status.Exited() {
							errVal = status.ExitStatus()
							exitStatus.ExitStatus = uint32(errVal)
							channel.SendRequest(ExitStatusRequest, false, ssh.Marshal(exitStatus))
						} else if status.Signaled() { // What is the difference between Siglnal and StopSignal
							e := exitSignalReq{}
							e.SignalName = status.Signal().String()
							e.CoreDumped = status.CoreDump()
							// TODO: Figure out other two fields
							channel.SendRequest(ExitSignalRequest, false, ssh.Marshal(e))
						}

					}

				} else {

					channel.SendRequest(ExitStatusRequest, false, ssh.Marshal(exitStatus))
				}

				req.Reply(ok, nil)
				close()
				return

			case EnvironmentRequest:
				ok = true
				e := envReq{}
				err = ssh.Unmarshal(req.Payload, &e)
				if err != nil {
					continue
				}

				env = append(env, fmt.Sprintf("%s=%s", e.Name, e.Value))

			case SignalRequest:
				ok = true
				sig := signalRequest{}
				ssh.Unmarshal(req.Payload, &sig)
				logger.Println("Received Signal: ", sig.Signal)

				s := signalsMap[sig.Signal]
				if command != nil {

					command.Process.Signal(s)
				} else {
					c.Process.Signal(s)
				}
			}
			req.Reply(ok, nil)

		}

	}(reqs)

}

var signalsMap = map[ssh.Signal]os.Signal{
	ssh.SIGABRT: syscall.SIGABRT,
	ssh.SIGALRM: syscall.SIGALRM,
	ssh.SIGFPE:  syscall.SIGFPE,
	ssh.SIGHUP:  syscall.SIGHUP,
	ssh.SIGILL:  syscall.SIGILL,
	ssh.SIGINT:  syscall.SIGINT,
	ssh.SIGKILL: syscall.SIGKILL,
	ssh.SIGPIPE: syscall.SIGPIPE,
	ssh.SIGQUIT: syscall.SIGQUIT,
	ssh.SIGSEGV: syscall.SIGSEGV,
	ssh.SIGTERM: syscall.SIGTERM,
	ssh.SIGUSR1: syscall.SIGUSR1,
	ssh.SIGUSR2: syscall.SIGUSR2,
}

// CopyReadWriters copies biderectionally - output from a to b, and output of b into a. Calls the close function when unable to copy in either direction
func CopyReadWriters(a, b io.ReadWriter, close func()) {
	var once sync.Once
	go func() {
		io.Copy(a, b)
		once.Do(close)
	}()

	go func() {
		io.Copy(b, a)
		once.Do(close)
	}()
}

// windowDimension represents channel request for window dimension change - RFC 4254 6.7
type windowDimensionReq struct {
	Width       uint32
	Height      uint32
	WidthPixel  uint32
	HeightPixel uint32
}

// ptyReq represents the channel request for a PTY. RFC 4254 6.2
type ptyReq struct {
	Term        string
	Width       uint32
	Height      uint32
	WidthPixel  uint32
	HeightPixel uint32
	TermModes   string
}

// envReq represents an "env" channel request - RFC 4254 6.4
type envReq struct {
	Name  string
	Value string
}

// execRequest represents an "exec" channel request - RFC 4254 6.5
type execRequest struct {
	Command string
}

// signalRequest represents a "signal" session channel request - RFC 4254 6.9
type signalRequest struct {
	Signal ssh.Signal
}

// exitStatusReq represents an exit status for "exec" requests - RFC 4254 6.10
type exitStatusReq struct {
	ExitStatus uint32
}

// exitSignalReq represents an exit signal for "exec" requests - RFC 4254 6.10
type exitSignalReq struct {
	SignalName   string
	CoreDumped   bool
	ErrorMessage string
	LanguageTag  string
}

// winsize stores the Height and Width of a terminal in rows/columns and pixels - for syscall - http://linux.die.net/man/4/tty_ioctl
type winsize struct {
	Row    uint16
	Col    uint16
	XPixel uint16 // unused
	YPixel uint16 // unused
}

// SetWinsize uses syscall to set pty window size
func setWinsize(fd uintptr, w, h uint32) {
	logger.Printf("Resize Window to %dx%d", w, h)
	ws := &winsize{Col: uint16(w), Row: uint16(h)}
	syscall.Syscall(syscall.SYS_IOCTL, fd, uintptr(syscall.TIOCSWINSZ), uintptr(unsafe.Pointer(ws)))
}
