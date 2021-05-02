package sshserver

import (
	//	"fmt"
	//	"encoding/hex"

	"time"

	log "github.com/sirupsen/logrus"

	gossh "github.com/gliderlabs/ssh"
	//	sw "github.com/sonnt85/gosutils/shellwords"
	"github.com/sonnt85/gosutils/sutils"
	"github.com/sonnt85/gosystem"
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
	//	defer close(winCh)
	shellbin := ""
	shellrunoption := ""
	//	os.Getenv("SHELL")
	log.Warn("permistion", s.Permissions())
	//	if len(shellbin) == 0 {
	if runtime.GOOS == "windows" {
		shellrunoption = "/c"
		shellbin = os.Getenv("COMSPEC")

		for k, v := range map[string]string{"cmd": "/c", "powershell": "-c"} {
			if _, err := exec.LookPath(k); err == nil {
				shellbin = v
				shellrunoption = v
				if len(commands) == 0 {
					commands = []string{shellbin}
				}
				break
			}
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
	//	}

	if isPty { //shell
		var f *os.File
		log.Printf("\nShell start %s[%s] ...\n", shellbin, ptyReq.Term)

		cmd = exec.Command(shellbin)
		cmd.Dir = sutils.GetHomeDir()
		cmd.Env = append(cmd.Env, "TERM="+ptyReq.Term, sutils.PathGetEnvPathKey()+"="+sutils.PathGetEnvPathValue())
		f, err := pty.Start(cmd) //start command via pty
		//		term.NewTerminal(c, prompt)
		//		term := terminal.NewTerminal("", s"")
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
			log.Infoln("Exit setWinsizeTerminal")
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

		if commands[0] == "ls" && runtime.GOOS == "windows" {
			commands = append([]string{shellbin, shellrunoption}, "dir /b", commands[3])
		}

		if commands[0] == "scmd" {
			if len(commands) > 1 {
				log.Info("Run scommand \n", commands)
				switch commands[1] {
				case "reboot":
					gosystem.Reboot(time.Second * 3)
				case "apprestart":
				case "upgrade":
				case "quit":
					os.Exit(0)
				}
			}
			return
		}

		if commands[0] == "scp" {
			if _, err := exec.LookPath(commands[0]); err != nil { //not found scp
				defer log.Warn("Exit scp server")
				log.Warn("Starting scp server ...", commands)
				scp := new(SecureCopier)
				if sutils.SlideHasElem(commands, "-r") {
					scp.IsRecursive = true
				} else {
					scp.IsRecursive = false
				}

				if sutils.SlideHasElem(commands, "-q") {
					scp.IsQuiet = true
				} else {
					scp.IsQuiet = false
				}
				scp.IsVerbose = !scp.IsQuiet
				scp.ignErr = false
				scp.inPipe = s.(io.WriteCloser)
				scp.outPipe = s.(io.ReadCloser)
				if sutils.SlideHasElem(commands, "-t") {
					scp.dstFile = commands[len(commands)-1]
					if err := scpFromClient(scp); err != nil {
						log.Error("Error scpFromClient", err)
					}
					return
				}
				if sutils.SlideHasElem(commands, "-f") {
					scp.srcFile = commands[len(commands)-1]
					if err := scpToClient(scp); err != nil {

					}
					return
				}
				return
			}
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

func DefaultChannelHandlers(srv *gossh.Server, conn *ssh.ServerConn, newChan ssh.NewChannel, ctx gossh.Context) {
	log.Info("Default channel handlers ")

	//	_, _, err := newChan.Accept()
	//	if err != nil {
	// TODO: trigger event callback
	//		return
	//	}
	//	sess := &gossh.session{
	//		Channel: ch,
	//	}

	//	sess.handleRequests(reqs)
	return
}

func DefaultRequestHandlers(ctx gossh.Context, srv *gossh.Server, req *ssh.Request) (bool, []byte) {
	log.Info("Default request handlers ", req.Type)

	if req.Type == "keepalive@openssh.com" {
		log.Info("Client send keepalive@openssh.com")
		return true, nil
	}
	return false, []byte{}
}

func NewServer(User, addr, keypass, Pubkeys string, timeouts ...time.Duration) *Server {
	timeout := time.Second * 60
	server := &Server{}
	if len(timeouts) != 0 {
		timeout = timeouts[0]
	}
	server.MaxTimeout = timeout
	if len(timeouts) >= 2 {
		server.IdleTimeout = timeouts[1]
	} else {
		server.IdleTimeout = timeout >> 1
	}
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

	server.LocalPortForwardingCallback = gossh.LocalPortForwardingCallback(func(ctx gossh.Context, dhost string, dport uint32) bool {
		log.Println("[ssh -L] Accepted forward", dhost, dport)
		return true
	})

	server.ReversePortForwardingCallback = gossh.ReversePortForwardingCallback(func(ctx gossh.Context, host string, port uint32) bool {
		log.Println("[ssh -R] attempt to bind", host, port, "granted")
		return true
	})
	server.ChannelHandlers = map[string]gossh.ChannelHandler{
		"default":                  DefaultChannelHandlers,
		"session":                  gossh.DefaultSessionHandler,
		gossh.DirectForwardRequest: gossh.DirectTCPIPHandler, //-L
		//		"subsystem":    gossh.SftpHandler,
	}

	forwardHandler := &gossh.ForwardedTCPHandler{}
	server.RequestHandlers = map[string]gossh.RequestHandler{
		"default":                        DefaultRequestHandlers,
		gossh.RemoteForwardRequest:       forwardHandler.HandleSSHRequest, //-R
		gossh.CancelRemoteForwardRequest: forwardHandler.HandleSSHRequest,
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
