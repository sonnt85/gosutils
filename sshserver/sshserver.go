package sshserver

import (
	"fmt"
	gossh "github.com/gliderlabs/ssh"
	"golang.org/x/crypto/ssh"

	"github.com/kr/pty"
	//	"github.com/sonnt85/gosutils/simplessh"
	//	"golang.org/x/crypto/ssh"

	"bytes"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"

	"crypto/rand"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"unsafe"
)

// Server wraps an SSH Client
type Server struct {
	gossh.Server
	//	config                     *ssh.ServerConfig
	Pubkeys                      string
	User, Password, AddresListen string
}

var SSHServer *Server

func setWinsizeTerminal(f *os.File, w, h int) {
	syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), uintptr(syscall.TIOCSWINSZ),
		uintptr(unsafe.Pointer(&struct{ h, w, x, y uint16 }{uint16(h), uint16(w), 0, 0})))
}

//"shell", "exec":
func sshSessionShellExecHandle(s gossh.Session) {
	cmd := exec.Command(os.Getenv("SHELL"))

	ptyReq, winCh, isPty := s.Pty()
	if isPty {
		cmd.Env = append(cmd.Env, fmt.Sprintf("TERM=%s", ptyReq.Term))
		f, err := pty.Start(cmd)
		if err != nil {
			fmt.Println("Can not start cmd with tpy")
			return
		}
		go func() {
			for win := range winCh {
				setWinsizeTerminal(f, win.Width, win.Height)
			}
		}()
		go func() {
			io.Copy(f, s) // stdin
		}()
		go func() {
			io.Copy(s, f) // stdout
		}()

		cmd.Wait()

	} else {

		io.WriteString(s, "No PTY requested....\n")
		//		s.Command()
		s.Exit(1)
	}
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
			fmt.Printf("User %s is not match\n", c.User())
			return false
		}
	}

	if SSHServer.Password != "" {
		if string(pass) != SSHServer.Password {
			fmt.Printf("Password %s is not match", pass)
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
	//	fmt.Printf("===============>server: %+v", server)
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
		fmt.Printf("New ssh connection from %s\n", conn.RemoteAddr().String())
		//				fmt.Printf("New ssh connection! %v\n", ctx)
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
