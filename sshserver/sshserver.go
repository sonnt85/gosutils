package sshserver

import (
	//	"fmt"
	//	"encoding/hex"

	"context"
	"fmt"
	"strings"
	"time"

	gossh "github.com/gliderlabs/ssh"
	"github.com/sirupsen/logrus"

	//	sw "github.com/sonnt85/gosutils/shellwords"
	"github.com/sonnt85/gofilepath"
	"github.com/sonnt85/gosutils/slogrus"
	"github.com/sonnt85/gosutils/sreflect"
	"github.com/sonnt85/gosutils/sregexp"
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

	// "path/filepath"
	filepath "github.com/sonnt85/gofilepath"

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
// type exitStatusReq struct {
// 	ExitStatus uint32
// }

var SSHServer *Server

// var Logger = slogrus.GetDefaultLogger()

//func setWinsizeTerminal(f *os.File, w, h int) {
//	syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), uintptr(syscall.TIOCSWINSZ),
//		uintptr(unsafe.Pointer(&struct{ h, w, x, y uint16 }{uint16(h), uint16(w), 0, 0})))
//}

//"shell", "exec":

func sshSessionShellExecHandle(s gossh.Session) {
	//	 var cmd *exec.Cmd
	debugEnable := false
	exitStatus := 0
	commands := s.Command()

	var cmd *exec.Cmd
	var err error
	ptyReq, winCh, isPty := s.Pty()
	defer func() {
		s.Exit(exitStatus)
		s.CloseWrite()
		s.Close()
	}()
	shellbin := ""
	shellrunoption := ""
	pwd := gosystem.Getwd()
	//	os.Getenv("SHELL")
	// slogrus.Warnf("permistion/path -> %v/%s", s.Permissions(), sutils.PathGetEnvPathValue())
	//	if len(shellbin) == 0 {
	TERM := "TERM"
	if runtime.GOOS == "windows" {
		shellrunoption = "/c"
		shellbin = os.Getenv("COMSPEC")
		TERM = "COMSPEC"
		if len(shellbin) == 0 {
			for k, v := range map[string]string{"cmd": "/c", "powershell": "-c"} {
				if _, err := exec.LookPath(k); err == nil {
					shellbin = k
					shellrunoption = v
					if len(commands) == 0 {
						commands = []string{shellbin}
					}
					break
				}
			}
		}
		if isPty {
			if _, err := exec.LookPath("powershell"); err == nil {
				shellbin = "powershell"
				commands = []string{shellbin}
			} else {
				s.Write([]byte(fmt.Sprintf("not suport pty, you can run with command %s\n", filepath.Base(shellbin))))
				exitStatus = 2
				// s.Exit(getExitCode(errors.New("not suport pty, you can run with command " + shellbin)))
				return
			}
		}
	} else { //linux
		shellrunoption = "-c"
		shellbin = os.Getenv("SHELL")
		shells := []string{"bash", "sh"}
		for i := 0; i < len(shells); i++ {
			if _, err := exec.LookPath(shells[i]); err == nil {
				shellbin = shells[i]
				break
			}
		}
	}
	//	}

	if isPty { //shell
		var f *os.File
		slogrus.Printf("Shell start %s[%s] ...", shellbin, ptyReq.Term)

		cmd = exec.Command(shellbin)
		cmd.Dir = pwd
		cmd.Env = append(cmd.Env, sutils.PathGetEnvPathKey()+"="+sutils.PathGetEnvPathValue())
		if runtime.GOOS != "windows" {
			cmd.Env = append(cmd.Env, TERM+"="+ptyReq.Term) //TODO, lelect terminal
		} else {
			cmd.Env = append(cmd.Env, TERM+"="+shellbin)
		}
		// else {
		// 	cmd.Env = append(cmd.Env, TERM+"="+ptyReq.Term)
		// }
		f, err := pty.Start(cmd) //start command via pty
		// term.NewTerminal(cmd, "> ")
		// term := terminal.NewTerminal(cmd, "> ")
		if err != nil {
			slogrus.Error("Swich to run command because can not start shell with pty: ", err)
			isPty = false
			shellrunoption = ""
			commands = []string{shellbin}
		} else {
			defer f.Close()
			if debugEnable {
				go sutils.TeeReadWriterOsFile(f, s.(io.ReadWriter), s.Stderr(), os.Stdout, nil)
			} else {
				go sutils.CopyReadWriters(f, s, nil)
			}

			go func() { //auto resize
				for win := range winCh {
					// pty.Setsize(f, win)
					pty.SetWinsizeTerminal(f, win.Width, win.Height)
				}
				slogrus.Info("Exit setWinsizeTerminal")
			}()
		}

	}
	if !isPty {
		if commands[0] == "command" {
			//			slogrus.Printf("commanraw:[%+v]", s.RawCommand())
			if runtime.GOOS == "windows" && len(commands) >= 2 {
				//command ls -aLdf *
				if commands[1] == "ls" && len(commands) >= 3 {
					pattern := "*"
					rootDir := pwd
					logrus.Debug("command linux: ", commands)
					var files []string
					if len(commands) >= 4 {
						if commands[3] == "/*" {
							files, err = gofilepath.GetDrives()
							if err != nil {
								logrus.Debug(err)
							}
						} else {
							commands[3] = sregexp.New("^/(.)\\*").ReplaceAllString(commands[3], "${1}/*")
							commands[3] = sregexp.New("^/(.)/").ReplaceAllString(commands[3], "${1}/")
							commands[3] = strings.Replace(commands[3], `:/`, `/`, 1) //for C:/
							commands[3] = strings.Replace(commands[3], `/`, `:/`, 1)
							logrus.Debug("command linux after: ", commands)
							pattern = filepath.Base(commands[3])
							rootDir = filepath.Dir(commands[3])
						}
					}
					logrus.Debugf("finding: '%s' '%s' ", rootDir, pattern)
					if len(files) == 0 {
						files = gofilepath.FindFilesMatchName(rootDir, pattern, 0, true, true)
					}
					filesStr := ""
					var file string
					var isDir bool
					for i := 0; i < len(files); i++ {
						isDir = sutils.PathIsDir(files[i])
						file = filepath.ToSlash(files[i])
						if isDir {
							file = file + "/"
						}

						file = strings.Replace(file, ":/", `/`, 1) + "\n"
						// slogrus.Debug(files[i], "->", file)
						filesStr += file
					}
					if len(filesStr) != 0 {
						s.Write([]byte(filesStr))
						// var n int
						// n, err = s.Write([]byte(filesStr))
						// slogrus.Print(n, err)
					}
					return
				} else if commands[1] == "pwd" { //never call
					home := filepath.ToSlash(pwd + "/")
					home = strings.Replace(home, ":/", `/`, 1)
					logrus.Debug("Sendding home dir fom command: ", home)
					s.Write([]byte(home))
					return
				}
			} else {
				commands = append([]string{shellbin, shellrunoption}, s.RawCommand())
			}
		} else if commands[0] == "pwd" && runtime.GOOS == "windows" { //user for autocomplete
			home := filepath.ToSlash(pwd + "/")
			home = strings.Replace(home, ":/", `/`, 1)
			// logrus.Debug("Sendding home dir fom: ", home)
			s.Write([]byte(home))
			return
		} else if commands[0] == "ls" && runtime.GOOS == "windows" {
			commands = []string{shellbin, shellrunoption, "dir", "/b"}
			commands = append(commands, commands[1:]...)
		} else if commands[0] == "scat" {
			if len(commands) >= 1 {
				file := filepath.FromSlashSmart(commands[1], true)
				if !sutils.PathIsFile(file) {
					file = filepath.Join(sutils.GetHomeDir(), filepath.FromSlash(commands[1]))
				}
				if bs, err := os.ReadFile(file); err == nil {
					s.Write(bs)
				} else {
					exitStatus = 2
				}
			} else {
				exitStatus = 2
			}
			return
		} else if commands[0] == "stouch" {
			if len(commands) >= 1 {
				file := commands[1]
				os.Getwd()
				sutils.TouchFile(file)
				if f, err := os.Open(file); err == nil {
					logrus.Info(f.Name())
					f.Close()
				}

			} else {
				exitStatus = 2
			}
			return
		} else if commands[0] == "scmd" {
			if len(commands) > 1 {
				slogrus.Info("Run scommand \n", commands)
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
		} else if commands[0] == "scp" {
			// if _, err := exec.LookPath(commands[0]); err != nil || runtime.GOOS == "windows" { //not found scp, use buil-in
			defer slogrus.Warn("Exit scp server")
			slogrus.Warn("Starting scp server ...", commands)
			scp := new(SecureCopier)
			if sreflect.SlideHasElem(commands, "-r") {
				scp.IsRecursive = true
			} else {
				scp.IsRecursive = false
			}

			if sreflect.SlideHasElem(commands, "-q") {
				scp.IsQuiet = true
			} else {
				scp.IsQuiet = false
			}
			scp.IsVerbose = !scp.IsQuiet
			scp.ignErr = false
			scp.inPipe = s.(io.WriteCloser)
			scp.outPipe = s.(io.ReadCloser)
			if sreflect.SlideHasElem(commands, "-t") {
				scp.dstFile = filepath.FromSlashSmart(commands[len(commands)-1], true)
				if err := scpFromClient(scp); err != nil {
					slogrus.Error("Error scpFromClient: ", err)
					// s.Stderr().Write([]byte(fmt.Sprintf("error scpFromClient: %s\n", err)))
					exitStatus = 2
				}
				return
			}
			if sreflect.SlideHasElem(commands, "-f") {
				scp.srcFile = filepath.FromSlashSmart(commands[len(commands)-1], true)
				if err := scpToClient(scp); err != nil {
					slogrus.Error("Error scpToClient: ", err)
					// s.Stderr().Write([]byte(fmt.Sprintf("error scpToClient: %s\n", err)))
					exitStatus = 2
				}
				return
			}
			return
			// }
		} else if commands[0] == "rsync" {
			if _, err := exec.LookPath(commands[0]); err != nil || runtime.GOOS == "windows" { //not found scp, use buil-in
				// if stats, err := rsyncssh.Rsyncssh(commands, s, s, s.Stderr()); err != nil {
				// 	slogrus.Error("Error rsync: ", err)
				// 	exitStatus = 2
				// } else {
				// 	slogrus.Debugf("Total read: %s bytes, Total writeten: %d bytes, Total size of files: %d", stats.Read, stats.Written, stats.Size)
				// }
				exitStatus = 2
				return
			}
		} else {
			if _, err := exec.LookPath(commands[0]); err != nil {
				slogrus.Debug("Run build-in command via shell")
				commands = append([]string{shellbin, shellrunoption}, commands...)
			}
		}

		slogrus.Infof("exec start: %v", commands)
		cmd = exec.Command(commands[0], commands[1:]...)
		cmd.Env = append(cmd.Env, sutils.PathGetEnvPathKey()+"="+sutils.PathGetEnvPathValue())

		cmd.Dir = pwd

		if debugEnable {
			if false { //use pty for any command
				cmd.Env = append(cmd.Env, "TERM=xterm", sutils.PathGetEnvPathKey()+"="+sutils.PathGetEnvPathValue())

				f, err := pty.Start(cmd) //start command via pty
				if err != nil {
					slogrus.Error("Can not start shell with tpy: ", err)
					exitStatus = 2
					return
				}
				defer f.Close()

				go func() { //auto resize
					for win := range winCh {
						pty.SetWinsizeTerminal(f, win.Width, win.Height)
					}
					slogrus.Info("Exit setWinsizeTerminal")
				}()
				go sutils.TeeReadWriterOsFile(f, s.(io.ReadWriter), s.Stderr(), os.Stdout, nil)
			} else {
				if nil != sutils.TeeReadWriterCmd(cmd, s.(io.ReadWriter), s.Stderr(), os.Stdout, nil) { //alredy gorountine {
					slogrus.Errorf("Can not start TeeReadWriterCmd: %v\n", err)
					exitStatus = 2
					return
				}
				err = cmd.Start() //start command
			}
		} else {
			// scp.inPipe = s.(io.WriteCloser)
			// scp.outPipe = s.(io.ReadCloser)
			cmd.Stderr = s.Stderr()
			cmd.Stdout = s
			// cmd.Stdin = s
			var inputWriter io.WriteCloser
			inputWriter, err = cmd.StdinPipe()
			if err != nil {
				exitStatus = 2
				return
			}
			err = cmd.Start() //start command
			if err == nil {
				go func() {
					io.Copy(inputWriter, s)
					inputWriter.Close()
					// logrus.Debug("Close inputWriter")
				}()
			}
		}

		if err != nil {
			slogrus.Errorf("Can not start command: %v", err)
			exitStatus = 2
			return
		}
	}

	err = cmd.Wait()
	if isPty {
		slogrus.Infof("Done shell secssion %v -> %v", s.Command(), commands)
	} else {
		slogrus.Infof("Done exec command %v -> %v", s.Command(), commands)
	}

	if err != nil {
		slogrus.Errorf("Command return err: %v", err)
		exitStatus = getExitCode(err)
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
			slogrus.Printf("Could not get exit code for failed program: use default %d", defaultFailedCode)
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
			slogrus.Printf("User %s is not match\n", c.User())
			return false
		}
	}

	if SSHServer.Password != "" {
		if string(pass) != SSHServer.Password {
			slogrus.Printf("Password %s is not match", pass)
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
	slogrus.Info("Default channel handlers ")

	//	_, _, err := newChan.Accept()
	//	if err != nil {
	// TODO: trigger event callback
	//		return
	//	}
	//	sess := &gossh.session{
	//		Channel: ch,
	//	}

	//	sess.handleRequests(reqs)
	// return
}

func DefaultRequestHandlers(ctx gossh.Context, srv *gossh.Server, req *ssh.Request) (bool, []byte) {
	slogrus.Info("Default request handlers ", req.Type)

	if req.Type == "keepalive@openssh.com" {
		slogrus.Info("Client send keepalive@openssh.com")
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
	//	slogrus.Printf("===============>server: %+v", server)
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
		slogrus.Printf("New ssh connection from %s\n", conn.RemoteAddr().String())
		//				slogrus.Printf("New ssh connection! %v\n", ctx)
		return conn
	}
	if len(Pubkeys) > 50 {
		server.PublicKeyHandler = publicKeyHandler
	}

	server.ConnectionFailedCallback = gossh.ConnectionFailedCallback(func(conn net.Conn, err error) {
		slogrus.Print("ConnectionFailedCallback ", err)
	})

	server.LocalPortForwardingCallback = gossh.LocalPortForwardingCallback(func(ctx gossh.Context, dhost string, dport uint32) bool {
		slogrus.Print("[ssh -L] Accepted forward", dhost, dport)
		return true
	})

	server.ReversePortForwardingCallback = gossh.ReversePortForwardingCallback(func(ctx gossh.Context, host string, port uint32) bool {
		slogrus.Print("[ssh -R] attempt to bind", host, port, "granted")
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
	ln, err := net.Listen("tcp4", s.Addr)
	if err != nil {
		return err
	}
	ctx, canFunc := context.WithTimeout(context.Background(), time.Second*30)
	go func() {
		select {
		case retport <- ln.Addr().(*net.TCPAddr).Port:
		case <-ctx.Done():
			retport <- -1
		}
		canFunc()
	}()
	err = s.Serve(ln)
	return err
	//	return s.ListenAndServe()
}
