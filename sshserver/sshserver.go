package sshserver

import (
	//	"fmt"
	//	"encoding/hex"

	log "github.com/sirupsen/logrus"

	gossh "github.com/gliderlabs/ssh"
	//	sw "github.com/sonnt85/gosutils/shellwords"
	"github.com/sonnt85/gosutils/sutils"
	"golang.org/x/crypto/ssh"

	//	"github.com/creack/pty"
	"github.com/sonnt85/gosutils/pty"

	//	"github.com/sonnt85/gosutils/simplessh"
	//	"golang.org/x/crypto/ssh"
	//	"sync"
	//	"bufio"
	"bytes"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"

	"crypto/rand"
	"io"
	"io/ioutil"

	//	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"
)

// Server wraps an SSH Client
type Server struct {
	gossh.Server
	//	config                     *ssh.ServerConfig
	Pubkeys                      string
	User, Password, AddresListen string
}

// exitStatusReq represents an exit status for "exec" requests - RFC 4254 6.10
type exitStatusReq struct {
	ExitStatus uint32
}

var SSHServer *Server

//func setWinsizeTerminal(f *os.File, w, h int) {
//	syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), uintptr(syscall.TIOCSWINSZ),
//		uintptr(unsafe.Pointer(&struct{ h, w, x, y uint16 }{uint16(h), uint16(w), 0, 0})))
//}

//"shell", "exec":

func sshSessionShellExecHandle(s gossh.Session) {
	//	 var cmd *exec.Cmd
	debugEnable := false

	commands := s.Command()

	var cmd *exec.Cmd
	var err error
	ptyReq, winCh, isPty := s.Pty()
	shellbin := ""
	shellrunoption := ""
	//	os.Getenv("SHELL")
	s.Permissions()
	if len(shellbin) == 0 {
		if runtime.GOOS == "windows" {
			shellrunoption = "/c"
			if shellbin = os.Getenv("COMSPEC"); shellbin == "" {
				shellbin = "cmd"
			}
		} else { //linux
			shellrunoption = "-c"
			shellbin = os.Getenv("SHELL")
			for _, v := range []string{"bash", "sh"} {
				if _, err := exec.LookPath(v); err == nil {
					shellbin = v
					break
				}
			}
		}
	}
	if isPty || (runtime.GOOS == "windows" && (len(commands) == 0)) { //shell
		var f *os.File
		log.Printf("\nShell start %s[%s] ...\n", shellbin, ptyReq.Term)

		cmd = exec.Command(shellbin)
		cmd.Dir = sutils.GetHomeDir()
		cmd.Env = append(cmd.Env, "TERM="+ptyReq.Term, sutils.PathGetEnvPathKey()+"="+sutils.PathGetEnvPathValue())
		f, err := pty.Start(cmd) //start command via pty
		if err != nil {
			log.Errorln("Can not start shell with tpy: ", err)
			return
		}
		//		execClose := func() {
		//			s.Close()
		//		}
		defer f.Close()
		if debugEnable {
			go sutils.TeeReadWriterOsFile(f, s.(io.ReadWriter), s.Stderr(), os.Stdout, nil)
		} else {
			go sutils.CopyReadWriters(f, s, nil)
		}

		go func() { //auto resize
			for win := range winCh {
				//				pty.Setsize(f, win)
				pty.SetWinsizeTerminal(f, win.Width, win.Height)
			}
			//			log.Infoln("Exit setWinsizeTerminal")
		}()

	} else {
		if commands[0] == "command" {
			//			log.Printf("commanraw:[%+v]", s.RawCommand())
			if runtime.GOOS == "windows" {
				if commands[1] == "ls" {
					commands = append([]string{shellbin, shellrunoption}, "dir /b", commands[3])
				} else if commands[1] == "pwd" {
					commands = append([]string{shellbin, shellrunoption}, "echo %cd%", commands[3])
				}
			} else {
				commands = append([]string{shellbin, shellrunoption}, s.RawCommand())
			}
		}

		if commands[0] == "pwd" && runtime.GOOS == "windows" {
			commands = append([]string{shellbin, shellrunoption}, "echo %cd%", commands[3])
		}

		log.Infof("\nexec start: %v\n", commands)
		cmd = exec.Command(commands[0], commands[1:]...)
		cmd.Env = append(cmd.Env, sutils.PathGetEnvPathKey()+"="+sutils.PathGetEnvPathValue())

		cmd.Dir = sutils.GetHomeDir()

		if debugEnable {
			if false { //use pty for any command
				cmd.Env = append(cmd.Env, "TERM=xterm", sutils.PathGetEnvPathKey()+"="+sutils.PathGetEnvPathValue())

				f, err := pty.Start(cmd) //start command via pty
				if err != nil {
					log.Errorln("Can not start shell with tpy: ", err)
					return
				}
				defer f.Close()

				go func() { //auto resize
					for win := range winCh {
						pty.SetWinsizeTerminal(f, win.Width, win.Height)
					}
					log.Infoln("Exit setWinsizeTerminal")
				}()
				go sutils.TeeReadWriterOsFile(f, s.(io.ReadWriter), s.Stderr(), os.Stdout, nil)
			} else {
				if nil != sutils.TeeReadWriterCmd(cmd, s.(io.ReadWriter), s.Stderr(), os.Stdout, nil) { //alredy gorountine {
					log.Errorf("Can not start TeeReadWriterCmd: %v\n", err)
					return
				}
				err = cmd.Start() //start command
			}
		} else {
			cmd.Stderr = s.Stderr()
			cmd.Stdout = s
			inputWriter, err := cmd.StdinPipe()
			if err != nil {
				return
			}
			go func() {
				io.Copy(inputWriter, s)
				inputWriter.Close()
			}()
			err = cmd.Start() //start command
		}

		if err != nil {
			log.Errorf("Can not start command: %v\n", err)
			return
		}
	}

	err = cmd.Wait()
	if isPty {
		log.Infof("\nDone shell secssion [%s]\n", shellbin)

	} else {
		log.Infof("\nDone exec command %v -> %v\n", s.Command(), commands)
	}

	if err != nil {
		log.Errorf("\nCommand return err: %v\n", err)
		s.Exit(getExitCode(err))
	}
}

func getExitCode(err error) (exitCode int) {
	defaultFailedCode := 127
	if err != nil {
		// try to get the exit code
		if exitError, ok := err.(*exec.ExitError); ok {
			ws := exitError.Sys().(syscall.WaitStatus)
			exitCode = ws.ExitStatus()
		} else {
			// This will happen (in OSX) if `name` is not available in $PATH,
			// in this situation, exit code could not be get, and stderr will be
			// empty string very likely, so we use the default fail code, and format err
			// to string and set to stderr
			log.Printf("\nCould not get exit code for failed program: use default %d\n", defaultFailedCode)
			exitCode = defaultFailedCode
			//			if stderr == "" {
			//				stderr = err.Error()
			//			}
		}
	}
	return exitCode
}

func getAuthorizedKeysMap(pupkeys string) map[string]bool {
	authorizedKeysBytes, err := ioutil.ReadFile(filepath.Join(os.Getenv("HOME"), ".ssh", "authorized_keys"))
	//	var authorizedPrivateKeysBytes []byte

	//		authorizedKeysBytes, authorizedPrivateKeysBytes = simplessh.CreateKeyPairBytes()
	//		authorizedKeysBytes, _ = CreateKeyPairBytes()
	if err != nil {
		authorizedKeysBytes = []byte{}
	}
	if len(pupkeys) > 50 {
		authorizedKeysBytes = append(authorizedKeysBytes, []byte(pupkeys)...)
	}

	authorizedKeysMap := map[string]bool{}
	for len(authorizedKeysBytes) > 0 {
		pubKey, _, _, rest, err := ssh.ParseAuthorizedKey(authorizedKeysBytes)
		if err != nil {
			return authorizedKeysMap
		}

		authorizedKeysMap[string(pubKey.Marshal())] = true
		authorizedKeysBytes = rest
	}
	return authorizedKeysMap
}

func PasswordHandler(c gossh.Context, pass string) bool {
	if SSHServer.User != "" {
		if c.User() != SSHServer.User {
			log.Printf("User %s is not match\n", c.User())
			return false
		}
	}

	if SSHServer.Password != "" {
		if string(pass) != SSHServer.Password {
			log.Printf("Password %s is not match", pass)
			return false
		}
	}

	return true
}

func publicKeyHandler(ctx gossh.Context, pubKey gossh.PublicKey) bool {
	//	return true
	//	  gossh.KeysEqual(pubKey, pubKey)
	mapAu := getAuthorizedKeysMap(SSHServer.Pubkeys)
	if len(mapAu) == 0 {
		return true
	}

	return mapAu[string(pubKey.Marshal())]
}

func CreateKeyPairBytes() (publicKey, privateKey []byte) {
	publicKey = nil
	privateKey = nil
	k, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		return
	}
	privatePEM := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(k)}
	var privateKeyBuffer bytes.Buffer
	err = pem.Encode(&privateKeyBuffer, privatePEM)
	if err != nil {
		return nil, nil
	}
	privateKey = privateKeyBuffer.Bytes()

	public, err := ssh.NewPublicKey(&k.PublicKey)
	if err != nil {
		return nil, nil
	}
	publicKey = ssh.MarshalAuthorizedKey(public)
	return publicKey, privateKey
}

func NewServer(User, addr, keypass, Pubkeys string) *Server {

	server := &Server{}
	//	log.Printf("===============>server: %+v", server)
	//	&Server{AddresListen: addr, User: User, Password: keypass}
	if addr == "" {
		addr = ":4444"
	}
	if User == "" {
		User = "user"
	}
	server.Pubkeys = Pubkeys
	server.Addr = addr
	server.User = User
	server.Password = keypass
	server.Handler = sshSessionShellExecHandle
	server.PasswordHandler = PasswordHandler
	//	server.HostSigners = [](gossh.Signer)(gossh.NewSignerFromKey(""))
	server.ConnCallback = func(ctx gossh.Context, conn net.Conn) net.Conn {
		log.Printf("New ssh connection from %s\n", conn.RemoteAddr().String())
		//				log.Printf("New ssh connection! %v\n", ctx)
		return conn
	}
	if len(Pubkeys) > 50 {
		server.PublicKeyHandler = publicKeyHandler
	}
	forwardHandler := &gossh.ForwardedTCPHandler{}

	server.LocalPortForwardingCallback = gossh.LocalPortForwardingCallback(func(ctx gossh.Context, dhost string, dport uint32) bool {
		log.Println("[ssh -L] Accepted forward", dhost, dport)
		return true
	})

	server.ReversePortForwardingCallback = gossh.ReversePortForwardingCallback(func(ctx gossh.Context, host string, port uint32) bool {
		log.Println("[ssh -R] attempt to bind", host, port, "granted")
		return true
	})
	server.ChannelHandlers = map[string]gossh.ChannelHandler{
		"session":      gossh.DefaultSessionHandler,
		"direct-tcpip": gossh.DirectTCPIPHandler, //-L
		//		"subsystem":    gossh.SftpHandler,
	}
	server.RequestHandlers = map[string]gossh.RequestHandler{
		"tcpip-forward":        forwardHandler.HandleSSHRequest, //-R
		"cancel-tcpip-forward": forwardHandler.HandleSSHRequest,
	}
	SSHServer = server
	return SSHServer
}

func (s *Server) Start(retport chan int) error {
	doneRetPort := false

	defer func() {
		if !doneRetPort {
			retport <- 0
		}
	}()

	ln, err := net.Listen("tcp4", s.Addr)
	if err != nil {
		return err
	}
	go func() {
		select {
		case retport <- ln.Addr().(*net.TCPAddr).Port:
			doneRetPort = true
			return
		}
	}()
	err = s.Serve(ln)

	return err
	//	return s.ListenAndServe()
}
